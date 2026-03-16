package app

import "testing"

func TestDetectPlatformSurface(t *testing.T) {
	cases := []struct {
		name      string
		goos      string
		env       map[string]string
		label     string
		supported bool
	}{
		{name: "mac", goos: "darwin", label: "macOS + Terminal.app", supported: true},
		{name: "mac iterm", goos: "darwin", env: map[string]string{"TERM_PROGRAM": "iTerm.app"}, label: "macOS + iTerm.app", supported: true},
		{name: "mac iterm override", goos: "darwin", env: map[string]string{"PRTR_TERMINAL_APP": "iTerm"}, label: "macOS + iTerm.app", supported: true},
		{name: "x11", goos: "linux", env: map[string]string{"DISPLAY": ":0"}, label: "Linux + X11", supported: true},
		{name: "wayland", goos: "linux", env: map[string]string{"WAYLAND_DISPLAY": "wayland-0"}, label: "Linux + Wayland", supported: true},
		{name: "windows interactive", goos: "windows", env: map[string]string{"SESSIONNAME": "Console"}, label: "Windows interactive session", supported: true},
		{name: "windows service", goos: "windows", env: map[string]string{"SESSIONNAME": "Services"}, label: "unsupported platform", supported: false},
		{name: "unknown", goos: "freebsd", label: "unsupported platform", supported: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			surface := detectPlatformSurface(tc.goos, func(key string) (string, bool) {
				value, ok := tc.env[key]
				return value, ok
			})
			if surface.Label != tc.label {
				t.Fatalf("Label = %q, want %q", surface.Label, tc.label)
			}
			if surface.Supported != tc.supported {
				t.Fatalf("Supported = %v, want %v", surface.Supported, tc.supported)
			}
		})
	}
}
