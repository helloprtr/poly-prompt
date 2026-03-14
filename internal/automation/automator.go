package automation

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"time"
)

var (
	ErrUnsupportedPlatform      = errors.New("automation is unsupported on this platform")
	ErrAccessibilityUnavailable = errors.New("automation accessibility permission is unavailable")
	ErrTerminalNotFrontmost     = errors.New("terminal is not frontmost")
	ErrUnsupportedSubmitMode    = errors.New("submit mode is not supported")
)

type SubmitMode string

const (
	SubmitManual  SubmitMode = "manual"
	SubmitConfirm SubmitMode = "confirm"
	SubmitAuto    SubmitMode = "auto"
)

type Request struct {
	Target           string
	TerminalApp      string
	PasteDelay       time.Duration
	RequireClipboard bool
	SubmitMode       SubmitMode
}

type Automator interface {
	Diagnose(req Request) error
	Describe(req Request) (string, error)
	Paste(ctx context.Context, req Request) error
	Submit(ctx context.Context, req Request) error
}

func New() Automator {
	switch runtime.GOOS {
	case "darwin":
		return &MacOSAutomator{
			goos:     runtime.GOOS,
			lookPath: exec.LookPath,
			runApple: runAppleScript,
		}
	case "linux":
		return &LinuxAutomator{
			goos:      runtime.GOOS,
			lookPath:  exec.LookPath,
			run:       runCommand,
			lookupEnv: lookupEnv,
		}
	case "windows":
		return &WindowsAutomator{
			goos:      runtime.GOOS,
			lookPath:  exec.LookPath,
			run:       runCommand,
			lookupEnv: lookupEnv,
		}
	default:
		return &unsupportedAutomator{}
	}
}

type unsupportedAutomator struct{}

func (a *unsupportedAutomator) Diagnose(Request) error {
	return ErrUnsupportedPlatform
}

func (a *unsupportedAutomator) Describe(Request) (string, error) {
	return "", ErrUnsupportedPlatform
}

func (a *unsupportedAutomator) Paste(context.Context, Request) error {
	return ErrUnsupportedPlatform
}

func (a *unsupportedAutomator) Submit(context.Context, Request) error {
	return ErrUnsupportedPlatform
}
