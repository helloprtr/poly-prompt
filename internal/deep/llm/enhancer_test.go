package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/helloprtr/poly-prompt/internal/deep/schema"
)

func testBundle() schema.PatchBundle {
	return schema.PatchBundle{
		Summary:      "fix nil dereference",
		TouchedFiles: []string{"internal/foo/foo.go"},
		Diff:         "- old line\n+ new line",
		Risks: []schema.RiskItem{
			{Title: "nil panic if input empty", Severity: "high", Detail: "check before deref"},
		},
		TestPlan: schema.TestPlan{
			TestCases: []string{"TestNilInput", "TestEmptySlice"},
		},
	}
}

func TestClaudeEnhancerCallsCorrectEndpoint(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{{"type": "text", "text": "enhanced prompt"}},
		})
	}))
	defer srv.Close()

	e := &claudeEnhancer{apiKey: "test-key", baseURL: srv.URL}
	result, err := e.Enhance(context.Background(), "source", testBundle(), "rule-based")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "enhanced prompt" {
		t.Errorf("got %q, want %q", result, "enhanced prompt")
	}
	if gotPath != "/v1/messages" {
		t.Errorf("got path %q, want /v1/messages", gotPath)
	}
}

func TestClaudeEnhancerErrorOnBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	e := &claudeEnhancer{apiKey: "test-key", baseURL: srv.URL}
	_, err := e.Enhance(context.Background(), "source", testBundle(), "rule-based")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGeminiEnhancerCallsCorrectEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{
					"parts": []map[string]string{{"text": "gemini enhanced"}},
				}},
			},
		})
	}))
	defer srv.Close()

	e := &geminiEnhancer{apiKey: "test-key", baseURL: srv.URL}
	result, err := e.Enhance(context.Background(), "source", testBundle(), "rule-based")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "gemini enhanced" {
		t.Errorf("got %q, want %q", result, "gemini enhanced")
	}
}

func TestCodexEnhancerCallsCorrectEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": "codex enhanced", "role": "assistant"}},
			},
		})
	}))
	defer srv.Close()

	e := &codexEnhancer{apiKey: "test-key", baseURL: srv.URL}
	result, err := e.Enhance(context.Background(), "source", testBundle(), "rule-based")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "codex enhanced" {
		t.Errorf("got %q, want %q", result, "codex enhanced")
	}
}

func TestNewUnknownProviderReturnsError(t *testing.T) {
	_, err := New("unknown-provider", "key")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "unknown LLM provider") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestBuildEnhancePromptIncludesRuleBased(t *testing.T) {
	bundle := testBundle()
	prompt := buildEnhancePrompt("my source", bundle, "my rule-based prompt")
	if !strings.Contains(prompt, "my rule-based prompt") {
		t.Error("prompt does not contain rule-based text")
	}
	if !strings.Contains(prompt, "my source") {
		t.Error("prompt does not contain source")
	}
}
