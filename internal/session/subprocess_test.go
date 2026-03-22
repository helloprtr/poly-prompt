// internal/session/subprocess_test.go
package session_test

import (
	"context"
	"testing"

	"github.com/helloprtr/poly-prompt/internal/session"
)

func TestRunForeground_EchoExitsZero(t *testing.T) {
	err := session.RunForeground(context.Background(), "echo", "hello")
	if err != nil {
		t.Errorf("expected nil error from echo, got: %v", err)
	}
}

func TestRunForeground_MissingBinary(t *testing.T) {
	err := session.RunForeground(context.Background(), "this-binary-does-not-exist-prtr-test")
	if err == nil {
		t.Error("expected error for missing binary")
	}
}

func TestFindBinary_ReturnsPathForEcho(t *testing.T) {
	path, err := session.FindBinary("echo")
	if err != nil {
		t.Fatalf("FindBinary(echo): %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path for echo")
	}
}

func TestFindBinary_ErrorForMissing(t *testing.T) {
	_, err := session.FindBinary("this-binary-does-not-exist-prtr-test")
	if err == nil {
		t.Error("expected error for missing binary")
	}
}

func TestModelBinaries_KnownModels(t *testing.T) {
	cases := []struct {
		model     string
		wantFirst string
	}{
		{"claude", "claude"},
		{"gemini", "gemini"},
		{"codex", "codex"},
	}
	for _, tc := range cases {
		bins := session.ModelBinaries(tc.model)
		if len(bins) == 0 {
			t.Errorf("ModelBinaries(%q): expected at least one candidate", tc.model)
		}
		if bins[0] != tc.wantFirst {
			t.Errorf("ModelBinaries(%q): first = %q, want %q", tc.model, bins[0], tc.wantFirst)
		}
	}
}
