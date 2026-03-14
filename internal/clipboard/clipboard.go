package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type Accessor interface {
	Copy(ctx context.Context, text string) error
	Read(ctx context.Context) (string, error)
}

type Diagnoser interface {
	Diagnose() error
}

type LookPathFunc func(string) (string, error)

type AutoClipboard struct {
	goos     string
	lookPath LookPathFunc
}

type commandClipboard struct {
	command string
	args    []string
}

func New() Accessor {
	return &AutoClipboard{
		goos:     runtime.GOOS,
		lookPath: exec.LookPath,
	}
}

func NewForTesting(goos string, lookPath LookPathFunc) *AutoClipboard {
	return &AutoClipboard{
		goos:     goos,
		lookPath: lookPath,
	}
}

func (c *AutoClipboard) Copy(ctx context.Context, text string) error {
	backend, err := detectClipboardWriter(c.goos, c.lookPath)
	if err != nil {
		return err
	}

	return backend.Copy(ctx, text)
}

func (c *AutoClipboard) Read(ctx context.Context) (string, error) {
	backend, err := detectClipboardReader(c.goos, c.lookPath)
	if err != nil {
		return "", err
	}

	return backend.Read(ctx)
}

func (c *AutoClipboard) Diagnose() error {
	_, err := detectClipboardWriter(c.goos, c.lookPath)
	return err
}

func (c *commandClipboard) Copy(ctx context.Context, text string) error {
	cmd := exec.CommandContext(ctx, c.command, c.args...)
	cmd.Stdin = bytes.NewBufferString(text)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("copy to clipboard with %s: %w: %s", c.command, err, string(output))
	}

	return nil
}

func (c *commandClipboard) Read(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, c.command, c.args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("read clipboard with %s: %w: %s", c.command, err, string(output))
	}

	return strings.TrimRight(string(output), "\r\n"), nil
}

func detectClipboardWriter(goos string, lookPath LookPathFunc) (copyBackend, error) {
	switch goos {
	case "darwin":
		return detectFirst(lookPath, []backendCandidate{
			{name: "pbcopy"},
		}, "clipboard support is unavailable on this macOS system because pbcopy was not found")
	case "linux":
		return detectFirst(lookPath, []backendCandidate{
			{name: "wl-copy"},
			{name: "xclip", args: []string{"-selection", "clipboard"}},
			{name: "xsel", args: []string{"--clipboard", "--input"}},
		}, "no clipboard tool found on Linux; install wl-clipboard, xclip, or xsel and try again")
	case "windows":
		return detectFirst(lookPath, []backendCandidate{
			{name: "clip.exe"},
		}, "clipboard support is unavailable on this Windows system because clip.exe was not found")
	default:
		return nil, fmt.Errorf("clipboard is not supported on %s", goos)
	}
}

func detectClipboardReader(goos string, lookPath LookPathFunc) (readBackend, error) {
	switch goos {
	case "darwin":
		return detectFirst(lookPath, []backendCandidate{
			{name: "pbpaste"},
		}, "clipboard read is unavailable on this macOS system because pbpaste was not found")
	case "linux":
		return detectFirst(lookPath, []backendCandidate{
			{name: "wl-paste", args: []string{"--no-newline"}},
			{name: "xclip", args: []string{"-selection", "clipboard", "-o"}},
			{name: "xsel", args: []string{"--clipboard", "--output"}},
		}, "no clipboard read tool found on Linux; install wl-clipboard, xclip, or xsel and try again")
	case "windows":
		return detectFirst(lookPath, []backendCandidate{
			{name: "powershell.exe", args: []string{"-NoProfile", "-Command", "Get-Clipboard"}},
		}, "clipboard read is unavailable on this Windows system because powershell.exe was not found")
	default:
		return nil, fmt.Errorf("clipboard is not supported on %s", goos)
	}
}

type backendCandidate struct {
	name string
	args []string
}

type copyBackend interface {
	Copy(ctx context.Context, text string) error
}

type readBackend interface {
	Read(ctx context.Context) (string, error)
}

func detectFirst(lookPath LookPathFunc, candidates []backendCandidate, notFoundMessage string) (*commandClipboard, error) {
	for _, candidate := range candidates {
		command, err := lookPath(candidate.name)
		if err == nil {
			return &commandClipboard{
				command: command,
				args:    candidate.args,
			}, nil
		}
	}

	return nil, fmt.Errorf("%s", notFoundMessage)
}
