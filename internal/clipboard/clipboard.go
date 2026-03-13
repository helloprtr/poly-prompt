package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

type Writer interface {
	Copy(ctx context.Context, text string) error
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

func New() Writer {
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
	backend, err := detectClipboard(c.goos, c.lookPath)
	if err != nil {
		return err
	}

	return backend.Copy(ctx, text)
}

func (c *AutoClipboard) Diagnose() error {
	_, err := detectClipboard(c.goos, c.lookPath)
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

func detectClipboard(goos string, lookPath LookPathFunc) (Writer, error) {
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

type backendCandidate struct {
	name string
	args []string
}

func detectFirst(lookPath LookPathFunc, candidates []backendCandidate, notFoundMessage string) (Writer, error) {
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
