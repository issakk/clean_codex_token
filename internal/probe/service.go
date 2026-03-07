package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"clean_codex_token/internal/mgmt"
	"clean_codex_token/internal/model"
)

type Service struct {
	Client *mgmt.Client
}

func NewService(client *mgmt.Client) *Service {
	return &Service{Client: client}
}

// errorCounter 用于统计每个 auth_index 的探测异常次数
type errorCounter struct {
	mu     sync.RWMutex
	counts map[string]int
}

func newErrorCounter() *errorCounter {
	return &errorCounter{counts: make(map[string]int)}
}

func (ec *errorCounter) Increment(authIndex string) int {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.counts[authIndex]++
	return ec.counts[authIndex]
}

func (s *Service) Run(ctx context.Context, opts *model.Options, progress func(string)) ([]model.ProbeResult, error) {
	files, err := s.Client.FetchAuthFiles(ctx)
	if err != nil {
		return nil, err
	}

	matchCandidate := func(f model.AuthFile) bool {
		if strings.ToLower(mgmt.GetItemType(f)) != strings.ToLower(opts.TargetType) {
			return false
		}
		if opts.Provider != "" {
			p, _ := f["provider"].(string)
			if strings.ToLower(p) != strings.ToLower(opts.Provider) {
				return false
			}
		}
		return true
	}

	candidateCount := 0
	for _, f := range files {
		if matchCandidate(f) {
			candidateCount++
		}
	}

	progress(fmt.Sprintf("总账号数: %d", len(files)))
	progress(fmt.Sprintf("符合过滤条件账号数: %d", candidateCount))
	progress(fmt.Sprintf("异步检测并发: workers=%d, timeout=%ds, retries=%d", opts.Workers, opts.Timeout, opts.Retries))

	if candidateCount == 0 {
		if err := writeJSON(opts.Output, []model.ProbeResult{}); err != nil {
			return nil, err
		}
		progress(fmt.Sprintf("已导出: %s", opts.Output))
		return []model.ProbeResult{}, nil
	}

	workers := opts.Workers
	if workers < 1 {
		workers = 1
	}
	if workers > candidateCount {
		workers = candidateCount
	}
	if workers > 64 {
		workers = 64
		progress("workers 过大，已自动限制为 64 以降低 CPU/内存压力")
	}

	taskCh := make(chan model.AuthFile, workers*2)
	resultCh := make(chan model.ProbeResult, workers*2)
	var wg sync.WaitGroup

	ec := newErrorCounter()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range taskCh {
				resultCh <- s.probeOneWithRetry(ctx, item, opts, ec)
			}
		}()
	}

	go func() {
		for _, f := range files {
			if matchCandidate(f) {
				taskCh <- f
			}
		}
		close(taskCh)
		wg.Wait()
		close(resultCh)
	}()

	invalid := make([]model.ProbeResult, 0)
	invalidByError := 0
	invalidByLimit := 0
	failed := 0
	done := 0
	nextReport := 100
	for r := range resultCh {
		done++
		if r.Invalid401 {
			invalid = append(invalid, r)
		}
		if r.InvalidByError {
			invalid = append(invalid, r)
			invalidByError++
		}
		if r.InvalidByLimit {
			invalid = append(invalid, r)
			invalidByLimit++
		}
		if r.Error != "" && !r.InvalidByError {
			failed++
		}
		if done >= nextReport || done == candidateCount {
			progress(fmt.Sprintf("检测进度: %d/%d", done, candidateCount))
			nextReport += 100
		}
	}

	sort.Slice(invalid, func(i, j int) bool { return invalid[i].Name < invalid[j].Name })
	progress(fmt.Sprintf("探测完成: 401失效=%d，异常10次=%d，限额为0=%d，探测异常=%d", len(invalid)-invalidByError-invalidByLimit, invalidByError, invalidByLimit, failed))
	for _, r := range invalid {
		if r.InvalidByError {
			progress(fmt.Sprintf("[ERR] %s | account=%s | auth_index=%s | error_count=%d", r.Name, r.Account, r.AuthIndex, r.ErrorCount))
		} else if r.InvalidByLimit {
			progress(fmt.Sprintf("[LIMIT] %s | account=%s | auth_index=%s | limit=0", r.Name, r.Account, r.AuthIndex))
		} else {
			progress(fmt.Sprintf("[401] %s | account=%s | auth_index=%s", r.Name, r.Account, r.AuthIndex))
		}
	}

	if err := writeJSON(opts.Output, invalid); err != nil {
		return nil, err
	}
	progress(fmt.Sprintf("已导出: %s", opts.Output))
	return invalid, nil
}

func (s *Service) probeOneWithRetry(ctx context.Context, item model.AuthFile, opts *model.Options, ec *errorCounter) model.ProbeResult {
	authIndex, _ := item["auth_index"].(string)
	name, _ := item["name"].(string)
	if name == "" {
		name, _ = item["id"].(string)
	}
	account, _ := item["account"].(string)
	if account == "" {
		account, _ = item["email"].(string)
	}

	result := model.ProbeResult{
		Name:      name,
		Account:   account,
		AuthIndex: authIndex,
		Type:      mgmt.GetItemType(item),
		Provider:  str(item["provider"]),
	}
	if authIndex == "" {
		result.Error = "missing auth_index"
		return result
	}

	chatID := mgmt.ExtractChatgptAccountID(item)
	if chatID == "" {
		chatID = opts.ChatgptAccountID
	}
	payload := mgmt.BuildProbePayload(authIndex, opts.UserAgent, chatID)

	for attempt := 0; attempt <= opts.Retries; attempt++ {
		_, data, err := s.Client.ProbeOne(ctx, payload)
		if err != nil {
			result.Error = err.Error()
			if attempt >= opts.Retries {
				// 重试耗尽，增加异常计数
				errCount := ec.Increment(authIndex)
				result.ErrorCount = errCount
				if errCount >= 10 {
					result.InvalidByError = true
				}
				return result
			}
			continue
		}
		sc, ok := asInt(data["status_code"])
		if !ok {
			result.StatusCode = nil
			result.Invalid401 = false
			result.Error = "missing status_code in api-call response"
			// 响应异常也计入错误次数
			errCount := ec.Increment(authIndex)
			result.ErrorCount = errCount
			if errCount >= 10 {
				result.InvalidByError = true
			}
			return result
		}
		result.StatusCode = &sc
		result.Invalid401 = sc == 401
		result.Error = ""

		// 检查限额是否为 0
		if isLimitZero(data) {
			result.InvalidByLimit = true
		}

		return result
	}

	return result
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	default:
		return 0, false
	}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

// isLimitZero 检查响应数据中的限额是否为 0
// 探测接口返回的数据结构: {"status_code": 200, "body": "..."}
// body 是 JSON 字符串，包含 usage 信息
func isLimitZero(data map[string]any) bool {
	bodyStr, ok := data["body"].(string)
	if !ok {
		return false
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(bodyStr), &body); err != nil {
		return false
	}

	// 检查 usage 对象中的 limit 字段
	usage, ok := body["usage"].(map[string]any)
	if !ok {
		return false
	}

	limit, ok := asInt(usage["limit"])
	if !ok {
		return false
	}

	return limit == 0
}
