package clipboard

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestDetectClipboardDarwinUsesPBcopy(t *testing.T) {
	t.Parallel()

	clipboard := NewForTesting("darwin", func(name string) (string, error) {
		if name == "pbcopy" {
			return "/usr/bin/pbcopy", nil
		}
		return "", exec.ErrNotFound
	})

	auto := clipboard
	backend, err := detectClipboardWriter(auto.goos, auto.lookPath)
	if err != nil {
		t.Fatalf("detectClipboardWriter() error = %v", err)
	}

	command, ok := backend.(*commandClipboard)
	if !ok {
		t.Fatalf("backend type = %T, want *commandClipboard", backend)
	}
	if command.command != "/usr/bin/pbcopy" {
		t.Fatalf("command = %q, want %q", command.command, "/usr/bin/pbcopy")
	}
}

func TestDetectClipboardLinuxFallbackOrder(t *testing.T) {
	t.Parallel()

	clipboard := NewForTesting("linux", func(name string) (string, error) {
		switch name {
		case "wl-copy":
			return "", exec.ErrNotFound
		case "xclip":
			return "/usr/bin/xclip", nil
		default:
			return "", exec.ErrNotFound
		}
	})

	backend, err := detectClipboardWriter(clipboard.goos, clipboard.lookPath)
	if err != nil {
		t.Fatalf("detectClipboardWriter() error = %v", err)
	}

	command, ok := backend.(*commandClipboard)
	if !ok {
		t.Fatalf("backend type = %T, want *commandClipboard", backend)
	}
	if command.command != "/usr/bin/xclip" {
		t.Fatalf("command = %q, want %q", command.command, "/usr/bin/xclip")
	}
	if got := strings.Join(command.args, " "); got != "-selection clipboard" {
		t.Fatalf("args = %q, want %q", got, "-selection clipboard")
	}
}

func TestDetectClipboardReturnsInstallGuidance(t *testing.T) {
	t.Parallel()

	clipboard := NewForTesting("linux", func(string) (string, error) {
		return "", exec.ErrNotFound
	})

	_, err := detectClipboardWriter(clipboard.goos, clipboard.lookPath)
	if err == nil {
		t.Fatal("detectClipboardWriter() expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "install wl-clipboard, xclip, or xsel") {
		t.Fatalf("detectClipboardWriter() error = %v", err)
	}
}

func TestDetectClipboardReaderLinuxFallbackOrder(t *testing.T) {
	t.Parallel()

	clipboard := NewForTesting("linux", func(name string) (string, error) {
		switch name {
		case "wl-paste":
			return "", exec.ErrNotFound
		case "xclip":
			return "/usr/bin/xclip", nil
		default:
			return "", exec.ErrNotFound
		}
	})

	backend, err := detectClipboardReader(clipboard.goos, clipboard.lookPath)
	if err != nil {
		t.Fatalf("detectClipboardReader() error = %v", err)
	}

	command, ok := backend.(*commandClipboard)
	if !ok {
		t.Fatalf("backend type = %T, want *commandClipboard", backend)
	}
	if command.command != "/usr/bin/xclip" {
		t.Fatalf("command = %q, want %q", command.command, "/usr/bin/xclip")
	}
	if got := strings.Join(command.args, " "); got != "-selection clipboard -o" {
		t.Fatalf("args = %q, want %q", got, "-selection clipboard -o")
	}
}

func TestCopyAndReadReturnDetectionError(t *testing.T) {
	t.Parallel()

	clipboard := NewForTesting("plan9", func(string) (string, error) {
		return "", errors.New("unused")
	})

	err := clipboard.Copy(context.Background(), "hello")
	if err == nil {
		t.Fatal("Copy() expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "clipboard is not supported on plan9") {
		t.Fatalf("Copy() error = %v", err)
	}

	_, err = clipboard.Read(context.Background())
	if err == nil {
		t.Fatal("Read() expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "clipboard is not supported on plan9") {
		t.Fatalf("Read() error = %v", err)
	}
}
