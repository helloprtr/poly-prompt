package automation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type runCommandFunc func(context.Context, string, ...string) (string, error)

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func lookupEnv(key string) string {
	return os.Getenv(key)
}
