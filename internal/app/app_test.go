package app

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/helloprtr/poly-prompt/internal/config"
	"github.com/helloprtr/poly-prompt/internal/editor"
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
	calls  int
	copied string
	err    error
}

func (s *stubClipboard) Copy(_ context.Context, text string) error {
	s.calls++
	s.copied = text
	return s.err
}

type stubEditor struct {
	calls    int
	gotInput string
	output   string
	err      error
}

func (s *stubEditor) Edit(_ context.Context, text string) (string, error) {
	s.calls++
	s.gotInput = text
	if s.err != nil {
		return "", s.err
	}
	return s.output, nil
}

func TestExecutePrefersArgsOverStdinAndSkipsClipboard(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Hello world"}
	clipboard := &stubClipboard{}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(Dependencies{
		Version:    "test",
		Stdout:     &stdout,
		Stderr:     &stderr,
		Translator: translator,
		Clipboard:  clipboard,
		Editor:     &stubEditor{},
		ConfigLoader: func() (config.Config, error) {
			return config.Config{
				Targets: map[string]config.TargetConfig{
					"claude": {Template: "{{prompt}}"},
				},
				Roles: map[string]config.RoleConfig{},
			}, nil
		},
		ConfigInit: func() (string, error) { return "", nil },
		LookupEnv:  func(string) (string, bool) { return "", false },
	})

	err := app.Execute(context.Background(), []string{"--no-copy", "hello", "world"}, strings.NewReader("ignored"), true)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if translator.gotInput != "hello world" {
		t.Fatalf("translator input = %q, want %q", translator.gotInput, "hello world")
	}
	if got := stdout.String(); got != "Hello world\n" {
		t.Fatalf("stdout = %q, want %q", got, "Hello world\n")
	}
	if clipboard.calls != 0 {
		t.Fatalf("clipboard calls = %d, want 0", clipboard.calls)
	}
	if !strings.Contains(stderr.String(), "clipboard skipped") {
		t.Fatalf("stderr = %q, want clipboard skipped notice", stderr.String())
	}
}

func TestExecuteReadsStdinAndCopiesRenderedPrompt(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "How do I start a Go server?"}
	clipboard := &stubClipboard{}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(Dependencies{
		Version:    "test",
		Stdout:     &stdout,
		Stderr:     &stderr,
		Translator: translator,
		Clipboard:  clipboard,
		Editor:     &stubEditor{},
		ConfigLoader: func() (config.Config, error) {
			return config.Config{
				DefaultTarget: "codex",
				Targets: map[string]config.TargetConfig{
					"codex": {Template: "Please answer in English.\n\n{{prompt}}"},
				},
				Roles: map[string]config.RoleConfig{},
			}, nil
		},
		ConfigInit: func() (string, error) { return "", nil },
		LookupEnv:  func(string) (string, bool) { return "", false },
	})

	err := app.Execute(context.Background(), nil, strings.NewReader("고 서버 시작법"), true)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := "Please answer in English.\n\nHow do I start a Go server?"
	if got := strings.TrimSuffix(stdout.String(), "\n"); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if clipboard.copied != want {
		t.Fatalf("clipboard copied = %q, want %q", clipboard.copied, want)
	}
	if !strings.Contains(stderr.String(), `target "codex"`) {
		t.Fatalf("stderr = %q, want target status", stderr.String())
	}
}

func TestExecuteReturnsUsageErrorWhenInputMissing(t *testing.T) {
	t.Parallel()

	app := New(Dependencies{
		Version:    "test",
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Translator: &stubTranslator{},
		Clipboard:  &stubClipboard{},
		ConfigLoader: func() (config.Config, error) {
			return config.Config{
				Targets: map[string]config.TargetConfig{"claude": {Template: "{{prompt}}"}},
				Roles:   map[string]config.RoleConfig{},
			}, nil
		},
		ConfigInit: func() (string, error) { return "", nil },
		LookupEnv:  func(string) (string, bool) { return "", false },
	})

	err := app.Execute(context.Background(), nil, strings.NewReader(""), false)
	if err == nil {
		t.Fatal("Execute() expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "missing prompt text") {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestExecuteReturnsUnknownTargetError(t *testing.T) {
	t.Parallel()

	app := New(Dependencies{
		Version:    "test",
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Translator: &stubTranslator{},
		Clipboard:  &stubClipboard{},
		ConfigLoader: func() (config.Config, error) {
			return config.Config{
				Targets: map[string]config.TargetConfig{
					"claude": {Template: "{{prompt}}"},
					"codex":  {Template: "{{prompt}}"},
				},
				Roles: map[string]config.RoleConfig{},
			}, nil
		},
		ConfigInit: func() (string, error) { return "", nil },
		LookupEnv:  func(string) (string, bool) { return "missing", true },
	})

	err := app.Execute(context.Background(), []string{"hello"}, strings.NewReader(""), false)
	if err == nil {
		t.Fatal("Execute() expected an error, got nil")
	}
	if !strings.Contains(err.Error(), `unknown target "missing"`) {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestExecuteVersionAndInitCommands(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	app := New(Dependencies{
		Version:    "1.2.3",
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		Translator: &stubTranslator{},
		Clipboard:  &stubClipboard{},
		ConfigLoader: func() (config.Config, error) {
			return config.Config{}, nil
		},
		ConfigInit: func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:  func(string) (string, bool) { return "", false },
	})

	if err := app.Execute(context.Background(), []string{"version"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("version Execute() error = %v", err)
	}
	if got := stdout.String(); got != "1.2.3\n" {
		t.Fatalf("version stdout = %q", got)
	}

	stdout.Reset()
	if err := app.Execute(context.Background(), []string{"init"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("init Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "/tmp/prtr/config.toml") {
		t.Fatalf("init stdout = %q", stdout.String())
	}
}

func TestExecutePropagatesClipboardErrors(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("clipboard unavailable")
	app := New(Dependencies{
		Version:    "test",
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Translator: &stubTranslator{output: "Hello"},
		Clipboard:  &stubClipboard{err: expectedErr},
		ConfigLoader: func() (config.Config, error) {
			return config.Config{
				Targets: map[string]config.TargetConfig{"claude": {Template: "{{prompt}}"}},
				Roles:   map[string]config.RoleConfig{},
			}, nil
		},
		ConfigInit: func() (string, error) { return "", nil },
		LookupEnv:  func(string) (string, bool) { return "", false },
	})

	err := app.Execute(context.Background(), []string{"hello"}, strings.NewReader(""), false)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("Execute() error = %v, want %v", err, expectedErr)
	}
}

func TestExecuteInjectsRoleIntoTemplate(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Review this service"}
	clipboard := &stubClipboard{}
	var stdout bytes.Buffer

	app := New(Dependencies{
		Version:    "test",
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		Translator: translator,
		Clipboard:  clipboard,
		Editor:     &stubEditor{},
		ConfigLoader: func() (config.Config, error) {
			return config.Config{
				Targets: map[string]config.TargetConfig{
					"claude": {Template: "<role>{{role}}</role>\n{{prompt}}"},
				},
				Roles: map[string]config.RoleConfig{
					"be": {Content: "Expert Backend Engineer & Tech Lead"},
				},
			}, nil
		},
		ConfigInit: func() (string, error) { return "", nil },
		LookupEnv:  func(string) (string, bool) { return "", false },
	})

	err := app.Execute(context.Background(), []string{"-r", "be", "--no-copy", "hello"}, strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := "<role>Expert Backend Engineer & Tech Lead</role>\nReview this service\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestExecuteReturnsUnknownRoleError(t *testing.T) {
	t.Parallel()

	app := New(Dependencies{
		Version:    "test",
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Translator: &stubTranslator{},
		Clipboard:  &stubClipboard{},
		Editor:     &stubEditor{},
		ConfigLoader: func() (config.Config, error) {
			return config.Config{
				Targets: map[string]config.TargetConfig{
					"claude": {Template: "{{prompt}}"},
				},
				Roles: map[string]config.RoleConfig{
					"be": {Content: "Expert Backend Engineer & Tech Lead"},
					"se": {Content: "Expert Security Engineer & Application Security Reviewer"},
				},
			}, nil
		},
		ConfigInit: func() (string, error) { return "", nil },
		LookupEnv:  func(string) (string, bool) { return "", false },
	})

	err := app.Execute(context.Background(), []string{"-r", "missing", "hello"}, strings.NewReader(""), false)
	if err == nil {
		t.Fatal("Execute() expected an error, got nil")
	}
	if !strings.Contains(err.Error(), `unknown role "missing"`) {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(err.Error(), "be, se") {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestExecuteInteractiveUsesEditedPromptForStdoutAndClipboard(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Initial"}
	clipboard := &stubClipboard{}
	editor := &stubEditor{output: "Edited"}
	var stdout bytes.Buffer

	app := New(Dependencies{
		Version:    "test",
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		Translator: translator,
		Clipboard:  clipboard,
		Editor:     editor,
		ConfigLoader: func() (config.Config, error) {
			return config.Config{
				Targets: map[string]config.TargetConfig{
					"claude": {Template: "{{prompt}}"},
				},
				Roles: map[string]config.RoleConfig{},
			}, nil
		},
		ConfigInit: func() (string, error) { return "", nil },
		LookupEnv:  func(string) (string, bool) { return "", false },
	})

	err := app.Execute(context.Background(), []string{"-i", "hello"}, strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if editor.calls != 1 {
		t.Fatalf("editor calls = %d, want 1", editor.calls)
	}
	if editor.gotInput != "Initial" {
		t.Fatalf("editor input = %q, want %q", editor.gotInput, "Initial")
	}
	if stdout.String() != "Edited\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "Edited\n")
	}
	if clipboard.copied != "Edited" {
		t.Fatalf("clipboard copied = %q, want %q", clipboard.copied, "Edited")
	}
}

func TestExecuteInteractiveNoCopySkipsClipboard(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Initial"}
	clipboard := &stubClipboard{}
	editor := &stubEditor{output: "Edited"}
	var stdout bytes.Buffer

	app := New(Dependencies{
		Version:    "test",
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		Translator: translator,
		Clipboard:  clipboard,
		Editor:     editor,
		ConfigLoader: func() (config.Config, error) {
			return config.Config{
				Targets: map[string]config.TargetConfig{
					"claude": {Template: "{{prompt}}"},
				},
				Roles: map[string]config.RoleConfig{},
			}, nil
		},
		ConfigInit: func() (string, error) { return "", nil },
		LookupEnv:  func(string) (string, bool) { return "", false },
	})

	err := app.Execute(context.Background(), []string{"-i", "--no-copy", "hello"}, strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stdout.String() != "Edited\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "Edited\n")
	}
	if clipboard.calls != 0 {
		t.Fatalf("clipboard calls = %d, want 0", clipboard.calls)
	}
}

func TestExecuteInteractiveCancelReturnsSentinelError(t *testing.T) {
	t.Parallel()

	app := New(Dependencies{
		Version:    "test",
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Translator: &stubTranslator{output: "Initial"},
		Clipboard:  &stubClipboard{},
		Editor:     &stubEditor{err: editor.ErrCanceled},
		ConfigLoader: func() (config.Config, error) {
			return config.Config{
				Targets: map[string]config.TargetConfig{
					"claude": {Template: "{{prompt}}"},
				},
				Roles: map[string]config.RoleConfig{},
			}, nil
		},
		ConfigInit: func() (string, error) { return "", nil },
		LookupEnv:  func(string) (string, bool) { return "", false },
	})

	err := app.Execute(context.Background(), []string{"-i", "hello"}, strings.NewReader(""), false)
	if !errors.Is(err, editor.ErrCanceled) {
		t.Fatalf("Execute() error = %v, want %v", err, editor.ErrCanceled)
	}
}
