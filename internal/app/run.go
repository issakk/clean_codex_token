package app

import (
	"context"
	"fmt"
	"io"

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
			invalid, err := probeSvc.Run(ctx, opts, progress)
			if err != nil {
				_, _ = fmt.Fprintf(errOut, "错误: %v\n", err)
				return 1
			}
			names := make([]string, 0, len(invalid))
			for _, r := range invalid {
				if r.Name != "" {
					names = append(names, r.Name)
				}
			}
			_ = deleteSvc.Run(ctx, names, opts.DeleteWorkers, !opts.Yes, in, out, progress)
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
