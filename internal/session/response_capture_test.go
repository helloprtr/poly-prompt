package session_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/helloprtr/poly-prompt/internal/session"
)

func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}

func TestReadClaudeResponseFromFile(t *testing.T) {
	path := filepath.Join(testdataDir(), "claude_session.jsonl")
	got := session.ReadClaudeResponseFromFile(path)
	want := "마지막 assistant 응답입니다."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReadClaudeResponseFromFile_Empty(t *testing.T) {
	got := session.ReadClaudeResponseFromFile("/nonexistent/path.jsonl")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestReadCodexResponseFromFile(t *testing.T) {
	path := filepath.Join(testdataDir(), "codex_rollout.jsonl")
	got := session.ReadCodexResponseFromFile(path)
	want := "Codex 최종 응답입니다."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReadCodexResponseFromFile_Empty(t *testing.T) {
	got := session.ReadCodexResponseFromFile("/nonexistent/path.jsonl")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestFindLatestJSONLInDir_ReturnsNewest(t *testing.T) {
	dir := t.TempDir()

	older := filepath.Join(dir, "a.jsonl")
	newer := filepath.Join(dir, "b.jsonl")
	if err := os.WriteFile(older, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newer, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Second)
	if err := os.Chtimes(newer, future, future); err != nil {
		t.Fatal(err)
	}

	got := session.FindLatestJSONLInDir(dir)
	if got != newer {
		t.Errorf("expected newest file %q, got %q", newer, got)
	}
}

func TestFindLatestJSONLInDir_Empty(t *testing.T) {
	dir := t.TempDir()
	got := session.FindLatestJSONLInDir(dir)
	if got != "" {
		t.Errorf("expected empty string for empty dir, got %q", got)
	}
}

func TestClaudeProjectSlug(t *testing.T) {
	tests := []struct {
		cwd  string
		want string
	}{
		{"/Users/koo/dev/foo", "-Users-koo-dev-foo"},
		{"/home/user/project", "-home-user-project"},
	}
	for _, tt := range tests {
		if got := session.ClaudeProjectSlug(tt.cwd); got != tt.want {
			t.Errorf("ClaudeProjectSlug(%q) = %q, want %q", tt.cwd, got, tt.want)
		}
	}
}

func TestReadClaudeResponse_Integration(t *testing.T) {
	projectsDir := t.TempDir()
	cwd := "/Users/test/myrepo"
	slug := session.ClaudeProjectSlug(cwd)
	projectDir := filepath.Join(projectsDir, slug)
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatal(err)
	}
	jsonlPath := filepath.Join(projectDir, "session.jsonl")
	content := "{\"parentUuid\":\"x\",\"message\":{\"role\":\"assistant\",\"content\":[{\"type\":\"text\",\"text\":\"통합테스트 응답\"}]}}\n"
	if err := os.WriteFile(jsonlPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got := session.ReadClaudeResponse(projectsDir, cwd)
	want := "통합테스트 응답"
	if got != want {
		t.Errorf("ReadClaudeResponse: got %q, want %q", got, want)
	}
}

func TestReadCodexResponse_Integration(t *testing.T) {
	sessionsDir := t.TempDir()
	dayDir := filepath.Join(sessionsDir, "2026", "03", "23")
	if err := os.MkdirAll(dayDir, 0o700); err != nil {
		t.Fatal(err)
	}
	rolloutPath := filepath.Join(dayDir, "rollout-2026-03-23T10-00-00-uuid.jsonl")
	content := "{\"type\":\"event_msg\",\"payload\":{\"type\":\"task_complete\",\"last_agent_message\":\"Codex 통합테스트 응답\"}}\n"
	if err := os.WriteFile(rolloutPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got := session.ReadCodexResponse(sessionsDir)
	want := "Codex 통합테스트 응답"
	if got != want {
		t.Errorf("ReadCodexResponse: got %q, want %q", got, want)
	}
}

func TestGeminiProjectHash(t *testing.T) {
	tests := []struct {
		cwd  string
		want string
	}{
		{"/Users/koo", "0cf7ace52b89043c4b8b6f10e62e1e6ae36a7f709da5cd7d67bc9f8432b5f0bf"},
		{"/home/user/project", "9dad1e4e08b0b11cbcd860257e8bdfa6b8e5f01790e10a6a0b1f4870c13e686b"},
	}
	for _, tt := range tests {
		got := session.GeminiProjectHash(tt.cwd)
		if got != tt.want {
			t.Errorf("GeminiProjectHash(%q) = %q, want %q", tt.cwd, got, tt.want)
		}
	}
}

func TestReadGeminiResponseFromFile(t *testing.T) {
	path := filepath.Join(testdataDir(), "gemini_session.json")
	got := session.ReadGeminiResponseFromFile(path)
	want := "Gemini 최종 응답입니다."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReadGeminiResponseFromFile_Empty(t *testing.T) {
	got := session.ReadGeminiResponseFromFile("/nonexistent/path.json")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestReadGeminiResponse_Integration(t *testing.T) {
	geminiDir := t.TempDir()
	cwd := "/Users/test/myrepo"
	hash := session.GeminiProjectHash(cwd)
	chatsDir := filepath.Join(geminiDir, hash, "chats")
	if err := os.MkdirAll(chatsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	sessionPath := filepath.Join(chatsDir, "session-2026-03-24T10-00-aabbccdd.json")
	content := `{"messages":[{"type":"user","content":"q"},{"type":"gemini","content":"통합테스트 Gemini 응답"}]}`
	if err := os.WriteFile(sessionPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got := session.ReadGeminiResponse(geminiDir, cwd)
	want := "통합테스트 Gemini 응답"
	if got != want {
		t.Errorf("ReadGeminiResponse: got %q, want %q", got, want)
	}
}

func TestReadGeminiResponse_NoSessions(t *testing.T) {
	geminiDir := t.TempDir()
	got := session.ReadGeminiResponse(geminiDir, "/no/such/project")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
