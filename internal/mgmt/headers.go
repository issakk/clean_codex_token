package mgmt

import "clean_codex_token/internal/model"

func MgmtHeaders(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/json",
	}
}

func GetItemType(item model.AuthFile) string {
	if v, _ := item["type"].(string); v != "" {
		return v
	}
	v, _ := item["typo"].(string)
	return v
}

func ExtractChatgptAccountID(item model.AuthFile) string {
	for _, key := range []string{"chatgpt_account_id", "chatgptAccountId", "account_id", "accountId"} {
		if v, _ := item[key].(string); v != "" {
			return v
		}
	}
	return ""
}

func BuildProbePayload(authIndex, userAgent, chatgptAccountID string) map[string]any {
	callHeader := map[string]any{
		"Authorization": "Bearer $TOKEN$",
		"Content-Type":  "application/json",
		"User-Agent":    userAgent,
	}
	if chatgptAccountID != "" {
		callHeader["Chatgpt-Account-Id"] = chatgptAccountID
	}

	return map[string]any{
		"authIndex": authIndex,
		"method":    "GET",
		"url":       "https://chatgpt.com/backend-api/wham/usage",
		"header":    callHeader,
	}
}
