package app

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"clean_codex_token/internal/cli"
	"clean_codex_token/internal/config"
	"clean_codex_token/internal/deleter"
	"clean_codex_token/internal/har"
	"clean_codex_token/internal/mgmt"
	"clean_codex_token/internal/model"
	"clean_codex_token/internal/output"
	"clean_codex_token/internal/probe"
)

func Run(args []string, in io.Reader, out io.Writer, errOut io.Writer) int {
	opts := cli.ParseFlags(args)

	conf, err := config.LoadConfigJSON(opts.ConfigPath)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "错误: %v\n", err)
		return 1
	}

	var harCtx *model.HarContext
	if opts.HarPath != "" {
		ctx, e := har.LoadContextFromHAR(opts.HarPath)
		if e != nil {
			_, _ = fmt.Fprintf(errOut, "错误: 解析 HAR 失败: %v\n", e)
			return 1
		}
		harCtx = ctx
	}

	cli.MergeOptions(opts, conf, harCtx)
	if opts.Token == "" {
		if opts.Cron != "" {
			_, _ = fmt.Fprintln(errOut, "错误: cron 无人值守模式下缺少管理 token。请提供 --har（从抓包提取）或 --token/MGMT_TOKEN。")
			return 1
		}
		opts.Token = cli.PromptToken(in, out)
	}
	if opts.Token == "" {
		_, _ = fmt.Fprintln(errOut, "错误: 缺少管理 token。请提供 --har（从抓包提取）或 --token/MGMT_TOKEN。")
		return 1
	}

	ctx := context.Background()
	client := mgmt.NewClient(opts.BaseURL, opts.Token, opts.Timeout)
	probeSvc := probe.NewService(client)
	deleteSvc := deleter.NewService(client)
	progress := func(s string) { _, _ = fmt.Fprintln(out, s) }

	if opts.Cron != "" {
		schedule, e := parseCron5(opts.Cron)
		if e != nil {
			_, _ = fmt.Fprintf(errOut, "错误: cron 表达式不合法: %v\n", e)
			return 1
		}
		opts.Delete = true
		opts.Yes = true
		_, _ = fmt.Fprintf(out, "已启用无人值守 cron 模式: %s\n", opts.Cron)
		_, _ = fmt.Fprintln(out, "模式固定为：检查401并自动删除（跳过确认）")
		return runCronLoop(ctx, schedule, opts, probeSvc, deleteSvc, out, errOut)
	}

	if !opts.Delete && !opts.DeleteFromOutput {
		mode := cli.ChooseModeInteractive(in, out)
		if mode == "exit" {
			_, _ = fmt.Fprintln(out, "已退出。")
			return 0
		}
		opts.Workers = cli.PromptInt(in, out, "请输入检测并发 workers", opts.Workers, 1)
		opts.DeleteWorkers = cli.PromptInt(in, out, "请输入删除并发 delete-workers", opts.DeleteWorkers, 1)
		opts.Timeout = cli.PromptInt(in, out, "请输入请求超时 timeout(秒)", opts.Timeout, 1)
		opts.Retries = cli.PromptInt(in, out, "请输入失败重试 retries", opts.Retries, 0)

		switch mode {
		case "check":
			if _, err := probeSvc.Run(ctx, opts, progress); err != nil {
				_, _ = fmt.Fprintf(errOut, "错误: %v\n", err)
				return 1
			}
			return 0
		case "check_delete":
			if err := runCheckDeleteOnce(ctx, opts, probeSvc, deleteSvc, in, out, progress); err != nil {
				_, _ = fmt.Fprintf(errOut, "错误: %v\n", err)
				return 1
			}
			return 0
		case "delete_from_output":
			names, err := output.LoadNamesFromOutput(opts.Output)
			if err != nil {
				_, _ = fmt.Fprintf(errOut, "错误: %v\n", err)
				return 1
			}
			_ = deleteSvc.Run(ctx, names, opts.DeleteWorkers, !opts.Yes, in, out, progress)
			return 0
		}
	}

	if opts.DeleteFromOutput {
		names, err := output.LoadNamesFromOutput(opts.Output)
		if err != nil {
			_, _ = fmt.Fprintf(errOut, "错误: %v\n", err)
			return 1
		}
		_ = deleteSvc.Run(ctx, names, opts.DeleteWorkers, !opts.Yes, in, out, progress)
		return 0
	}

	invalid, err := probeSvc.Run(ctx, opts, progress)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "错误: %v\n", err)
		return 1
	}
	if opts.Delete {
		names := make([]string, 0, len(invalid))
		for _, r := range invalid {
			if r.Name != "" {
				names = append(names, r.Name)
			}
		}
		_ = deleteSvc.Run(ctx, names, opts.DeleteWorkers, !opts.Yes, in, out, progress)
	} else {
		_, _ = fmt.Fprintln(out, "当前为仅检查模式。")
	}
	return 0
}

func runCheckDeleteOnce(ctx context.Context, opts *model.Options, probeSvc *probe.Service, deleteSvc *deleter.Service, in io.Reader, out io.Writer, progress func(string)) error {
	invalid, err := probeSvc.Run(ctx, opts, progress)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(invalid))
	for _, r := range invalid {
		if r.Name != "" {
			names = append(names, r.Name)
		}
	}
	_ = deleteSvc.Run(ctx, names, opts.DeleteWorkers, !opts.Yes, in, out, progress)
	return nil
}

func runCronLoop(ctx context.Context, schedule cronSchedule, opts *model.Options, probeSvc *probe.Service, deleteSvc *deleter.Service, out io.Writer, errOut io.Writer) int {
	lastKey := ""
	for {
		now := time.Now()
		if schedule.match(now) {
			key := now.Format("2006-01-02 15:04")
			if key != lastKey {
				lastKey = key
				_, _ = fmt.Fprintf(out, "[%s] 开始执行: 401检测+自动删除\n", key)
				err := runCheckDeleteOnce(ctx, opts, probeSvc, deleteSvc, strings.NewReader(""), out, func(s string) {
					_, _ = fmt.Fprintln(out, s)
				})
				if err != nil {
					_, _ = fmt.Fprintf(errOut, "[%s] 执行失败: %v\n", key, err)
				} else {
					_, _ = fmt.Fprintf(out, "[%s] 执行完成\n", key)
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
}

type cronSchedule struct {
	minute fieldMatcher
	hour   fieldMatcher
	dom    fieldMatcher
	month  fieldMatcher
	dow    fieldMatcher
}

func (c cronSchedule) match(t time.Time) bool {
	return c.minute.match(t.Minute()) &&
		c.hour.match(t.Hour()) &&
		c.dom.match(t.Day()) &&
		c.month.match(int(t.Month())) &&
		c.dow.match(int(t.Weekday()))
}

type fieldMatcher struct {
	any    bool
	values map[int]struct{}
}

func (f fieldMatcher) match(v int) bool {
	if f.any {
		return true
	}
	_, ok := f.values[v]
	return ok
}

func parseCron5(expr string) (cronSchedule, error) {
	parts := strings.Fields(strings.TrimSpace(expr))
	if len(parts) != 5 {
		return cronSchedule{}, fmt.Errorf("需要 5 段表达式，例如 */5 * * * *")
	}
	min, err := parseField(parts[0], 0, 59)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("minute: %w", err)
	}
	hour, err := parseField(parts[1], 0, 23)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("hour: %w", err)
	}
	dom, err := parseField(parts[2], 1, 31)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("day-of-month: %w", err)
	}
	month, err := parseField(parts[3], 1, 12)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("month: %w", err)
	}
	dow, err := parseField(parts[4], 0, 6)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("day-of-week: %w", err)
	}
	return cronSchedule{minute: min, hour: hour, dom: dom, month: month, dow: dow}, nil
}

func parseField(part string, min int, max int) (fieldMatcher, error) {
	if part == "*" {
		return fieldMatcher{any: true}, nil
	}
	values := make(map[int]struct{})
	segments := strings.Split(part, ",")
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			return fieldMatcher{}, fmt.Errorf("空段")
		}
		if strings.HasPrefix(seg, "*/") {
			step, err := strconv.Atoi(strings.TrimPrefix(seg, "*/"))
			if err != nil || step <= 0 {
				return fieldMatcher{}, fmt.Errorf("非法步长: %s", seg)
			}
			for i := min; i <= max; i += step {
				values[i] = struct{}{}
			}
			continue
		}
		if strings.Contains(seg, "-") {
			r := strings.SplitN(seg, "-", 2)
			if len(r) != 2 {
				return fieldMatcher{}, fmt.Errorf("非法范围: %s", seg)
			}
			start, err1 := strconv.Atoi(r[0])
			end, err2 := strconv.Atoi(r[1])
			if err1 != nil || err2 != nil || start > end || start < min || end > max {
				return fieldMatcher{}, fmt.Errorf("非法范围: %s", seg)
			}
			for i := start; i <= end; i++ {
				values[i] = struct{}{}
			}
			continue
		}
		n, err := strconv.Atoi(seg)
		if err != nil || n < min || n > max {
			return fieldMatcher{}, fmt.Errorf("非法值: %s", seg)
		}
		values[n] = struct{}{}
	}
	if len(values) == 0 {
		return fieldMatcher{}, fmt.Errorf("没有可用值")
	}
	return fieldMatcher{values: values}, nil
}
