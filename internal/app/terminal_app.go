package app

import "strings"

func preferredTerminalApp(lookupEnv LookupEnv) string {
	for _, key := range []string{"PRTR_TERMINAL_APP", "TERM_PROGRAM", "LC_TERMINAL"} {
		if value, ok := lookupEnv(key); ok {
			if app := normalizeTerminalApp(value); app != "" {
				return app
			}
		}
	}
	return "Terminal"
}

func preferredTerminalDisplayName(lookupEnv LookupEnv) string {
	switch preferredTerminalApp(lookupEnv) {
	case "iTerm":
		return "iTerm.app"
	default:
		return "Terminal.app"
	}
}

func normalizeTerminalApp(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	switch {
	case lower == "":
		return ""
	case strings.Contains(lower, "iterm"):
		return "iTerm"
	case strings.Contains(lower, "terminal"):
		return "Terminal"
	default:
		return ""
	}
}
