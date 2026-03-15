package app

import (
	"strings"

	"github.com/helloprtr/poly-prompt/internal/config"
	"github.com/helloprtr/poly-prompt/internal/repoctx"
)

type routeDecision struct {
	Target string
	Source string
	Reason string
}

func selectGoTarget(cfg config.Config, mode, explicitTarget, promptText string, repoSummary repoctx.Summary) routeDecision {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "ask"
	}

	if target := strings.TrimSpace(explicitTarget); target != "" {
		return routeDecision{Target: target, Source: "cli", Reason: "explicit --to override"}
	}

	if target := strings.TrimSpace(cfg.Routing.FixedTargets[mode]); target != "" {
		return routeDecision{Target: target, Source: "fixed_target", Reason: mode + " fixed target from routing config"}
	}

	if cfg.Routing.Enabled && strings.EqualFold(blankDefault(cfg.Routing.Policy, "deterministic-v1"), "deterministic-v1") {
		if decision, ok := applyDeterministicRoute(mode, promptText, repoSummary); ok {
			return decision
		}
		if target := strings.TrimSpace(cfg.Routing.ModeDefaults[mode]); target != "" {
			return routeDecision{Target: target, Source: "mode_default", Reason: mode + " default from routing config"}
		}
	}

	if target := strings.TrimSpace(cfg.DefaultTarget); target != "" {
		return routeDecision{Target: target, Source: "config default", Reason: "routing disabled; config default target"}
	}
	return routeDecision{Target: "claude", Source: "built-in default", Reason: "routing disabled; built-in default target"}
}

func applyDeterministicRoute(mode, promptText string, repoSummary repoctx.Summary) (routeDecision, bool) {
	switch mode {
	case "fix":
		return routeDecision{Target: "codex", Source: "auto_route", Reason: "fix mode always prefers codex"}, true
	case "review":
		if isCodeHeavy(promptText) || len(repoSummary.Changes) > 0 {
			return routeDecision{Target: "codex", Source: "auto_route", Reason: "review includes code or repo changes"}, true
		}
		return routeDecision{Target: "claude", Source: "auto_route", Reason: "review without code-heavy evidence prefers claude"}, true
	case "design":
		if isUIHeavy(promptText) {
			return routeDecision{Target: "gemini", Source: "auto_route", Reason: "design request has UI or product signals"}, true
		}
		return routeDecision{Target: "claude", Source: "auto_route", Reason: "design request without UI signals prefers claude"}, true
	case "ask":
		if isCodeHeavy(promptText) {
			return routeDecision{Target: "codex", Source: "auto_route", Reason: "request includes code or log-heavy signals"}, true
		}
		if isUIHeavy(promptText) {
			return routeDecision{Target: "gemini", Source: "auto_route", Reason: "request includes UI or product signals"}, true
		}
		return routeDecision{Target: "claude", Source: "auto_route", Reason: "generic ask request prefers claude"}, true
	default:
		return routeDecision{}, false
	}
}

func isCodeHeavy(text string) bool {
	text = strings.ToLower(text)
	if text == "" {
		return false
	}
	markers := []string{
		"```", "traceback", "panic", "error:", "fail", "diff --git", "+++ ", "--- ", "@@",
		"exception", "stack trace", "stderr", "stdout", "line ", " at ", "undefined:",
		"cannot ", "module ", "test failed", ".go:", ".ts:", ".tsx:", ".js:", ".py:",
		"npm ", "pytest", "go test", "build failed", "compile error", "/internal/", "/src/",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func isUIHeavy(text string) bool {
	text = strings.ToLower(text)
	if text == "" {
		return false
	}
	markers := []string{
		"ui", "ux", "layout", "screen", "wireframe", "onboarding", "landing", "hierarchy",
		"spacing", "copy", "interaction", "visual", "component", "flow", "navigation",
		"design system", "dashboard", "hero", "mobile", "responsive",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
