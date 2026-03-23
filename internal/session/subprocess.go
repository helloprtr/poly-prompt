// internal/session/subprocess.go
package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// ModelBinaries returns ordered binary candidates for a model name.
// Delegates to the provider registry in provider.go.
func ModelBinaries(model string) []string {
	if p, ok := GetProvider(model); ok {
		return p.Binaries
	}
	return []string{model}
}

// FindBinary searches $PATH for the first available candidate binary name.
func FindBinary(candidates ...string) (string, error) {
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("binary not found: tried %v", candidates)
}

// RunForeground runs binary with args in the foreground, inheriting stdin/stdout/stderr.
// Blocks until the process exits.
func RunForeground(ctx context.Context, binary string, args ...string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
