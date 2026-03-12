package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/helloprtr/poly-prompt/internal/clipboard"
	"github.com/helloprtr/poly-prompt/internal/config"
	"github.com/helloprtr/poly-prompt/internal/input"
	prompttemplate "github.com/helloprtr/poly-prompt/internal/template"
	"github.com/helloprtr/poly-prompt/internal/translate"
)

type ConfigLoader func() (config.Config, error)
type ConfigInit func() (string, error)
type LookupEnv func(string) (string, bool)

type Dependencies struct {
	Version      string
	Stdout       io.Writer
	Stderr       io.Writer
	Translator   translate.Translator
	Clipboard    clipboard.Writer
	ConfigLoader ConfigLoader
	ConfigInit   ConfigInit
	LookupEnv    LookupEnv
}

type App struct {
	version      string
	stdout       io.Writer
	stderr       io.Writer
	translator   translate.Translator
	clipboard    clipboard.Writer
	configLoader ConfigLoader
	configInit   ConfigInit
	lookupEnv    LookupEnv
}

type usageError struct {
	message string
}

type runOptions struct {
	target       string
	noCopy       bool
	showOriginal bool
}

func New(deps Dependencies) *App {
	return &App{
		version:      deps.Version,
		stdout:       deps.Stdout,
		stderr:       deps.Stderr,
		translator:   deps.Translator,
		clipboard:    deps.Clipboard,
		configLoader: deps.ConfigLoader,
		configInit:   deps.ConfigInit,
		lookupEnv:    deps.LookupEnv,
	}
}

func (a *App) Execute(ctx context.Context, args []string, stdin io.Reader, stdinPiped bool) error {
	if len(args) > 0 {
		switch args[0] {
		case "init":
			return a.runInit()
		case "version":
			return a.runVersion()
		}
	}

	return a.runMain(ctx, args, stdin, stdinPiped)
}

func (a *App) runInit() error {
	path, err := a.configInit()
	if err != nil {
		if errors.Is(err, config.ErrConfigExists) {
			return fmt.Errorf("config already exists at %s", path)
		}
		return err
	}

	_, _ = fmt.Fprintf(a.stdout, "created config at %s\n", path)
	return nil
}

func (a *App) runVersion() error {
	version := strings.TrimSpace(a.version)
	if version == "" {
		version = "dev"
	}

	_, _ = fmt.Fprintln(a.stdout, version)
	return nil
}

func (a *App) runMain(ctx context.Context, args []string, stdin io.Reader, stdinPiped bool) error {
	opts, positional, err := parseRunOptions(args)
	if err != nil {
		return err
	}

	text, err := input.Resolve(positional, stdin, stdinPiped)
	if err != nil {
		if errors.Is(err, input.ErrNoInput) {
			return usageError{message: "missing prompt text"}
		}
		return fmt.Errorf("read input: %w", err)
	}

	cfg, err := a.configLoader()
	if err != nil {
		return err
	}

	envTarget, _ := a.lookupEnv("PRTR_TARGET")
	target := config.ResolveTarget(opts.target, cfg, envTarget)

	targetConfig, ok := cfg.Targets[target]
	if !ok {
		available := config.AvailableTargets(cfg)
		return fmt.Errorf("unknown target %q (available: %s)", target, strings.Join(available, ", "))
	}

	translated, err := a.translator.Translate(ctx, text)
	if err != nil {
		return err
	}

	finalPrompt, err := prompttemplate.Render(targetConfig.Template, translated)
	if err != nil {
		return fmt.Errorf("render target template %q: %w", target, err)
	}

	if opts.showOriginal {
		_, _ = fmt.Fprintf(a.stderr, "Original:\n%s\n\n", text)
	}

	_, _ = fmt.Fprintln(a.stdout, finalPrompt)

	if opts.noCopy {
		_, _ = fmt.Fprintf(a.stderr, "target %q ready; clipboard skipped\n", target)
		return nil
	}

	if err := a.clipboard.Copy(ctx, finalPrompt); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.stderr, "copied prompt for target %q to clipboard\n", target)
	return nil
}

func parseRunOptions(args []string) (runOptions, []string, error) {
	fs := flag.NewFlagSet("prtr", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts runOptions
	fs.StringVar(&opts.target, "target", "", "target profile name")
	fs.StringVar(&opts.target, "t", "", "target profile name")
	fs.BoolVar(&opts.noCopy, "no-copy", false, "skip clipboard copy")
	fs.BoolVar(&opts.showOriginal, "show-original", false, "print the original input to stderr")

	if err := fs.Parse(args); err != nil {
		return runOptions{}, nil, usageError{message: err.Error()}
	}

	return opts, fs.Args(), nil
}

func (e usageError) Error() string {
	return fmt.Sprintf("%s\n\n%s", e.message, usageText())
}

func usageText() string {
	return strings.Join([]string{
		"Usage:",
		"  prtr [flags] [text...]",
		"  prtr init",
		"  prtr version",
		"",
		"Flags:",
		"  -t, --target <name>  target profile name",
		"      --no-copy       print the translated prompt without copying it",
		"      --show-original print the original input to stderr",
		"",
		"Examples:",
		`  prtr -t codex "한국어 질문"`,
		`  echo "한국어 질문" | prtr`,
	}, "\n")
}
