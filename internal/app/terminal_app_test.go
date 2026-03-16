package app

import "testing"

func TestPreferredTerminalAppPreference(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		env         map[string]string
		appName     string
		displayName string
		source      string
		supported   bool
	}{
		{
			name:        "default terminal",
			env:         map[string]string{},
			appName:     "Terminal",
			displayName: "Terminal.app",
			source:      "default",
			supported:   true,
		},
		{
			name:        "bundle id override",
			env:         map[string]string{"PRTR_TERMINAL_APP": "com.googlecode.iterm2"},
			appName:     "iTerm",
			displayName: "iTerm.app",
			source:      "PRTR_TERMINAL_APP",
			supported:   true,
		},
		{
			name:        "custom app override",
			env:         map[string]string{"PRTR_TERMINAL_APP": "Ghostty"},
			appName:     "Ghostty",
			displayName: "Ghostty",
			source:      "PRTR_TERMINAL_APP",
			supported:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pref := preferredTerminalAppPreference(func(key string) (string, bool) {
				value, ok := tc.env[key]
				return value, ok
			})
			if pref.AppName != tc.appName {
				t.Fatalf("AppName = %q, want %q", pref.AppName, tc.appName)
			}
			if pref.DisplayName != tc.displayName {
				t.Fatalf("DisplayName = %q, want %q", pref.DisplayName, tc.displayName)
			}
			if pref.Source != tc.source {
				t.Fatalf("Source = %q, want %q", pref.Source, tc.source)
			}
			if pref.Supported != tc.supported {
				t.Fatalf("Supported = %v, want %v", pref.Supported, tc.supported)
			}
		})
	}
}
