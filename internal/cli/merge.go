package cli

import (
	"strings"

	"clean_codex_token/internal/model"
)

func MergeOptions(opts *model.Options, conf map[string]any, harCtx *model.HarContext) {
	if v, ok := conf["base_url"].(string); ok && v != "" && opts.BaseURL == model.DefaultBaseURL {
		opts.BaseURL = v
	}
	if v, ok := conf["token"].(string); ok && v != "" && opts.Token == "" {
		opts.Token = v
	}
	if v, ok := conf["cpa_password"].(string); ok && v != "" && opts.Token == "" {
		opts.Token = v
	}
	if v, ok := conf["user_agent"].(string); ok && v != "" && opts.UserAgent == model.DefaultUA {
		opts.UserAgent = v
	}
	if v, ok := conf["chatgpt_account_id"].(string); ok && v != "" && opts.ChatgptAccountID == "" {
		opts.ChatgptAccountID = v
	}
	if v, ok := conf["target_type"].(string); ok && v != "" && opts.TargetType == "codex" {
		opts.TargetType = v
	}
	if v, ok := conf["provider"].(string); ok && v != "" && opts.Provider == "" {
		opts.Provider = v
	}
	if v, ok := asInt(conf["workers"]); ok && opts.Workers == 120 {
		opts.Workers = v
	}
	if v, ok := asInt(conf["delete_workers"]); ok && opts.DeleteWorkers == 20 {
		opts.DeleteWorkers = v
	}
	if v, ok := asInt(conf["timeout"]); ok && opts.Timeout == model.DefaultTimeout {
		opts.Timeout = v
	}
	if v, ok := asInt(conf["retries"]); ok && opts.Retries == 1 {
		opts.Retries = v
	}
	if v, ok := conf["output"].(string); ok && v != "" && opts.Output == model.DefaultOutput {
		opts.Output = v
	}

	if harCtx != nil {
		if opts.Token == "" && harCtx.Token != "" {
			opts.Token = harCtx.Token
		}
		if opts.BaseURL == model.DefaultBaseURL && harCtx.BaseURL != "" {
			opts.BaseURL = harCtx.BaseURL
		}
		if opts.UserAgent == model.DefaultUA && harCtx.UserAgent != "" {
			opts.UserAgent = harCtx.UserAgent
		}
		if opts.ChatgptAccountID == "" && harCtx.ChatgptAccountID != "" {
			opts.ChatgptAccountID = harCtx.ChatgptAccountID
		}
	}

	opts.BaseURL = strings.TrimRight(opts.BaseURL, "/")
	if opts.BaseURL == "" {
		opts.BaseURL = strings.TrimRight(model.DefaultBaseURL, "/")
	}
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
