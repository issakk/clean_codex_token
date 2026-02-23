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

func (s *Service) Run(ctx context.Context, opts *model.Options, progress func(string)) ([]model.ProbeResult, error) {
	files, err := s.Client.FetchAuthFiles(ctx)
	if err != nil {
		return nil, err
	}

	candidates := make([]model.AuthFile, 0)
	for _, f := range files {
		if strings.ToLower(mgmt.GetItemType(f)) != strings.ToLower(opts.TargetType) {
			continue
		}
		if opts.Provider != "" {
			p, _ := f["provider"].(string)
			if strings.ToLower(p) != strings.ToLower(opts.Provider) {
				continue
			}
		}
		candidates = append(candidates, f)
	}

	progress(fmt.Sprintf("总账号数: %d", len(files)))
	progress(fmt.Sprintf("符合过滤条件账号数: %d", len(candidates)))
	progress(fmt.Sprintf("异步检测并发: workers=%d, timeout=%ds, retries=%d", opts.Workers, opts.Timeout, opts.Retries))

	if len(candidates) == 0 {
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

	taskCh := make(chan model.AuthFile)
	resultCh := make(chan model.ProbeResult, len(candidates))
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range taskCh {
				resultCh <- s.probeOneWithRetry(ctx, item, opts)
			}
		}()
	}

	go func() {
		for _, c := range candidates {
			taskCh <- c
		}
		close(taskCh)
		wg.Wait()
		close(resultCh)
	}()

	results := make([]model.ProbeResult, 0, len(candidates))
	done := 0
	nextReport := 100
	total := len(candidates)
	for r := range resultCh {
		results = append(results, r)
		done++
		if done >= nextReport || done == total {
			progress(fmt.Sprintf("检测进度: %d/%d", done, total))
			nextReport += 100
		}
	}

	invalid := make([]model.ProbeResult, 0)
	failed := 0
	for _, r := range results {
		if r.Invalid401 {
			invalid = append(invalid, r)
		}
		if r.Error != "" {
			failed++
		}
	}

	sort.Slice(invalid, func(i, j int) bool { return invalid[i].Name < invalid[j].Name })
	progress(fmt.Sprintf("探测完成: 401失效=%d，探测异常=%d", len(invalid), failed))
	for _, r := range invalid {
		progress(fmt.Sprintf("[401] %s | account=%s | auth_index=%s", r.Name, r.Account, r.AuthIndex))
	}

	if err := writeJSON(opts.Output, invalid); err != nil {
		return nil, err
	}
	progress(fmt.Sprintf("已导出: %s", opts.Output))
	return invalid, nil
}

func (s *Service) probeOneWithRetry(ctx context.Context, item model.AuthFile, opts *model.Options) model.ProbeResult {
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
				return result
			}
			continue
		}
		sc, ok := asInt(data["status_code"])
		if !ok {
			result.StatusCode = nil
			result.Invalid401 = false
			result.Error = "missing status_code in api-call response"
			return result
		}
		result.StatusCode = &sc
		result.Invalid401 = sc == 401
		result.Error = ""
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
