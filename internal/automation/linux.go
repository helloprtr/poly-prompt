package automation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrGraphicalSessionRequired = errors.New("automation requires a graphical Linux session")

type LinuxAutomator struct {
	goos      string
	lookPath  func(string) (string, error)
	run       runCommandFunc
	lookupEnv func(string) string
}

type linuxPasteBackend struct {
	name    string
	session string
}

func NewLinuxForTesting(goos string, lookPath func(string) (string, error), run runCommandFunc, lookupEnv func(string) string) *LinuxAutomator {
	return &LinuxAutomator{
		goos:      goos,
		lookPath:  lookPath,
		run:       run,
		lookupEnv: lookupEnv,
	}
}

func (a *LinuxAutomator) Diagnose(req Request) error {
	_, err := a.describeBackend(req)
	return err
}

func (a *LinuxAutomator) Describe(req Request) (string, error) {
	backend, err := a.describeBackend(req)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s via %s", backend.session, backend.name), nil
}

func (a *LinuxAutomator) Paste(ctx context.Context, req Request) error {
	backend, err := a.describeBackend(req)
	if err != nil {
		return err
	}

	if err := waitForDelay(ctx, req.PasteDelay); err != nil {
		return err
	}

	switch backend.name {
	case "xdotool":
		title, err := a.run(ctx, "xdotool", "getactivewindow", "getwindowname")
		if err != nil {
			return fmt.Errorf("confirm active terminal window: %w", err)
		}
		if !looksLikeTerminalWindow(title, req.Target) {
			return ErrTerminalNotFrontmost
		}
		if _, err := a.run(ctx, "xdotool", "key", "--clearmodifiers", "ctrl+shift+v"); err != nil {
			return fmt.Errorf("paste into terminal: %w", err)
		}
		return nil
	case "wtype":
		if _, err := a.run(ctx, "wtype", "-M", "ctrl", "-M", "shift", "v", "-m", "shift", "-m", "ctrl"); err != nil {
			return fmt.Errorf("paste into terminal: %w", err)
		}
		return nil
	default:
		return ErrUnsupportedPlatform
	}
}

func (a *LinuxAutomator) Submit(context.Context, Request) error {
	return ErrUnsupportedSubmitMode
}

func (a *LinuxAutomator) describeBackend(req Request) (linuxPasteBackend, error) {
	if a.goos != "linux" {
		return linuxPasteBackend{}, ErrUnsupportedPlatform
	}

	waylandDisplay := strings.TrimSpace(a.lookupEnv("WAYLAND_DISPLAY"))
	display := strings.TrimSpace(a.lookupEnv("DISPLAY"))

	if waylandDisplay != "" {
		if _, err := a.lookPath("wtype"); err == nil {
			return linuxPasteBackend{name: "wtype", session: "wayland"}, nil
		}
	}
	if display != "" {
		if _, err := a.lookPath("xdotool"); err == nil {
			return linuxPasteBackend{name: "xdotool", session: "x11"}, nil
		}
	}

	if waylandDisplay == "" && display == "" {
		return linuxPasteBackend{}, ErrGraphicalSessionRequired
	}

	_ = req
	return linuxPasteBackend{}, errors.New("automation requires xdotool on X11 or wtype on Wayland")
}

func looksLikeTerminalWindow(title, target string) bool {
	lower := strings.ToLower(strings.TrimSpace(title))
	if lower == "" {
		return false
	}

	hints := []string{
		"terminal", "konsole", "kitty", "wezterm", "shell", "bash", "zsh", "fish", "claude", "codex", "gemini",
	}
	if strings.TrimSpace(target) != "" {
		hints = append(hints, strings.ToLower(target))
	}
	for _, hint := range hints {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

func waitForDelay(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
