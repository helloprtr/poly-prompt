package app

import "strings"

type terminalAppPreference struct {
	AppName     string
	DisplayName string
	Source      string
	Raw         string
	Supported   bool
}

func preferredTerminalAppPreference(lookupEnv LookupEnv) terminalAppPreference {
	for _, key := range []string{"PRTR_TERMINAL_APP", "TERM_PROGRAM", "LC_TERMINAL"} {
		if value, ok := lookupEnv(key); ok {
			if pref, ok := parseTerminalAppPreference(value, key); ok {
				return pref
			}
		}
	}

	return terminalAppPreference{
		AppName:     "Terminal",
		DisplayName: "Terminal.app",
		Source:      "default",
		Raw:         "Terminal",
		Supported:   true,
	}
}

func preferredTerminalApp(lookupEnv LookupEnv) string {
	return preferredTerminalAppPreference(lookupEnv).AppName
}

func preferredTerminalDisplayName(lookupEnv LookupEnv) string {
	return preferredTerminalAppPreference(lookupEnv).DisplayName
}

func parseTerminalAppPreference(value, source string) (terminalAppPreference, bool) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return terminalAppPreference{}, false
	}

	lower := strings.ToLower(raw)
	switch {
	case strings.Contains(lower, "iterm"), lower == "com.googlecode.iterm2":
		return terminalAppPreference{
			AppName:     "iTerm",
			DisplayName: "iTerm.app",
			Source:      source,
			Raw:         raw,
			Supported:   true,
		}, true
	case strings.Contains(lower, "terminal"), lower == "com.apple.terminal":
		return terminalAppPreference{
			AppName:     "Terminal",
			DisplayName: "Terminal.app",
			Source:      source,
			Raw:         raw,
			Supported:   true,
		}, true
	default:
		return terminalAppPreference{
			AppName:     raw,
			DisplayName: raw,
			Source:      source,
			Raw:         raw,
			Supported:   false,
		}, true
	}
}
