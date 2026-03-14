package automation

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestLinuxAutomatorDescribeX11Backend(t *testing.T) {
	t.Parallel()

	automator := NewLinuxForTesting("linux", func(name string) (string, error) {
		if name == "xdotool" {
			return "/usr/bin/xdotool", nil
		}
		return "", exec.ErrNotFound
	}, func(context.Context, string, ...string) (string, error) {
		return "", nil
	}, func(key string) string {
		if key == "DISPLAY" {
			return ":0"
		}
		return ""
	})

	description, err := automator.Describe(Request{Target: "claude"})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if description != "x11 via xdotool" {
		t.Fatalf("description = %q", description)
	}
}

func TestLinuxAutomatorDescribeWaylandBackend(t *testing.T) {
	t.Parallel()

	automator := NewLinuxForTesting("linux", func(name string) (string, error) {
		if name == "wtype" {
			return "/usr/bin/wtype", nil
		}
		return "", exec.ErrNotFound
	}, func(context.Context, string, ...string) (string, error) {
		return "", nil
	}, func(key string) string {
		if key == "WAYLAND_DISPLAY" {
			return "wayland-0"
		}
		return ""
	})

	description, err := automator.Describe(Request{Target: "codex"})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if description != "wayland via wtype" {
		t.Fatalf("description = %q", description)
	}
}

func TestLinuxAutomatorPasteUsesXdotool(t *testing.T) {
	t.Parallel()

	var calls [][]string
	automator := NewLinuxForTesting("linux", func(name string) (string, error) {
		if name == "xdotool" {
			return "/usr/bin/xdotool", nil
		}
		return "", exec.ErrNotFound
	}, func(_ context.Context, name string, args ...string) (string, error) {
		call := append([]string{name}, args...)
		calls = append(calls, call)
		if len(args) >= 2 && args[0] == "getactivewindow" && args[1] == "getwindowname" {
			return "Claude Terminal", nil
		}
		return "", nil
	}, func(key string) string {
		if key == "DISPLAY" {
			return ":0"
		}
		return ""
	})

	err := automator.Paste(context.Background(), Request{Target: "claude", PasteDelay: 1 * time.Millisecond})
	if err != nil {
		t.Fatalf("Paste() error = %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %#v", calls)
	}
	if calls[1][0] != "xdotool" || calls[1][1] != "key" {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestLinuxAutomatorPasteRejectsNonTerminalWindow(t *testing.T) {
	t.Parallel()

	automator := NewLinuxForTesting("linux", func(name string) (string, error) {
		if name == "xdotool" {
			return "/usr/bin/xdotool", nil
		}
		return "", exec.ErrNotFound
	}, func(_ context.Context, name string, args ...string) (string, error) {
		if name == "xdotool" && len(args) >= 2 && args[0] == "getactivewindow" && args[1] == "getwindowname" {
			return "Figma", nil
		}
		return "", nil
	}, func(key string) string {
		if key == "DISPLAY" {
			return ":0"
		}
		return ""
	})

	err := automator.Paste(context.Background(), Request{Target: "claude"})
	if !errors.Is(err, ErrTerminalNotFrontmost) {
		t.Fatalf("Paste() error = %v", err)
	}
}

func TestLinuxAutomatorRequiresGraphicalSession(t *testing.T) {
	t.Parallel()

	automator := NewLinuxForTesting("linux", func(string) (string, error) {
		return "", exec.ErrNotFound
	}, func(context.Context, string, ...string) (string, error) {
		return "", nil
	}, func(string) string {
		return ""
	})

	err := automator.Diagnose(Request{Target: "gemini"})
	if !errors.Is(err, ErrGraphicalSessionRequired) {
		t.Fatalf("Diagnose() error = %v", err)
	}
}

func TestLinuxAutomatorSubmitUnsupported(t *testing.T) {
	t.Parallel()

	automator := NewLinuxForTesting("linux", func(string) (string, error) {
		return "", nil
	}, func(context.Context, string, ...string) (string, error) {
		return "", nil
	}, func(string) string {
		return ":0"
	})

	if !errors.Is(automator.Submit(context.Background(), Request{SubmitMode: SubmitConfirm}), ErrUnsupportedSubmitMode) {
		t.Fatal("Submit() expected unsupported submit mode error")
	}
}

func TestLooksLikeTerminalWindow(t *testing.T) {
	t.Parallel()

	if !looksLikeTerminalWindow("Codex - Terminal", "codex") {
		t.Fatal("looksLikeTerminalWindow() expected true")
	}
	if looksLikeTerminalWindow("Browser", "codex") {
		t.Fatal("looksLikeTerminalWindow() expected false")
	}
}

func TestLinuxAutomatorMissingPasteTool(t *testing.T) {
	t.Parallel()

	automator := NewLinuxForTesting("linux", func(string) (string, error) {
		return "", exec.ErrNotFound
	}, func(context.Context, string, ...string) (string, error) {
		return "", nil
	}, func(key string) string {
		if key == "WAYLAND_DISPLAY" {
			return "wayland-0"
		}
		return ""
	})

	err := automator.Diagnose(Request{})
	if err == nil || !strings.Contains(err.Error(), "xdotool") && !strings.Contains(err.Error(), "wtype") {
		t.Fatalf("Diagnose() error = %v", err)
	}
}
