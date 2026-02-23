package har

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadContextFromHAR(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.har")
	content := `{
  "log": {
    "entries": [
      {
        "request": {
          "url": "https://api.example.com/v0/management/auth-files",
          "method": "GET",
          "headers": [
            {"name": "Authorization", "value": "Bearer T1"},
            {"name": "User-Agent", "value": "UA1"},
            {"name": "Chatgpt-Account-Id", "value": "CID1"}
          ]
        }
      }
    ]
  }
}`
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, err := LoadContextFromHAR(p)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Token != "T1" || ctx.UserAgent != "UA1" || ctx.ChatgptAccountID != "CID1" {
		t.Fatalf("unexpected context: %+v", ctx)
	}
	if ctx.BaseURL != "https://api.example.com" {
		t.Fatalf("unexpected base_url: %q", ctx.BaseURL)
	}
}
