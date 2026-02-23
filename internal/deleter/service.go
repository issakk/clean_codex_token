package deleter

import (
	"context"
	"fmt"
	"io"
	"sync"

	"clean_codex_token/internal/cli"
	"clean_codex_token/internal/mgmt"
	"clean_codex_token/internal/model"
)

type Service struct {
	Client *mgmt.Client
}

func NewService(client *mgmt.Client) *Service {
	return &Service{Client: client}
}

func (s *Service) Run(ctx context.Context, names []string, deleteWorkers int, needConfirm bool, in io.Reader, out io.Writer, progress func(string)) []model.DeleteResult {
	if len(names) == 0 {
		progress("没有可删除账号。")
		return nil
	}

	progress(fmt.Sprintf("待删除账号数: %d", len(names)))
	if needConfirm {
		if !cli.ConfirmDelete(in, out, len(names)) {
			progress("已取消删除。")
			return nil
		}
	}

	workers := deleteWorkers
	if workers < 1 {
		workers = 1
	}
	taskCh := make(chan string)
	resultCh := make(chan model.DeleteResult, len(names))
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for name := range taskCh {
				resultCh <- s.deleteOne(ctx, name)
			}
		}()
	}

	go func() {
		for _, n := range names {
			taskCh <- n
		}
		close(taskCh)
		wg.Wait()
		close(resultCh)
	}()

	results := make([]model.DeleteResult, 0, len(names))
	done := 0
	nextReport := 100
	for r := range resultCh {
		results = append(results, r)
		done++
		if done >= nextReport || done == len(names) {
			progress(fmt.Sprintf("删除进度: %d/%d", done, len(names)))
			nextReport += 100
		}
	}

	success := 0
	failed := make([]model.DeleteResult, 0)
	for _, r := range results {
		if r.Deleted {
			success++
		} else {
			failed = append(failed, r)
		}
	}
	progress(fmt.Sprintf("删除完成: 成功=%d，失败=%d", success, len(failed)))
	for _, r := range failed {
		progress(fmt.Sprintf("[删除失败] %s | %s", r.Name, r.Error))
	}
	return results
}

func (s *Service) deleteOne(ctx context.Context, name string) model.DeleteResult {
	if name == "" {
		return model.DeleteResult{Name: "", Deleted: false, Error: "missing name"}
	}
	status, data, text, err := s.Client.DeleteOne(ctx, name)
	if err != nil {
		return model.DeleteResult{Name: name, Deleted: false, Error: err.Error()}
	}
	ok := status == 200 && str(data["status"]) == "ok"
	errText := ""
	if !ok {
		if len(text) > 200 {
			text = text[:200]
		}
		errText = "delete failed, response=" + text
	}
	return model.DeleteResult{Name: name, Deleted: ok, StatusCode: status, Error: errText}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
