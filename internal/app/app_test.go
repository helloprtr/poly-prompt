package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/helloprtr/poly-prompt/internal/automation"
	"github.com/helloprtr/poly-prompt/internal/config"
	"github.com/helloprtr/poly-prompt/internal/editor"
	"github.com/helloprtr/poly-prompt/internal/history"
	"github.com/helloprtr/poly-prompt/internal/launcher"
	"github.com/helloprtr/poly-prompt/internal/repoctx"
	"github.com/helloprtr/poly-prompt/internal/termbook"
	"github.com/helloprtr/poly-prompt/internal/translate"
)

type stubTranslator struct {
	gotInput translate.Request
	output   string
	err      error
}

func (s *stubTranslator) Translate(_ context.Context, req translate.Request) (string, error) {
	s.gotInput = req
	if s.err != nil {
		return "", s.err
	}
	return s.output, nil
}

type stubClipboard struct {
	calls   int
	copied  string
	err     error
	read    string
	readErr error
	diagErr error
}

func (s *stubClipboard) Copy(_ context.Context, text string) error {
	s.calls++
	s.copied = text
	return s.err
}

func (s *stubClipboard) Read(_ context.Context) (string, error) {
	if s.readErr != nil {
		return "", s.readErr
	}
	return s.read, nil
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

type stubLauncher struct {
	calls   int
	gotReq  launcher.Request
	err     error
	diagErr error
	desc    string
}

func (s *stubLauncher) Launch(_ context.Context, req launcher.Request) error {
	s.calls++
	s.gotReq = req
	return s.err
}

func (s *stubLauncher) Diagnose(req launcher.Request) error {
	s.gotReq = req
	return s.diagErr
}

func (s *stubLauncher) Describe(req launcher.Request) (string, error) {
	s.gotReq = req
	if s.diagErr != nil {
		return "", s.diagErr
	}
	return s.desc, nil
}

type stubAutomator struct {
	pasteCalls  int
	submitCalls int
	gotReq      automation.Request
	pasteErr    error
	submitErr   error
	diagErr     error
	desc        string
}

func (s *stubAutomator) Diagnose(req automation.Request) error {
	s.gotReq = req
	return s.diagErr
}

func (s *stubAutomator) Describe(req automation.Request) (string, error) {
	s.gotReq = req
	if s.diagErr != nil {
		return "", s.diagErr
	}
	return s.desc, nil
}

func (s *stubAutomator) Paste(_ context.Context, req automation.Request) error {
	s.pasteCalls++
	s.gotReq = req
	return s.pasteErr
}

func (s *stubAutomator) Submit(_ context.Context, req automation.Request) error {
	s.submitCalls++
	s.gotReq = req
	return s.submitErr
}

type stubConfirmer struct {
	calls  int
	allow  bool
	err    error
	target string
}

type stubRepoContext struct {
	summary repoctx.Summary
	err     error
}

func (s *stubRepoContext) Collect(_ context.Context) (repoctx.Summary, error) {
	if s.err != nil {
		return repoctx.Summary{}, s.err
	}
	return s.summary, nil
}

func (s *stubConfirmer) ConfirmSubmit(target string) (bool, error) {
	s.calls++
	s.target = target
	return s.allow, s.err
}

func testConfig() config.Config {
	return config.Config{
		TranslationSourceLang: "auto",
		TranslationTargetLang: "en",
		DefaultTarget:         "claude",
		DefaultTemplatePreset: "claude-structured",
		Targets: map[string]config.TargetConfig{
			"claude": {Family: "claude", DefaultTemplatePreset: "claude-structured", TranslationTargetLang: "en", DefaultDelivery: "open-copy"},
			"codex":  {Family: "codex", DefaultTemplatePreset: "codex-implement", TranslationTargetLang: "en", DefaultDelivery: "open-copy"},
			"gemini": {Family: "gemini", DefaultTemplatePreset: "gemini-stepwise", TranslationTargetLang: "en", DefaultDelivery: "open-copy"},
		},
		TemplatePresets: map[string]config.TemplatePresetConfig{
			"claude-structured": {Template: "<role>\n{{role}}\n</role>\n<input_prompt>\n{{prompt}}\n</input_prompt>"},
			"claude-review":     {Template: "<task>\nReview carefully.\n</task>\n<input_prompt>\n{{prompt}}\n</input_prompt>"},
			"codex-implement":   {Template: "// Target: {{target}}\n\n{{role}}\n\n{{prompt}}"},
			"codex-review":      {Template: "// Target: {{target}}\n\n{{role}}\n\n{{prompt}}"},
			"gemini-stepwise":   {Template: "{{role}}\n\nUser Request:\n{{prompt}}"},
		},
		Roles: map[string]config.RoleConfig{
			"be": {
				Prompt: "Expert Backend Engineer & Tech Lead",
				Targets: map[string]config.RoleTargetConfig{
					"claude": {Prompt: "Claude BE", TemplatePreset: "claude-review"},
					"codex":  {Prompt: "Codex BE", TemplatePreset: "codex-implement"},
				},
			},
			"ui": {Prompt: "Expert Product Designer & UI Systems Specialist"},
			"review": {Prompt: "Expert Reviewer", Targets: map[string]config.RoleTargetConfig{
				"codex": {Prompt: "Codex Review", TemplatePreset: "codex-review"},
			}},
		},
		Profiles: map[string]config.ProfileConfig{
			"backend_review": {Target: "claude", Role: "be", TemplatePreset: "claude-review", TranslationTargetLang: "ja"},
		},
		Shortcuts: map[string]config.ShortcutConfig{
			"ask":    {Target: "claude", TemplatePreset: "claude-structured"},
			"review": {Target: "claude", Role: "be", TemplatePreset: "claude-review"},
			"fix":    {Target: "codex", Role: "be", TemplatePreset: "codex-implement"},
			"design": {Target: "gemini", Role: "ui", TemplatePreset: "gemini-stepwise"},
		},
		Launchers: map[string]config.LauncherConfig{
			"claude": {Command: "claude", PasteDelayMS: 700, SubmitMode: "manual"},
			"codex":  {Command: "codex", PasteDelayMS: 800, SubmitMode: "manual"},
			"gemini": {Command: "gemini", PasteDelayMS: 700, SubmitMode: "manual"},
		},
	}
}

func newTestApp(t *testing.T, cfg config.Config, translator *stubTranslator, clipboard *stubClipboard, ed *stubEditor, historyStore *history.Store) *App {
	t.Helper()
	return New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      translator,
		Clipboard:       clipboard,
		Editor:          ed,
		Launcher:        &stubLauncher{},
		Automator:       &stubAutomator{},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader: func() (config.Config, error) {
			return cfg, nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: historyStore,
		RepoContext:  &stubRepoContext{},
		RepoRootFinder: func() (string, error) {
			return "", termbook.ErrNotGitRepo
		},
		TermbookLoader: func(string) (termbook.Book, error) {
			return termbook.Book{}, os.ErrNotExist
		},
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
	if !strings.Contains(got, "<role>\nClaude BE\n</role>") {
		t.Fatalf("stdout = %q", got)
	}
	if !strings.Contains(got, "<input_prompt>\nTranslated prompt\n</input_prompt>") {
		t.Fatalf("stdout = %q", got)
	}
	if !strings.Contains(stderr.String(), "clipboard skipped") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteRootHelp(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))
	stdout, _ := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"--help"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Translate intent into the next AI action.") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `prtr go [mode] [message...]`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExecuteGoHelp(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))
	stdout, _ := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"go", "--help"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "`prtr go` is the fastest way to use prtr.") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "If you pipe text and also pass a message, the piped text becomes evidence.") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExecuteTakeHelp(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))
	stdout, _ := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"take", "--help"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "prtr take <action>") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "patch     Turn the answer into an implementation prompt") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExecuteLearnHelp(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))
	stdout, _ := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"learn", "--help"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "prtr learn [paths...]") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "--dry-run") {
		t.Fatalf("stdout = %q", stdout.String())
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
	if !strings.Contains(got, "Codex BE") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestExecuteGoUsesLastAppAndCompactStatus(t *testing.T) {
	t.Parallel()

	store := history.New(filepath.Join(t.TempDir(), "history.json"))
	if err := store.Append(history.Entry{
		ID:        "last",
		CreatedAt: time.Unix(200, 0).UTC(),
		Target:    "codex",
		Shortcut:  "fix",
		Original:  "older prompt",
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	translator := &stubTranslator{output: "Translated prompt"}
	clipboard := &stubClipboard{}
	editorStub := &stubEditor{}
	launchStub := &stubLauncher{}
	autoStub := &stubAutomator{}
	app := New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      translator,
		Clipboard:       clipboard,
		Editor:          editorStub,
		Launcher:        launchStub,
		Automator:       autoStub,
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader: func() (config.Config, error) {
			return testConfig(), nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: store,
	})
	stdout, stderr := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"go", "review", "원문"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "// Target: codex") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if launchStub.calls != 1 || autoStub.pasteCalls != 1 {
		t.Fatalf("launch=%d paste=%d, want 1/1", launchStub.calls, autoStub.pasteCalls)
	}
	if !strings.Contains(stderr.String(), "-> review | codex | prompt | launch+paste | auto->en") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteGoAttachesPipedInputAsEvidence(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Translated prompt"}
	app := newTestApp(t, testConfig(), translator, &stubClipboard{}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))

	if err := app.Execute(context.Background(), []string{"go", "fix", "왜 실패하는지 찾아줘", "--dry-run"}, strings.NewReader("stack trace"), true); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(translator.gotInput.Text, "왜 실패하는지 찾아줘") {
		t.Fatalf("translate input = %q", translator.gotInput.Text)
	}
	if !strings.Contains(translator.gotInput.Text, "Evidence:\nstack trace") {
		t.Fatalf("translate input = %q", translator.gotInput.Text)
	}
}

func TestExecuteGoAttachesRepoContext(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Translated prompt"}
	app := New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      translator,
		Clipboard:       &stubClipboard{},
		Editor:          &stubEditor{},
		Launcher:        &stubLauncher{},
		Automator:       &stubAutomator{},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader: func() (config.Config, error) {
			return testConfig(), nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: history.New(filepath.Join(t.TempDir(), "history.json")),
		RepoContext: &stubRepoContext{summary: repoctx.Summary{
			RepoName: "poly-prompt",
			Branch:   "main",
			Changes:  []string{"M internal/app/app.go", "?? notes.txt"},
		}},
	})
	stdout, stderr := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"go", "fix", "왜 실패하는지 찾아줘", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if translator.gotInput.Text != "왜 실패하는지 찾아줘" {
		t.Fatalf("translate input = %q", translator.gotInput.Text)
	}
	if !strings.Contains(stdout.String(), "Repo context:\nrepo: poly-prompt\nbranch: main") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "- M internal/app/app.go") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "-> fix | codex | prompt+repo | preview | auto->en") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteGoLoadsLearnedTerms(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Explain BuildPrompt and PRTR_TARGET"}
	app := New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      translator,
		Clipboard:       &stubClipboard{},
		Editor:          &stubEditor{},
		Launcher:        &stubLauncher{},
		Automator:       &stubAutomator{},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader: func() (config.Config, error) {
			return testConfig(), nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: history.New(filepath.Join(t.TempDir(), "history.json")),
		RepoContext:  &stubRepoContext{},
		RepoRootFinder: func() (string, error) {
			return "/tmp/repo", nil
		},
		TermbookLoader: func(string) (termbook.Book, error) {
			return termbook.Book{
				ProtectedTerms: []string{"BuildPrompt", "PRTR_TARGET"},
			}, nil
		},
	})
	stdout, _ := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"go", "ask", "BuildPrompt와 PRTR_TARGET를 설명해줘", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "Protected project terms: BuildPrompt, PRTR_TARGET") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExecuteGoNoContextSkipsRepoContext(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Translated prompt"}
	app := New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      translator,
		Clipboard:       &stubClipboard{},
		Editor:          &stubEditor{},
		Launcher:        &stubLauncher{},
		Automator:       &stubAutomator{},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader: func() (config.Config, error) {
			return testConfig(), nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: history.New(filepath.Join(t.TempDir(), "history.json")),
		RepoContext: &stubRepoContext{summary: repoctx.Summary{
			RepoName: "poly-prompt",
			Branch:   "main",
			Changes:  []string{"M internal/app/app.go"},
		}},
	})

	if err := app.Execute(context.Background(), []string{"go", "fix", "왜 실패하는지 찾아줘", "--dry-run", "--no-context"}, strings.NewReader("stack trace"), true); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if strings.Contains(translator.gotInput.Text, "Repo context:") {
		t.Fatalf("translate input = %q", translator.gotInput.Text)
	}
	if strings.Contains(translator.gotInput.Text, "Evidence:\nstack trace") {
		t.Fatalf("translate input = %q", translator.gotInput.Text)
	}
	if len(translator.gotInput.ProtectedTerms) != 0 {
		t.Fatalf("ProtectedTerms = %v", translator.gotInput.ProtectedTerms)
	}
}

func TestExecuteLearnDryRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("Use BuildPrompt and PRTR_TARGET.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	app := New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      &stubTranslator{},
		Clipboard:       &stubClipboard{},
		Editor:          &stubEditor{},
		Launcher:        &stubLauncher{},
		Automator:       &stubAutomator{},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader: func() (config.Config, error) {
			return testConfig(), nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: history.New(filepath.Join(t.TempDir(), "history.json")),
		RepoContext:  &stubRepoContext{},
		RepoRootFinder: func() (string, error) {
			return root, nil
		},
	})
	stdout, _ := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"learn", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "protected_terms = [") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "BuildPrompt") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".prtr", "termbook.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("termbook should not be written, stat err = %v", err)
	}
}

func TestExecuteLearnWritesTermbook(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("Use BuildPrompt and PRTR_TARGET.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	app := New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      &stubTranslator{},
		Clipboard:       &stubClipboard{},
		Editor:          &stubEditor{},
		Launcher:        &stubLauncher{},
		Automator:       &stubAutomator{},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader: func() (config.Config, error) {
			return testConfig(), nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: history.New(filepath.Join(t.TempDir(), "history.json")),
		RepoContext:  &stubRepoContext{},
		RepoRootFinder: func() (string, error) {
			return root, nil
		},
	})
	stdout, _ := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"learn"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "saved") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	book, err := termbook.Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !strings.Contains(strings.Join(book.ProtectedTerms, ","), "BuildPrompt") {
		t.Fatalf("ProtectedTerms = %v", book.ProtectedTerms)
	}
}

func TestExecuteAgainUsesLatestHistoryEntry(t *testing.T) {
	t.Parallel()

	store := history.New(filepath.Join(t.TempDir(), "history.json"))
	if err := store.Append(history.Entry{
		ID:              "last",
		CreatedAt:       time.Unix(200, 0).UTC(),
		Original:        "원문",
		Target:          "codex",
		Role:            "be",
		TemplatePreset:  "codex-implement",
		Shortcut:        "fix",
		SourceLang:      "auto",
		TargetLang:      "en",
		TranslationMode: "auto",
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	translator := &stubTranslator{output: "Translated again"}
	app := newTestApp(t, testConfig(), translator, &stubClipboard{}, &stubEditor{}, store)
	stdout, stderr := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"again", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if translator.gotInput.Text != "원문" {
		t.Fatalf("translate input = %q, want %q", translator.gotInput.Text, "원문")
	}
	if !strings.Contains(stdout.String(), "// Target: codex") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "-> fix | codex | history | preview | auto->en") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteSwapOverridesApp(t *testing.T) {
	t.Parallel()

	store := history.New(filepath.Join(t.TempDir(), "history.json"))
	if err := store.Append(history.Entry{
		ID:              "last",
		CreatedAt:       time.Unix(200, 0).UTC(),
		Original:        "원문",
		Target:          "claude",
		Role:            "be",
		TemplatePreset:  "claude-review",
		Shortcut:        "review",
		SourceLang:      "auto",
		TargetLang:      "en",
		TranslationMode: "auto",
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	translator := &stubTranslator{output: "Translated prompt"}
	app := newTestApp(t, testConfig(), translator, &stubClipboard{}, &stubEditor{}, store)
	stdout, stderr := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"swap", "gemini", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "User Request:\nTranslated prompt") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "-> review | gemini | history | preview | auto->en") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteTakeUsesLatestTargetAndSkipsTranslation(t *testing.T) {
	t.Parallel()

	store := history.New(filepath.Join(t.TempDir(), "history.json"))
	if err := store.Append(history.Entry{
		ID:        "last",
		CreatedAt: time.Unix(200, 0).UTC(),
		Target:    "codex",
		Shortcut:  "fix",
		Original:  "older prompt",
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	translator := &stubTranslator{output: "should not be used"}
	clipboard := &stubClipboard{read: "Use ripgrep in /internal/app first."}
	app := newTestApp(t, testConfig(), translator, clipboard, &stubEditor{}, store)
	stdout, stderr := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"take", "patch", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if translator.gotInput.Text != "" {
		t.Fatalf("translator should not be called, got input %q", translator.gotInput.Text)
	}
	if !strings.Contains(stdout.String(), "// Target: codex") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Turn the material below into an implementation prompt.") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Source material:\nUse ripgrep in /internal/app first.") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "-> take:patch | codex | clipboard | preview | en->en") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteTakeAllowsAppOverrideAndEdit(t *testing.T) {
	t.Parallel()

	clipboard := &stubClipboard{read: "Need a tighter summary."}
	editorStub := &stubEditor{output: "Edited take prompt"}
	app := newTestApp(t, testConfig(), &stubTranslator{}, clipboard, editorStub, history.New(filepath.Join(t.TempDir(), "history.json")))
	stdout, stderr := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"take", "summary", "--to", "gemini", "--edit", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if editorStub.calls != 1 {
		t.Fatalf("editor calls = %d, want 1", editorStub.calls)
	}
	if !strings.Contains(editorStub.gotInput.Initial, "Need a tighter summary.") {
		t.Fatalf("editor initial = %q", editorStub.gotInput.Initial)
	}
	if stdout.String() != "Edited take prompt\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "-> take:summary | gemini | clipboard | preview | en->en") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteTakeRejectsEmptyClipboard(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{read: "   "}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))

	err := app.Execute(context.Background(), []string{"take", "commit", "--dry-run"}, strings.NewReader(""), false)
	if err == nil {
		t.Fatal("Execute() expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "clipboard is empty; copy an answer and try again") {
		t.Fatalf("error = %v", err)
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

func TestExecuteInspectExplainsWithoutClipboard(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Translated prompt"}
	clipboard := &stubClipboard{}
	app := newTestApp(t, testConfig(), translator, clipboard, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))
	stdout, stderr := buffersFromApp(app)

	err := app.Execute(context.Background(), []string{"inspect", "원문"}, strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "<input_prompt>\nTranslated prompt\n</input_prompt>") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Resolved configuration:") || !strings.Contains(stderr.String(), "Final Prompt:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if clipboard.calls != 0 {
		t.Fatalf("clipboard calls = %d, want 0", clipboard.calls)
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

func TestExecuteRerunEditUsesStoredPrompt(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	store := history.New(filepath.Join(tempDir, "history.json"))
	translator := &stubTranslator{output: "Translated prompt"}
	editorStub := &stubEditor{output: "Edited final prompt"}
	app := newTestApp(t, testConfig(), translator, &stubClipboard{}, editorStub, store)
	stdout, _ := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"--no-copy", "원문"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("first execute error = %v", err)
	}
	entries, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	stdout.Reset()
	if err := app.Execute(context.Background(), []string{"rerun", entries[0].ID, "--edit", "--no-copy"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("rerun edit error = %v", err)
	}
	if stdout.String() != "Edited final prompt\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExecuteLaunchUsesLauncher(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	translator := &stubTranslator{output: "Translated prompt"}
	clipboard := &stubClipboard{}
	launchStub := &stubLauncher{}
	autoStub := &stubAutomator{}
	app := New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      translator,
		Clipboard:       clipboard,
		Editor:          &stubEditor{},
		Launcher:        launchStub,
		Automator:       autoStub,
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader: func() (config.Config, error) {
			return cfg, nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: history.New(filepath.Join(t.TempDir(), "history.json")),
	})

	if err := app.Execute(context.Background(), []string{"--launch", "원문"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if launchStub.calls != 1 {
		t.Fatalf("launcher calls = %d", launchStub.calls)
	}
	if launchStub.gotReq.Command != "claude" {
		t.Fatalf("launcher command = %q", launchStub.gotReq.Command)
	}
	if autoStub.pasteCalls != 0 {
		t.Fatalf("paste calls = %d, want 0", autoStub.pasteCalls)
	}
}

func TestExecutePasteUsesAutomator(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	translator := &stubTranslator{output: "Translated prompt"}
	clipboard := &stubClipboard{}
	launchStub := &stubLauncher{}
	autoStub := &stubAutomator{}
	app := New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      translator,
		Clipboard:       clipboard,
		Editor:          &stubEditor{},
		Launcher:        launchStub,
		Automator:       autoStub,
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader: func() (config.Config, error) {
			return cfg, nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: history.New(filepath.Join(t.TempDir(), "history.json")),
	})

	if err := app.Execute(context.Background(), []string{"--paste", "원문"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if launchStub.calls != 1 {
		t.Fatalf("launcher calls = %d", launchStub.calls)
	}
	if autoStub.pasteCalls != 1 {
		t.Fatalf("paste calls = %d", autoStub.pasteCalls)
	}
	if autoStub.gotReq.Target != "claude" {
		t.Fatalf("automation target = %q", autoStub.gotReq.Target)
	}
	if autoStub.gotReq.PasteDelay != 700*time.Millisecond {
		t.Fatalf("paste delay = %v", autoStub.gotReq.PasteDelay)
	}
}

func TestExecuteSubmitConfirmUsesConfirmerAndAutomator(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	translator := &stubTranslator{output: "Translated prompt"}
	clipboard := &stubClipboard{}
	launchStub := &stubLauncher{}
	autoStub := &stubAutomator{}
	confirmStub := &stubConfirmer{allow: true}
	app := New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      translator,
		Clipboard:       clipboard,
		Editor:          &stubEditor{},
		Launcher:        launchStub,
		Automator:       autoStub,
		SubmitConfirmer: confirmStub,
		ConfigLoader: func() (config.Config, error) {
			return cfg, nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: history.New(filepath.Join(t.TempDir(), "history.json")),
	})

	if err := app.Execute(context.Background(), []string{"--paste", "--submit", "confirm", "원문"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if confirmStub.calls != 1 {
		t.Fatalf("confirm calls = %d", confirmStub.calls)
	}
	if autoStub.submitCalls != 1 {
		t.Fatalf("submit calls = %d", autoStub.submitCalls)
	}
}

func TestExecuteSubmitAutoRejected(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, testConfig(), &stubTranslator{output: "ok"}, &stubClipboard{}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))

	err := app.Execute(context.Background(), []string{"--paste", "--submit", "auto", "원문"}, strings.NewReader(""), false)
	if err == nil {
		t.Fatal("Execute() expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "not supported yet") {
		t.Fatalf("error = %v", err)
	}
}

func TestExecuteRejectsSubmitWithoutPaste(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, testConfig(), &stubTranslator{output: "ok"}, &stubClipboard{}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))

	err := app.Execute(context.Background(), []string{"--submit", "confirm", "원문"}, strings.NewReader(""), false)
	if err == nil {
		t.Fatal("Execute() expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "--submit requires --paste") {
		t.Fatalf("error = %v", err)
	}
}

func TestExecuteHistorySearchAndFavorite(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	store := history.New(filepath.Join(tempDir, "history.json"))
	translator := &stubTranslator{output: "Translated prompt"}
	app := newTestApp(t, testConfig(), translator, &stubClipboard{}, &stubEditor{}, store)
	stdout, _ := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"--no-copy", "원문"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("first execute error = %v", err)
	}
	entries, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	stdout.Reset()
	if err := app.Execute(context.Background(), []string{"favorite", entries[0].ID}, strings.NewReader(""), false); err != nil {
		t.Fatalf("favorite error = %v", err)
	}
	if !strings.Contains(stdout.String(), "favorited") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	stdout.Reset()
	if err := app.Execute(context.Background(), []string{"history", "search", "translated"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("history search error = %v", err)
	}
	if !strings.Contains(stdout.String(), "lang=auto->en") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExecuteSetupWritesDefaults(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cfg := testConfig()
	app := newTestApp(t, cfg, &stubTranslator{}, &stubClipboard{}, &stubEditor{}, history.New(filepath.Join(t.TempDir(), "history.json")))
	stdout, _ := buffersFromApp(app)

	input := strings.NewReader("saved-key\nko\nen\ncodex\nbe\ncodex-implement\n")
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
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      &stubTranslator{output: "ok"},
		Clipboard:       &stubClipboard{diagErr: errors.New("clipboard unavailable")},
		Editor:          &stubEditor{},
		Launcher:        &stubLauncher{diagErr: errors.New("launcher unavailable")},
		Automator:       &stubAutomator{diagErr: errors.New("automation unavailable")},
		SubmitConfirmer: &stubConfirmer{},
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

func TestTruncateOneLineUnicode(t *testing.T) {
	input := "안녕하세요 세계입니다"
	result := truncateOneLine(input, 8)
	if !utf8.ValidString(result) {
		t.Fatalf("truncateOneLine produced invalid UTF-8: %q", result)
	}
	runes := []rune(result)
	if len(runes) > 8 {
		t.Fatalf("truncateOneLine exceeded rune limit: got %d runes", len(runes))
	}
}
