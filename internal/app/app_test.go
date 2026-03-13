package app

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helloprtr/poly-prompt/internal/config"
	"github.com/helloprtr/poly-prompt/internal/editor"
	"github.com/helloprtr/poly-prompt/internal/history"
)

type stubTranslator struct {
	gotInput string
	output   string
	err      error
}

func (s *stubTranslator) Translate(_ context.Context, text string) (string, error) {
	s.gotInput = text
	if s.err != nil {
		return "", s.err
	}
	return s.output, nil
}

type stubClipboard struct {
	calls   int
	copied  string
	err     error
	diagErr error
}

func (s *stubClipboard) Copy(_ context.Context, text string) error {
	s.calls++
	s.copied = text
	return s.err
}

func (s *stubClipboard) Diagnose() error {
	return s.diagErr
}

type stubEditor struct {
	calls    int
	gotInput editor.Request
	output   string
	err      error
}

func (s *stubEditor) Edit(_ context.Context, req editor.Request) (string, error) {
	s.calls++
	s.gotInput = req
	if s.err != nil {
		return "", s.err
	}
	return s.output, nil
}

func testConfig() config.Config {
	return config.Config{
		DefaultTarget:         "claude",
		DefaultTemplatePreset: "claude-structured",
		Targets: map[string]config.TargetConfig{
			"claude": {Family: "claude", DefaultTemplatePreset: "claude-structured"},
			"codex":  {Family: "codex", DefaultTemplatePreset: "codex-implement"},
			"gemini": {Family: "gemini", DefaultTemplatePreset: "gemini-stepwise"},
		},
		TemplatePresets: map[string]config.TemplatePresetConfig{
			"claude-structured": {Template: "<role>\n{{role}}\n</role>\n<input_prompt>\n{{prompt}}\n</input_prompt>"},
			"claude-review":     {Template: "<task>\nReview carefully.\n</task>\n<input_prompt>\n{{prompt}}\n</input_prompt>"},
			"codex-implement":   {Template: "// Target: {{target}}\n\n{{role}}\n\n{{prompt}}"},
			"gemini-stepwise":   {Template: "{{role}}\n\nUser Request:\n{{prompt}}"},
		},
		Roles: map[string]config.RoleConfig{
			"be": {Prompt: "Expert Backend Engineer & Tech Lead"},
			"ui": {Prompt: "Expert Product Designer & UI Systems Specialist"},
		},
		Profiles: map[string]config.ProfileConfig{
			"backend_review": {Target: "claude", Role: "be", TemplatePreset: "claude-review"},
		},
		Shortcuts: map[string]config.ShortcutConfig{
			"ask":    {Target: "claude", TemplatePreset: "claude-structured"},
			"review": {Target: "claude", Role: "be", TemplatePreset: "claude-review"},
			"fix":    {Target: "codex", Role: "be", TemplatePreset: "codex-implement"},
			"design": {Target: "gemini", Role: "ui", TemplatePreset: "gemini-stepwise"},
		},
	}
}

func newTestApp(t *testing.T, cfg config.Config, translator *stubTranslator, clipboard *stubClipboard, ed *stubEditor, historyStore *history.Store) *App {
	t.Helper()
	return New(Dependencies{
		Version:    "test",
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Translator: translator,
		Clipboard:  clipboard,
		Editor:     ed,
		ConfigLoader: func() (config.Config, error) {
			return cfg, nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: historyStore,
	})
}

func buffersFromApp(app *App) (*bytes.Buffer, *bytes.Buffer) {
	return app.stdout.(*bytes.Buffer), app.stderr.(*bytes.Buffer)
}

func TestExecuteRendersPresetAndSkipsClipboard(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Translated prompt"}
	app := newTestApp(t, testConfig(), translator, &stubClipboard{}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))
	stdout, stderr := buffersFromApp(app)

	err := app.Execute(context.Background(), []string{"--template", "claude-structured", "-r", "be", "--no-copy", "원문"}, strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "<role>\nExpert Backend Engineer & Tech Lead\n</role>") {
		t.Fatalf("stdout = %q", got)
	}
	if !strings.Contains(got, "<input_prompt>\nTranslated prompt\n</input_prompt>") {
		t.Fatalf("stdout = %q", got)
	}
	if !strings.Contains(stderr.String(), "clipboard skipped") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteShortcutUsesShortcutDefaults(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Translated prompt"}
	app := newTestApp(t, testConfig(), translator, &stubClipboard{}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(), []string{"fix", "--no-copy", "원문"}, strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "// Target: codex") {
		t.Fatalf("stdout = %q", got)
	}
	if !strings.Contains(got, "Expert Backend Engineer & Tech Lead") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestExecuteCodexTemplateKeepsMultilineRoleReadable(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.Roles["be"] = config.RoleConfig{
		Prompt: "Expert Backend Engineer & Tech Lead.\nFocus on reliability and maintainability.",
	}

	translator := &stubTranslator{output: "Refactor this function."}
	app := newTestApp(t, cfg, translator, &stubClipboard{}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(), []string{"-t", "codex", "--template", "codex-implement", "-r", "be", "--no-copy", "원문"}, strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "// Target: codex") {
		t.Fatalf("stdout = %q", got)
	}
	if strings.Contains(got, "// Role:") {
		t.Fatalf("stdout = %q, want multiline role outside comment header", got)
	}
	if !strings.Contains(got, "Focus on reliability and maintainability.") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestExecuteJSONExplainAndDiff(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Translated prompt"}
	clipboard := &stubClipboard{}
	app := newTestApp(t, testConfig(), translator, clipboard, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))
	stdout, stderr := buffersFromApp(app)

	err := app.Execute(context.Background(), []string{"--json", "--explain", "--diff", "원문"}, strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), `"final_prompt"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"copied": true`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Resolved configuration:") || !strings.Contains(stderr.String(), "Final Prompt:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if clipboard.calls != 1 {
		t.Fatalf("clipboard calls = %d, want 1", clipboard.calls)
	}
}

func TestExecuteInteractiveUsesEditorRequestStatus(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Translated prompt"}
	editorStub := &stubEditor{output: "Edited prompt"}
	app := newTestApp(t, testConfig(), translator, &stubClipboard{}, editorStub, history.New(filepath.Join(t.TempDir(), "history.json")))
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(), []string{"-i", "--no-copy", "-r", "be", "원문"}, strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if editorStub.calls != 1 {
		t.Fatalf("editor calls = %d", editorStub.calls)
	}
	if !strings.Contains(editorStub.gotInput.Status, "Target: claude") || !strings.Contains(editorStub.gotInput.Status, "Role: be") {
		t.Fatalf("editor status = %q", editorStub.gotInput.Status)
	}
	if stdout.String() != "Edited prompt\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExecuteTemplatesCommands(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))
	stdout, _ := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"templates", "list"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("templates list error = %v", err)
	}
	if !strings.Contains(stdout.String(), "claude-structured") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	stdout.Reset()
	if err := app.Execute(context.Background(), []string{"templates", "show", "codex-implement"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("templates show error = %v", err)
	}
	if !strings.Contains(stdout.String(), "// Target: {{target}}") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExecuteProfilesCommandsAndUse(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cfg := testConfig()
	app := newTestApp(t, cfg, &stubTranslator{}, &stubClipboard{}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))
	stdout, _ := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"profiles", "list"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("profiles list error = %v", err)
	}
	if !strings.Contains(stdout.String(), "backend_review") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	stdout.Reset()
	if err := app.Execute(context.Background(), []string{"profiles", "show", "backend_review"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("profiles show error = %v", err)
	}
	if !strings.Contains(stdout.String(), "template_preset: claude-review") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	stdout.Reset()
	if err := app.Execute(context.Background(), []string{"profiles", "use", "backend_review"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("profiles use error = %v", err)
	}
	if !strings.Contains(stdout.String(), "set defaults from profile") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExecuteHistoryAndRerun(t *testing.T) {
	tempDir := t.TempDir()
	store := history.New(filepath.Join(tempDir, "history.json"))
	translator := &stubTranslator{output: "Translated prompt"}
	app := newTestApp(t, testConfig(), translator, &stubClipboard{}, &stubEditor{}, store)
	stdout, _ := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"--no-copy", "원문"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("first execute error = %v", err)
	}

	stdout.Reset()
	if err := app.Execute(context.Background(), []string{"history"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("history error = %v", err)
	}
	if !strings.Contains(stdout.String(), "target=claude") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	entries, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}

	stdout.Reset()
	translator.output = "Translated again"
	if err := app.Execute(context.Background(), []string{"rerun", entries[0].ID, "--no-copy"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("rerun error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Translated again") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExecuteSetupWritesDefaults(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cfg := testConfig()
	app := newTestApp(t, cfg, &stubTranslator{}, &stubClipboard{}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))
	stdout, _ := buffersFromApp(app)

	input := strings.NewReader("saved-key\ncodex\nbe\ncodex-implement\n")
	if err := app.Execute(context.Background(), []string{"setup"}, input, false); err != nil {
		t.Fatalf("setup error = %v", err)
	}
	if !strings.Contains(stdout.String(), "updated config") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExecuteDoctorReportsFailures(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.DefaultTemplatePreset = "missing"

	app := New(Dependencies{
		Version:    "test",
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Translator: &stubTranslator{output: "ok"},
		Clipboard:  &stubClipboard{diagErr: errors.New("clipboard unavailable")},
		Editor:     &stubEditor{},
		ConfigLoader: func() (config.Config, error) {
			return cfg, nil
		},
		ConfigInit:   func() (string, error) { return "", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: history.New(filepath.Join(t.TempDir(), "history.json")),
	})
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(), []string{"doctor"}, strings.NewReader(""), false)
	if err == nil {
		t.Fatal("doctor expected an error, got nil")
	}
	if !strings.Contains(stdout.String(), "FAIL deepl api key") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExecuteReturnsUnknownTemplateError(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, testConfig(), &stubTranslator{output: "ok"}, &stubClipboard{}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))

	err := app.Execute(context.Background(), []string{"--template", "missing", "--no-copy", "원문"}, strings.NewReader(""), false)
	if err == nil {
		t.Fatal("Execute() expected an error, got nil")
	}
	if !strings.Contains(err.Error(), `unknown template preset "missing"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestExecuteInteractiveCancelReturnsSentinelError(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, testConfig(), &stubTranslator{output: "Rendered"}, &stubClipboard{}, &stubEditor{err: editor.ErrCanceled}, history.New(filepath.Join(t.TempDir(), "history.json")))

	err := app.Execute(context.Background(), []string{"-i", "원문"}, strings.NewReader(""), false)
	if !errors.Is(err, editor.ErrCanceled) {
		t.Fatalf("Execute() error = %v", err)
	}
}
