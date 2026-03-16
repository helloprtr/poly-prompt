package automation

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type runAppleScriptFunc func(context.Context, string) (string, error)

type MacOSAutomator struct {
	goos     string
	lookPath func(string) (string, error)
	runApple runAppleScriptFunc
}

func NewForTesting(goos string, lookPath func(string) (string, error), runApple runAppleScriptFunc) *MacOSAutomator {
	return &MacOSAutomator{
		goos:     goos,
		lookPath: lookPath,
		runApple: runApple,
	}
}

func (a *MacOSAutomator) Diagnose(req Request) error {
	if a.goos != "darwin" {
		return ErrUnsupportedPlatform
	}
	if strings.TrimSpace(req.TerminalApp) == "" {
		return errors.New("terminal app is empty")
	}
	if _, err := a.lookPath("osascript"); err != nil {
		return fmt.Errorf("automation requires osascript: %w", err)
	}
	output, err := a.runApple(context.Background(), `tell application "System Events" to return UI elements enabled`)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAccessibilityUnavailable, err)
	}
	if strings.TrimSpace(strings.ToLower(output)) != "true" {
		return ErrAccessibilityUnavailable
	}
	if _, err := a.runApple(context.Background(), fmt.Sprintf(`tell application %q to get name`, req.TerminalApp)); err != nil {
		return fmt.Errorf("terminal app %q is unavailable: %w", req.TerminalApp, err)
	}
	return nil
}

func (a *MacOSAutomator) Describe(req Request) (string, error) {
	if err := a.Diagnose(req); err != nil {
		return "", err
	}
	return "macOS via osascript", nil
}

func (a *MacOSAutomator) Paste(ctx context.Context, req Request) error {
	if err := a.Diagnose(req); err != nil {
		return err
	}

	_, err := a.runApple(ctx, fmt.Sprintf(`tell application %q to activate
delay %s
tell application "System Events"
if name of first application process whose frontmost is true is not %q then error "terminal-not-frontmost"
keystroke "v" using command down
end tell`, req.TerminalApp, formatDelay(req.PasteDelay), req.TerminalApp))
	if err != nil {
		if strings.Contains(err.Error(), "terminal-not-frontmost") {
			return ErrTerminalNotFrontmost
		}
		return fmt.Errorf("paste into terminal: %w", err)
	}
	return nil
}

func (a *MacOSAutomator) Submit(ctx context.Context, req Request) error {
	if err := a.Diagnose(req); err != nil {
		return err
	}
	if req.SubmitMode != SubmitConfirm && req.SubmitMode != SubmitAuto {
		return ErrUnsupportedSubmitMode
	}

	_, err := a.runApple(ctx, fmt.Sprintf(`tell application %q to activate
tell application "System Events"
if name of first application process whose frontmost is true is not %q then error "terminal-not-frontmost"
key code 36
end tell`, req.TerminalApp, req.TerminalApp))
	if err != nil {
		if strings.Contains(err.Error(), "terminal-not-frontmost") {
			return ErrTerminalNotFrontmost
		}
		return fmt.Errorf("submit from terminal: %w", err)
	}
	return nil
}

func runAppleScript(ctx context.Context, script string) (string, error) {
	output, err := exec.CommandContext(ctx, "osascript", "-e", script).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func formatDelay(delay time.Duration) string {
	if delay <= 0 {
		return "0"
	}
	return strconv.FormatFloat(delay.Seconds(), 'f', 3, 64)
}
