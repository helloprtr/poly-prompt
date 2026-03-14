package automation

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestWindowsAutomatorDescribe(t *testing.T) {
	t.Parallel()

	automator := NewWindowsForTesting("windows", func(name string) (string, error) {
		if name == "powershell.exe" {
			return `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`, nil
		}
		return "", exec.ErrNotFound
	}, func(context.Context, string, ...string) (string, error) {
		return "", nil
	}, func(key string) string {
		if key == "SESSIONNAME" {
			return "Console"
		}
		return ""
	})

	description, err := automator.Describe(Request{Target: "claude"})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if description != "sendkeys via powershell.exe" {
		t.Fatalf("description = %q", description)
	}
}

func TestWindowsAutomatorPasteUsesPowerShellScript(t *testing.T) {
	t.Parallel()

	var (
		name string
		args []string
	)
	automator := NewWindowsForTesting("windows", func(command string) (string, error) {
		if command == "pwsh.exe" {
			return `C:\Program Files\PowerShell\7\pwsh.exe`, nil
		}
		if command == "powershell.exe" {
			return "", exec.ErrNotFound
		}
		return "", exec.ErrNotFound
	}, func(_ context.Context, command string, commandArgs ...string) (string, error) {
		name = command
		args = append([]string(nil), commandArgs...)
		return "", nil
	}, func(key string) string {
		if key == "SESSIONNAME" {
			return "Console"
		}
		return ""
	})

	err := automator.Paste(context.Background(), Request{Target: "codex"})
	if err != nil {
		t.Fatalf("Paste() error = %v", err)
	}
	if name != "pwsh.exe" {
		t.Fatalf("name = %q", name)
	}
	if len(args) < 4 || args[2] != "-Command" {
		t.Fatalf("args = %#v", args)
	}
	if !strings.Contains(args[3], "SendKeys") || !strings.Contains(args[3], "terminal-not-frontmost") {
		t.Fatalf("script = %q", args[3])
	}
}

func TestWindowsAutomatorRejectsServiceSession(t *testing.T) {
	t.Parallel()

	automator := NewWindowsForTesting("windows", func(name string) (string, error) {
		if name == "powershell.exe" {
			return "powershell.exe", nil
		}
		return "", exec.ErrNotFound
	}, func(context.Context, string, ...string) (string, error) {
		return "", nil
	}, func(key string) string {
		if key == "SESSIONNAME" {
			return "Services"
		}
		return ""
	})

	err := automator.Diagnose(Request{})
	if !errors.Is(err, ErrInteractiveDesktopRequired) {
		t.Fatalf("Diagnose() error = %v", err)
	}
}

func TestWindowsAutomatorSubmitUnsupported(t *testing.T) {
	t.Parallel()

	automator := NewWindowsForTesting("windows", func(string) (string, error) {
		return "", nil
	}, func(context.Context, string, ...string) (string, error) {
		return "", nil
	}, func(string) string {
		return "Console"
	})

	if !errors.Is(automator.Submit(context.Background(), Request{SubmitMode: SubmitConfirm}), ErrUnsupportedSubmitMode) {
		t.Fatal("Submit() expected unsupported submit mode error")
	}
}

func TestBuildWindowsPasteScriptIncludesTargetHint(t *testing.T) {
	t.Parallel()

	script := buildWindowsPasteScript("claude")
	if !strings.Contains(script, "claude") {
		t.Fatalf("script = %q", script)
	}
	if !strings.Contains(script, "SendWait") {
		t.Fatalf("script = %q", script)
	}
}
