package launcher

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestTerminalLauncherDiagnose(t *testing.T) {
	t.Parallel()

	launcher := NewForTesting("darwin", func(name string) (string, error) {
		if name == "claude" {
			return "/usr/local/bin/claude", nil
		}
		return "", exec.ErrNotFound
	}, func(context.Context, string) error { return nil }, func(context.Context, string, ...string) error { return nil })

	if err := launcher.Diagnose(Request{Command: "claude"}); err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	err := launcher.Diagnose(Request{Command: "missing"})
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("Diagnose() error = %v", err)
	}
}

func TestTerminalLauncherLaunchBuildsAppleScript(t *testing.T) {
	t.Parallel()

	var script string
	launcher := NewForTesting("darwin", func(name string) (string, error) {
		return "/usr/local/bin/" + name, nil
	}, func(_ context.Context, value string) error {
		script = value
		return nil
	}, func(context.Context, string, ...string) error { return nil })

	if err := launcher.Launch(context.Background(), Request{Command: "codex", Args: []string{"chat", "--model", "gpt-5"}}); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}

	if !strings.Contains(script, "tell application \"Terminal\"") {
		t.Fatalf("script = %q", script)
	}
	if !strings.Contains(script, "'codex' 'chat' '--model' 'gpt-5'") {
		t.Fatalf("script = %q", script)
	}
}

func TestTerminalLauncherLaunchUsesITermOnDarwin(t *testing.T) {
	t.Parallel()

	var script string
	launcher := NewForTesting("darwin", func(name string) (string, error) {
		return "/usr/local/bin/" + name, nil
	}, func(_ context.Context, value string) error {
		script = value
		return nil
	}, func(context.Context, string, ...string) error { return nil })

	if err := launcher.Launch(context.Background(), Request{Command: "codex", Args: []string{"chat"}, TerminalApp: "iTerm"}); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}

	if !strings.Contains(script, `tell application id "com.googlecode.iterm2"`) {
		t.Fatalf("script = %q", script)
	}
	if !strings.Contains(script, `create window with default profile command`) {
		t.Fatalf("script = %q", script)
	}
}

func TestTerminalLauncherDescribeDarwinITerm(t *testing.T) {
	t.Parallel()

	launcher := NewForTesting("darwin", func(name string) (string, error) {
		return "/usr/local/bin/" + name, nil
	}, func(context.Context, string) error { return nil }, func(context.Context, string, ...string) error { return nil })

	description, err := launcher.Describe(Request{Command: "codex", TerminalApp: "iTerm"})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if description != "iTerm.app" {
		t.Fatalf("description = %q", description)
	}
}

func TestTerminalLauncherDiagnoseRejectsUnsupportedDarwinTerminalApp(t *testing.T) {
	t.Parallel()

	launcher := NewForTesting("darwin", func(name string) (string, error) {
		return "/usr/local/bin/" + name, nil
	}, func(context.Context, string) error { return nil }, func(context.Context, string, ...string) error { return nil })

	err := launcher.Diagnose(Request{Command: "codex", TerminalApp: "Ghostty"})
	if err == nil {
		t.Fatal("Diagnose() expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported for macOS launch handoff") {
		t.Fatalf("error = %v", err)
	}
}

func TestTerminalLauncherLaunchUsesLinuxTerminal(t *testing.T) {
	t.Parallel()

	var (
		startedName string
		startedArgs []string
	)

	launcher := NewForTesting("linux", func(name string) (string, error) {
		switch name {
		case "codex":
			return "/usr/local/bin/codex", nil
		case "gnome-terminal":
			return "/usr/bin/gnome-terminal", nil
		default:
			return "", exec.ErrNotFound
		}
	}, func(context.Context, string) error { return nil }, func(_ context.Context, name string, args ...string) error {
		startedName = name
		startedArgs = append([]string(nil), args...)
		return nil
	})

	if err := launcher.Launch(context.Background(), Request{Command: "codex", Args: []string{"chat", "--model", "gpt-5"}}); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}

	if startedName != "gnome-terminal" {
		t.Fatalf("startedName = %q", startedName)
	}
	if len(startedArgs) < 4 {
		t.Fatalf("startedArgs = %#v", startedArgs)
	}
	if startedArgs[0] != "--" || startedArgs[1] != "/bin/sh" || startedArgs[2] != "-lc" {
		t.Fatalf("startedArgs = %#v", startedArgs)
	}
	if !strings.Contains(startedArgs[3], "'codex' 'chat' '--model' 'gpt-5'") {
		t.Fatalf("startedArgs = %#v", startedArgs)
	}
}

func TestTerminalLauncherLinuxPrefersFirstSupportedBackend(t *testing.T) {
	t.Parallel()

	launcher := NewForTesting("linux", func(name string) (string, error) {
		switch name {
		case "claude":
			return "/usr/local/bin/claude", nil
		case "x-terminal-emulator":
			return "/usr/bin/x-terminal-emulator", nil
		case "gnome-terminal":
			return "/usr/bin/gnome-terminal", nil
		default:
			return "", exec.ErrNotFound
		}
	}, func(context.Context, string) error { return nil }, func(context.Context, string, ...string) error { return nil })

	if err := launcher.Diagnose(Request{Command: "claude"}); err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	backend, err := launcher.selectLinuxBackend()
	if err != nil {
		t.Fatalf("selectLinuxBackend() error = %v", err)
	}
	if backend.name != "x-terminal-emulator" {
		t.Fatalf("backend.name = %q", backend.name)
	}
}

func TestTerminalLauncherDescribeLinuxBackend(t *testing.T) {
	t.Parallel()

	launcher := NewForTesting("linux", func(name string) (string, error) {
		switch name {
		case "gemini":
			return "/usr/local/bin/gemini", nil
		case "wezterm":
			return "/usr/bin/wezterm", nil
		default:
			return "", exec.ErrNotFound
		}
	}, func(context.Context, string) error { return nil }, func(context.Context, string, ...string) error { return nil })

	description, err := launcher.Describe(Request{Command: "gemini"})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if description != "wezterm" {
		t.Fatalf("description = %q", description)
	}
}

func TestTerminalLauncherLaunchUsesWindowsTerminal(t *testing.T) {
	t.Parallel()

	var (
		startedName string
		startedArgs []string
	)

	launcher := NewForTesting("windows", func(name string) (string, error) {
		switch name {
		case "claude":
			return `C:\tools\claude.exe`, nil
		case "wt.exe":
			return `C:\Windows\System32\wt.exe`, nil
		default:
			return "", exec.ErrNotFound
		}
	}, func(context.Context, string) error { return nil }, func(_ context.Context, name string, args ...string) error {
		startedName = name
		startedArgs = append([]string(nil), args...)
		return nil
	})

	if err := launcher.Launch(context.Background(), Request{Command: "claude", Args: []string{"chat", "--model", "gpt-5"}}); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}

	if startedName != "wt.exe" {
		t.Fatalf("startedName = %q", startedName)
	}
	expected := []string{"new-tab", "claude", "chat", "--model", "gpt-5"}
	if strings.Join(startedArgs, "|") != strings.Join(expected, "|") {
		t.Fatalf("startedArgs = %#v", startedArgs)
	}
}

func TestTerminalLauncherDescribeWindowsBackend(t *testing.T) {
	t.Parallel()

	launcher := NewForTesting("windows", func(name string) (string, error) {
		switch name {
		case "codex":
			return `C:\tools\codex.exe`, nil
		case "powershell.exe":
			return `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`, nil
		default:
			return "", exec.ErrNotFound
		}
	}, func(context.Context, string) error { return nil }, func(context.Context, string, ...string) error { return nil })

	description, err := launcher.Describe(Request{Command: "codex"})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if description != "powershell.exe" {
		t.Fatalf("description = %q", description)
	}
}

func TestWindowsCommandFallbacksUseExpectedQuoting(t *testing.T) {
	t.Parallel()

	req := Request{Command: "codex", Args: []string{"say hello", `a"b`, "plain"}}

	if got := powerShellCommandLine(req); got != `& 'codex' 'say hello' 'a"b' 'plain'` {
		t.Fatalf("powerShellCommandLine() = %q", got)
	}
	if got := cmdCommandLine(req); got != `"codex" "say hello" "a\"b" "plain"` {
		t.Fatalf("cmdCommandLine() = %q", got)
	}
}

func TestTerminalLauncherRejectsUnsupportedPlatform(t *testing.T) {
	t.Parallel()

	launcher := NewForTesting("plan9", func(string) (string, error) {
		return "", nil
	}, func(context.Context, string) error { return nil }, func(context.Context, string, ...string) error { return nil })

	if !errors.Is(launcher.Diagnose(Request{Command: "claude"}), ErrUnsupportedPlatform) {
		t.Fatal("Diagnose() expected unsupported platform error")
	}
}
