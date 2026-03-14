package automation

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestMacOSAutomatorDiagnose(t *testing.T) {
	t.Parallel()

	automator := NewForTesting("darwin", func(name string) (string, error) {
		if name == "osascript" {
			return "/usr/bin/osascript", nil
		}
		return "", exec.ErrNotFound
	}, func(_ context.Context, script string) (string, error) {
		if strings.Contains(script, "UI elements enabled") {
			return "true", nil
		}
		if strings.Contains(script, `tell application "Terminal" to get name`) {
			return "Terminal", nil
		}
		return "", nil
	})

	if err := automator.Diagnose(Request{TerminalApp: "Terminal"}); err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
}

func TestMacOSAutomatorPasteBuildsScript(t *testing.T) {
	t.Parallel()

	var scripts []string
	automator := NewForTesting("darwin", func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}, func(_ context.Context, script string) (string, error) {
		scripts = append(scripts, script)
		if strings.Contains(script, "UI elements enabled") {
			return "true", nil
		}
		return "", nil
	})

	err := automator.Paste(context.Background(), Request{
		Target:      "claude",
		TerminalApp: "Terminal",
		PasteDelay:  700 * time.Millisecond,
		SubmitMode:  SubmitManual,
	})
	if err != nil {
		t.Fatalf("Paste() error = %v", err)
	}
	if len(scripts) < 3 {
		t.Fatalf("scripts = %d, want at least 3", len(scripts))
	}
	if !strings.Contains(scripts[len(scripts)-1], `keystroke "v" using command down`) {
		t.Fatalf("script = %q", scripts[len(scripts)-1])
	}
}

func TestMacOSAutomatorSubmitRejectsUnsupportedMode(t *testing.T) {
	t.Parallel()

	automator := NewForTesting("darwin", func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}, func(_ context.Context, script string) (string, error) {
		if strings.Contains(script, "UI elements enabled") {
			return "true", nil
		}
		return "", nil
	})

	err := automator.Submit(context.Background(), Request{
		TerminalApp: "Terminal",
		SubmitMode:  SubmitManual,
	})
	if !errors.Is(err, ErrUnsupportedSubmitMode) {
		t.Fatalf("Submit() error = %v", err)
	}
}

func TestMacOSAutomatorUnsupportedPlatform(t *testing.T) {
	t.Parallel()

	automator := NewForTesting("linux", func(string) (string, error) {
		return "", nil
	}, func(context.Context, string) (string, error) {
		return "", nil
	})

	if !errors.Is(automator.Diagnose(Request{TerminalApp: "Terminal"}), ErrUnsupportedPlatform) {
		t.Fatal("Diagnose() expected unsupported platform error")
	}
}

func TestMacOSAutomatorAccessibilityFailure(t *testing.T) {
	t.Parallel()

	automator := NewForTesting("darwin", func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}, func(_ context.Context, script string) (string, error) {
		if strings.Contains(script, "UI elements enabled") {
			return "", errors.New("not authorized")
		}
		return "", nil
	})

	err := automator.Diagnose(Request{TerminalApp: "Terminal"})
	if !errors.Is(err, ErrAccessibilityUnavailable) {
		t.Fatalf("Diagnose() error = %v", err)
	}
}
