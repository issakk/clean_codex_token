package cli

import (
	"testing"

	"clean_codex_token/internal/model"
)

func TestMergeOptionsPriorityRules(t *testing.T) {
	opts := &model.Options{
		BaseURL:       model.DefaultBaseURL,
		TargetType:    "codex",
		Workers:       120,
		DeleteWorkers: 20,
		Timeout:       model.DefaultTimeout,
		Retries:       1,
		UserAgent:     model.DefaultUA,
		Output:        model.DefaultOutput,
	}
	conf := map[string]any{
		"base_url":           "https://cfg.example.com",
		"token":              "cfg-token",
		"user_agent":         "cfg-ua",
		"chatgpt_account_id": "cfg-cid",
		"target_type":        "other",
		"provider":           "openai",
		"workers":            float64(88),
		"delete_workers":     float64(9),
		"timeout":            float64(30),
		"retries":            float64(2),
		"output":             "cfg.json",
	}
	har := &model.HarContext{Token: "har-token", BaseURL: "https://har.example.com", UserAgent: "har-ua", ChatgptAccountID: "har-cid"}

	MergeOptions(opts, conf, har)

	if opts.Token != "cfg-token" {
		t.Fatalf("token expected cfg-token, got %q", opts.Token)
	}
	if opts.BaseURL != "https://cfg.example.com" {
		t.Fatalf("base_url expected config value, got %q", opts.BaseURL)
	}
	if opts.UserAgent != "cfg-ua" {
		t.Fatalf("user_agent expected config value, got %q", opts.UserAgent)
	}
	if opts.ChatgptAccountID != "cfg-cid" {
		t.Fatalf("chatgpt_account_id expected config value, got %q", opts.ChatgptAccountID)
	}
	if opts.TargetType != "other" || opts.Provider != "openai" {
		t.Fatalf("target/provider merge failed: %q/%q", opts.TargetType, opts.Provider)
	}
	if opts.Workers != 88 || opts.DeleteWorkers != 9 || opts.Timeout != 30 || opts.Retries != 2 {
		t.Fatalf("int merge failed: %+v", opts)
	}
	if opts.Output != "cfg.json" {
		t.Fatalf("output merge failed: %q", opts.Output)
	}
}
