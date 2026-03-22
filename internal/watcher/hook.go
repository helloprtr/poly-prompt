// internal/watcher/hook.go
package watcher

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed hook.zsh
var zshHook string

//go:embed hook.bash
var bashHook string

const hookMarkerStart = "# prtr watch hook — managed by prtr, do not edit manually"
const hookMarkerEnd = "# end prtr watch hook"

// InstallHook appends the shell hook to the user's shell config if not already present.
func InstallHook(shellConfig string) error {
	data, err := os.ReadFile(shellConfig)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read shell config: %w", err)
	}

	if strings.Contains(string(data), hookMarkerStart) {
		return nil // already installed
	}

	hook := hookForShell(shellConfig)
	f, err := os.OpenFile(shellConfig, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open shell config: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "\n%s\n%s\n%s\n", hookMarkerStart, hook, hookMarkerEnd)
	return err
}

// DetectShellConfig returns the user's primary shell config path.
// For zsh it honours $ZDOTDIR when set, matching zsh's own lookup order.
func DetectShellConfig() string {
	home, _ := os.UserHomeDir()
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		base := os.Getenv("ZDOTDIR")
		if base == "" {
			base = home
		}
		return filepath.Join(base, ".zshrc")
	}
	// Prefer ~/.bashrc; fall back to ~/.bash_profile for login-only shells.
	bashrc := filepath.Join(home, ".bashrc")
	if _, err := os.Stat(bashrc); err == nil {
		return bashrc
	}
	return filepath.Join(home, ".bash_profile")
}

func hookForShell(configPath string) string {
	if strings.Contains(configPath, "zsh") {
		return zshHook
	}
	return bashHook
}
