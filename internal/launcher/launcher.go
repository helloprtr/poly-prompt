package launcher

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

var ErrUnsupportedPlatform = errors.New("launcher is unsupported on this platform")

type Request struct {
	Command string
	Args    []string
}

type Launcher interface {
	Launch(ctx context.Context, req Request) error
	Diagnose(req Request) error
	Describe(req Request) (string, error)
}

type TerminalLauncher struct {
	goos     string
	lookPath func(string) (string, error)
	runApple func(context.Context, string) error
	runStart func(context.Context, string, ...string) error
}

func New() *TerminalLauncher {
	return &TerminalLauncher{
		goos:     runtime.GOOS,
		lookPath: exec.LookPath,
		runApple: runAppleScript,
		runStart: runDetachedProcess,
	}
}

func NewForTesting(goos string, lookPath func(string) (string, error), runApple func(context.Context, string) error, runStart func(context.Context, string, ...string) error) *TerminalLauncher {
	return &TerminalLauncher{
		goos:     goos,
		lookPath: lookPath,
		runApple: runApple,
		runStart: runStart,
	}
}

func (l *TerminalLauncher) Diagnose(req Request) error {
	if strings.TrimSpace(req.Command) == "" {
		return errors.New("launcher command is empty")
	}
	if _, err := l.lookPath(req.Command); err != nil {
		return fmt.Errorf("launcher command %q was not found: %w", req.Command, err)
	}
	switch l.goos {
	case "darwin":
		return nil
	case "linux":
		_, err := l.selectLinuxBackend()
		return err
	case "windows":
		_, err := l.selectWindowsBackend()
		return err
	default:
		return ErrUnsupportedPlatform
	}
}

func (l *TerminalLauncher) Describe(req Request) (string, error) {
	if err := l.Diagnose(req); err != nil {
		return "", err
	}

	switch l.goos {
	case "darwin":
		return "Terminal.app", nil
	case "linux":
		backend, err := l.selectLinuxBackend()
		if err != nil {
			return "", err
		}
		return backend.name, nil
	case "windows":
		backend, err := l.selectWindowsBackend()
		if err != nil {
			return "", err
		}
		return backend.name, nil
	default:
		return "", ErrUnsupportedPlatform
	}
}

func (l *TerminalLauncher) Launch(ctx context.Context, req Request) error {
	if err := l.Diagnose(req); err != nil {
		return err
	}

	switch l.goos {
	case "darwin":
		commandLine := shellQuote(req.Command)
		for _, arg := range req.Args {
			commandLine += " " + shellQuote(arg)
		}
		script := fmt.Sprintf(`tell application "Terminal"
activate
do script %q
end tell`, commandLine)

		if err := l.runApple(ctx, script); err != nil {
			return fmt.Errorf("launch target command: %w", err)
		}
		return nil
	case "linux":
		commandLine := shellQuote(req.Command)
		for _, arg := range req.Args {
			commandLine += " " + shellQuote(arg)
		}
		backend, err := l.selectLinuxBackend()
		if err != nil {
			return err
		}
		if err := l.runStart(ctx, backend.command, backend.args(commandLine)...); err != nil {
			return fmt.Errorf("launch target command via %s: %w", backend.name, err)
		}
		return nil
	case "windows":
		backend, err := l.selectWindowsBackend()
		if err != nil {
			return err
		}
		if err := l.runStart(ctx, backend.command, backend.args(req)...); err != nil {
			return fmt.Errorf("launch target command via %s: %w", backend.name, err)
		}
		return nil
	default:
		return ErrUnsupportedPlatform
	}
}

func runAppleScript(ctx context.Context, script string) error {
	output, err := exec.CommandContext(ctx, "osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func runDetachedProcess(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
