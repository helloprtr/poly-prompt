package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/helloprtr/poly-prompt/internal/automation"
	"github.com/helloprtr/poly-prompt/internal/capsule"
	"github.com/helloprtr/poly-prompt/internal/clipboard"
	"github.com/helloprtr/poly-prompt/internal/config"
	"github.com/helloprtr/poly-prompt/internal/deep"
	"github.com/helloprtr/poly-prompt/internal/editor"
	"github.com/helloprtr/poly-prompt/internal/history"
	"github.com/helloprtr/poly-prompt/internal/input"
	"github.com/helloprtr/poly-prompt/internal/launcher"
	"github.com/helloprtr/poly-prompt/internal/repoctx"
	"github.com/helloprtr/poly-prompt/internal/session"
	prompttemplate "github.com/helloprtr/poly-prompt/internal/template"
	"github.com/helloprtr/poly-prompt/internal/termbook"
	"github.com/helloprtr/poly-prompt/internal/translate"
)

type ConfigLoader func() (config.Config, error)
type ConfigInit func() (string, error)
type LookupEnv func(string) (string, bool)
type TranslatorFactory func(string) translate.Translator
type RepoRootFinder func() (string, error)
type TermbookLoader func(string) (termbook.Book, error)

type Dependencies struct {
	Version           string
	Stdout            io.Writer
	Stderr            io.Writer
	Translator        translate.Translator
	TranslatorFactory TranslatorFactory
	Clipboard         clipboard.Accessor
	Editor            editor.Editor
	Launcher          launcher.Launcher
	Automator         automation.Automator
	SubmitConfirmer   SubmitConfirmer
	ConfigLoader      ConfigLoader
	ConfigInit        ConfigInit
	LookupEnv         LookupEnv
	HistoryStore      *history.Store
	RepoContext       repoctx.Collector
	RepoRootFinder    RepoRootFinder
	TermbookLoader    TermbookLoader
	SessionStore      *session.Store
}

type App struct {
	version           string
	stdout            io.Writer
	stderr            io.Writer
	translator        translate.Translator
	translatorFactory TranslatorFactory
	clipboard         clipboard.Accessor
	editor            editor.Editor
	launcher          launcher.Launcher
	automator         automation.Automator
	submitConfirmer   SubmitConfirmer
	configLoader      ConfigLoader
	configInit        ConfigInit
	lookupEnv         LookupEnv
	historyStore      *history.Store
	repoContext       repoctx.Collector
	repoRootFinder    RepoRootFinder
	termbookLoader    TermbookLoader
	sessionStore      *session.Store
}

type usageError struct {
	message  string
	helpText string
}

type runOptions struct {
	target               string
	role                 string
	templatePreset       string
	sourceLang           string
	targetLang           string
	translationMode      string
	noCopy               bool
	showOriginal         bool
	interactive          bool
	explain              bool
	diff                 bool
	jsonOutput           bool
	launch               bool
	paste                bool
	submitMode           string
	rerunEdit            bool
	compactStatus        bool
	surfaceMode          string
	surfaceInput         string
	surfaceDelivery      string
	promptSuffix         string
	protectedTerms       []string
	preferTargetTemplate bool
	engine               string
	parentID             string
	runID                string
	resultType           string
	artifactRoot         string
	runStatus            string
	eventLogPath         string
	statusNotes          []string
	nextSteps            []string
}

type goCommandOptions struct {
	mode      string
	app       string
	edit      bool
	dryRun    bool
	noContext bool
	noCopy    bool
	prompt    []string
}

type startCommandOptions struct {
	app    string
	dryRun bool
	prompt []string
}

type replayCommandOptions struct {
	app       string
	edit      bool
	dryRun    bool
	noContext bool
	noCopy    bool
	prompt    []string
}

type takeCommandOptions struct {
	action      string
	app         string
	edit        bool
	dryRun      bool
	deep        bool
	llm         bool
	llmProvider string
}

type learnCommandOptions struct {
	dryRun bool
	reset  bool
	paths  []string
}

type resolvedRun struct {
	original            string
	translated          string
	finalPrompt         string
	targetName          string
	targetSource        string
	targetConfig        config.TargetConfig
	roleName            string
	rolePrompt          string
	roleSource          string
	roleVariantSource   string
	templateName        string
	templateSource      string
	layout              string
	context             string
	outputFormat        string
	shortcut            string
	copied              bool
	launched            bool
	launchedTarget      string
	deliveryMode        string
	pasted              bool
	submitMode          string
	submitted           bool
	apiKeySource        string
	sourceLang          string
	targetLang          string
	translationMode     string
	translationDecision string
	config              config.Config
	engine              string
	parentID            string
	runID               string
	resultType          string
	artifactRoot        string
	runStatus           string
	eventLogPath        string
	statusNotes         []string
	nextSteps           []string
}

var sourceLangOptions = []languageOption{
	{Label: "auto", Value: "auto", Description: "automatic detection"},
	{Label: "ko", Value: "ko", Description: "Korean"},
	{Label: "ja", Value: "ja", Description: "Japanese"},
	{Label: "zh", Value: "zh", Description: "Chinese"},
	{Label: "en", Value: "en", Description: "English"},
}

var targetLangOptions = []languageOption{
	{Label: "en", Value: "en", Description: "English"},
	{Label: "ja", Value: "ja", Description: "Japanese"},
	{Label: "zh", Value: "zh", Description: "Chinese"},
	{Label: "de", Value: "de", Description: "German"},
	{Label: "fr", Value: "fr", Description: "French"},
}

const (
	demoOriginalRequest = "왜 깨지는지 정확한 원인만 찾아줘"
	demoTranslatedGoal  = "Find only the precise root cause of why this npm test run is failing. Do not suggest fixes yet."
	demoEvidence        = `> prtr@0.7.0 test
> go test ./...

--- FAIL: TestExecuteTakePatch (0.02s)
    app_test.go:751: expected compact status to include translation route
FAIL
exit status 1`
)

func New(deps Dependencies) *App {
	store := deps.HistoryStore
	if store == nil {
		if path, err := history.DefaultPath(); err == nil {
			store = history.New(path)
		}
	}

	return &App{
		version:           deps.Version,
		stdout:            deps.Stdout,
		stderr:            deps.Stderr,
		translator:        deps.Translator,
		translatorFactory: deps.TranslatorFactory,
		clipboard:         deps.Clipboard,
		editor:            deps.Editor,
		launcher:          deps.Launcher,
		automator:         deps.Automator,
		submitConfirmer:   deps.SubmitConfirmer,
		configLoader:      deps.ConfigLoader,
		configInit:        deps.ConfigInit,
		lookupEnv:         deps.LookupEnv,
		historyStore:      store,
		repoContext:       deps.RepoContext,
		repoRootFinder:    deps.RepoRootFinder,
		termbookLoader:    deps.TermbookLoader,
		sessionStore:      deps.SessionStore,
	}
}

func (a *App) Execute(ctx context.Context, args []string, stdin io.Reader, stdinPiped bool) error {
	// Bare invocation (no args) → interactive session start.
	if len(args) == 0 {
		return a.runBare(ctx, stdin)
	}
	// @model → handoff to named model.
	if strings.HasPrefix(strings.TrimSpace(args[0]), "@") {
		model := strings.TrimPrefix(strings.TrimSpace(args[0]), "@")
		return a.runHandoff(ctx, model)
	}
	if a.shouldRunRootDirect(args) {
		return a.runMain(ctx, args, stdin, stdinPiped, "")
	}
	cmd := a.Command(ctx, stdin, stdinPiped)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func (a *App) shouldRunRootDirect(args []string) bool {
	// len(args)==0 and @model are handled in Execute before this is called.
	first := strings.TrimSpace(args[0])
	if first == "" {
		return true
	}
	if first == "-h" || first == "--help" || first == "help" {
		return false
	}
	if strings.HasPrefix(first, "-") {
		return true
	}

	switch first {
	case "init", "version", "start", "setup", "lang", "doctor", "templates", "profiles",
		"history", "rerun", "pin", "favorite", "go", "demo", "again", "swap", "take",
		"learn", "inspect",
		// v0.8 commands:
		"watch", "save", "resume", "list", "prune",
		"dip", "taste", "plate", "marinate", "prep",
		// v1.0 commands:
		"review", "edit", "fix", "design", "checkpoint", "done", "sessions", "status":
		return false
	}

	return !a.builtInShortcutNames()[first]
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

func (a *App) runDemo(ctx context.Context) error {
	previewInput := demoTranslatedGoal + "\n\nEvidence:\n" + demoEvidence
	resolved, err := a.prepareRun(ctx, runOptions{
		target:          "codex",
		role:            "be",
		templatePreset:  "codex-implement",
		sourceLang:      "en",
		targetLang:      "en",
		translationMode: string(translate.ModeSkip),
		noCopy:          true,
		launch:          false,
		paste:           false,
	}, previewInput, "")
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(a.stdout, "prtr demo")
	_, _ = fmt.Fprintln(a.stdout, "The command layer for AI work.")
	_, _ = fmt.Fprintln(a.stdout, "No API key required. Safe preview only.")
	_, _ = fmt.Fprintln(a.stdout)
	_, _ = fmt.Fprintln(a.stdout, "Loop:")
	_, _ = fmt.Fprintln(a.stdout, `  npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"`)
	_, _ = fmt.Fprintln(a.stdout, "  prtr swap gemini")
	_, _ = fmt.Fprintln(a.stdout, "  prtr take patch")
	_, _ = fmt.Fprintln(a.stdout)
	_, _ = fmt.Fprintln(a.stdout, "Sample input:")
	_, _ = fmt.Fprintf(a.stdout, "  request (ko): %s\n", demoOriginalRequest)
	_, _ = fmt.Fprintln(a.stdout, "  evidence: npm test output")
	_, _ = fmt.Fprintln(a.stdout)
	_, _ = fmt.Fprintln(a.stdout, "Preview prompt:")
	_, _ = fmt.Fprintln(a.stdout, resolved.finalPrompt)
	_, _ = fmt.Fprintln(a.stdout)
	_, _ = fmt.Fprintln(a.stdout, "Try next:")
	_, _ = fmt.Fprintln(a.stdout, `  prtr go "explain this error" --dry-run`)
	_, _ = fmt.Fprintln(a.stdout, "  prtr setup")

	return nil
}

func (a *App) runStart(ctx context.Context, args []string, stdin io.Reader, stdinPiped bool) error {
	command, err := parseStartCommand(args)
	if err != nil {
		return err
	}

	cfg, err := a.configLoader()
	if err != nil {
		return err
	}

	envAPIKey, _ := a.lookupEnv("DEEPL_API_KEY")
	currentAPIKey, _ := config.ResolveAPIKey(envAPIKey, cfg)
	reader := bufio.NewReader(stdin)

	if startNeedsOnboarding(cfg, currentAPIKey) {
		if stdinPiped {
			return errors.New("prtr start requires an interactive terminal when onboarding is needed; rerun without piped stdin or use `prtr setup` first")
		}
		if err := a.runStartOnboarding(reader, cfg, currentAPIKey); err != nil {
			return err
		}
		cfg, err = a.configLoader()
		if err != nil {
			return err
		}
	}

	_, _ = fmt.Fprintln(a.stdout, "running prtr doctor before the first send...")
	if err := a.runDoctorForStart(ctx, command.dryRun); err != nil {
		return err
	}

	goArgs := make([]string, 0, len(command.prompt)+3)
	if command.app != "" {
		goArgs = append(goArgs, "--to", command.app)
	}
	if command.dryRun {
		goArgs = append(goArgs, "--dry-run")
	}

	startReader := stdin
	startStdinPiped := stdinPiped
	switch {
	case len(command.prompt) > 0:
		goArgs = append(goArgs, command.prompt...)
		startReader = strings.NewReader("")
		startStdinPiped = false
	case !stdinPiped:
		_, _ = fmt.Fprintln(a.stdout, "")
		firstPrompt, err := promptInput(reader, a.stdout, "First request [default: 이 함수 왜 느린지 설명해줘]")
		if err != nil {
			return err
		}
		firstPrompt = strings.TrimSpace(firstPrompt)
		if firstPrompt == "" {
			firstPrompt = "이 함수 왜 느린지 설명해줘"
		}
		goArgs = append(goArgs, firstPrompt)
		startReader = strings.NewReader("")
		startStdinPiped = false
	}

	if err := a.runGo(ctx, goArgs, startReader, startStdinPiped); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(a.stdout, "")
	_, _ = fmt.Fprintln(a.stdout, "next actions:")
	_, _ = fmt.Fprintln(a.stdout, "  prtr swap <app>")
	_, _ = fmt.Fprintln(a.stdout, "  prtr take patch")
	_, _ = fmt.Fprintln(a.stdout, "  prtr again")
	_, _ = fmt.Fprintln(a.stdout, "  prtr learn")
	return nil
}

func startNeedsOnboarding(cfg config.Config, currentAPIKey string) bool {
	if !cfg.HasUserConfig {
		return true
	}
	if strings.TrimSpace(currentAPIKey) == "" {
		return true
	}
	if strings.TrimSpace(cfg.TranslationSourceLang) == "" {
		return true
	}
	if strings.TrimSpace(cfg.TranslationTargetLang) == "" {
		return true
	}
	if strings.TrimSpace(config.ResolveTarget("", cfg, "")) == "" {
		return true
	}
	return false
}

func (a *App) runDoctorForStart(ctx context.Context, dryRun bool) error {
	cfg, err := a.configLoader()
	if err != nil {
		return err
	}

	report := a.buildDoctorReport(ctx, cfg)
	a.writeDoctorSection("Platform matrix", report.Platform)
	a.writeDoctorSection("Checks", report.Checks)

	failures := 0
	for _, check := range append(append([]doctorCheck{}, report.Platform...), report.Checks...) {
		if check.Severity != doctorBlocking {
			continue
		}
		if dryRun && startAllowsDryRunBlockingCheck(check.Label) {
			continue
		}
		failures++
	}
	if failures > 0 {
		return fmt.Errorf("doctor found %d issue(s)", failures)
	}
	return nil
}

func startAllowsDryRunBlockingCheck(label string) bool {
	switch {
	case label == "user config":
		return true
	case label == "clipboard":
		return true
	case label == "launcher surface":
		return true
	case label == "paste surface":
		return true
	case strings.HasPrefix(label, "launcher "):
		return true
	case strings.HasPrefix(label, "automation "):
		return true
	default:
		return false
	}
}

func (a *App) runStartOnboarding(reader *bufio.Reader, cfg config.Config, currentAPIKey string) error {
	currentTarget := config.ResolveTarget("", cfg, "")
	currentSourceLang := cfg.TranslationSourceLang
	if currentSourceLang == "" {
		currentSourceLang = "auto"
	}
	currentTargetLang := cfg.TranslationTargetLang
	if currentTargetLang == "" {
		currentTargetLang = "en"
	}

	_, _ = fmt.Fprintln(a.stdout, "prtr start")
	_, _ = fmt.Fprintln(a.stdout, "Let's get your first send ready. Press Enter to keep the current value.")

	apiPrompt := "DeepL API key"
	if currentAPIKey != "" {
		apiPrompt += " [configured]"
	}
	apiValue, err := promptInput(reader, a.stdout, apiPrompt)
	if err != nil {
		return err
	}

	sourceLangValue, err := promptLanguage(reader, a.stdout, "Default input language", []languageOption{
		{Label: "auto", Value: "auto", Description: "automatic detection"},
		{Label: "ko", Value: "ko", Description: "Korean"},
		{Label: "ja", Value: "ja", Description: "Japanese"},
		{Label: "zh", Value: "zh", Description: "Chinese"},
		{Label: "en", Value: "en", Description: "English"},
	}, currentSourceLang, true)
	if err != nil {
		return err
	}

	targetLangValue, err := promptLanguage(reader, a.stdout, "Default output language", []languageOption{
		{Label: "en", Value: "en", Description: "English"},
		{Label: "ja", Value: "ja", Description: "Japanese"},
		{Label: "zh", Value: "zh", Description: "Chinese"},
		{Label: "de", Value: "de", Description: "German"},
		{Label: "fr", Value: "fr", Description: "French"},
	}, currentTargetLang, false)
	if err != nil {
		return err
	}

	targetValue, err := promptChoice(reader, a.stdout, "Default app", config.AvailableTargets(cfg), currentTarget, false)
	if err != nil {
		return err
	}

	update := config.DefaultsUpdate{
		TranslationSourceLang: stringPtr(sourceLangValue),
		TranslationTargetLang: stringPtr(targetLangValue),
		DefaultTarget:         stringPtr(targetValue),
	}
	if strings.TrimSpace(apiValue) != "" {
		update.APIKey = stringPtr(apiValue)
	}

	path, err := config.SaveDefaults(update)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.stdout, "\nupdated config at %s\n", path)
	return nil
}

func (a *App) runSetup(stdin io.Reader) error {
	cfg, err := a.configLoader()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(stdin)
	envAPIKey, _ := a.lookupEnv("DEEPL_API_KEY")
	currentAPIKey, _ := config.ResolveAPIKey(envAPIKey, cfg)
	currentTarget := config.ResolveTarget("", cfg, "")
	currentRole := config.ResolveRole("", cfg)
	currentSourceLang := cfg.TranslationSourceLang
	if currentSourceLang == "" {
		currentSourceLang = "auto"
	}
	currentTargetLang := cfg.TranslationTargetLang
	if currentTargetLang == "" {
		currentTargetLang = "en"
	}
	currentPreset := cfg.DefaultTemplatePreset
	if currentPreset == "" {
		if targetConfig, ok := cfg.Targets[currentTarget]; ok {
			currentPreset = targetConfig.DefaultTemplatePreset
		}
	}

	_, _ = fmt.Fprintln(a.stdout, "prtr setup")
	_, _ = fmt.Fprintln(a.stdout, "Press Enter to keep the current value.")

	apiPrompt := "DeepL API key"
	if currentAPIKey != "" {
		apiPrompt += " [configured]"
	}
	apiValue, err := promptInput(reader, a.stdout, apiPrompt)
	if err != nil {
		return err
	}

	sourceLangValue, err := promptLanguage(reader, a.stdout, "Default input language", sourceLangOptions, currentSourceLang, true)
	if err != nil {
		return err
	}

	targetLangValue, err := promptLanguage(reader, a.stdout, "Default output language", targetLangOptions, currentTargetLang, false)
	if err != nil {
		return err
	}

	targetValue, err := promptChoice(reader, a.stdout, "Default target", config.AvailableTargets(cfg), currentTarget, false)
	if err != nil {
		return err
	}
	roleValue, err := promptChoice(reader, a.stdout, "Default role", config.AvailableRoles(cfg), currentRole, true)
	if err != nil {
		return err
	}
	presetDefault := currentPreset
	if presetDefault == "" {
		presetDefault = "claude-structured"
	}
	presetValue, err := promptChoice(reader, a.stdout, "Default template preset", config.AvailableTemplatePresets(cfg), presetDefault, false)
	if err != nil {
		return err
	}

	update := config.DefaultsUpdate{
		TranslationSourceLang: stringPtr(sourceLangValue),
		TranslationTargetLang: stringPtr(targetLangValue),
		DefaultTarget:         stringPtr(targetValue),
		DefaultRole:           stringPtr(roleValue),
		DefaultTemplatePreset: stringPtr(presetValue),
	}
	if strings.TrimSpace(apiValue) != "" {
		update.APIKey = stringPtr(apiValue)
	}

	path, err := config.SaveDefaults(update)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.stdout, "\nupdated config at %s\n", path)
	if strings.TrimSpace(apiValue) == "" && currentAPIKey == "" {
		_, _ = fmt.Fprintln(a.stdout, "No DeepL key set. prtr works without one — AI targets handle multilingual input natively. Add DEEPL_API_KEY or rerun setup if you want dedicated translation quality.")
	}
	return nil
}

func (a *App) runLang(stdin io.Reader) error {
	cfg, err := a.configLoader()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(stdin)
	currentSourceLang := cfg.TranslationSourceLang
	if currentSourceLang == "" {
		currentSourceLang = "auto"
	}
	currentTargetLang := cfg.TranslationTargetLang
	if currentTargetLang == "" {
		currentTargetLang = "en"
	}

	sourceLangValue, err := promptLanguage(reader, a.stdout, "Default input language", sourceLangOptions, currentSourceLang, true)
	if err != nil {
		return err
	}

	targetLangValue, err := promptLanguage(reader, a.stdout, "Default output language", targetLangOptions, currentTargetLang, false)
	if err != nil {
		return err
	}

	path, err := config.SaveDefaults(config.DefaultsUpdate{
		TranslationSourceLang: stringPtr(sourceLangValue),
		TranslationTargetLang: stringPtr(targetLangValue),
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.stdout, "updated language defaults in %s\n", path)
	return nil
}
func (a *App) runTemplates(args []string) error {
	if len(args) == 0 {
		return usageError{message: "templates requires a subcommand: list or show"}
	}

	cfg, err := a.configLoader()
	if err != nil {
		return err
	}

	switch args[0] {
	case "list":
		for _, name := range config.AvailableTemplatePresets(cfg) {
			preset := cfg.TemplatePresets[name]
			if preset.Description != "" {
				_, _ = fmt.Fprintf(a.stdout, "%s\t%s\n", name, preset.Description)
				continue
			}
			_, _ = fmt.Fprintln(a.stdout, name)
		}
		return nil
	case "show":
		if len(args) < 2 {
			return usageError{message: "templates show requires a template preset name"}
		}
		preset, ok := cfg.TemplatePresets[args[1]]
		if !ok {
			return fmt.Errorf("unknown template preset %q (available: %s)", args[1], strings.Join(config.AvailableTemplatePresets(cfg), ", "))
		}
		if preset.Description != "" {
			_, _ = fmt.Fprintf(a.stdout, "# %s\n%s\n\n", args[1], preset.Description)
		}
		_, _ = fmt.Fprintln(a.stdout, preset.Template)
		return nil
	default:
		return usageError{message: fmt.Sprintf("unknown templates subcommand %q", args[0])}
	}
}

func (a *App) runProfiles(args []string) error {
	if len(args) == 0 {
		return usageError{message: "profiles requires a subcommand: list, show, or use"}
	}

	cfg, err := a.configLoader()
	if err != nil {
		return err
	}

	switch args[0] {
	case "list":
		for _, name := range config.AvailableProfiles(cfg) {
			profile := cfg.Profiles[name]
			_, _ = fmt.Fprintf(a.stdout, "%s\ttarget=%s role=%s template=%s\n", name, profile.Target, profile.Role, profile.TemplatePreset)
		}
		return nil
	case "show":
		if len(args) < 2 {
			return usageError{message: "profiles show requires a profile name"}
		}
		profile, ok := cfg.Profiles[args[1]]
		if !ok {
			return fmt.Errorf("unknown profile %q (available: %s)", args[1], strings.Join(config.AvailableProfiles(cfg), ", "))
		}
		_, _ = fmt.Fprintf(a.stdout, "name: %s\n", args[1])
		_, _ = fmt.Fprintf(a.stdout, "target: %s\n", profile.Target)
		_, _ = fmt.Fprintf(a.stdout, "role: %s\n", profile.Role)
		_, _ = fmt.Fprintf(a.stdout, "template_preset: %s\n", profile.TemplatePreset)
		if profile.Context != "" {
			_, _ = fmt.Fprintf(a.stdout, "context: %s\n", profile.Context)
		}
		if profile.OutputFormat != "" {
			_, _ = fmt.Fprintf(a.stdout, "output_format: %s\n", profile.OutputFormat)
		}
		return nil
	case "use":
		if len(args) < 2 {
			return usageError{message: "profiles use requires a profile name"}
		}
		profile, ok := cfg.Profiles[args[1]]
		if !ok {
			return fmt.Errorf("unknown profile %q (available: %s)", args[1], strings.Join(config.AvailableProfiles(cfg), ", "))
		}
		path, err := config.SaveDefaults(config.DefaultsUpdate{
			TranslationTargetLang: stringPtr(profile.TranslationTargetLang),
			DefaultTarget:         stringPtr(profile.Target),
			DefaultRole:           stringPtr(profile.Role),
			DefaultTemplatePreset: stringPtr(profile.TemplatePreset),
		})
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(a.stdout, "set defaults from profile %q in %s\n", args[1], path)
		return nil
	default:
		return usageError{message: fmt.Sprintf("unknown profiles subcommand %q", args[0])}
	}
}

func (a *App) runHistory(args []string) error {
	if a.historyStore == nil {
		return errors.New("history store is not configured")
	}

	entries, err := a.historyStore.List()
	if err != nil {
		return err
	}
	if len(args) > 0 {
		if args[0] != "search" {
			return usageError{message: "history requires no args or the search subcommand"}
		}
		if len(args) < 2 {
			return usageError{message: "history search requires a query"}
		}
		entries, err = a.historyStore.Search(strings.Join(args[1:], " "))
		if err != nil {
			return err
		}
	}

	if len(entries) == 0 {
		_, _ = fmt.Fprintln(a.stdout, "no history yet")
		return nil
	}

	limit := minInt(10, len(entries))
	for _, entry := range entries[:limit] {
		preview := truncateOneLine(entry.Original, 60)
		flags := historyFlags(entry)
		extra := flags
		if strings.TrimSpace(entry.Engine) == "deep" {
			extra = strings.TrimSpace(strings.Join([]string{flags, "deep:" + blankDefault(entry.RunStatus, "completed")}, ","))
		}
		_, _ = fmt.Fprintf(a.stdout, "%s\t%s\ttarget=%s role=%s template=%s lang=%s->%s %s\t%s\n", entry.ID, entry.CreatedAt.Format("2006-01-02 15:04:05"), entry.Target, entry.Role, entry.TemplatePreset, blankDefault(entry.SourceLang, "auto"), blankDefault(entry.TargetLang, "en"), extra, preview)
	}
	return nil
}

func (a *App) runRerun(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usageError{message: "rerun requires a history id"}
	}
	if a.historyStore == nil {
		return errors.New("history store is not configured")
	}

	entry, err := a.historyStore.Get(args[0])
	if err != nil {
		if errors.Is(err, history.ErrNotFound) {
			return fmt.Errorf("history entry %q not found", args[0])
		}
		return err
	}

	opts, _, err := parseRunOptions(args[1:])
	if err != nil {
		return err
	}
	if strings.TrimSpace(opts.target) == "" {
		opts.target = entry.Target
	}
	if strings.TrimSpace(opts.role) == "" {
		opts.role = entry.Role
	}
	if strings.TrimSpace(opts.templatePreset) == "" {
		opts.templatePreset = entry.TemplatePreset
	}
	if strings.TrimSpace(opts.sourceLang) == "" {
		opts.sourceLang = entry.SourceLang
	}
	if strings.TrimSpace(opts.targetLang) == "" {
		opts.targetLang = entry.TargetLang
	}
	if strings.TrimSpace(opts.translationMode) == "" {
		opts.translationMode = entry.TranslationMode
	}
	if opts.rerunEdit {
		return a.executeStoredPrompt(ctx, opts, entry)
	}

	return a.executePrompt(ctx, opts, entry.Original, entry.Shortcut)
}

func (a *App) runPin(args []string) error {
	if len(args) == 0 {
		return usageError{message: "pin requires a history id"}
	}
	if a.historyStore == nil {
		return errors.New("history store is not configured")
	}

	entry, err := a.historyStore.TogglePinned(args[0])
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(a.stdout, "%s %s\n", pinVerb(entry.Pinned), entry.ID)
	return nil
}

func (a *App) runFavorite(args []string) error {
	if len(args) == 0 {
		return usageError{message: "favorite requires a history id"}
	}
	if a.historyStore == nil {
		return errors.New("history store is not configured")
	}

	entry, err := a.historyStore.ToggleFavorite(args[0])
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(a.stdout, "%s %s\n", favoriteVerb(entry.Favorite), entry.ID)
	return nil
}

func (a *App) runShortcut(ctx context.Context, shortcut string, args []string, stdin io.Reader, stdinPiped bool) error {
	return a.runMain(ctx, args, stdin, stdinPiped, shortcut)
}

func (a *App) runGo(ctx context.Context, args []string, stdin io.Reader, stdinPiped bool) error {
	command, err := parseGoCommand(args, a.builtInShortcutNames())
	if err != nil {
		return err
	}

	text, inputSource, err := resolveSurfaceInput(command.prompt, stdin, stdinPiped, !command.noContext)
	if err != nil {
		if errors.Is(err, input.ErrNoInput) {
			return usageError{message: "missing prompt text"}
		}
		return fmt.Errorf("read input: %w", err)
	}
	repoSuffix, inputSource := a.resolveRepoContext(ctx, inputSource, command.noContext)
	protectedTerms, protectedSuffix := a.resolveLearnedTerms(command.noContext)

	if !command.noContext {
		if repoRoot, err := a.resolveRepoRoot(); err == nil && repoRoot != "" {
			patterns := repoctx.LoadIgnorePatterns(repoRoot)

			if diff, err := repoctx.GitDiff(ctx, repoRoot); err == nil && diff != "" {
				filtered := repoctx.FilterDiffHunks(diff, patterns)
				if filtered != "" {
					repoSuffix += "\n\nGit diff:\n" + filtered
				}
			}

			testOutputPath := filepath.Join(os.TempDir(), "prtr-last-output")
			if out, err := repoctx.LastTestOutput(testOutputPath); err == nil && out != "" {
				repoSuffix += "\n\nLast test output:\n" + out
			}
		}
	}

	target := strings.TrimSpace(command.app)
	if target == "" {
		if entry, err := a.latestHistoryEntry(); err == nil {
			target = entry.Target
		}
	}

	opts := runOptions{
		target:               target,
		interactive:          command.edit,
		noCopy:               command.dryRun || command.noCopy,
		launch:               !command.dryRun,
		paste:                !command.dryRun,
		compactStatus:        true,
		surfaceMode:          command.mode,
		surfaceInput:         inputSource,
		surfaceDelivery:      surfaceDeliveryLabel(command.dryRun),
		promptSuffix:         joinPromptSections(repoSuffix, protectedSuffix),
		protectedTerms:       protectedTerms,
		preferTargetTemplate: true,
	}

	return a.executePrompt(ctx, opts, text, command.mode)
}

func (a *App) resolveRepoContext(ctx context.Context, inputSource string, disabled bool) (string, string) {
	if disabled || a.repoContext == nil {
		return "", inputSource
	}

	summary, err := a.repoContext.Collect(ctx)
	if err != nil {
		return "", inputSource
	}

	contextBlock := formatRepoContext(summary)
	if contextBlock == "" {
		return "", inputSource
	}

	label := inputSource
	if label == "" {
		label = "prompt"
	}
	if !strings.Contains(label, "repo") {
		label += "+repo"
	}

	return contextBlock, label
}

func (a *App) resolveLearnedTerms(disabled bool) ([]string, string) {
	if disabled {
		return nil, ""
	}

	repoRoot, err := a.resolveRepoRoot()
	if err != nil {
		return nil, ""
	}

	book, err := a.loadTermbook(repoRoot)
	if err != nil {
		return nil, ""
	}
	if len(book.ProtectedTerms) == 0 {
		return nil, ""
	}

	return book.ProtectedTerms, formatProtectedTerms(book.ProtectedTerms)
}

func (a *App) resolveRepoRoot() (string, error) {
	if a.repoRootFinder != nil {
		return a.repoRootFinder()
	}
	return termbook.FindRepoRoot("")
}

func (a *App) loadTermbook(repoRoot string) (termbook.Book, error) {
	if a.termbookLoader != nil {
		return a.termbookLoader(repoRoot)
	}
	return termbook.Load(repoRoot)
}

func (a *App) runAgain(ctx context.Context, args []string, stdin io.Reader, stdinPiped bool) error {
	command, err := parseReplayCommand(args, false)
	if err != nil {
		return err
	}

	entry, err := a.latestHistoryEntry()
	if err != nil {
		return err
	}

	text := entry.Original
	inputSource := "history"
	if len(command.prompt) > 0 || stdinPiped {
		text, inputSource, err = resolveSurfaceInput(command.prompt, stdin, stdinPiped, !command.noContext)
		if err != nil {
			if errors.Is(err, input.ErrNoInput) {
				return usageError{message: "missing prompt text"}
			}
			return fmt.Errorf("read input: %w", err)
		}
	}

	opts := runOptions{
		target:          entry.Target,
		role:            entry.Role,
		templatePreset:  entry.TemplatePreset,
		sourceLang:      entry.SourceLang,
		targetLang:      entry.TargetLang,
		translationMode: entry.TranslationMode,
		interactive:     command.edit,
		noCopy:          command.dryRun || command.noCopy,
		launch:          !command.dryRun,
		paste:           !command.dryRun,
		compactStatus:   true,
		surfaceMode:     blankDefault(entry.Shortcut, "ask"),
		surfaceInput:    inputSource,
		surfaceDelivery: surfaceDeliveryLabel(command.dryRun),
		engine:          "classic",
		parentID:        entry.ID,
	}

	return a.executePrompt(ctx, opts, text, entry.Shortcut)
}

func (a *App) runSwap(ctx context.Context, args []string, stdin io.Reader, stdinPiped bool) error {
	command, err := parseReplayCommand(args, true)
	if err != nil {
		return err
	}

	entry, err := a.latestHistoryEntry()
	if err != nil {
		return err
	}

	text := entry.Original
	inputSource := "history"
	if len(command.prompt) > 0 || stdinPiped {
		text, inputSource, err = resolveSurfaceInput(command.prompt, stdin, stdinPiped, !command.noContext)
		if err != nil {
			if errors.Is(err, input.ErrNoInput) {
				return usageError{message: "missing prompt text"}
			}
			return fmt.Errorf("read input: %w", err)
		}
	}

	opts := runOptions{
		target:               command.app,
		role:                 entry.Role,
		sourceLang:           entry.SourceLang,
		targetLang:           entry.TargetLang,
		translationMode:      entry.TranslationMode,
		interactive:          command.edit,
		noCopy:               command.dryRun || command.noCopy,
		launch:               !command.dryRun,
		paste:                !command.dryRun,
		compactStatus:        true,
		surfaceMode:          blankDefault(entry.Shortcut, "ask"),
		surfaceInput:         inputSource,
		surfaceDelivery:      surfaceDeliveryLabel(command.dryRun),
		preferTargetTemplate: true,
		engine:               blankDefault(entry.Engine, "classic"),
		parentID:             entry.ID,
	}

	return a.executePrompt(ctx, opts, text, entry.Shortcut)
}

func (a *App) runTake(ctx context.Context, args []string) error {
	command, err := parseTakeCommand(args)
	if err != nil {
		return err
	}

	clipboardText, err := a.clipboard.Read(ctx)
	if err != nil {
		return err
	}
	clipboardText = strings.TrimSpace(clipboardText)
	if clipboardText == "" {
		return errors.New("clipboard is empty; copy an answer and try again")
	}

	target := strings.TrimSpace(command.app)
	var parentEntry history.Entry
	var hasParent bool
	if target == "" {
		if entry, err := a.latestHistoryEntry(); err == nil {
			target = entry.Target
			parentEntry = entry
			hasParent = true
		}
	} else if entry, err := a.latestHistoryEntry(); err == nil {
		parentEntry = entry
		hasParent = true
	}

	if command.deep {
		deepCfg, err := a.configLoader()
		if err != nil {
			return err
		}
		repoRoot, _ := a.resolveRepoRoot()
		repoSummary := repoctx.Summary{}
		if a.repoContext != nil {
			if summary, err := a.repoContext.Collect(ctx); err == nil {
				repoSummary = summary
			}
		}
		protectedTerms, _ := a.resolveLearnedTerms(false)
		parentHistoryID := ""
		var historyRef *history.Entry
		if hasParent {
			parentHistoryID = parentEntry.ID
			entryCopy := parentEntry
			historyRef = &entryCopy
		}

		// Resolve LLM provider: --llm flag > config llm_provider > PRTR_LLM_PROVIDER env.
		cfg, _ := a.configLoader()
		envLLMProvider, _ := a.lookupEnv("PRTR_LLM_PROVIDER")
		llmProvider := config.ResolveLLMProvider(command.llmProvider, cfg, envLLMProvider)

		_, _ = fmt.Fprintf(a.stderr, "-> take:%s --deep | %s | clipboard | running\n", command.action, target)
		result, err := deep.ExecutePatchRun(ctx, deep.Options{
			Action:          command.action,
			Source:          clipboardText,
			SourceKind:      "clipboard",
			TargetApp:       target,
			RepoRoot:        repoRoot,
			ParentHistoryID: parentHistoryID,
			ProtectedTerms:  protectedTerms,
			HistoryEntry:    historyRef,
			RepoSummary:     repoSummary,
			LLMProvider:     resolvedLLMProvider(command, deepCfg),
			LLMAPIKey:       resolvedLLMAPIKey(command, deepCfg),
			Progress: func(progress deep.Progress) {
				_, _ = fmt.Fprintf(a.stderr, "   step: %s (%d/%d)\n", progress.Step, progress.Index, progress.Total)
			},
		})
		if err != nil {
			return err
		}

		if result.LLMEnhanced {
			_, _ = fmt.Fprintf(a.stderr, "   prompt: enhanced for %s\n", llmProvider)
		} else {
			_, _ = fmt.Fprintf(a.stderr, "   prompt: rule-based (use --llm=claude|gemini|codex or set llm_provider in config)\n")
		}

		opts := runOptions{
			target:               target,
			sourceLang:           "en",
			targetLang:           "en",
			translationMode:      string(translate.ModeSkip),
			interactive:          command.edit,
			noCopy:               command.dryRun,
			launch:               !command.dryRun,
			paste:                !command.dryRun,
			compactStatus:        true,
			surfaceMode:          "take:" + command.action + " --deep",
			surfaceInput:         "clipboard",
			surfaceDelivery:      surfaceDeliveryLabel(command.dryRun),
			preferTargetTemplate: true,
			engine:               "deep",
			parentID:             parentHistoryID,
			runID:                result.Run.ID,
			resultType:           result.Run.ResultType,
			artifactRoot:         result.Run.ArtifactRoot,
			runStatus:            string(result.Run.Status),
			eventLogPath:         result.Run.EventLogPath,
			statusNotes: []string{
				"artifact: " + result.Run.ArtifactRoot + "/result/patch_bundle.json",
			},
			nextSteps: []string{
				"review: cat " + result.Run.ArtifactRoot + "/result/summary.md",
				"inspect: cat " + result.Run.EventLogPath,
				"log: prtr list",
			},
		}
		return a.executePrompt(ctx, opts, result.DeliveryPrompt, "take:"+command.action)
	}

	opts := runOptions{
		target:               target,
		sourceLang:           "en",
		targetLang:           "en",
		translationMode:      string(translate.ModeSkip),
		interactive:          command.edit,
		noCopy:               command.dryRun,
		launch:               !command.dryRun,
		paste:                !command.dryRun,
		compactStatus:        true,
		surfaceMode:          "take:" + command.action,
		surfaceInput:         "clipboard",
		surfaceDelivery:      surfaceDeliveryLabel(command.dryRun),
		preferTargetTemplate: true,
		engine:               "classic",
	}

	return a.executePrompt(ctx, opts, takePrompt(command.action, clipboardText), "take:"+command.action)
}

func (a *App) runLearn(args []string) error {
	command, err := parseLearnCommand(args)
	if err != nil {
		return err
	}

	repoRoot, err := a.resolveRepoRoot()
	if err != nil {
		return err
	}

	extraction, err := termbook.Extract(repoRoot, command.paths)
	if err != nil {
		return err
	}

	book := termbook.Book{
		GeneratedAt:    time.Now().UTC(),
		Sources:        extraction.Sources,
		ProtectedTerms: extraction.Terms,
	}

	if !command.reset {
		existing, err := a.loadTermbook(repoRoot)
		switch {
		case err == nil:
			book = termbook.Merge(existing, book)
		case errors.Is(err, os.ErrNotExist), errors.Is(err, termbook.ErrNotGitRepo):
		default:
			return err
		}
	}

	if command.dryRun {
		data, err := termbook.Encode(book)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(a.stdout, string(data))
		return nil
	}

	path, err := termbook.Save(repoRoot, book)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.stdout, "saved %d protected terms to %s\n", len(book.ProtectedTerms), path)
	return nil
}

func (a *App) runInspect(ctx context.Context, args []string, stdin io.Reader, stdinPiped bool) error {
	opts, positional, err := parseRunOptions(args)
	if err != nil {
		return err
	}

	opts.noCopy = true
	opts.launch = false
	opts.paste = false
	opts.explain = true
	if !opts.jsonOutput {
		opts.diff = true
	}

	text, err := input.Resolve(positional, stdin, stdinPiped)
	if err != nil {
		if errors.Is(err, input.ErrNoInput) {
			return usageError{message: "missing prompt text"}
		}
		return fmt.Errorf("read input: %w", err)
	}

	return a.executePrompt(ctx, opts, text, "")
}

func (a *App) runMain(ctx context.Context, args []string, stdin io.Reader, stdinPiped bool, shortcut string) error {
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

	return a.executePrompt(ctx, opts, text, shortcut)
}

func (a *App) executePrompt(ctx context.Context, opts runOptions, text, shortcut string) error {
	resolved, err := a.prepareRun(ctx, opts, text, shortcut)
	if err != nil {
		return err
	}
	resolved.engine = blankDefault(opts.engine, "classic")
	resolved.parentID = strings.TrimSpace(opts.parentID)
	resolved.runID = strings.TrimSpace(opts.runID)
	resolved.resultType = strings.TrimSpace(opts.resultType)
	resolved.artifactRoot = strings.TrimSpace(opts.artifactRoot)
	resolved.runStatus = strings.TrimSpace(opts.runStatus)
	resolved.eventLogPath = strings.TrimSpace(opts.eventLogPath)
	resolved.statusNotes = append([]string{}, opts.statusNotes...)
	resolved.nextSteps = append([]string{}, opts.nextSteps...)

	if opts.showOriginal {
		_, _ = fmt.Fprintf(a.stderr, "Original:\n%s\n\n", resolved.original)
	}
	if opts.diff {
		a.writeDiff(resolved)
	}

	if opts.interactive {
		if a.editor == nil {
			return errors.New("interactive mode is unavailable: editor is not configured")
		}

		finalPrompt, err := a.editor.Edit(ctx, editor.Request{
			Initial: resolved.finalPrompt,
			Status:  interactiveStatus(resolved),
		})
		if err != nil {
			return err
		}
		resolved.finalPrompt = finalPrompt
	}

	if !opts.noCopy {
		if err := a.clipboard.Copy(ctx, resolved.finalPrompt); err != nil {
			return err
		}
		resolved.copied = true
	}
	if err := a.appendDeepEvent(resolved.eventLogPath, deep.Event{
		Type:      deep.EventDeliveryStarted,
		Timestamp: time.Now().UTC(),
		Data: map[string]any{
			"target":        resolved.targetName,
			"delivery_mode": blankDefault(resolved.deliveryMode, blankDefault(opts.surfaceDelivery, "copy")),
		},
	}); err != nil {
		return err
	}
	if err := a.applyDelivery(ctx, opts, &resolved); err != nil {
		_ = a.appendDeepEvent(resolved.eventLogPath, deep.Event{
			Type:      deep.EventRunFailed,
			Timestamp: time.Now().UTC(),
			Data: map[string]any{
				"error": err.Error(),
			},
		})
		return err
	}
	if err := a.appendDeepEvent(resolved.eventLogPath, deep.Event{
		Type:      deep.EventDeliveryCompleted,
		Timestamp: time.Now().UTC(),
		Data: map[string]any{
			"target":        resolved.targetName,
			"pasted":        resolved.pasted,
			"submitted":     resolved.submitted,
			"delivery_mode": resolved.deliveryMode,
		},
	}); err != nil {
		return err
	}
	if opts.explain {
		a.writeExplain(resolved)
	}

	if err := a.appendHistory(resolved); err != nil {
		return err
	}

	// Auto-save capsule after successful run.
	// Use context.Background() — the command's ctx may be cancelled before
	// the goroutine finishes, which would silently drop the auto-save.
	if latestEntry, err := a.historyStore.Latest(); err == nil {
		entry := latestEntry // capture loop variable
		go a.tryAutoSave(context.Background(), &entry)
	}

	if opts.jsonOutput {
		payload := map[string]any{
			"original":             resolved.original,
			"translated":           resolved.translated,
			"target":               resolved.targetName,
			"role":                 resolved.roleName,
			"template_preset":      resolved.templateName,
			"final_prompt":         resolved.finalPrompt,
			"copied":               resolved.copied,
			"launched":             resolved.launched,
			"source_lang":          resolved.sourceLang,
			"target_lang":          resolved.targetLang,
			"translation_mode":     resolved.translationMode,
			"translation_decision": resolved.translationDecision,
			"delivery_mode":        resolved.deliveryMode,
			"pasted":               resolved.pasted,
			"submit_mode":          resolved.submitMode,
			"submitted":            resolved.submitted,
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("encode json output: %w", err)
		}
		_, _ = fmt.Fprintln(a.stdout, string(data))
	} else {
		_, _ = fmt.Fprintln(a.stdout, resolved.finalPrompt)
	}

	if opts.compactStatus {
		a.writeCompactStatus(opts, resolved)
		return nil
	}

	if opts.noCopy {
		_, _ = fmt.Fprintf(a.stderr, "target %q ready; clipboard skipped\n", resolved.targetName)
	} else {
		_, _ = fmt.Fprintf(a.stderr, "copied prompt for target %q to clipboard\n", resolved.targetName)
	}
	if opts.launch && resolved.launched {
		_, _ = fmt.Fprintf(a.stderr, "opened %s CLI session\n", resolved.targetName)
	}
	if opts.paste && resolved.pasted {
		_, _ = fmt.Fprintf(a.stderr, "pasted prompt into %s terminal session\n", resolved.targetName)
	}
	if resolved.submitted {
		_, _ = fmt.Fprintf(a.stderr, "submitted prompt to %s\n", resolved.targetName)
	}

	return nil
}

func (a *App) executeStoredPrompt(ctx context.Context, opts runOptions, entry history.Entry) error {
	finalPrompt := entry.FinalPrompt
	if opts.rerunEdit {
		if a.editor == nil {
			return errors.New("interactive mode is unavailable: editor is not configured")
		}
		edited, err := a.editor.Edit(ctx, editor.Request{
			Initial: entry.FinalPrompt,
			Status:  "Rerun Edit | Target: " + entry.Target,
		})
		if err != nil {
			return err
		}
		finalPrompt = edited
	}

	run := resolvedRun{
		original:            entry.Original,
		translated:          entry.Translated,
		finalPrompt:         finalPrompt,
		targetName:          entry.Target,
		roleName:            entry.Role,
		templateName:        entry.TemplatePreset,
		shortcut:            entry.Shortcut,
		sourceLang:          blankDefault(entry.SourceLang, "auto"),
		targetLang:          blankDefault(entry.TargetLang, "en"),
		translationMode:     blankDefault(entry.TranslationMode, string(translate.ModeAuto)),
		translationDecision: blankDefault(entry.TranslationDecision, translate.DecisionTranslated),
		launchedTarget:      entry.LaunchedTarget,
		deliveryMode:        blankDefault(entry.DeliveryMode, "open-copy"),
		submitMode:          blankDefault(entry.SubmitMode, string(automation.SubmitManual)),
		pasted:              entry.Pasted,
		submitted:           entry.Submitted,
		engine:              blankDefault(entry.Engine, "classic"),
		parentID:            entry.ID,
		runID:               entry.RunID,
		resultType:          entry.ResultType,
		artifactRoot:        entry.ArtifactRoot,
		runStatus:           entry.RunStatus,
	}
	cfg, err := a.configLoader()
	if err != nil {
		return err
	}
	resolvedSubmitMode, err := resolveSubmitMode(opts.submitMode, cfg.Launchers[entry.Target].SubmitMode)
	if err != nil {
		return err
	}
	run.config = cfg
	run.submitMode = resolvedSubmitMode
	if opts.paste {
		run.deliveryMode = "open-copy-paste"
	} else {
		run.deliveryMode = "open-copy"
	}

	if !opts.noCopy {
		if err := a.clipboard.Copy(ctx, finalPrompt); err != nil {
			return err
		}
		run.copied = true
	}
	if err := a.applyDelivery(ctx, opts, &run); err != nil {
		return err
	}
	if opts.explain {
		a.writeExplain(run)
	}

	if opts.jsonOutput {
		payload := map[string]any{
			"original":        run.original,
			"translated":      run.translated,
			"target":          run.targetName,
			"role":            run.roleName,
			"template_preset": run.templateName,
			"final_prompt":    run.finalPrompt,
			"copied":          run.copied,
			"launched":        run.launched,
			"delivery_mode":   run.deliveryMode,
			"pasted":          run.pasted,
			"submit_mode":     run.submitMode,
			"submitted":       run.submitted,
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("encode json output: %w", err)
		}
		_, _ = fmt.Fprintln(a.stdout, string(data))
	} else {
		_, _ = fmt.Fprintln(a.stdout, finalPrompt)
	}

	if err := a.appendHistory(run); err != nil {
		return err
	}

	// Auto-save capsule after successful run.
	// Use context.Background() — the command's ctx may be cancelled before
	// the goroutine finishes, which would silently drop the auto-save.
	if latestEntry, err := a.historyStore.Latest(); err == nil {
		entry := latestEntry // capture loop variable
		go a.tryAutoSave(context.Background(), &entry)
	}

	if opts.compactStatus {
		a.writeCompactStatus(opts, run)
	}
	return nil
}

func (a *App) prepareRun(ctx context.Context, opts runOptions, text, shortcutName string) (resolvedRun, error) {
	cfg, err := a.configLoader()
	if err != nil {
		return resolvedRun{}, err
	}

	envTarget, _ := a.lookupEnv("PRTR_TARGET")
	envAPIKey, _ := a.lookupEnv("DEEPL_API_KEY")

	shortcut := config.ShortcutConfig{}
	if shortcutName != "" {
		shortcuts := a.builtInShortcutNames()
		if strings.HasPrefix(shortcutName, "take:") {
			shortcut = config.ShortcutConfig{}
		} else if !shortcuts[shortcutName] {
			return resolvedRun{}, fmt.Errorf("unknown shortcut %q", shortcutName)
		} else {
			shortcut = cfg.Shortcuts[shortcutName]
			if requestedTarget := strings.TrimSpace(opts.target); requestedTarget != "" && requestedTarget != strings.TrimSpace(shortcut.Target) {
				shortcut.Target = ""
				shortcut.TemplatePreset = ""
			}
		}
	}

	target, targetSource := resolveRunTarget(opts.target, shortcut, cfg, envTarget)
	targetConfig, ok := cfg.Targets[target]
	if !ok {
		return resolvedRun{}, fmt.Errorf("unknown target %q (available: %s)", target, strings.Join(config.AvailableTargets(cfg), ", "))
	}
	if opts.launch || opts.paste || strings.TrimSpace(opts.submitMode) != "" {
		if _, hasLauncher := cfg.Launchers[target]; !hasLauncher {
			return resolvedRun{}, fmt.Errorf("target %q does not have a launcher configured; launch, paste, and submit require a [launchers.%s] config entry", target, target)
		}
	}

	role, roleSource := resolveRunRole(opts.role, shortcut, cfg)
	rolePrompt := ""
	roleVariantSource := ""
	roleTemplatePreset := ""
	if role != "" {
		roleConfig, ok := cfg.Roles[role]
		if !ok {
			return resolvedRun{}, fmt.Errorf("unknown role %q (available: %s)", role, strings.Join(config.AvailableRoles(cfg), ", "))
		}
		rolePrompt = roleConfig.Prompt
		roleVariantSource = "base role"
		if targetOverride, ok := roleConfig.Targets[target]; ok {
			if strings.TrimSpace(targetOverride.Prompt) != "" {
				rolePrompt = targetOverride.Prompt
				roleVariantSource = "role target override"
			}
			roleTemplatePreset = strings.TrimSpace(targetOverride.TemplatePreset)
		}
	}

	templateName, templateSource := resolveRunTemplate(opts.templatePreset, shortcut, roleTemplatePreset, cfg, targetConfig, opts.preferTargetTemplate)
	layout, err := resolveTemplateLayout(templateName, targetConfig, cfg)
	if err != nil {
		return resolvedRun{}, err
	}

	apiKey, apiKeySource := config.ResolveAPIKey(envAPIKey, cfg)
	translator := a.resolveTranslator(apiKey)

	sourceLang, targetLang := resolveRunLanguages(opts, shortcut, targetConfig, cfg)
	resolvedSubmitMode, err := resolveSubmitMode(opts.submitMode, cfg.Launchers[target].SubmitMode)
	if err != nil {
		return resolvedRun{}, err
	}
	deliveryMode := "open-copy"
	if opts.paste {
		deliveryMode = "open-copy-paste"
	}
	outcome, err := translate.ApplyPolicy(ctx, translator, translate.Request{
		Text:           text,
		SourceLang:     sourceLang,
		TargetLang:     targetLang,
		ProtectedTerms: opts.protectedTerms,
	}, translate.Mode(blankDefault(opts.translationMode, string(translate.ModeAuto))))
	if err != nil {
		return resolvedRun{}, err
	}

	finalPrompt, err := prompttemplate.RenderData(layout, prompttemplate.Data{
		Prompt:       joinPromptSections(outcome.Text, opts.promptSuffix),
		Role:         rolePrompt,
		Target:       target,
		Context:      shortcut.Context,
		OutputFormat: shortcut.OutputFormat,
	})
	if err != nil {
		return resolvedRun{}, fmt.Errorf("render template for target %q: %w", target, err)
	}

	return resolvedRun{
		original:            text,
		translated:          outcome.Text,
		finalPrompt:         finalPrompt,
		targetName:          target,
		targetSource:        targetSource,
		targetConfig:        targetConfig,
		roleName:            role,
		rolePrompt:          rolePrompt,
		roleSource:          roleSource,
		roleVariantSource:   roleVariantSource,
		templateName:        templateName,
		templateSource:      templateSource,
		layout:              layout,
		context:             shortcut.Context,
		outputFormat:        shortcut.OutputFormat,
		shortcut:            shortcutName,
		apiKeySource:        apiKeySource,
		sourceLang:          outcome.SourceLang,
		targetLang:          outcome.TargetLang,
		translationMode:     blankDefault(opts.translationMode, string(translate.ModeAuto)),
		translationDecision: outcome.Decision,
		deliveryMode:        deliveryMode,
		submitMode:          resolvedSubmitMode,
		config:              cfg,
		engine:              blankDefault(opts.engine, "classic"),
		parentID:            strings.TrimSpace(opts.parentID),
		runID:               strings.TrimSpace(opts.runID),
		resultType:          strings.TrimSpace(opts.resultType),
		artifactRoot:        strings.TrimSpace(opts.artifactRoot),
		runStatus:           strings.TrimSpace(opts.runStatus),
		eventLogPath:        strings.TrimSpace(opts.eventLogPath),
		statusNotes:         append([]string{}, opts.statusNotes...),
		nextSteps:           append([]string{}, opts.nextSteps...),
	}, nil
}

func (a *App) writeExplain(run resolvedRun) {
	_, _ = fmt.Fprintln(a.stderr, "Resolved configuration:")
	_, _ = fmt.Fprintf(a.stderr, "- target: %s (%s)\n", run.targetName, run.targetSource)
	if run.roleName != "" {
		_, _ = fmt.Fprintf(a.stderr, "- role: %s (%s)\n", run.roleName, run.roleSource)
		if run.roleVariantSource != "" {
			_, _ = fmt.Fprintf(a.stderr, "- role variant: %s\n", run.roleVariantSource)
		}
	} else {
		_, _ = fmt.Fprintf(a.stderr, "- role: none (%s)\n", run.roleSource)
	}
	_, _ = fmt.Fprintf(a.stderr, "- template preset: %s (%s)\n", run.templateName, run.templateSource)
	_, _ = fmt.Fprintf(a.stderr, "- language route: %s -> %s\n", run.sourceLang, run.targetLang)
	_, _ = fmt.Fprintf(a.stderr, "- translation decision: %s\n", run.translationDecision)
	_, _ = fmt.Fprintf(a.stderr, "- delivery mode: %s\n", blankDefault(run.deliveryMode, "open-copy"))
	_, _ = fmt.Fprintf(a.stderr, "- paste: %t\n", run.pasted)
	_, _ = fmt.Fprintf(a.stderr, "- submit mode: %s\n", blankDefault(run.submitMode, string(automation.SubmitManual)))
	_, _ = fmt.Fprintf(a.stderr, "- submitted: %t\n", run.submitted)
	if run.shortcut != "" {
		_, _ = fmt.Fprintf(a.stderr, "- shortcut: %s\n", run.shortcut)
	}
	if run.config.HasUserConfig {
		_, _ = fmt.Fprintf(a.stderr, "- user config: %s\n", run.config.UserPath)
	}
	if run.config.HasProjectConfig {
		_, _ = fmt.Fprintf(a.stderr, "- project config: %s\n", run.config.ProjectPath)
	}
	if run.apiKeySource != "" {
		_, _ = fmt.Fprintf(a.stderr, "- api key: %s\n", run.apiKeySource)
	}
	if run.launchedTarget != "" {
		_, _ = fmt.Fprintf(a.stderr, "- launch target: %s\n", run.launchedTarget)
	}
	_, _ = fmt.Fprintln(a.stderr)
}

func (a *App) writeDiff(run resolvedRun) {
	_, _ = fmt.Fprintf(a.stderr, "Original:\n%s\n\nTranslated:\n%s\n\nFinal Prompt:\n%s\n\n", run.original, run.translated, run.finalPrompt)
}

func (a *App) appendHistory(run resolvedRun) error {
	if a.historyStore == nil {
		return nil
	}

	return a.historyStore.Append(history.Entry{
		Original:            run.original,
		Translated:          run.translated,
		FinalPrompt:         run.finalPrompt,
		Target:              run.targetName,
		Role:                run.roleName,
		TemplatePreset:      run.templateName,
		Shortcut:            run.shortcut,
		SourceLang:          run.sourceLang,
		TargetLang:          run.targetLang,
		TranslationMode:     run.translationMode,
		TranslationDecision: run.translationDecision,
		LaunchedTarget:      run.launchedTarget,
		DeliveryMode:        run.deliveryMode,
		Pasted:              run.pasted,
		SubmitMode:          run.submitMode,
		Submitted:           run.submitted,
		ParentID:            run.parentID,
		RunID:               run.runID,
		Engine:              run.engine,
		ResultType:          run.resultType,
		ArtifactRoot:        run.artifactRoot,
		RunStatus:           run.runStatus,
	})
}

func (a *App) applyDelivery(ctx context.Context, opts runOptions, run *resolvedRun) error {
	if run == nil {
		return nil
	}
	if opts.paste {
		run.deliveryMode = "open-copy-paste"
	} else {
		run.deliveryMode = "open-copy"
	}
	launcherCfg, ok := run.config.Launchers[run.targetName]
	if !ok {
		launcherCfg = config.LauncherConfig{}
	}
	if launcherCfg.PasteDelayMS < 0 {
		return fmt.Errorf("launcher %q has negative paste_delay_ms", run.targetName)
	}

	if opts.launch {
		if a.launcher == nil {
			return errors.New("launch mode is unavailable: launcher is not configured")
		}
		if strings.TrimSpace(launcherCfg.Command) == "" {
			return fmt.Errorf("launcher is not configured for target %q", run.targetName)
		}
		if err := a.launcher.Launch(ctx, launcher.Request{
			Command: launcherCfg.Command,
			Args:    launcherCfg.Args,
		}); err != nil {
			return err
		}
		run.launched = true
		run.launchedTarget = run.targetName
	}

	if opts.paste {
		if a.automator == nil {
			return errors.New("paste mode is unavailable: automator is not configured")
		}
		if run.submitMode == string(automation.SubmitAuto) {
			return fmt.Errorf("--submit=auto is not supported yet")
		}
		autoReq := automation.Request{
			Target:           run.targetName,
			TerminalApp:      "Terminal",
			PasteDelay:       time.Duration(maxInt(0, launcherCfg.PasteDelayMS)) * time.Millisecond,
			RequireClipboard: true,
			SubmitMode:       automation.SubmitMode(run.submitMode),
		}
		if err := a.automator.Paste(ctx, autoReq); err != nil {
			return err
		}
		run.pasted = true
	}

	if opts.paste && strings.TrimSpace(run.submitMode) == string(automation.SubmitConfirm) {
		if runtime.GOOS != "darwin" {
			return fmt.Errorf("--submit=confirm is only supported on macOS right now")
		}
		if a.submitConfirmer == nil {
			return errors.New("submit confirmation is unavailable")
		}
		confirmed, err := a.submitConfirmer.ConfirmSubmit(run.targetName)
		if err != nil {
			return err
		}
		if confirmed {
			if a.automator == nil {
				return errors.New("submit mode is unavailable: automator is not configured")
			}
			if err := a.automator.Submit(ctx, automation.Request{
				Target:      run.targetName,
				TerminalApp: "Terminal",
				SubmitMode:  automation.SubmitConfirm,
			}); err != nil {
				return err
			}
			run.submitted = true
		}
	}

	return nil
}

func (a *App) latestHistoryEntry() (history.Entry, error) {
	if a.historyStore == nil {
		return history.Entry{}, errors.New("history store is not configured")
	}

	entry, err := a.historyStore.Latest()
	if err != nil {
		if errors.Is(err, history.ErrNotFound) {
			return history.Entry{}, errors.New("no history yet; run `prtr go` first")
		}
		return history.Entry{}, err
	}

	return entry, nil
}

func (a *App) writeCompactStatus(opts runOptions, run resolvedRun) {
	mode := blankDefault(opts.surfaceMode, "ask")
	inputSource := blankDefault(opts.surfaceInput, "prompt")

	deliveryLabel := blankDefault(opts.surfaceDelivery, run.deliveryMode)
	parts := []string{mode, run.targetName, inputSource, deliveryLabel, run.sourceLang + "->" + run.targetLang}
	if strings.TrimSpace(run.engine) == "deep" {
		parts = append(parts, blankDefault(run.runStatus, string(deep.RunStatusCompleted)))
	}
	_, _ = fmt.Fprintf(a.stderr, "-> %s\n", strings.Join(parts, " | "))
	for _, note := range run.statusNotes {
		_, _ = fmt.Fprintf(a.stderr, "   %s\n", note)
	}
	for _, next := range run.nextSteps {
		_, _ = fmt.Fprintf(a.stderr, "   next: %s\n", next)
	}
}

func (a *App) appendDeepEvent(path string, event deep.Event) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return deep.AppendEvent(path, event)
}

func (a *App) diagnoseClipboard() error {
	if diagnoser, ok := a.clipboard.(clipboard.Diagnoser); ok {
		return diagnoser.Diagnose()
	}
	return nil
}

func (a *App) resolveTranslator(apiKey string) translate.Translator {
	if a.translatorFactory != nil {
		if strings.TrimSpace(apiKey) == "" {
			return nil
		}
		return a.translatorFactory(apiKey)
	}
	return a.translator
}

func (a *App) builtInShortcutNames() map[string]bool {
	return map[string]bool{
		"ask":    true,
		"review": true,
		"fix":    true,
		"design": true,
	}
}

func parseGoCommand(args []string, builtInShortcuts map[string]bool) (goCommandOptions, error) {
	command := goCommandOptions{mode: "ask"}
	if len(args) > 0 && builtInShortcuts[args[0]] {
		command.mode = args[0]
		args = args[1:]
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--edit":
			command.edit = true
		case arg == "--interactive" || arg == "-i":
			command.edit = true
		case arg == "--dry-run":
			command.dryRun = true
		case arg == "--no-copy":
			command.noCopy = true
		case arg == "--no-context":
			command.noContext = true
		case arg == "--to" || arg == "--app" || arg == "--target" || arg == "-t":
			i++
			if i >= len(args) {
				return goCommandOptions{}, usageError{message: fmt.Sprintf("%s requires a value", arg), helpText: goHelpText()}
			}
			command.app = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--to="):
			command.app = strings.TrimSpace(strings.TrimPrefix(arg, "--to="))
		case strings.HasPrefix(arg, "--app="):
			command.app = strings.TrimSpace(strings.TrimPrefix(arg, "--app="))
		case strings.HasPrefix(arg, "--target="):
			command.app = strings.TrimSpace(strings.TrimPrefix(arg, "--target="))
		case arg == "--":
			command.prompt = append(command.prompt, args[i+1:]...)
			return command, nil
		case strings.HasPrefix(arg, "-"):
			return goCommandOptions{}, usageError{message: fmt.Sprintf("unknown go flag %q", arg), helpText: goHelpText()}
		default:
			command.prompt = append(command.prompt, arg)
		}
	}
	return command, nil
}

func parseStartCommand(args []string) (startCommandOptions, error) {
	command := startCommandOptions{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--dry-run":
			command.dryRun = true
		case arg == "--to" || arg == "--app" || arg == "--target" || arg == "-t":
			i++
			if i >= len(args) {
				return startCommandOptions{}, usageError{message: fmt.Sprintf("%s requires a value", arg), helpText: startHelpText()}
			}
			command.app = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--to="):
			command.app = strings.TrimSpace(strings.TrimPrefix(arg, "--to="))
		case strings.HasPrefix(arg, "--app="):
			command.app = strings.TrimSpace(strings.TrimPrefix(arg, "--app="))
		case strings.HasPrefix(arg, "--target="):
			command.app = strings.TrimSpace(strings.TrimPrefix(arg, "--target="))
		case arg == "--":
			command.prompt = append(command.prompt, args[i+1:]...)
			return command, nil
		case strings.HasPrefix(arg, "-"):
			return startCommandOptions{}, usageError{message: fmt.Sprintf("unknown start flag %q", arg), helpText: startHelpText()}
		default:
			command.prompt = append(command.prompt, arg)
		}
	}

	return command, nil
}

func parseReplayCommand(args []string, requireApp bool) (replayCommandOptions, error) {
	command := replayCommandOptions{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--edit":
			command.edit = true
		case arg == "--interactive" || arg == "-i":
			command.edit = true
		case arg == "--dry-run":
			command.dryRun = true
		case arg == "--no-copy":
			command.noCopy = true
		case arg == "--no-context":
			command.noContext = true
		case arg == "--to" || arg == "--app" || arg == "--target" || arg == "-t":
			i++
			if i >= len(args) {
				return replayCommandOptions{}, usageError{message: fmt.Sprintf("%s requires a value", arg), helpText: swapHelpText()}
			}
			command.app = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--to="):
			command.app = strings.TrimSpace(strings.TrimPrefix(arg, "--to="))
		case strings.HasPrefix(arg, "--app="):
			command.app = strings.TrimSpace(strings.TrimPrefix(arg, "--app="))
		case strings.HasPrefix(arg, "--target="):
			command.app = strings.TrimSpace(strings.TrimPrefix(arg, "--target="))
		case arg == "--":
			command.prompt = append(command.prompt, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(arg, "-"):
			return replayCommandOptions{}, usageError{message: fmt.Sprintf("unknown replay flag %q", arg), helpText: swapHelpText()}
		default:
			if requireApp && strings.TrimSpace(command.app) == "" {
				command.app = strings.TrimSpace(arg)
				continue
			}
			command.prompt = append(command.prompt, arg)
		}
	}

	if requireApp && strings.TrimSpace(command.app) == "" {
		return replayCommandOptions{}, usageError{message: "swap requires a target app such as claude, codex, or gemini", helpText: swapHelpText()}
	}
	return command, nil
}

func parseTakeCommand(args []string) (takeCommandOptions, error) {
	command := takeCommandOptions{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--edit":
			command.edit = true
		case arg == "--interactive" || arg == "-i":
			command.edit = true
		case arg == "--dry-run":
			command.dryRun = true
		case arg == "--dip", arg == "--deep":
			command.deep = true
		case arg == "--llm":
			command.llm = true
		case strings.HasPrefix(arg, "--llm="):
			command.llm = true
			command.llmProvider = strings.TrimPrefix(arg, "--llm=")
		case arg == "--to" || arg == "--app" || arg == "--target" || arg == "-t":
			i++
			if i >= len(args) {
				return takeCommandOptions{}, usageError{message: fmt.Sprintf("%s requires a value", arg), helpText: takeHelpText()}
			}
			command.app = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--to="):
			command.app = strings.TrimSpace(strings.TrimPrefix(arg, "--to="))
		case strings.HasPrefix(arg, "--app="):
			command.app = strings.TrimSpace(strings.TrimPrefix(arg, "--app="))
		case strings.HasPrefix(arg, "--target="):
			command.app = strings.TrimSpace(strings.TrimPrefix(arg, "--target="))
		case strings.HasPrefix(arg, "-"):
			return takeCommandOptions{}, usageError{message: fmt.Sprintf("unknown take flag %q", arg), helpText: takeHelpText()}
		case strings.TrimSpace(command.action) == "":
			command.action = normalizeTakeAction(arg)
		default:
			return takeCommandOptions{}, usageError{message: fmt.Sprintf("take only accepts one action; got unexpected argument %q", arg), helpText: takeHelpText()}
		}
	}

	if command.action == "" {
		return takeCommandOptions{}, usageError{message: "take requires an action such as patch, test, commit, summary, issue, or plan", helpText: takeHelpText()}
	}
	if !isSupportedTakeAction(command.action) {
		return takeCommandOptions{}, usageError{message: fmt.Sprintf("unknown take action %q (available: patch, test, commit, summary, clarify, issue, plan, debug, refactor)", command.action), helpText: takeHelpText()}
	}
	if command.deep && !isSupportedDeepAction(command.action) {
		return takeCommandOptions{}, usageError{message: fmt.Sprintf("deep execution supports: patch, test, debug, refactor; got %q", command.action), helpText: takeHelpText()}
	}

	return command, nil
}

func parseLearnCommand(args []string) (learnCommandOptions, error) {
	command := learnCommandOptions{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--dry-run":
			command.dryRun = true
		case arg == "--reset":
			command.reset = true
		case arg == "--":
			command.paths = append(command.paths, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(arg, "-"):
			return learnCommandOptions{}, usageError{message: fmt.Sprintf("unknown learn flag %q", arg), helpText: learnHelpText()}
		default:
			command.paths = append(command.paths, arg)
		}
	}

	return command, nil
}

func resolveSurfaceInput(promptParts []string, stdin io.Reader, stdinPiped bool, attachStdin bool) (string, string, error) {
	prompt := strings.TrimSpace(strings.Join(promptParts, " "))
	if !stdinPiped {
		if prompt == "" {
			return "", "", input.ErrNoInput
		}
		return prompt, "prompt", nil
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", "", err
	}
	stdinText := strings.TrimSpace(string(data))

	switch {
	case prompt == "" && stdinText == "":
		return "", "", input.ErrNoInput
	case prompt == "":
		return stdinText, "stdin", nil
	case stdinText == "" || !attachStdin:
		return prompt, "prompt", nil
	default:
		return prompt + "\n\nEvidence:\n" + stdinText, "prompt+stdin", nil
	}
}

func surfaceDeliveryLabel(dryRun bool) string {
	if dryRun {
		return "preview"
	}
	return "launch+paste"
}

func formatRepoContext(summary repoctx.Summary) string {
	lines := []string{"Repo context:"}

	if strings.TrimSpace(summary.RepoName) != "" {
		lines = append(lines, "repo: "+strings.TrimSpace(summary.RepoName))
	}
	if strings.TrimSpace(summary.Branch) != "" {
		lines = append(lines, "branch: "+strings.TrimSpace(summary.Branch))
	}
	if len(summary.Changes) == 0 {
		lines = append(lines, "changes:", "- working tree clean")
	} else {
		lines = append(lines, "changes:")
		for _, change := range summary.Changes {
			lines = append(lines, "- "+change)
		}
		if summary.Truncated > 0 {
			lines = append(lines, fmt.Sprintf("- +%d more", summary.Truncated))
		}
	}

	return strings.Join(lines, "\n")
}

func formatProtectedTerms(terms []string) string {
	if len(terms) == 0 {
		return ""
	}
	limit := minInt(20, len(terms))
	return "Protected project terms: " + strings.Join(terms[:limit], ", ")
}

func joinPromptSections(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}

	return strings.Join(filtered, "\n\n")
}

func parseRunOptions(args []string) (runOptions, []string, error) {
	fs := flag.NewFlagSet("prtr", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts runOptions
	fs.StringVar(&opts.target, "target", "", "target profile name")
	fs.StringVar(&opts.target, "t", "", "target profile name")
	fs.StringVar(&opts.sourceLang, "source-lang", "", "source language code or auto")
	fs.StringVar(&opts.targetLang, "lang", "", "target language code (e.g. en, ja, zh)")
	fs.StringVar(&opts.targetLang, "to", "", "target language code (deprecated: use --lang)")
	fs.StringVar(&opts.role, "role", "", "role profile alias")
	fs.StringVar(&opts.role, "r", "", "role profile alias")
	fs.StringVar(&opts.templatePreset, "template", "", "template preset name")
	fs.StringVar(&opts.translationMode, "translation-mode", "", "translation mode: auto, force, or skip")
	fs.BoolVar(&opts.noCopy, "no-copy", false, "skip clipboard copy")
	fs.BoolVar(&opts.showOriginal, "show-original", false, "print the original input to stderr")
	fs.BoolVar(&opts.interactive, "interactive", false, "open an interactive editor before writing the final prompt")
	fs.BoolVar(&opts.interactive, "i", false, "open an interactive editor before writing the final prompt")
	fs.BoolVar(&opts.explain, "explain", false, "print resolved config details to stderr")
	fs.BoolVar(&opts.diff, "diff", false, "print original, translated, and final prompt to stderr")
	fs.BoolVar(&opts.jsonOutput, "json", false, "emit JSON output")
	fs.BoolVar(&opts.launch, "launch", false, "launch the target CLI after copying the prompt")
	fs.BoolVar(&opts.paste, "paste", false, "launch the target CLI and paste the copied prompt on macOS")
	fs.StringVar(&opts.submitMode, "submit", "", "submit mode: manual, confirm, or auto")
	fs.BoolVar(&opts.rerunEdit, "edit", false, "edit the stored final prompt before rerun")

	if err := fs.Parse(args); err != nil {
		return runOptions{}, nil, usageError{message: err.Error()}
	}
	if opts.paste {
		opts.launch = true
	}
	if strings.TrimSpace(opts.submitMode) != "" && !opts.paste {
		return runOptions{}, nil, usageError{message: "--submit requires --paste"}
	}
	if opts.launch && opts.noCopy {
		return runOptions{}, nil, usageError{message: "--launch requires clipboard copy; remove --no-copy"}
	}
	if opts.paste && opts.noCopy {
		return runOptions{}, nil, usageError{message: "--paste requires clipboard copy; remove --no-copy"}
	}
	if opts.submitMode != "" && opts.noCopy {
		return runOptions{}, nil, usageError{message: "--submit requires clipboard copy; remove --no-copy"}
	}

	return opts, fs.Args(), nil
}

func resolveRunTarget(cliTarget string, shortcut config.ShortcutConfig, cfg config.Config, envTarget string) (string, string) {
	if target := strings.TrimSpace(cliTarget); target != "" {
		return target, "cli flag"
	}
	if target := strings.TrimSpace(shortcut.Target); target != "" {
		return target, "shortcut"
	}
	if target := strings.TrimSpace(cfg.DefaultTarget); target != "" {
		source := cfg.DefaultTargetSource
		if source == "" {
			source = "config default"
		}
		return target, source
	}
	if target := strings.TrimSpace(envTarget); target != "" {
		return target, "environment"
	}
	return "claude", "built-in default"
}

func resolveRunRole(cliRole string, shortcut config.ShortcutConfig, cfg config.Config) (string, string) {
	if role := strings.TrimSpace(cliRole); role != "" {
		return role, "cli flag"
	}
	if role := strings.TrimSpace(shortcut.Role); role != "" {
		return role, "shortcut"
	}
	if role := strings.TrimSpace(cfg.DefaultRole); role != "" {
		source := cfg.DefaultRoleSource
		if source == "" {
			source = "config default"
		}
		return role, source
	}
	return "", "not set"
}

func resolveRunTemplate(cliPreset string, shortcut config.ShortcutConfig, roleTemplatePreset string, cfg config.Config, target config.TargetConfig, preferTargetTemplate bool) (string, string) {
	if preset := strings.TrimSpace(cliPreset); preset != "" {
		return preset, "cli flag"
	}
	if preset := strings.TrimSpace(shortcut.TemplatePreset); preset != "" {
		return preset, "shortcut"
	}
	if preset := strings.TrimSpace(roleTemplatePreset); preset != "" {
		return preset, "role target override"
	}
	if preferTargetTemplate {
		if preset := strings.TrimSpace(target.DefaultTemplatePreset); preset != "" {
			return preset, "target default"
		}
	}
	if preset := strings.TrimSpace(cfg.DefaultTemplatePreset); preset != "" {
		source := cfg.DefaultPresetSource
		if source == "" {
			source = "config default"
		}
		return preset, source
	}
	if !preferTargetTemplate {
		if preset := strings.TrimSpace(target.DefaultTemplatePreset); preset != "" {
			return preset, "target default"
		}
	}
	return "", "target template"
}

func resolveRunLanguages(opts runOptions, shortcut config.ShortcutConfig, target config.TargetConfig, cfg config.Config) (string, string) {
	sourceLang := strings.TrimSpace(opts.sourceLang)
	if sourceLang == "" {
		sourceLang = strings.TrimSpace(cfg.TranslationSourceLang)
	}
	if sourceLang == "" {
		sourceLang = "auto"
	}

	targetLang := strings.TrimSpace(opts.targetLang)
	if targetLang == "" {
		targetLang = strings.TrimSpace(shortcut.TranslationTargetLang)
	}
	if targetLang == "" {
		targetLang = strings.TrimSpace(target.TranslationTargetLang)
	}
	if targetLang == "" {
		targetLang = strings.TrimSpace(cfg.TranslationTargetLang)
	}
	if targetLang == "" {
		targetLang = "en"
	}

	return strings.ToLower(sourceLang), strings.ToLower(targetLang)
}

func resolveTemplateLayout(templateName string, target config.TargetConfig, cfg config.Config) (string, error) {
	if strings.TrimSpace(templateName) != "" {
		preset, ok := cfg.TemplatePresets[templateName]
		if !ok {
			return "", fmt.Errorf("unknown template preset %q (available: %s)", templateName, strings.Join(config.AvailableTemplatePresets(cfg), ", "))
		}
		return preset.Template, nil
	}
	if strings.TrimSpace(target.Template) != "" {
		return target.Template, nil
	}
	return "", errors.New("no template could be resolved")
}

func validateTargetDefaults(cfg config.Config) error {
	for name, target := range cfg.Targets {
		if target.Template == "" && target.DefaultTemplatePreset == "" {
			return fmt.Errorf("target %q has no template or default template preset", name)
		}
		if target.DefaultTemplatePreset != "" {
			if _, ok := cfg.TemplatePresets[target.DefaultTemplatePreset]; !ok {
				return fmt.Errorf("target %q references unknown template preset %q", name, target.DefaultTemplatePreset)
			}
		}
	}
	return nil
}

func validateTemplatePresets(cfg config.Config) error {
	for name, preset := range cfg.TemplatePresets {
		if err := prompttemplate.Validate(preset.Template); err != nil {
			return fmt.Errorf("template preset %q: %w", name, err)
		}
	}
	return nil
}

func validateProfiles(cfg config.Config) error {
	for name, profile := range cfg.Profiles {
		if profile.Target != "" {
			if _, ok := cfg.Targets[profile.Target]; !ok {
				return fmt.Errorf("profile %q references unknown target %q", name, profile.Target)
			}
		}
		if profile.Role != "" {
			if _, ok := cfg.Roles[profile.Role]; !ok {
				return fmt.Errorf("profile %q references unknown role %q", name, profile.Role)
			}
		}
		if profile.TemplatePreset != "" {
			if _, ok := cfg.TemplatePresets[profile.TemplatePreset]; !ok {
				return fmt.Errorf("profile %q references unknown template preset %q", name, profile.TemplatePreset)
			}
		}
	}
	return nil
}

func validateShortcuts(cfg config.Config) error {
	for name, shortcut := range cfg.Shortcuts {
		if shortcut.Target != "" {
			if _, ok := cfg.Targets[shortcut.Target]; !ok {
				return fmt.Errorf("shortcut %q references unknown target %q", name, shortcut.Target)
			}
		}
		if shortcut.Role != "" {
			if _, ok := cfg.Roles[shortcut.Role]; !ok {
				return fmt.Errorf("shortcut %q references unknown role %q", name, shortcut.Role)
			}
		}
		if shortcut.TemplatePreset != "" {
			if _, ok := cfg.TemplatePresets[shortcut.TemplatePreset]; !ok {
				return fmt.Errorf("shortcut %q references unknown template preset %q", name, shortcut.TemplatePreset)
			}
		}
	}
	return nil
}

func validateLaunchers(cfg config.Config) error {
	for name, launcherCfg := range cfg.Launchers {
		if strings.TrimSpace(launcherCfg.Command) == "" {
			return fmt.Errorf("launcher %q has an empty command", name)
		}
		if launcherCfg.PasteDelayMS < 0 {
			return fmt.Errorf("launcher %q has negative paste_delay_ms", name)
		}
		switch blankDefault(launcherCfg.SubmitMode, string(automation.SubmitManual)) {
		case string(automation.SubmitManual), string(automation.SubmitConfirm), string(automation.SubmitAuto):
		default:
			return fmt.Errorf("launcher %q has invalid submit_mode %q", name, launcherCfg.SubmitMode)
		}
	}
	return nil
}

func resolveSubmitMode(cliMode, configMode string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(cliMode))
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(configMode))
	}
	if mode == "" {
		mode = string(automation.SubmitManual)
	}
	switch mode {
	case string(automation.SubmitManual), string(automation.SubmitConfirm), string(automation.SubmitAuto):
		return mode, nil
	default:
		return "", fmt.Errorf("invalid submit mode %q (expected manual, confirm, or auto)", mode)
	}
}

func interactiveStatus(run resolvedRun) string {
	parts := []string{"Target: " + run.targetName}
	if run.roleName != "" {
		parts = append(parts, "Role: "+run.roleName)
	}
	if run.templateName != "" {
		parts = append(parts, "Template: "+run.templateName)
	}
	return strings.Join(parts, " | ")
}

func promptInput(reader *bufio.Reader, output io.Writer, label string) (string, error) {
	_, _ = fmt.Fprintf(output, "%s: ", label)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read setup input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func promptChoice(reader *bufio.Reader, output io.Writer, label string, choices []string, defaultValue string, allowBlank bool) (string, error) {
	suffix := strings.Join(choices, ", ")
	if allowBlank {
		suffix += ", none"
	}
	if defaultValue != "" {
		suffix += fmt.Sprintf(" [default: %s]", defaultValue)
	}
	value, err := promptInput(reader, output, fmt.Sprintf("%s (%s)", label, suffix))
	if err != nil {
		return "", err
	}
	if value == "" {
		return defaultValue, nil
	}
	if allowBlank && strings.EqualFold(value, "none") {
		return "", nil
	}
	for _, choice := range choices {
		if value == choice {
			return value, nil
		}
	}
	_, _ = fmt.Fprintf(output, "  unrecognized value %q — using default %q\n", value, defaultValue)
	return defaultValue, nil
}

type languageOption struct {
	Label       string
	Value       string
	Description string
}

func promptLanguage(reader *bufio.Reader, output io.Writer, label string, choices []languageOption, defaultValue string, allowAuto bool) (string, error) {
	parts := make([]string, 0, len(choices)+1)
	for _, choice := range choices {
		if choice.Description != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", choice.Label, choice.Description))
		} else {
			parts = append(parts, choice.Label)
		}
	}
	parts = append(parts, "custom")
	value, err := promptInput(reader, output, fmt.Sprintf("%s (%s) [default: %s]", label, strings.Join(parts, ", "), defaultValue))
	if err != nil {
		return "", err
	}
	if value == "" {
		return defaultValue, nil
	}
	value = strings.ToLower(strings.TrimSpace(value))
	if allowAuto && value == "auto" {
		return "auto", nil
	}
	for _, choice := range choices {
		if value == choice.Label || value == choice.Value {
			return choice.Value, nil
		}
	}
	return value, nil
}

func truncateOneLine(text string, limit int) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
}

func blankDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func historyFlags(entry history.Entry) string {
	flags := make([]string, 0, 2)
	if entry.Pinned {
		flags = append(flags, "pinned")
	}
	if entry.Favorite {
		flags = append(flags, "favorite")
	}
	if len(flags) == 0 {
		return "-"
	}
	return strings.Join(flags, ",")
}

func pinVerb(pinned bool) string {
	if pinned {
		return "pinned"
	}
	return "unpinned"
}

func favoriteVerb(favorite bool) string {
	if favorite {
		return "favorited"
	}
	return "unfavorited"
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			return true
		}
	}
	return false
}

func stringPtr(value string) *string {
	v := value
	return &v
}

func (a *App) lookupEnvOrDefault(key, fallback string) string {
	if v, ok := a.lookupEnv(key); ok {
		return v
	}
	return fallback
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (e usageError) Error() string {
	helpText := e.helpText
	if strings.TrimSpace(helpText) == "" {
		helpText = rootHelpText()
	}
	return fmt.Sprintf("%s\n\n%s", e.message, helpText)
}


func isSupportedDeepAction(action string) bool {
	switch normalizeTakeAction(action) {
	case "patch", "test", "debug", "refactor":
		return true
	default:
		return false
	}
}

func isSupportedTakeAction(action string) bool {
	switch normalizeTakeAction(action) {
	case "patch", "test", "commit", "summary", "clarify", "issue", "plan", "debug", "refactor":
		return true
	default:
		return false
	}
}

func normalizeTakeAction(action string) string {
	return strings.ToLower(strings.TrimSpace(action))
}

func takePrompt(action, clipboardText string) string {
	var goal string
	switch normalizeTakeAction(action) {
	case "patch":
		goal = "Turn the material below into an implementation prompt. Focus on concrete code changes, files to touch, constraints, and validation steps. Avoid unnecessary explanation."
	case "test":
		goal = "Turn the material below into a testing prompt. Ask for targeted unit, integration, or regression tests and the most important verification steps."
	case "commit":
		goal = "Turn the material below into a prompt that produces a single concise commit message. Ask for one strong commit message only."
	case "summary":
		goal = "Turn the material below into a prompt that produces a short reusable summary. Keep it compact and easy to share."
	case "clarify":
		goal = "Turn the material below into a prompt that asks for a clearer explanation. Request plain language, assumptions, and a couple of concrete examples."
	case "issue":
		goal = "Turn the material below into a prompt that produces a clean issue or task description with summary, context, acceptance criteria, and open questions."
	case "plan":
		goal = "Turn the material below into a prompt that produces a step-by-step implementation plan with risks, dependencies, file areas, and validation steps."
	case "debug":
		goal = "Turn the material below into a debugging prompt. Ask for root cause identification, a minimal reproduction, and a targeted fix with verification steps."
	case "refactor":
		goal = "Turn the material below into a refactoring prompt. Ask for a safe, scoped refactor with rollback strategy and test coverage plan."
	default:
		goal = "Turn the material below into the next useful prompt."
	}

	return strings.Join([]string{
		goal,
		"",
		"Source material:",
		clipboardText,
	}, "\n")
}

func resolvedLLMProvider(cmd takeCommandOptions, cfg config.Config) string {
	if cmd.llmProvider != "" {
		return cmd.llmProvider
	}
	if cmd.llm {
		return cfg.LLMProvider
	}
	return ""
}

func resolvedLLMAPIKey(cmd takeCommandOptions, cfg config.Config) string {
	if cmd.llm || cmd.llmProvider != "" {
		return cfg.LLMAPIKey
	}
	return ""
}

func (a *App) runSave(label, note string) error {
	cfg, err := a.configLoader()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if !cfg.Memory.Enabled {
		_, _ = fmt.Fprintln(a.stdout, "capsules are disabled (memory.enabled = false)")
		return nil
	}

	repoRoot, err := a.resolveRepoRoot()
	if err != nil {
		return fmt.Errorf("find repo root: %w", err)
	}

	ctx := context.Background()
	repoSummary, err := a.repoContext.Collect(ctx)
	if err != nil {
		return fmt.Errorf("collect repo context: %w", err)
	}

	// Latest history entry is optional — proceed without it on error.
	var histEntry *history.Entry
	if a.historyStore != nil {
		if e, err := a.historyStore.Latest(); err == nil {
			histEntry = &e
		}
	}

	in := capsule.BuildInput{
		Label:        label,
		Note:         note,
		Kind:         capsule.KindManual,
		HistoryEntry: histEntry,
		RepoSummary:  repoSummary,
		RepoRoot:     repoRoot,
	}
	c := capsule.Build(in)

	dir, err := capsule.DefaultDir(repoRoot)
	if err != nil {
		return fmt.Errorf("resolve capsule dir: %w", err)
	}
	store := capsule.NewStore(dir)
	if err := store.Save(c); err != nil {
		return fmt.Errorf("save capsule: %w", err)
	}

	todoCount := len(c.Work.Todos)
	displayLabel := c.Label
	if displayLabel == "" {
		displayLabel = "[auto]"
	}
	_, _ = fmt.Fprintf(a.stdout, "✓ capsule saved  %s  %s\n  branch: %s  sha: %s  %d todos\n",
		c.ID, displayLabel, c.Repo.Branch, c.Repo.HeadSHA, todoCount)

	if cfg.Memory.PruneOnWrite {
		_ = a.runPrune("", false) // best-effort, ignore error
	}

	return nil
}

func (a *App) runResume(ctx context.Context, id, to string, dryRun bool) error {
	repoRoot, err := a.resolveRepoRoot()
	if err != nil {
		return fmt.Errorf("find repo root: %w", err)
	}

	dir, err := capsule.DefaultDir(repoRoot)
	if err != nil {
		return fmt.Errorf("resolve capsule dir: %w", err)
	}
	store := capsule.NewStore(dir)

	var c capsule.Capsule
	if id == "" {
		c, err = store.Latest()
	} else {
		c, err = store.Load(id)
	}
	if err != nil {
		if errors.Is(err, capsule.ErrNotFound) {
			return fmt.Errorf("no capsules found for this repo — run `prtr save` first")
		}
		return fmt.Errorf("load capsule: %w", err)
	}

	// Detect drift
	current, _ := a.repoContext.Collect(ctx)
	drift := capsule.DetectDrift(c, current)

	// Resolve target
	target := to
	if target == "" {
		target = c.Session.TargetApp
	}
	if target == "" {
		cfg, _ := a.configLoader()
		target = cfg.DefaultTarget
	}
	if target == "" {
		target = "claude"
	}

	// Render resume prompt
	prompt := capsule.RenderResumePrompt(c, target, drift)

	if dryRun {
		_, _ = fmt.Fprintln(a.stdout, prompt)
		return nil
	}

	// Deliver: copy to clipboard + launch target app
	if err := a.clipboard.Copy(ctx, prompt); err != nil {
		return fmt.Errorf("copy to clipboard: %w", err)
	}

	cfg, err := a.configLoader()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	launcherCfg, hasLauncher := cfg.Launchers[target]
	if hasLauncher && strings.TrimSpace(launcherCfg.Command) != "" && a.launcher != nil {
		if err := a.launcher.Launch(ctx, launcher.Request{
			Command: launcherCfg.Command,
			Args:    launcherCfg.Args,
		}); err != nil {
			return fmt.Errorf("launch %s: %w", target, err)
		}
	}

	// Persist the target that was actually used so that `prtr status` and the
	// next resume show the correct app, especially when --to overrode the
	// capsule's saved target.
	if target != c.Session.TargetApp {
		_ = store.Update(c.ID, func(cap *capsule.Capsule) {
			cap.Session.TargetApp = target
		})
	}

	displayLabel := c.Label
	if displayLabel == "" {
		displayLabel = "[auto]"
	}
	_, _ = fmt.Fprintf(a.stdout, "✓ resume prompt copied  %s  → %s\n", displayLabel, target)
	if drift.HasDrift() {
		_, _ = fmt.Fprintln(a.stdout, "  ⚠ repo has drifted since save — review the warning in the prompt")
	}

	if cfg.Memory.PruneOnResume {
		_ = a.runPrune("", false)
	}

	return nil
}

func (a *App) runCapsuleStatus() error {
	repoRoot, err := a.resolveRepoRoot()
	if err != nil {
		return fmt.Errorf("find repo root: %w", err)
	}

	dir, err := capsule.DefaultDir(repoRoot)
	if err != nil {
		return fmt.Errorf("resolve capsule dir: %w", err)
	}
	store := capsule.NewStore(dir)

	c, err := store.Latest()
	if err != nil {
		if errors.Is(err, capsule.ErrNotFound) {
			_, _ = fmt.Fprintln(a.stdout, "no capsules saved for this repo")
			return nil
		}
		return fmt.Errorf("load latest capsule: %w", err)
	}

	ctx := context.Background()
	current, _ := a.repoContext.Collect(ctx)

	drift := capsule.DetectDrift(c, current)

	displayLabel := c.Label
	if displayLabel == "" {
		displayLabel = "[auto]"
	}
	_, _ = fmt.Fprintf(a.stdout, "store:      %s\n", dir)
	_, _ = fmt.Fprintf(a.stdout, "last save:  %s  %s\n",
		c.CreatedAt.Local().Format("2006-01-02 15:04"), displayLabel)

	if drift.BranchChanged {
		_, _ = fmt.Fprintf(a.stdout, "branch:     %s → %s  ⚠ branch changed\n",
			drift.SavedBranch, drift.CurrentBranch)
	} else {
		_, _ = fmt.Fprintf(a.stdout, "branch:     %s  (no drift)\n", c.Repo.Branch)
	}

	if drift.SHAChanged {
		_, _ = fmt.Fprintf(a.stdout, "sha:        %s → %s  ⚠ commits since save\n",
			drift.SavedSHA, drift.CurrentSHA)
	} else {
		_, _ = fmt.Fprintf(a.stdout, "sha:        %s  (no drift)\n", c.Repo.HeadSHA)
	}

	open, done := 0, 0
	for _, t := range c.Work.Todos {
		if t.Status == "completed" {
			done++
		} else {
			open++
		}
	}
	_, _ = fmt.Fprintf(a.stdout, "todos:      %d open, %d done\n", open, done)
	_, _ = fmt.Fprintf(a.stdout, "target:     %s\n", c.Session.TargetApp)

	return nil
}

func (a *App) runCapsuleList() error {
	repoRoot, err := a.resolveRepoRoot()
	if err != nil {
		return fmt.Errorf("find repo root: %w", err)
	}

	dir, err := capsule.DefaultDir(repoRoot)
	if err != nil {
		return fmt.Errorf("resolve capsule dir: %w", err)
	}
	store := capsule.NewStore(dir)

	caps, err := store.List()
	if err != nil {
		return fmt.Errorf("list capsules: %w", err)
	}

	if len(caps) == 0 {
		_, _ = fmt.Fprintln(a.stdout, "no capsules saved for this repo")
		return nil
	}

	for _, c := range caps {
		label := c.Label
		if label == "" {
			label = "[auto]"
		}
		pinMark := ""
		if c.Pinned {
			pinMark = "  📌"
		}
		_, _ = fmt.Fprintf(a.stdout, "%s  %s  %-30s  %-8s  %dt%s\n",
			c.ID,
			c.CreatedAt.Local().Format("2006-01-02 15:04"),
			label,
			c.Session.TargetApp,
			len(c.Work.Todos),
			pinMark,
		)
	}

	return nil
}

func (a *App) runPrune(olderThan string, dryRun bool) error {
	repoRoot, err := a.resolveRepoRoot()
	if err != nil {
		return fmt.Errorf("find repo root: %w", err)
	}

	dir, err := capsule.DefaultDir(repoRoot)
	if err != nil {
		return fmt.Errorf("resolve capsule dir: %w", err)
	}
	store := capsule.NewStore(dir)

	caps, err := store.List()
	if err != nil {
		return fmt.Errorf("list capsules: %w", err)
	}

	var toDelete []string
	if olderThan != "" {
		d, err := parseDuration(olderThan)
		if err != nil {
			return fmt.Errorf("parse --older-than: %w", err)
		}
		toDelete = capsule.ApplyOlderThan(caps, d)
	} else {
		cfg, err := a.configLoader()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		toDelete = capsule.ApplyRetentionPolicy(caps, cfg.Memory)
	}

	if len(toDelete) == 0 {
		_, _ = fmt.Fprintln(a.stdout, "nothing to prune")
		return nil
	}

	if dryRun {
		_, _ = fmt.Fprintf(a.stdout, "would delete %d capsule(s):\n", len(toDelete))
		for _, id := range toDelete {
			_, _ = fmt.Fprintf(a.stdout, "  %s\n", id)
		}
		return nil
	}

	for _, id := range toDelete {
		if err := store.Delete(id); err != nil {
			_, _ = fmt.Fprintf(a.stderr, "warning: failed to delete %s: %v\n", id, err)
		}
	}
	_, _ = fmt.Fprintf(a.stdout, "pruned %d capsule(s)\n", len(toDelete))
	return nil
}

// parseDuration parses a duration string like "30d", "14d", "7d".
// Only day units are supported.
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") {
		n := 0
		if _, err := fmt.Sscanf(s[:len(s)-1], "%d", &n); err != nil {
			return 0, fmt.Errorf("invalid day count in %q", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	return 0, fmt.Errorf("unsupported duration format %q — use Nd (e.g. 30d)", s)
}

// tryAutoSave creates or deduplicates an auto-save capsule after a successful run.
// Errors are non-fatal — auto-save must never block the main flow.
func (a *App) tryAutoSave(ctx context.Context, histEntry *history.Entry) {
	if a.configLoader == nil || a.repoContext == nil {
		return
	}

	cfg, err := a.configLoader()
	if err != nil || !cfg.Memory.Enabled || !cfg.Memory.AutoSave {
		return
	}

	repoRoot, err := a.resolveRepoRoot()
	if err != nil {
		return
	}

	repoSummary, err := a.repoContext.Collect(ctx)
	if err != nil {
		return
	}

	dir, err := capsule.DefaultDir(repoRoot)
	if err != nil {
		return
	}
	store := capsule.NewStore(dir)

	in := capsule.BuildInput{
		Kind:         capsule.KindAuto,
		HistoryEntry: histEntry,
		RepoSummary:  repoSummary,
		RepoRoot:     repoRoot,
	}
	c := capsule.Build(in)

	// Dedup: check if we have a recent auto-save with the same key fields.
	if existing := findDedupeTarget(store, c); existing != nil {
		_ = store.Update(existing.ID, func(old *capsule.Capsule) {
			old.Repo = c.Repo
			old.Session = c.Session
			old.Work = c.Work
		})
		return
	}

	_ = store.Save(c)

	if cfg.Memory.PruneOnWrite {
		_ = a.runPrune("", false)
	}
}

// findDedupeTarget returns the existing auto-save capsule to update, or nil
// if no dedup target is found. Dedup condition: same repo + branch +
// normalized_goal + target_app, and created within the last 10 minutes.
func findDedupeTarget(store *capsule.Store, incoming capsule.Capsule) *capsule.Capsule {
	caps, err := store.List()
	if err != nil {
		return nil
	}
	cutoff := time.Now().UTC().Add(-10 * time.Minute)
	for i := range caps {
		c := &caps[i]
		if c.Kind != capsule.KindAuto {
			continue
		}
		if c.Pinned {
			continue // never update a pinned auto-save
		}
		if c.CreatedAt.Before(cutoff) {
			continue
		}
		if c.Repo.Name != incoming.Repo.Name {
			continue
		}
		if c.Repo.Branch != incoming.Repo.Branch {
			continue
		}
		if c.Work.NormalizedGoal != incoming.Work.NormalizedGoal {
			continue
		}
		if c.Session.TargetApp != incoming.Session.TargetApp {
			continue
		}
		return c
	}
	return nil
}
