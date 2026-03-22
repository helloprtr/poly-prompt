package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/helloprtr/poly-prompt/internal/history"
)

func TestSaveAndReadLastResponse(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{}, &stubEditor{},
		history.New(filepath.Join(t.TempDir(), "h.json")))
	// Override lookupEnv to return an isolated temp dir for XDG_CONFIG_HOME
	a.lookupEnv = func(key string) (string, bool) {
		if key == "XDG_CONFIG_HOME" {
			return dir, true
		}
		return "", false
	}
	a.saveLastResponse("테스트 응답 내용")

	got := a.readLastResponse()
	want := "테스트 응답 내용"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSaveLastResponse_EmptySkips(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{}, &stubEditor{},
		history.New(filepath.Join(t.TempDir(), "h.json")))
	a.lookupEnv = func(key string) (string, bool) {
		if key == "XDG_CONFIG_HOME" {
			return dir, true
		}
		return "", false
	}
	a.saveLastResponse("   ") // whitespace only

	// File should not be created
	path := filepath.Join(dir, "prtr", "last-response.json")
	if _, err := os.Stat(path); err == nil {
		t.Error("expected no file for empty/whitespace input")
	}
}
