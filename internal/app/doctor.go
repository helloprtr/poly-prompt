package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/helloprtr/poly-prompt/internal/automation"
	"github.com/helloprtr/poly-prompt/internal/clipboard"
	"github.com/helloprtr/poly-prompt/internal/config"
	"github.com/helloprtr/poly-prompt/internal/launcher"
	"github.com/helloprtr/poly-prompt/internal/session"
	"github.com/helloprtr/poly-prompt/internal/translate"
)

type doctorSeverity string

const (
	doctorOK       doctorSeverity = "ok"
	doctorInfo     doctorSeverity = "info"
	doctorWarning  doctorSeverity = "warning"
	doctorBlocking doctorSeverity = "blocking"
)

type doctorCheck struct {
	Severity doctorSeverity
	Label    string
	Detail   string
	Err      error
}

type doctorReport struct {
	Platform []doctorCheck
	Checks   []doctorCheck
}

func (a *App) runDoctor(ctx context.Context, applyFix bool) error {
	cfg, fixMessages, err := a.loadDoctorConfig(applyFix)
	if err != nil {
		return err
	}

	report := a.buildDoctorReport(ctx, cfg)
	readyChecks, optionalChecks := splitDoctorChecks(report.Checks)
	for _, message := range fixMessages {
		_, _ = fmt.Fprintf(a.stdout, "FIX  user config: %s\n", message)
	}
	a.writeDoctorSection("Platform matrix", report.Platform)
	a.writeDoctorSection("Ready now", readyChecks)
	a.writeDoctorSection("Optional unlocks", optionalChecks)
	_, _ = fmt.Fprintln(a.stdout, "Try now: prtr demo")
	_, _ = fmt.Fprintln(a.stdout, `Or use the English fast path: prtr go "explain this error" --dry-run`)
	if applyFix {
		a.writeDoctorSuggestions(doctorReport{Platform: report.Platform, Checks: readyChecks})
	}

	requiredFailures := 0
	optionalWarnings := 0
	for _, check := range readyChecks {
		if check.Severity == doctorBlocking {
			requiredFailures++
		}
	}
	for _, check := range optionalChecks {
		if check.Severity == doctorWarning || check.Severity == doctorBlocking {
			optionalWarnings++
		}
	}
	if requiredFailures > 0 {
		return fmt.Errorf("doctor found %d required issue(s) and %d optional warning(s)", requiredFailures, optionalWarnings)
	}
	return nil
}

func splitDoctorChecks(checks []doctorCheck) ([]doctorCheck, []doctorCheck) {
	ready := make([]doctorCheck, 0, len(checks))
	optional := make([]doctorCheck, 0, len(checks))
	hasTranslation := false

	for _, check := range checks {
		current := check
		switch {
		case current.Label == "user config":
			if current.Severity == doctorBlocking {
				current.Severity = doctorWarning
				current.Err = nil
				current.Detail = "not found; defaults still work, or run `prtr setup` later"
			}
			ready = append(ready, current)
		case current.Label == "project config":
			ready = append(ready, current)
		case current.Label == "deepl api key":
			// Keep INFO as-is; downgrade WARN (blocking) to WARN; keep OK as-is.
			if current.Severity == doctorBlocking {
				current.Severity = doctorWarning
				current.Err = nil
			}
			optional = append(optional, current)
		case current.Label == "translation":
			hasTranslation = true
			current.Severity = doctorWarning
			if strings.TrimSpace(current.Detail) == "" {
				current.Detail = "rerun `prtr setup` or verify the API key"
			}
			optional = append(optional, current)
		case strings.HasPrefix(current.Label, "launcher "):
			if current.Severity == doctorBlocking {
				current.Severity = doctorWarning
			}
			optional = append(optional, current)
		case strings.HasPrefix(current.Label, "automation "):
			if current.Severity == doctorBlocking {
				current.Severity = doctorWarning
			}
			optional = append(optional, current)
		default:
			ready = append(ready, current)
		}
	}

	if !hasTranslation {
		optional = append(optional, doctorCheck{
			Severity: doctorWarning,
			Label:    "translation",
			Detail:   "live check skipped until a DeepL key is configured",
		})
	}

	return ready, optional
}

func (a *App) loadDoctorConfig(applyFix bool) (config.Config, []string, error) {
	cfg, err := a.configLoader()
	if err == nil {
		if !applyFix || cfg.HasUserConfig {
			return cfg, nil, nil
		}

		createdPath, initErr := a.configInit()
		if initErr != nil {
			return cfg, nil, nil
		}
		cfg, reloadErr := a.configLoader()
		if reloadErr != nil {
			return config.Config{}, nil, reloadErr
		}
		return cfg, []string{fmt.Sprintf("created starter config at %s", createdPath)}, nil
	}
	if !applyFix {
		return config.Config{}, nil, err
	}

	path, pathErr := config.Path()
	if pathErr != nil {
		return config.Config{}, nil, err
	}

	if _, statErr := os.Stat(path); errors.Is(statErr, os.ErrNotExist) {
		createdPath, initErr := a.configInit()
		if initErr != nil {
			return config.Config{}, nil, err
		}
		cfg, reloadErr := a.configLoader()
		if reloadErr != nil {
			return config.Config{}, nil, reloadErr
		}
		return cfg, []string{fmt.Sprintf("created starter config at %s", createdPath)}, nil
	}

	resetPath, resetErr := config.Reset()
	if resetErr != nil {
		return config.Config{}, nil, err
	}
	cfg, reloadErr := a.configLoader()
	if reloadErr != nil {
		return config.Config{}, nil, reloadErr
	}
	return cfg, []string{fmt.Sprintf("reset invalid config at %s", resetPath)}, nil
}

func (a *App) buildDoctorReport(ctx context.Context, cfg config.Config) doctorReport {
	report := doctorReport{
		Platform: a.buildPlatformMatrix(cfg),
	}

	envAPIKey, _ := a.lookupEnv("DEEPL_API_KEY")
	apiKey, apiSource := config.ResolveAPIKey(envAPIKey, cfg)

	if cfg.HasUserConfig {
		report.Checks = append(report.Checks, doctorCheck{Severity: doctorOK, Label: "user config", Detail: cfg.UserPath})
	} else {
		report.Checks = append(report.Checks, doctorCheck{Severity: doctorBlocking, Label: "user config", Err: errors.New("not found"), Detail: "run `prtr start` or `prtr setup`"})
	}
	if cfg.HasProjectConfig {
		report.Checks = append(report.Checks, doctorCheck{Severity: doctorOK, Label: "project config", Detail: cfg.ProjectPath})
	} else {
		report.Checks = append(report.Checks, doctorCheck{Severity: doctorWarning, Label: "project config", Detail: "not found"})
	}

	if apiKey == "" {
		// When a pre-built translator is injected (e.g. in tests or native-AI mode),
		// translation works without a DeepL key — show as INFO (optional).
		// Otherwise show as WARN so the user knows a key is needed for translation.
		if a.translator != nil {
			report.Checks = append(report.Checks, doctorCheck{Severity: doctorInfo, Label: "deepl api key", Detail: "optional — AI targets handle multilingual input natively"})
		} else {
			report.Checks = append(report.Checks, doctorCheck{Severity: doctorWarning, Label: "deepl api key", Detail: "not configured; you can try prtr now without a key with `prtr demo` or `prtr go \"explain this error\" --dry-run`"})
		}
	} else {
		report.Checks = append(report.Checks, doctorCheck{Severity: doctorOK, Label: "deepl api key", Detail: apiSource})
	}

	if diagnoser, ok := a.clipboard.(clipboard.Diagnoser); ok {
		if err := diagnoser.Diagnose(); err != nil {
			report.Checks = append(report.Checks, doctorCheck{Severity: doctorBlocking, Label: "clipboard", Err: err, Detail: "fix clipboard support or use `prtr go --dry-run`"})
		} else {
			report.Checks = append(report.Checks, doctorCheck{Severity: doctorOK, Label: "clipboard", Detail: "available"})
		}
	} else {
		report.Checks = append(report.Checks, doctorCheck{Severity: doctorWarning, Label: "clipboard", Detail: "diagnostic unavailable"})
	}

	appendValidationCheck := func(label string, err error, detail string) {
		if err != nil {
			report.Checks = append(report.Checks, doctorCheck{Severity: doctorBlocking, Label: label, Err: err, Detail: detail})
			return
		}
		report.Checks = append(report.Checks, doctorCheck{Severity: doctorOK, Label: label, Detail: detail})
	}

	appendValidationCheck("targets", validateTargetDefaults(cfg), "")
	appendValidationCheck("template presets", validateTemplatePresets(cfg), fmt.Sprintf("%d presets", len(cfg.TemplatePresets)))
	appendValidationCheck("profiles", validateProfiles(cfg), fmt.Sprintf("%d profiles", len(cfg.Profiles)))
	appendValidationCheck("shortcuts", validateShortcuts(cfg), fmt.Sprintf("%d shortcuts", len(cfg.Shortcuts)))
	appendValidationCheck("launchers", validateLaunchers(cfg), fmt.Sprintf("%d launchers", len(cfg.Launchers)))

	if apiKey != "" {
		translator := a.resolveTranslator(apiKey)
		if translator == nil {
			report.Checks = append(report.Checks, doctorCheck{Severity: doctorBlocking, Label: "translation", Err: errors.New("translator is not configured")})
		} else if _, err := translator.Translate(ctx, translate.Request{Text: "안녕하세요", SourceLang: cfg.TranslationSourceLang, TargetLang: cfg.TranslationTargetLang}); err != nil {
			report.Checks = append(report.Checks, doctorCheck{Severity: doctorBlocking, Label: "translation", Err: err, Detail: "rerun `prtr start` or verify the API key"})
		} else {
			report.Checks = append(report.Checks, doctorCheck{Severity: doctorOK, Label: "translation", Detail: "ready"})
		}
	}

	for _, targetName := range []string{"claude", "codex", "gemini"} {
		launcherCfg := cfg.Launchers[targetName]
		if strings.TrimSpace(launcherCfg.Command) == "" {
			bins := session.ModelBinaries(targetName)
			if _, err := session.FindBinary(bins...); err == nil {
				report.Checks = append(report.Checks, doctorCheck{
					Severity: doctorOK,
					Label:    "launcher " + targetName,
					Detail:   "binary found; run `prtr setup` to configure",
				})
			} else {
				report.Checks = append(report.Checks, doctorCheck{
					Severity: doctorWarning,
					Label:    "launcher " + targetName,
					Detail:   fmt.Sprintf("install %s and run `prtr setup` to configure", targetName),
				})
			}
			continue
		}
		if a.launcher == nil {
			report.Checks = append(report.Checks, doctorCheck{Severity: doctorBlocking, Label: "launcher " + targetName, Err: errors.New("launcher is not configured")})
			continue
		}
		req := launcher.Request{Command: launcherCfg.Command, Args: launcherCfg.Args}
		detail := launcherCfg.Command
		if description, err := a.launcher.Describe(req); err == nil && strings.TrimSpace(description) != "" {
			detail = fmt.Sprintf("%s via %s", launcherCfg.Command, description)
		}
		if err := a.launcher.Diagnose(req); err != nil {
			report.Checks = append(report.Checks, doctorCheck{Severity: doctorBlocking, Label: "launcher " + targetName, Err: err, Detail: "fix launcher support or use `prtr go --dry-run`"})
		} else {
			report.Checks = append(report.Checks, doctorCheck{Severity: doctorOK, Label: "launcher " + targetName, Detail: detail})
		}
		if launcherCfg.SubmitMode == string(automation.SubmitAuto) {
			report.Checks = append(report.Checks, doctorCheck{Severity: doctorWarning, Label: "launcher " + targetName + " submit mode", Detail: "auto is not supported yet; use manual or confirm"})
		}

		if a.automator == nil {
			report.Checks = append(report.Checks, doctorCheck{Severity: doctorBlocking, Label: "automation " + targetName, Err: errors.New("automator is not configured")})
			continue
		}
		autoReq := automation.Request{
			Target:      targetName,
			TerminalApp: "Terminal",
			PasteDelay:  time.Duration(maxInt(0, launcherCfg.PasteDelayMS)) * time.Millisecond,
			SubmitMode:  automation.SubmitMode(blankDefault(launcherCfg.SubmitMode, string(automation.SubmitManual))),
		}
		autoDetail := fmt.Sprintf("delay=%dms", launcherCfg.PasteDelayMS)
		if description, err := a.automator.Describe(autoReq); err == nil && strings.TrimSpace(description) != "" {
			autoDetail = fmt.Sprintf("%s delay=%dms", description, launcherCfg.PasteDelayMS)
		}
		if err := a.automator.Diagnose(autoReq); err != nil {
			report.Checks = append(report.Checks, doctorCheck{Severity: doctorBlocking, Label: "automation " + targetName, Err: err, Detail: "install the paste dependency or use `prtr go --dry-run`"})
		} else {
			report.Checks = append(report.Checks, doctorCheck{Severity: doctorOK, Label: "automation " + targetName, Detail: autoDetail})
		}
	}

	return report
}

func (a *App) buildPlatformMatrix(cfg config.Config) []doctorCheck {
	checks := []doctorCheck{
		{Severity: doctorOK, Label: "current surface", Detail: a.currentPlatformLabel()},
	}

	if diagnoser, ok := a.clipboard.(clipboard.Diagnoser); ok {
		if err := diagnoser.Diagnose(); err != nil {
			checks = append(checks, doctorCheck{Severity: doctorBlocking, Label: "clipboard", Err: err, Detail: "install clipboard support"})
		} else {
			checks = append(checks, doctorCheck{Severity: doctorOK, Label: "clipboard", Detail: "ready"})
		}
	}

	if a.launcher == nil {
		checks = append(checks, doctorCheck{Severity: doctorBlocking, Label: "launcher surface", Err: errors.New("launcher is not configured")})
	} else if req, ok := firstLauncherRequest(cfg); ok {
		detail := req.Command
		if description, err := a.launcher.Describe(req); err == nil && strings.TrimSpace(description) != "" {
			detail = description
		}
		if err := a.launcher.Diagnose(req); err != nil {
			checks = append(checks, doctorCheck{Severity: doctorBlocking, Label: "launcher surface", Err: err, Detail: "use a supported terminal backend"})
		} else {
			checks = append(checks, doctorCheck{Severity: doctorOK, Label: "launcher surface", Detail: detail})
		}
	}

	if a.automator == nil {
		checks = append(checks, doctorCheck{Severity: doctorBlocking, Label: "paste surface", Err: errors.New("automator is not configured")})
	} else if req, ok := firstAutomationRequest(cfg); ok {
		detail := "manual"
		if description, err := a.automator.Describe(req); err == nil && strings.TrimSpace(description) != "" {
			detail = description
		}
		if err := a.automator.Diagnose(req); err != nil {
			checks = append(checks, doctorCheck{Severity: doctorBlocking, Label: "paste surface", Err: err, Detail: "install the required paste dependency or use preview mode"})
		} else {
			checks = append(checks, doctorCheck{Severity: doctorOK, Label: "paste surface", Detail: detail})
		}
	}

	checks = append(checks, doctorCheck{Severity: doctorWarning, Label: "submit surface", Detail: "manual only in the current release"})
	return checks
}

func (a *App) currentPlatformLabel() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS + Terminal.app"
	case "linux":
		if value, ok := a.lookupEnv("WAYLAND_DISPLAY"); ok && strings.TrimSpace(value) != "" {
			return "Linux + Wayland"
		}
		return "Linux + X11"
	case "windows":
		return "Windows interactive session"
	default:
		return "unsupported platform"
	}
}

func firstLauncherRequest(cfg config.Config) (launcher.Request, bool) {
	for _, target := range []string{"claude", "codex", "gemini"} {
		launcherCfg := cfg.Launchers[target]
		if strings.TrimSpace(launcherCfg.Command) == "" {
			continue
		}
		return launcher.Request{Command: launcherCfg.Command, Args: launcherCfg.Args}, true
	}
	return launcher.Request{}, false
}

func firstAutomationRequest(cfg config.Config) (automation.Request, bool) {
	for _, target := range []string{"claude", "codex", "gemini"} {
		launcherCfg := cfg.Launchers[target]
		if strings.TrimSpace(launcherCfg.Command) == "" {
			continue
		}
		return automation.Request{
			Target:      target,
			TerminalApp: "Terminal",
			PasteDelay:  time.Duration(maxInt(0, launcherCfg.PasteDelayMS)) * time.Millisecond,
			SubmitMode:  automation.SubmitMode(blankDefault(launcherCfg.SubmitMode, string(automation.SubmitManual))),
		}, true
	}
	return automation.Request{}, false
}

func (a *App) writeDoctorSection(title string, checks []doctorCheck) {
	_, _ = fmt.Fprintln(a.stdout, title)
	for _, check := range checks {
		status := "OK  "
		switch check.Severity {
		case doctorInfo:
			status = "INFO"
		case doctorWarning:
			status = "WARN"
		case doctorBlocking:
			status = "FAIL"
		}
		switch {
		case check.Err != nil && strings.TrimSpace(check.Detail) != "":
			_, _ = fmt.Fprintf(a.stdout, "%s %s: %v (%s)\n", status, check.Label, check.Err, check.Detail)
		case check.Err != nil:
			_, _ = fmt.Fprintf(a.stdout, "%s %s: %v\n", status, check.Label, check.Err)
		case strings.TrimSpace(check.Detail) != "":
			_, _ = fmt.Fprintf(a.stdout, "%s %s: %s\n", status, check.Label, check.Detail)
		default:
			_, _ = fmt.Fprintf(a.stdout, "%s %s\n", status, check.Label)
		}
	}
}

func (a *App) writeDoctorSuggestions(report doctorReport) {
	_, _ = fmt.Fprintln(a.stdout, "Fix suggestions")
	for _, check := range append(report.Platform, report.Checks...) {
		if check.Severity != doctorBlocking {
			continue
		}
		_, _ = fmt.Fprintf(a.stdout, "SUGGEST %s: %s\n", check.Label, doctorSuggestion(check))
	}
}

func doctorSuggestion(check doctorCheck) string {
	switch check.Label {
	case "deepl api key", "translation":
		return "rerun `prtr start` or `prtr setup`, or set DEEPL_API_KEY"
	case "clipboard":
		return "install the platform clipboard dependency or use `prtr go --dry-run`"
	case "launcher surface":
		return "install a supported terminal backend for this platform"
	case "paste surface":
		return "install the paste dependency or fall back to preview/manual paste"
	case "user config":
		return "rerun `prtr doctor --fix` or use `prtr start`"
	default:
		if strings.HasPrefix(check.Label, "launcher ") {
			return "fix the launcher command or use `prtr go --dry-run`"
		}
		if strings.HasPrefix(check.Label, "automation ") {
			return "install the paste dependency or use preview/manual paste"
		}
		return "review the diagnostic output and rerun `prtr doctor --fix`"
	}
}
