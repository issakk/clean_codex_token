package mgmt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"clean_codex_token/internal/model"
)

type Client struct {
	HTTPClient *http.Client
	BaseURL    string
	Token      string
}

func NewClient(baseURL, token string, timeoutSec int) *Client {
	t := timeoutSec
	if t < 1 {
		t = 1
	}
	return &Client{
		HTTPClient: &http.Client{Timeout: time.Duration(t) * time.Second},
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Token:      token,
	}
}

func safeJSONBytes(b []byte) map[string]any {
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func (c *Client) FetchAuthFiles(ctx context.Context) ([]model.AuthFile, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v0/management/auth-files", nil)
	for k, v := range MgmtHeaders(c.Token) {
		req.Header.Set(k, v)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("management auth-files http %d: %s", resp.StatusCode, string(body))
	}
	body, _ := io.ReadAll(resp.Body)
	data := safeJSONBytes(body)
	filesAny, _ := data["files"].([]any)
	files := make([]model.AuthFile, 0, len(filesAny))
	for _, v := range filesAny {
		m, _ := v.(map[string]any)
		files = append(files, model.AuthFile(m))
	}
	return files, nil
}

func (c *Client) ProbeOne(ctx context.Context, payload map[string]any) (int, map[string]any, error) {
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v0/management/api-call", bytes.NewReader(b))
	headers := MgmtHeaders(c.Token)
	headers["Content-Type"] = "application/json"
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	text := string(body)
	if resp.StatusCode >= 400 {
		if len(text) > 200 {
			text = text[:200]
		}
		return resp.StatusCode, nil, fmt.Errorf("management api-call http %d: %s", resp.StatusCode, text)
	}
	return resp.StatusCode, safeJSONBytes(body), nil
}

func (c *Client) DeleteOne(ctx context.Context, name string) (int, map[string]any, string, error) {
	if name == "" {
		return 0, nil, "", fmt.Errorf("missing name")
	}
	u := c.BaseURL + "/v0/management/auth-files?name=" + url.QueryEscape(name)
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	for k, v := range MgmtHeaders(c.Token) {
		req.Header.Set(k, v)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, nil, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, safeJSONBytes(body), string(body), nil
}
