package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMergesUserAndProjectConfig(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "repo", "nested")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", tempDir)
	oldWD, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	userPath := filepath.Join(tempDir, "prtr", "config.toml")
	if err := os.MkdirAll(filepath.Dir(userPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	userContent := `deepl_api_key = "user-key"
default_target = "gemini"
default_role = "be"

[template_presets.team]
template = "Team:\n{{prompt}}"

[roles.writer]
content = "Expert Technical Writer"
`
	if err := os.WriteFile(userPath, []byte(userContent), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	projectPath := filepath.Join(tempDir, "repo", ".prtr.toml")
	projectContent := `default_target = "codex"
default_template_preset = "team"

[profiles.backend_review]
target = "claude"
role = "be"
template_preset = "claude-review"

[shortcuts.fix]
target = "codex"
role = "fe"
template_preset = "codex-implement"
`
	if err := os.WriteFile(projectPath, []byte(projectContent), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.APIKey != "user-key" {
		t.Fatalf("APIKey = %q, want %q", cfg.APIKey, "user-key")
	}
	if cfg.DefaultTarget != "codex" {
		t.Fatalf("DefaultTarget = %q, want %q", cfg.DefaultTarget, "codex")
	}
	if cfg.DefaultRole != "be" {
		t.Fatalf("DefaultRole = %q, want %q", cfg.DefaultRole, "be")
	}
	if cfg.DefaultTemplatePreset != "team" {
		t.Fatalf("DefaultTemplatePreset = %q, want %q", cfg.DefaultTemplatePreset, "team")
	}
	if cfg.DefaultTargetSource != "project config" {
		t.Fatalf("DefaultTargetSource = %q", cfg.DefaultTargetSource)
	}
	if !cfg.HasUserConfig || !cfg.HasProjectConfig {
		t.Fatalf("expected both user and project configs to be detected")
	}
	if cfg.Roles["writer"].Prompt != "Expert Technical Writer" {
		t.Fatalf("writer role = %q", cfg.Roles["writer"].Prompt)
	}
	if cfg.TemplatePresets["team"].Template != "Team:\n{{prompt}}" {
		t.Fatalf("team preset = %q", cfg.TemplatePresets["team"].Template)
	}
	if cfg.Profiles["backend_review"].TemplatePreset != "claude-review" {
		t.Fatalf("profile template preset = %q", cfg.Profiles["backend_review"].TemplatePreset)
	}
	if cfg.Shortcuts["fix"].Role != "fe" {
		t.Fatalf("shortcut role = %q", cfg.Shortcuts["fix"].Role)
	}
}

func TestResolveFunctions(t *testing.T) {
	t.Parallel()

	cfg := Config{
		APIKey:                "config-key",
		DefaultTarget:         "gemini",
		DefaultRole:           "be",
		DefaultTemplatePreset: "gemini-stepwise",
	}

	target := TargetConfig{DefaultTemplatePreset: "codex-implement"}

	if got := ResolveTarget("codex", cfg, "claude"); got != "codex" {
		t.Fatalf("ResolveTarget() = %q", got)
	}
	if got := ResolveRole("", cfg); got != "be" {
		t.Fatalf("ResolveRole() = %q", got)
	}
	if got := ResolveTemplatePreset("", cfg, target); got != "gemini-stepwise" {
		t.Fatalf("ResolveTemplatePreset() = %q", got)
	}
	if key, source := ResolveAPIKey("", cfg); key != "config-key" || source == "" {
		t.Fatalf("ResolveAPIKey() = %q, %q", key, source)
	}
}

func TestSaveDefaultsWritesConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	path, err := SaveDefaults(DefaultsUpdate{
		APIKey:                stringPtr("saved-key"),
		DefaultTarget:         stringPtr("claude"),
		DefaultRole:           stringPtr("ui"),
		DefaultTemplatePreset: stringPtr("claude-structured"),
	})
	if err != nil {
		t.Fatalf("SaveDefaults() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "deepl_api_key") || !strings.Contains(content, "saved-key") {
		t.Fatalf("config = %q", content)
	}
	if !strings.Contains(content, "default_role") || !strings.Contains(content, "ui") {
		t.Fatalf("config = %q", content)
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
	if !strings.Contains(string(data), "[template_presets.claude-structured]") {
		t.Fatalf("starter config = %q, want template presets", string(data))
	}
}

func TestLoadRejectsEmptyRolePrompt(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	path := filepath.Join(tempDir, "prtr", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	content := "[roles.bad]\nprompt = \"   \"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected an error, got nil")
	}
	if !strings.Contains(err.Error(), `config role "bad" has empty prompt`) {
		t.Fatalf("Load() error = %v", err)
	}
}

func stringPtr(value string) *string {
	v := value
	return &v
}
