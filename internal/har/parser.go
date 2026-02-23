package har

import (
	"encoding/json"
	"os"
	"strings"

	"clean_codex_token/internal/model"
)

func LoadContextFromHAR(harPath string) (*model.HarContext, error) {
	b, err := os.ReadFile(harPath)
	if err != nil {
		return nil, err
	}

	var har map[string]any
	if err := json.Unmarshal(b, &har); err != nil {
		return nil, err
	}

	ctx := &model.HarContext{}
	logObj, _ := har["log"].(map[string]any)
	entries, _ := logObj["entries"].([]any)
	for _, entryAny := range entries {
		entry, _ := entryAny.(map[string]any)
		req, _ := entry["request"].(map[string]any)
		url, _ := req["url"].(string)
		method, _ := req["method"].(string)
		method = strings.ToUpper(method)
		headers := headersToDict(req["headers"])

		auth := headers["authorization"]
		if ctx.Token == "" && strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			ctx.Token = strings.TrimSpace(strings.SplitN(auth, " ", 2)[1])
		}

		if ctx.BaseURL == "" && url != "" {
			if base := parseBaseURL(url); base != "" {
				ctx.BaseURL = base
			}
		}

		if ctx.UserAgent == "" {
			ctx.UserAgent = headers["user-agent"]
		}
		if ctx.ChatgptAccountID == "" {
			ctx.ChatgptAccountID = headers["chatgpt-account-id"]
		}

		if strings.Contains(url, "/v0/management/api-call") && method == "POST" {
			postData, _ := req["postData"].(map[string]any)
			postText, _ := postData["text"].(string)
			if postText != "" {
				var payload map[string]any
				if json.Unmarshal([]byte(postText), &payload) == nil {
					hdr, _ := payload["header"].(map[string]any)
					if ctx.ChatgptAccountID == "" {
						if v, _ := hdr["Chatgpt-Account-Id"].(string); v != "" {
							ctx.ChatgptAccountID = v
						}
					}
					if ctx.UserAgent == "" {
						if v, _ := hdr["User-Agent"].(string); v != "" {
							ctx.UserAgent = v
						}
					}
				}
			}
		}
	}

	return ctx, nil
}

func headersToDict(headersAny any) map[string]string {
	result := map[string]string{}
	headers, _ := headersAny.([]any)
	for _, hAny := range headers {
		h, _ := hAny.(map[string]any)
		name, _ := h["name"].(string)
		value, ok := h["value"]
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" || !ok {
			continue
		}
		if _, exists := result[key]; exists {
			continue
		}
		result[key] = toString(value)
	}
	return result
}

func parseBaseURL(url string) string {
	idx := strings.Index(url, "://")
	if idx <= 0 {
		return ""
	}
	rest := url[idx+3:]
	slash := strings.Index(rest, "/")
	if slash < 0 {
		return url
	}
	return url[:idx+3+slash]
}

func toString(v any) string {
	s, _ := v.(string)
	return s
}
