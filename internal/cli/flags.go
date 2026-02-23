package cli

import (
	"flag"
	"os"

	"clean_codex_token/internal/model"
)

func ParseFlags(args []string) *model.Options {
	fs := flag.NewFlagSet("clean-codex-accounts", flag.ContinueOnError)
	opts := &model.Options{}

	fs.StringVar(&opts.ConfigPath, "config", model.DefaultConfigPath, "配置文件路径（默认: config.json）")
	fs.StringVar(&opts.BaseURL, "base-url", model.DefaultBaseURL, "")
	fs.StringVar(&opts.Token, "token", os.Getenv("MGMT_TOKEN"), "")
	fs.StringVar(&opts.HarPath, "har", "", "从浏览器导出的 HAR 自动提取 token/base-url/UA/Chatgpt-Account-Id")
	fs.StringVar(&opts.TargetType, "target-type", "codex", "按 files[].type（或 typo）过滤")
	fs.StringVar(&opts.Provider, "provider", "", "可选：再按 provider 过滤")
	fs.IntVar(&opts.Workers, "workers", 120, "并发数（401检测）")
	fs.IntVar(&opts.DeleteWorkers, "delete-workers", 20, "并发数（删除）")
	fs.IntVar(&opts.Timeout, "timeout", model.DefaultTimeout, "每次请求超时秒数")
	fs.IntVar(&opts.Retries, "retries", 1, "单账号探测失败重试次数")
	fs.StringVar(&opts.UserAgent, "user-agent", model.DefaultUA, "")
	fs.StringVar(&opts.ChatgptAccountID, "chatgpt-account-id", os.Getenv("CHATGPT_ACCOUNT_ID"), "")
	fs.StringVar(&opts.Output, "output", model.DefaultOutput, "")
	fs.BoolVar(&opts.Delete, "delete", false, "开启后执行删除")
	fs.BoolVar(&opts.DeleteFromOutput, "delete-from-output", false, "从 output 文件读取账号直接删除（跳过401检测）")
	fs.BoolVar(&opts.Yes, "yes", false, "删除时跳过二次确认")

	_ = fs.Parse(args)
	return opts
}
