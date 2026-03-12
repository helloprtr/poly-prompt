package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadParsesConfigFileAndMergesTargets(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	path := filepath.Join(tempDir, "prtr", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	content := `default_target = "gemini"

[targets.codex]
template = "Review this carefully:\n{{prompt}}"

[targets.custom]
template = "Custom:\n{{prompt}}"

[roles.be]
content = "Custom Backend Role"

[roles.writer]
content = "Expert Technical Writer"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DefaultTarget != "gemini" {
		t.Fatalf("DefaultTarget = %q, want %q", cfg.DefaultTarget, "gemini")
	}
	if !strings.Contains(cfg.Targets["claude"].Template, "<input_prompt>") {
		t.Fatalf("claude template = %q, want rich default template", cfg.Targets["claude"].Template)
	}
	if cfg.Targets["codex"].Template != "Review this carefully:\n{{prompt}}" {
		t.Fatalf("codex template = %q", cfg.Targets["codex"].Template)
	}
	if cfg.Targets["custom"].Template != "Custom:\n{{prompt}}" {
		t.Fatalf("custom template = %q", cfg.Targets["custom"].Template)
	}
	if cfg.Roles["be"].Content != "Custom Backend Role" {
		t.Fatalf("be role = %q", cfg.Roles["be"].Content)
	}
	if cfg.Roles["writer"].Content != "Expert Technical Writer" {
		t.Fatalf("writer role = %q", cfg.Roles["writer"].Content)
	}
	if cfg.Roles["da"].Content == "" {
		t.Fatal("expected default roles to be present")
	}
}

func TestResolveTargetPrecedence(t *testing.T) {
	t.Parallel()

	cfg := Config{DefaultTarget: "gemini"}

	if got := ResolveTarget("codex", cfg, "claude"); got != "codex" {
		t.Fatalf("ResolveTarget() with CLI override = %q, want %q", got, "codex")
	}
	if got := ResolveTarget("", cfg, "claude"); got != "gemini" {
		t.Fatalf("ResolveTarget() with config default = %q, want %q", got, "gemini")
	}
	if got := ResolveTarget("", Config{}, "claude"); got != "claude" {
		t.Fatalf("ResolveTarget() with env default = %q, want %q", got, "claude")
	}
	if got := ResolveTarget("", Config{}, ""); got != "claude" {
		t.Fatalf("ResolveTarget() fallback = %q, want %q", got, "claude")
	}
}

func TestInitCreatesStarterConfigAndRefusesOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	path, err := Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != starterConfig {
		t.Fatalf("starter config mismatch:\n%s", string(data))
	}

	secondPath, err := Init()
	if !errors.Is(err, ErrConfigExists) {
		t.Fatalf("second Init() error = %v, want %v", err, ErrConfigExists)
	}
	if secondPath != path {
		t.Fatalf("second Init() path = %q, want %q", secondPath, path)
	}
}

func TestLoadRejectsEmptyRoleContent(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	path := filepath.Join(tempDir, "prtr", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	content := "[roles.bad]\ncontent = \"   \"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected an error, got nil")
	}
	if !strings.Contains(err.Error(), `config role "bad" has empty content`) {
		t.Fatalf("Load() error = %v", err)
	}
}
