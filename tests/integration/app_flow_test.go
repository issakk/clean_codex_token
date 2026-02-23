package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"clean_codex_token/internal/app"
)

func TestAppFlowInteractiveCheck(t *testing.T) {
	type delState struct{}
	_ = delState{}

	srv := newMockServer(t)
	defer srv.Close()

	dir := t.TempDir()
	outFile := filepath.Join(dir, "invalid.json")
	stdin := bytes.NewBufferString("1\n120\n20\n12\n1\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := app.Run([]string{
		"--token", "t",
		"--base-url", srv.URL(),
		"--output", outFile,
		"--target-type", "codex",
	}, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code=%d stderr=%s", code, stderr.String())
	}

	b, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(b, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 invalid rows, got %d", len(rows))
	}
	names := []string{rows[0]["name"].(string), rows[1]["name"].(string)}
	if !sort.StringsAreSorted(names) {
		t.Fatalf("names should be sorted: %+v", names)
	}
	if names[0] != "a-401" || names[1] != "c-401" {
		t.Fatalf("unexpected invalid names: %+v", names)
	}
}

func TestAppFlowCheckDelete(t *testing.T) {
	srv := newMockServer(t)
	defer srv.Close()

	dir := t.TempDir()
	outFile := filepath.Join(dir, "invalid.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := app.Run([]string{
		"--token", "t",
		"--base-url", srv.URL(),
		"--output", outFile,
		"--target-type", "codex",
		"--delete",
		"--yes",
	}, strings.NewReader(""), stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code=%d stderr=%s", code, stderr.String())
	}

	d := srv.deleteNames()
	sort.Strings(d)
	if len(d) != 2 || d[0] != "a-401" || d[1] != "c-401" {
		t.Fatalf("unexpected deleted names: %+v", d)
	}
}

func TestAppFlowDeleteFromOutput(t *testing.T) {
	srv := newMockServer(t)
	defer srv.Close()

	dir := t.TempDir()
	outFile := filepath.Join(dir, "invalid.json")
	content := `[{"name":"a-401"},{"name":"c-401"}]`
	if err := os.WriteFile(outFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := app.Run([]string{
		"--token", "t",
		"--base-url", srv.URL(),
		"--output", outFile,
		"--delete-from-output",
		"--yes",
	}, strings.NewReader(""), stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code=%d stderr=%s", code, stderr.String())
	}

	d := srv.deleteNames()
	sort.Strings(d)
	if len(d) != 2 || d[0] != "a-401" || d[1] != "c-401" {
		t.Fatalf("unexpected deleted names: %+v", d)
	}
}

type mockServer struct {
	ts          *httptest.Server
	mu          sync.Mutex
	deleted     []string
	authIndexes map[string]int
}

func newMockServer(t *testing.T) *mockServer {
	t.Helper()
	m := &mockServer{
		authIndexes: map[string]int{
			"idx-a": 401,
			"idx-b": 200,
			"idx-c": 401,
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v0/management/auth-files", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]any{"files": []map[string]any{
				{"name": "a-401", "account": "a@test", "auth_index": "idx-a", "type": "codex", "provider": "openai"},
				{"name": "b-200", "account": "b@test", "auth_index": "idx-b", "typo": "codex", "provider": "openai"},
				{"name": "c-401", "account": "c@test", "auth_index": "idx-c", "type": "codex", "provider": "openai"},
			}})
			return
		}
		if r.Method == http.MethodDelete {
			name := r.URL.Query().Get("name")
			m.mu.Lock()
			m.deleted = append(m.deleted, name)
			m.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("/v0/management/api-call", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		authIndex, _ := payload["authIndex"].(string)
		sc, ok := m.authIndexes[authIndex]
		if !ok {
			sc = 200
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status_code": sc})
	})
	m.ts = httptest.NewServer(mux)
	return m
}

func (m *mockServer) URL() string { return m.ts.URL }
func (m *mockServer) Close()      { m.ts.Close() }

func (m *mockServer) deleteNames() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.deleted))
	copy(out, m.deleted)
	return out
}
