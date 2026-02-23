package mgmt

import (
	"reflect"
	"testing"

	"clean_codex_token/internal/model"
)

func TestHelpers(t *testing.T) {
	item := model.AuthFile{
		"typo":               "codex",
		"chatgpt_account_id": "cid",
	}
	if GetItemType(item) != "codex" {
		t.Fatalf("expected codex")
	}
	if ExtractChatgptAccountID(item) != "cid" {
		t.Fatalf("expected cid")
	}
	p := BuildProbePayload("idx", "ua", "cid")
	h := p["header"].(map[string]any)
	if h["Chatgpt-Account-Id"] != "cid" {
		t.Fatalf("missing chatgpt id in header")
	}
	if !reflect.DeepEqual(MgmtHeaders("t"), map[string]string{"Authorization": "Bearer t", "Accept": "application/json"}) {
		t.Fatalf("headers mismatch")
	}
}
