package model

const (
	DefaultBaseURL    = "http://fnos.740110.xyz:8317/"
	DefaultUA         = "codex_cli_rs/0.76.0 (Debian 13.0.0; x86_64) WindowsTerminal"
	DefaultTimeout    = 12
	DefaultConfigPath = "config.json"
	DefaultOutput     = "invalid_codex_accounts.json"
)

type AuthFile map[string]any

type ProbeResult struct {
	Name       string `json:"name"`
	Account    string `json:"account"`
	AuthIndex  string `json:"auth_index"`
	Type       string `json:"type"`
	Provider   string `json:"provider"`
	StatusCode *int   `json:"status_code"`
	Invalid401 bool   `json:"invalid_401"`
	Error      string `json:"error"`
}

type DeleteResult struct {
	Name       string `json:"name"`
	Deleted    bool   `json:"deleted"`
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error"`
}

type Options struct {
	ConfigPath       string
	BaseURL          string
	Token            string
	HarPath          string
	TargetType       string
	Provider         string
	Workers          int
	DeleteWorkers    int
	Timeout          int
	Retries          int
	UserAgent        string
	ChatgptAccountID string
	Output           string
	Delete           bool
	DeleteFromOutput bool
	Yes              bool
}

type HarContext struct {
	Token            string
	BaseURL          string
	ChatgptAccountID string
	UserAgent        string
}
