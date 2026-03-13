package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/helloprtr/poly-prompt/internal/clipboard"
	"github.com/helloprtr/poly-prompt/internal/config"
	"github.com/helloprtr/poly-prompt/internal/editor"
	"github.com/helloprtr/poly-prompt/internal/history"
	"github.com/helloprtr/poly-prompt/internal/input"
	prompttemplate "github.com/helloprtr/poly-prompt/internal/template"
	"github.com/helloprtr/poly-prompt/internal/translate"
)

type ConfigLoader func() (config.Config, error)
type ConfigInit func() (string, error)
type LookupEnv func(string) (string, bool)
type TranslatorFactory func(string) translate.Translator

type Dependencies struct {
	Version           string
	Stdout            io.Writer
	Stderr            io.Writer
	Translator        translate.Translator
	TranslatorFactory TranslatorFactory
	Clipboard         clipboard.Writer
	Editor            editor.Editor
	ConfigLoader      ConfigLoader
	ConfigInit        ConfigInit
	LookupEnv         LookupEnv
	HistoryStore      *history.Store
}

type App struct {
	version           string
	stdout            io.Writer
	stderr            io.Writer
	translator        translate.Translator
	translatorFactory TranslatorFactory
	clipboard         clipboard.Writer
	editor            editor.Editor
	configLoader      ConfigLoader
	configInit        ConfigInit
	lookupEnv         LookupEnv
	historyStore      *history.Store
}

type usageError struct {
	message string
}

type runOptions struct {
	target         string
	role           string
	templatePreset string
	noCopy         bool
	showOriginal   bool
	interactive    bool
	explain        bool
	diff           bool
	jsonOutput     bool
}

type resolvedRun struct {
	original       string
	translated     string
	finalPrompt    string
	targetName     string
	targetSource   string
	targetConfig   config.TargetConfig
	roleName       string
	rolePrompt     string
	roleSource     string
	templateName   string
	templateSource string
	layout         string
	context        string
	outputFormat   string
	shortcut       string
	copied         bool
	apiKeySource   string
	config         config.Config
}

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
		configLoader:      deps.ConfigLoader,
		configInit:        deps.ConfigInit,
		lookupEnv:         deps.LookupEnv,
		historyStore:      store,
	}
}

func (a *App) Execute(ctx context.Context, args []string, stdin io.Reader, stdinPiped bool) error {
	if len(args) > 0 {
		switch args[0] {
		case "init":
			return a.runInit()
		case "version":
			return a.runVersion()
		case "setup":
			return a.runSetup(stdin)
		case "doctor":
			return a.runDoctor(ctx)
		case "templates":
			return a.runTemplates(args[1:])
		case "profiles":
			return a.runProfiles(args[1:])
		case "history":
			return a.runHistory()
		case "rerun":
			return a.runRerun(ctx, args[1:])
		}
	}

	if len(args) > 0 {
		if _, ok := a.builtInShortcutNames()[args[0]]; ok {
			return a.runShortcut(ctx, args[0], args[1:], stdin, stdinPiped)
		}
	}

	return a.runMain(ctx, args, stdin, stdinPiped, "")
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
		_, _ = fmt.Fprintln(a.stdout, "DeepL API key is still empty. Add DEEPL_API_KEY or rerun setup to save it in config.")
	}
	return nil
}

func (a *App) runDoctor(ctx context.Context) error {
	cfg, err := a.configLoader()
	if err != nil {
		return err
	}

	failures := 0
	writeCheck := func(label string, err error, detail string) {
		if err != nil {
			failures++
			_, _ = fmt.Fprintf(a.stdout, "FAIL %s: %v\n", label, err)
			return
		}
		if strings.TrimSpace(detail) != "" {
			_, _ = fmt.Fprintf(a.stdout, "OK   %s: %s\n", label, detail)
			return
		}
		_, _ = fmt.Fprintf(a.stdout, "OK   %s\n", label)
	}

	if cfg.HasUserConfig {
		writeCheck("user config", nil, cfg.UserPath)
	} else {
		writeCheck("user config", errors.New("not found"), "run `prtr init` or `prtr setup`")
	}
	if cfg.HasProjectConfig {
		writeCheck("project config", nil, cfg.ProjectPath)
	} else {
		writeCheck("project config", nil, "not found")
	}

	envAPIKey, _ := a.lookupEnv("DEEPL_API_KEY")
	apiKey, apiSource := config.ResolveAPIKey(envAPIKey, cfg)
	if apiKey == "" {
		writeCheck("deepl api key", translate.ErrMissingAPIKey, "set DEEPL_API_KEY or run `prtr setup`")
	} else {
		writeCheck("deepl api key", nil, apiSource)
	}

	if diagnoser, ok := a.clipboard.(clipboard.Diagnoser); ok {
		writeCheck("clipboard", diagnoser.Diagnose(), "")
	} else {
		writeCheck("clipboard", nil, "diagnostic unavailable")
	}

	writeCheck("targets", validateTargetDefaults(cfg), "")
	writeCheck("template presets", validateTemplatePresets(cfg), fmt.Sprintf("%d presets", len(cfg.TemplatePresets)))
	writeCheck("profiles", validateProfiles(cfg), fmt.Sprintf("%d profiles", len(cfg.Profiles)))
	writeCheck("shortcuts", validateShortcuts(cfg), fmt.Sprintf("%d shortcuts", len(cfg.Shortcuts)))

	if apiKey != "" {
		translator := a.resolveTranslator(apiKey)
		if translator == nil {
			writeCheck("translation", errors.New("translator is not configured"), "")
		} else {
			_, err := translator.Translate(ctx, "안녕하세요")
			writeCheck("translation", err, "")
		}
	}

	if failures > 0 {
		return fmt.Errorf("doctor found %d issue(s)", failures)
	}
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

func (a *App) runHistory() error {
	if a.historyStore == nil {
		return errors.New("history store is not configured")
	}

	entries, err := a.historyStore.List()
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		_, _ = fmt.Fprintln(a.stdout, "no history yet")
		return nil
	}

	limit := minInt(10, len(entries))
	for _, entry := range entries[:limit] {
		preview := truncateOneLine(entry.Original, 60)
		_, _ = fmt.Fprintf(a.stdout, "%s\t%s\ttarget=%s role=%s template=%s\t%s\n", entry.ID, entry.CreatedAt.Format("2006-01-02 15:04:05"), entry.Target, entry.Role, entry.TemplatePreset, preview)
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

	return a.executePrompt(ctx, opts, entry.Original, "")
}

func (a *App) runShortcut(ctx context.Context, shortcut string, args []string, stdin io.Reader, stdinPiped bool) error {
	return a.runMain(ctx, args, stdin, stdinPiped, shortcut)
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

	if opts.showOriginal {
		_, _ = fmt.Fprintf(a.stderr, "Original:\n%s\n\n", resolved.original)
	}
	if opts.explain {
		a.writeExplain(resolved)
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

	if err := a.appendHistory(resolved); err != nil {
		return err
	}

	if opts.jsonOutput {
		payload := map[string]any{
			"original":        resolved.original,
			"translated":      resolved.translated,
			"target":          resolved.targetName,
			"role":            resolved.roleName,
			"template_preset": resolved.templateName,
			"final_prompt":    resolved.finalPrompt,
			"copied":          resolved.copied,
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("encode json output: %w", err)
		}
		_, _ = fmt.Fprintln(a.stdout, string(data))
	} else {
		_, _ = fmt.Fprintln(a.stdout, resolved.finalPrompt)
	}

	if opts.noCopy {
		_, _ = fmt.Fprintf(a.stderr, "target %q ready; clipboard skipped\n", resolved.targetName)
	} else {
		_, _ = fmt.Fprintf(a.stderr, "copied prompt for target %q to clipboard\n", resolved.targetName)
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
		if !shortcuts[shortcutName] {
			return resolvedRun{}, fmt.Errorf("unknown shortcut %q", shortcutName)
		}
		shortcut = cfg.Shortcuts[shortcutName]
	}

	target, targetSource := resolveRunTarget(opts.target, shortcut, cfg, envTarget)
	targetConfig, ok := cfg.Targets[target]
	if !ok {
		return resolvedRun{}, fmt.Errorf("unknown target %q (available: %s)", target, strings.Join(config.AvailableTargets(cfg), ", "))
	}

	role, roleSource := resolveRunRole(opts.role, shortcut, cfg)
	rolePrompt := ""
	if role != "" {
		roleConfig, ok := cfg.Roles[role]
		if !ok {
			return resolvedRun{}, fmt.Errorf("unknown role %q (available: %s)", role, strings.Join(config.AvailableRoles(cfg), ", "))
		}
		rolePrompt = roleConfig.Prompt
	}

	templateName, templateSource := resolveRunTemplate(opts.templatePreset, shortcut, cfg, targetConfig)
	layout, err := resolveTemplateLayout(templateName, targetConfig, cfg)
	if err != nil {
		return resolvedRun{}, err
	}

	apiKey, apiKeySource := config.ResolveAPIKey(envAPIKey, cfg)
	translator := a.resolveTranslator(apiKey)
	if translator == nil {
		return resolvedRun{}, errors.New("translator is not configured")
	}

	translated, err := translator.Translate(ctx, text)
	if err != nil {
		return resolvedRun{}, err
	}

	finalPrompt, err := prompttemplate.RenderData(layout, prompttemplate.Data{
		Prompt:       translated,
		Role:         rolePrompt,
		Target:       target,
		Context:      shortcut.Context,
		OutputFormat: shortcut.OutputFormat,
	})
	if err != nil {
		return resolvedRun{}, fmt.Errorf("render template for target %q: %w", target, err)
	}

	return resolvedRun{
		original:       text,
		translated:     translated,
		finalPrompt:    finalPrompt,
		targetName:     target,
		targetSource:   targetSource,
		targetConfig:   targetConfig,
		roleName:       role,
		rolePrompt:     rolePrompt,
		roleSource:     roleSource,
		templateName:   templateName,
		templateSource: templateSource,
		layout:         layout,
		context:        shortcut.Context,
		outputFormat:   shortcut.OutputFormat,
		shortcut:       shortcutName,
		apiKeySource:   apiKeySource,
		config:         cfg,
	}, nil
}

func (a *App) writeExplain(run resolvedRun) {
	_, _ = fmt.Fprintln(a.stderr, "Resolved configuration:")
	_, _ = fmt.Fprintf(a.stderr, "- target: %s (%s)\n", run.targetName, run.targetSource)
	if run.roleName != "" {
		_, _ = fmt.Fprintf(a.stderr, "- role: %s (%s)\n", run.roleName, run.roleSource)
	} else {
		_, _ = fmt.Fprintf(a.stderr, "- role: none (%s)\n", run.roleSource)
	}
	_, _ = fmt.Fprintf(a.stderr, "- template preset: %s (%s)\n", run.templateName, run.templateSource)
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
		Original:       run.original,
		Translated:     run.translated,
		FinalPrompt:    run.finalPrompt,
		Target:         run.targetName,
		Role:           run.roleName,
		TemplatePreset: run.templateName,
		Shortcut:       run.shortcut,
	})
}

func (a *App) resolveTranslator(apiKey string) translate.Translator {
	if a.translatorFactory != nil {
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

func parseRunOptions(args []string) (runOptions, []string, error) {
	fs := flag.NewFlagSet("prtr", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts runOptions
	fs.StringVar(&opts.target, "target", "", "target profile name")
	fs.StringVar(&opts.target, "t", "", "target profile name")
	fs.StringVar(&opts.role, "role", "", "role profile alias")
	fs.StringVar(&opts.role, "r", "", "role profile alias")
	fs.StringVar(&opts.templatePreset, "template", "", "template preset name")
	fs.BoolVar(&opts.noCopy, "no-copy", false, "skip clipboard copy")
	fs.BoolVar(&opts.showOriginal, "show-original", false, "print the original input to stderr")
	fs.BoolVar(&opts.interactive, "interactive", false, "open an interactive editor before writing the final prompt")
	fs.BoolVar(&opts.interactive, "i", false, "open an interactive editor before writing the final prompt")
	fs.BoolVar(&opts.explain, "explain", false, "print resolved config details to stderr")
	fs.BoolVar(&opts.diff, "diff", false, "print original, translated, and final prompt to stderr")
	fs.BoolVar(&opts.jsonOutput, "json", false, "emit JSON output")

	if err := fs.Parse(args); err != nil {
		return runOptions{}, nil, usageError{message: err.Error()}
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

func resolveRunTemplate(cliPreset string, shortcut config.ShortcutConfig, cfg config.Config, target config.TargetConfig) (string, string) {
	if preset := strings.TrimSpace(cliPreset); preset != "" {
		return preset, "cli flag"
	}
	if preset := strings.TrimSpace(shortcut.TemplatePreset); preset != "" {
		return preset, "shortcut"
	}
	if preset := strings.TrimSpace(cfg.DefaultTemplatePreset); preset != "" {
		source := cfg.DefaultPresetSource
		if source == "" {
			source = "config default"
		}
		return preset, source
	}
	if preset := strings.TrimSpace(target.DefaultTemplatePreset); preset != "" {
		return preset, "target default"
	}
	return "", "target template"
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

func truncateOneLine(text string, limit int) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.TrimSpace(text)
	if len(text) <= limit {
		return text
	}
	return text[:limit-3] + "..."
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

func (e usageError) Error() string {
	return fmt.Sprintf("%s\n\n%s", e.message, usageText())
}

func usageText() string {
	return strings.Join([]string{
		"Usage:",
		"  prtr [flags] [text...]",
		"  prtr setup",
		"  prtr doctor",
		"  prtr templates <list|show>",
		"  prtr profiles <list|show|use>",
		"  prtr history",
		"  prtr rerun <id> [flags]",
		"  prtr ask|review|fix|design [flags] [text...]",
		"  prtr init",
		"  prtr version",
		"",
		"Flags:",
		"  -t, --target <name>    target profile name",
		"  -r, --role <alias>     role profile alias",
		"      --template <name>  template preset name",
		"  -i, --interactive      edit the final prompt in a TUI before output",
		"      --no-copy          print the translated prompt without copying it",
		"      --show-original    print the original input to stderr",
		"      --explain          print resolved configuration details to stderr",
		"      --diff             print original, translated, and final prompt to stderr",
		"      --json             emit structured JSON output",
		"",
		"Examples:",
		`  prtr -t codex --template codex-implement "한국어 질문"`,
		`  prtr review -i "한국어 질문"`,
		`  prtr templates list`,
	}, "\n")
}
