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
	"runtime"
	"strings"
	"time"

	"github.com/helloprtr/poly-prompt/internal/automation"
	"github.com/helloprtr/poly-prompt/internal/clipboard"
	"github.com/helloprtr/poly-prompt/internal/config"
	"github.com/helloprtr/poly-prompt/internal/deep"
	"github.com/helloprtr/poly-prompt/internal/editor"
	"github.com/helloprtr/poly-prompt/internal/history"
	"github.com/helloprtr/poly-prompt/internal/input"
	"github.com/helloprtr/poly-prompt/internal/lastresponse"
	"github.com/helloprtr/poly-prompt/internal/launcher"
	"github.com/helloprtr/poly-prompt/internal/repoctx"
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
	LastResponseStore *lastresponse.Store
	RepoContext       repoctx.Collector
	RepoRootFinder    RepoRootFinder
	TermbookLoader    TermbookLoader
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
	lastResponseStore *lastresponse.Store
	repoContext       repoctx.Collector
	repoRootFinder    RepoRootFinder
	termbookLoader    TermbookLoader
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

type resumeCommandOptions struct {
	prompt []string
	dryRun bool
	noCopy bool
	edit   bool
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
	demoEvidence        = `> poly-prompt@0.6.2 test
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
		lastResponseStore: deps.LastResponseStore,
		repoContext:       deps.RepoContext,
		repoRootFinder:    deps.RepoRootFinder,
		termbookLoader:    deps.TermbookLoader,
	}
}

func (a *App) Execute(ctx context.Context, args []string, stdin io.Reader, stdinPiped bool) error {
	if a.shouldRunRootDirect(args) {
		return a.runMain(ctx, args, stdin, stdinPiped, "")
	}
	cmd := a.Command(ctx, stdin, stdinPiped)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func (a *App) shouldRunRootDirect(args []string) bool {
	if len(args) == 0 {
		return true
	}

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
	case "init", "version", "start", "setup", "lang", "doctor", "templates", "profiles", "history", "rerun", "pin", "favorite", "go", "demo", "again", "resume", "swap", "take", "learn", "inspect":
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

// readLastResponse returns (responseText, source, error).
// It tries lastResponseStore first (if configured), then falls back to clipboard.
// Emits a warning to stderr when the stored response is >= 5 minutes old.
// Returns ("", "", nil) when lastResponseStore is absent and clipboard is empty.
func (a *App) readLastResponse(ctx context.Context) (text, source string, err error) {
	if a.lastResponseStore != nil {
		entry, ok, readErr := a.lastResponseStore.Read()
		if readErr != nil {
			return "", "", fmt.Errorf("read last-response: %w", readErr)
		}
		if ok && strings.TrimSpace(entry.Response) != "" {
			age := time.Since(entry.CapturedAt)
			if age >= 5*time.Minute {
				_, _ = fmt.Fprintf(a.stderr, "Using response from %s ago\n",
					age.Round(time.Minute).String())
			}
			return strings.TrimSpace(entry.Response), entry.Source, nil
		}
	}
	// Fallback: clipboard
	clipText, clipErr := a.clipboard.Read(ctx)
	if clipErr != nil {
		return "", "", clipErr
	}
	return strings.TrimSpace(clipText), "clipboard", nil
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
		engine:          blankDefault(entry.Engine, "classic"),
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

	responseText, responseSource, err := a.readLastResponse(ctx)
	if err != nil {
		return err
	}
	if responseText == "" {
		return errors.New("No response captured yet. Run prtr go first, then copy the AI's response.")
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

		_, _ = fmt.Fprintf(a.stderr, "-> take:%s --deep | %s | %s | running\n", command.action, target, responseSource)
		result, err := deep.ExecutePatchRun(ctx, deep.Options{
			Action:          command.action,
			Source:          responseText,
			SourceKind:      responseSource,
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
			surfaceInput:         responseSource,
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
		surfaceInput:         responseSource,
		surfaceDelivery:      surfaceDeliveryLabel(command.dryRun),
		preferTargetTemplate: true,
		engine:               "classic",
	}

	return a.executePrompt(ctx, opts, takePrompt(command.action, responseText), "take:"+command.action)
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
			run.launchedTarget = run.targetName
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
	if command.noCopy && !command.dryRun {
		return goCommandOptions{}, usageError{message: "--no-copy currently requires --dry-run with `prtr go`", helpText: goHelpText()}
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
	if command.noCopy && !command.dryRun {
		return replayCommandOptions{}, usageError{message: "--no-copy currently requires --dry-run", helpText: swapHelpText()}
	}

	return command, nil
}

func parseResumeCommand(args []string) (resumeCommandOptions, error) {
	command := resumeCommandOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--dry-run":
			command.dryRun = true
		case arg == "--no-copy":
			command.noCopy = true
		case arg == "--edit" || arg == "--interactive" || arg == "-i":
			command.edit = true
		case strings.HasPrefix(arg, "-"):
			return resumeCommandOptions{}, usageError{message: fmt.Sprintf("unknown resume flag %q", arg)}
		default:
			command.prompt = append(command.prompt, arg)
		}
	}
	return command, nil
}

func resumePrompt(response, userMessage string) string {
	return strings.Join([]string{
		response,
		"",
		"---",
		userMessage,
	}, "\n")
}

func (a *App) runResume(ctx context.Context, args []string, stdin io.Reader, stdinPiped bool) error {
	command, err := parseResumeCommand(args)
	if err != nil {
		return err
	}

	// No-arg: re-run last prompt verbatim (same behavior as legacy `again`)
	if len(command.prompt) == 0 && !stdinPiped {
		entry, err := a.latestHistoryEntry()
		if err != nil {
			return errors.New("No previous prompt found. Run prtr go first.")
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
			surfaceInput:    "history",
			surfaceDelivery: surfaceDeliveryLabel(command.dryRun),
			engine:          blankDefault(entry.Engine, "classic"),
			parentID:        entry.ID,
		}
		return a.executePrompt(ctx, opts, entry.Original, entry.Shortcut)
	}

	// With message: prepend last response as context
	responseText, _, err := a.readLastResponse(ctx)
	if err != nil {
		return err
	}
	if responseText == "" {
		return errors.New("No response captured yet. Run prtr go first, then copy the AI's response.")
	}

	userMsg := strings.Join(command.prompt, " ")
	if stdinPiped {
		stdinBytes, readErr := io.ReadAll(stdin)
		if readErr != nil {
			return fmt.Errorf("read stdin: %w", readErr)
		}
		if userMsg == "" {
			userMsg = strings.TrimSpace(string(stdinBytes))
		}
	}

	// latestHistoryEntry error is intentionally ignored here: if history is
	// unavailable, we proceed with zero-value Entry (empty target, no parent).
	entry, _ := a.latestHistoryEntry()
	opts := runOptions{
		target:          entry.Target,
		sourceLang:      entry.SourceLang,
		targetLang:      entry.TargetLang,
		translationMode: entry.TranslationMode,
		interactive:     command.edit,
		noCopy:          command.dryRun || command.noCopy,
		launch:          !command.dryRun,
		paste:           !command.dryRun,
		compactStatus:   true,
		surfaceMode:     "resume",
		surfaceInput:    "last-response",
		surfaceDelivery: surfaceDeliveryLabel(command.dryRun),
		engine:          blankDefault(entry.Engine, "classic"),
		parentID:        entry.ID,
	}
	return a.executePrompt(ctx, opts, resumePrompt(responseText, userMsg), "")
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
	fs.StringVar(&opts.targetLang, "to", "", "target language code")
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

func usageText() string {
	return rootHelpText()
}

func rootHelpText() string {
	return strings.Join([]string{
		"prtr is the command layer for AI work.",
		"",
		"Turn logs, diffs, and intent into the next AI action across Claude, Codex, and Gemini.",
		"",
		"Start with the beginner-first flow or jump straight into a safe preview:",
		"  prtr start",
		"  prtr demo",
		`  prtr go "explain this error" --dry-run`,
		"",
		"Core loop:",
		`  npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"`,
		"  prtr swap gemini",
		"  prtr take patch",
		"",
		"Then keep moving with:",
		"  prtr learn",
		"  prtr again",
		"  prtr inspect",
		"",
		"Usage:",
		"  prtr start [message...]",
		"  prtr go [mode] [message...]",
		"  prtr demo",
		"  prtr swap <app>",
		"  prtr take <action>",
		"  prtr learn [paths...]",
		"  prtr again",
		"  prtr inspect [message...]",
		"  prtr history [search <query>]",
		"  prtr doctor [--fix]",
		"  prtr version",
		"",
		"Compatibility aliases:",
		"  prtr ask",
		"  prtr review",
		"  prtr fix",
		"  prtr design",
		"",
		"Advanced commands:",
		"  prtr templates <list|show>",
		"  prtr profiles <list|show|use>",
		"  prtr rerun <id> [flags]",
		"  prtr pin <id>",
		"  prtr favorite <id>",
		"  prtr setup",
		"  prtr lang",
		"  prtr init",
	}, "\n")
}

func setupHelpText() string {
	return strings.Join([]string{
		"Run the advanced guided setup flow.",
		"",
		"`prtr setup` is the compatibility path for people who want the full",
		"interactive defaults flow instead of the smaller first-run path in `start`.",
		"",
		"What `setup` asks for:",
		"  - DeepL API key",
		"  - default input language",
		"  - default output language",
		"  - default app",
		"  - default role",
		"  - default template preset",
	}, "\n")
}

func doctorHelpText() string {
	return strings.Join([]string{
		"Run environment and configuration diagnostics.",
		"",
		"`prtr doctor` checks config, translation readiness, clipboard support,",
		"launchers, automation, and the current platform matrix.",
		"",
		"Usage:",
		"  prtr doctor",
		"  prtr doctor --fix",
		"",
		"`--fix` applies safe automatic fixes when possible, then prints fallback",
		"suggestions for anything it cannot repair automatically.",
	}, "\n")
}

func startHelpText() string {
	return strings.Join([]string{
		"Run the beginner-first first-send flow.",
		"",
		"`prtr start` helps you get from setup to a real first send with as little",
		"friction as possible.",
		"",
		"What `start` does:",
		"  1. Prompts for minimal onboarding settings when needed",
		"  2. Runs `prtr doctor` before the first send",
		"  3. Asks for a first request if you do not pass one",
		"  4. Sends the request through the same flow as `prtr go`",
		"",
		"Usage:",
		"  prtr start [message...]",
		"",
		"Examples:",
		"  prtr start",
		`  prtr start "이 함수 왜 느린지 설명해줘"`,
		`  prtr start --to codex "왜 테스트가 깨지는지 찾아줘"`,
		"  prtr start --dry-run",
		"",
		"Flags:",
		"      --to <app>        Choose the app: claude | codex | gemini",
		"      --dry-run         Show the first-send prompt without opening any app",
		"  -h, --help            Help for start",
		"",
		"Advanced setup:",
		"  Use `prtr setup` when you want the full configuration flow for role,",
		"  template preset, and other advanced defaults.",
	}, "\n")
}

func goHelpText() string {
	return strings.Join([]string{
		"Turn logs, diffs, and intent into the next AI action.",
		"",
		"`prtr go` is the fastest way to use prtr for real work.",
		"",
		"Write in your language when you need translation.",
		"Pipe logs or stack traces when you have evidence.",
		"prtr shapes the request for Claude, Codex, or Gemini and keeps the next step cheaper.",
		"",
		"Works without a DeepL key — AI targets handle multilingual input natively.",
		"Use `prtr demo` if you want a safe preview before setup.",
		"",
		"Usage:",
		"  prtr go [mode] [message...]",
		"",
		"Modes:",
		"  ask       General questions and everyday prompting (default)",
		"  review    Review code, docs, PRs, or plans",
		"  fix       Diagnose errors, logs, and broken tests",
		"  design    Plan implementations, architecture, or UX flows",
		"",
		"How input works:",
		"  - If you pass a message, that message is your request.",
		"  - If you pipe text and also pass a message, the piped text becomes evidence.",
		"  - If you only pipe text, the piped text becomes the request.",
		"  - If you are inside a Git repo, prtr may attach lightweight repo context",
		"    unless --no-context is used.",
		"",
		"What `go` does:",
		"  1. Reads your request",
		"  2. Collects useful context from stdin and the current repo",
		"  3. Translates only when needed",
		"  4. Applies the selected mode",
		"  5. Opens Claude, Codex, or Gemini",
		"  6. Pastes the final prompt",
		"  7. Saves the run so you can swap or run again",
		"",
		"Examples:",
		`  prtr go "이 에러 원인 분석해줘"`,
		`  prtr go review "이 PR에서 위험한 부분만 짚어줘"`,
		`  npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"`,
		`  prtr go design "이 기능 구조 설계해줘" --to gemini --edit`,
		`  prtr go "이 문서 설명해줘" --dry-run`,
		`  cat crash.log | prtr go fix`,
		"",
		"Flags:",
		"      --to <app>        Choose the app: claude | codex | gemini",
		"      --edit            Review and edit before sending",
		"      --dry-run         Show the final prompt without opening any app",
		"      --no-context      Do not attach repo or piped context automatically",
		"      --no-copy         Do not copy the final prompt to the clipboard",
		"  -h, --help            Help for go",
		"",
		"Next steps:",
		"  prtr swap <app>       Send the last prompt to another app",
		"  prtr take <action>    Turn clipboard text into the next action",
		"  prtr learn            Teach prtr your repo terms and style",
		"  prtr again            Run the latest flow again",
		"  prtr inspect          Inspect the compiled prompt and config",
	}, "\n")
}

func demoHelpText() string {
	return strings.Join([]string{
		"Preview prtr's core loop.",
		"",
		"`prtr demo` is a safe preview path.",
		"It shows the command-layer story with a Korean fix request, sample npm test evidence,",
		"and a Codex-ready prompt preview without launching an app or touching the clipboard.",
		"",
		"Usage:",
		"  prtr demo",
		"",
		"Notes:",
		"  - No API key required",
		"  - No launch, paste, or clipboard writes",
		"  - Uses sample evidence and preview output only",
		"",
		"Try next:",
		`  prtr go "explain this error" --dry-run`,
		"  prtr setup",
	}, "\n")
}

func againHelpText() string {
	return strings.Join([]string{
		"Run the latest prompt flow again.",
		"",
		"`prtr again` reuses the latest request, mode, app, and translation settings.",
		"",
		"Usage:",
		"  prtr again [message...]",
		"",
		"If you pass a new message, it replaces the last request.",
		"If you pipe text and also pass a message, the piped text becomes evidence.",
		"",
		"Examples:",
		"  prtr again",
		"  prtr again --edit",
		`  prtr again "이전 질문을 더 날카롭게 다시 물어봐줘"`,
		"",
		"Flags:",
		"      --edit            Review and edit before sending",
		"      --dry-run         Show the final prompt without opening any app",
		"      --no-context      Do not attach repo or piped context automatically",
		"      --no-copy         Do not copy the final prompt to the clipboard",
		"  -h, --help            Help for again",
	}, "\n")
}

func swapHelpText() string {
	return strings.Join([]string{
		"Send the latest prompt to another app.",
		"",
		"`prtr swap` reuses the latest request and mode, then recompiles it for",
		"Claude, Codex, or Gemini without rebuilding the flow manually.",
		"",
		"Usage:",
		"  prtr swap <app> [message...]",
		"",
		"Examples:",
		"  prtr swap claude",
		"  prtr swap codex",
		"  prtr swap gemini",
		"  prtr swap gemini --edit",
		"  prtr swap claude --dry-run",
		"",
		"Flags:",
		"      --edit            Review and edit before sending",
		"      --dry-run         Show the final prompt without opening any app",
		"      --no-context      Do not attach repo or piped context automatically",
		"      --no-copy         Do not copy the final prompt to the clipboard",
		"  -h, --help            Help for swap",
	}, "\n")
}

func takeHelpText() string {
	return strings.Join([]string{
		"Turn the latest answer or clipboard text into the next action.",
		"",
		"`prtr take` has two paths:",
		"  - classic: turn clipboard text into the next ready-to-send prompt",
		"  - deep: build a typed execution run with artifacts, plan, and progress",
		"",
		"Usage:",
		"  prtr take <action> [--dip] [flags]",
		"",
		"Classic actions (all support --deep for patch/test/debug/refactor):",
		"  patch     Turn the answer into an implementation prompt",
		"  test      Turn the answer into a testing prompt",
		"  debug     Turn the answer into a debugging prompt",
		"  refactor  Turn the answer into a refactoring prompt",
		"  commit    Turn the answer into a commit message prompt",
		"  summary   Turn the answer into a short reusable summary prompt",
		"  clarify   Turn the answer into a clarification prompt",
		"  issue     Turn the answer into an issue or task prompt",
		"  plan      Turn the answer into an implementation plan prompt",
		"",
		"Classic examples:",
		"  prtr take patch",
		"  prtr take patch --dip --dry-run",
		"  prtr take test --to codex",
		"  prtr take commit --dry-run",
		"  prtr take summary --edit",
		"  prtr take issue",
		"  prtr take plan",
		"",
		"Deep execution engine examples:",
		"  prtr take patch --dip",
		"  prtr take patch --dip --dry-run    # write artifacts; skip opening app",
		"  prtr take patch --dip --to claude  # route delivery to Claude",
		"  prtr take patch --dip --llm        # enable LLM enhancement (uses config)",
		"  prtr take patch --dip --llm=claude # specify LLM provider",
		"",
		"Flags:",
		"      --to <app>        Choose the app: claude | codex | gemini",
		"      --dip            Run the internal execution engine for `take patch`",
		"      --deep           Alias for --dip (deprecated)",
		"      --llm            Enable LLM enhancement (provider from config or PRTR_LLM_PROVIDER)",
		"      --llm=<provider> Enable LLM enhancement with explicit provider",
		"      --edit            Review and edit before sending",
		"      --dry-run         Show the final prompt without opening any app",
		"                        Deep mode still writes plan and artifact files locally",
		"  -h, --help            Help for take",
	}, "\n")
}

func learnHelpText() string {
	return strings.Join([]string{
		"Teach prtr your repo terms and style.",
		"",
		"`prtr learn` scans your project and builds a local termbook with",
		"protected project terms that should not be translated away.",
		"",
		"Usage:",
		"  prtr learn [paths...]",
		"",
		"If no paths are provided, prtr learns from README, docs, cmd, and internal.",
		"",
		"Examples:",
		"  prtr learn",
		"  prtr learn README.md docs",
		"  prtr learn --dry-run",
		"  prtr learn --reset",
		"",
		"Flags:",
		"      --dry-run         Show the generated termbook without saving",
		"      --reset           Rebuild the termbook instead of merging it",
		"  -h, --help            Help for learn",
	}, "\n")
}

func inspectHelpText() string {
	return strings.Join([]string{
		"Inspect the compiled prompt and resolved config without sending it anywhere.",
		"",
		"`prtr inspect` is the expert path for diff, explain, JSON output, and",
		"advanced prompt-shaping flags.",
		"",
		"Usage:",
		"  prtr inspect [flags] [message...]",
		"",
		"Examples:",
		`  prtr inspect "이 PR 리뷰해줘"`,
		`  prtr inspect --json "이 에러 분석해줘"`,
		`  prtr inspect -t codex --template codex-implement -r be "이 함수 개선해줘"`,
		"",
		"Flags:",
		"  -t, --target <name>    target profile name",
		"      --source-lang <code> advanced source language override",
		"      --to <code>        target language override",
		"  -r, --role <alias>     role profile alias",
		"      --template <name>  template preset name",
		"      --translation-mode auto|force|skip",
		"  -i, --interactive      edit the final prompt in a TUI before output",
		"      --show-original    print the original input to stderr",
		"      --explain          print resolved configuration details to stderr",
		"      --diff             print original, translated, and final prompt to stderr",
		"      --json             emit structured JSON output",
		"      --no-copy          print the translated prompt without copying it",
		"  -h, --help             Help for inspect",
	}, "\n")
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
