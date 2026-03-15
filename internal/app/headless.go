package app

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type HeadlessRequest struct {
	Target     string
	Prompt     string
	JSONOutput bool
}

type HeadlessResult struct {
	Command string
	Args    []string
	Stdout  string
	Stderr  string
}

type CLIHeadlessRunner struct{}

func (CLIHeadlessRunner) Run(ctx context.Context, req HeadlessRequest) (HeadlessResult, error) {
	command, args, err := headlessCommandFor(req)
	if err != nil {
		return HeadlessResult{}, err
	}

	cmd := exec.CommandContext(ctx, command, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return HeadlessResult{
			Command: command,
			Args:    args,
			Stdout:  stdout.String(),
			Stderr:  stderr.String(),
		}, fmt.Errorf("headless %s run failed: %w", req.Target, err)
	}

	return HeadlessResult{
		Command: command,
		Args:    args,
		Stdout:  stdout.String(),
		Stderr:  stderr.String(),
	}, nil
}

func headlessCommandFor(req HeadlessRequest) (string, []string, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return "", nil, fmt.Errorf("headless prompt is empty")
	}

	switch strings.ToLower(strings.TrimSpace(req.Target)) {
	case "claude":
		args := []string{"-p", prompt}
		if req.JSONOutput {
			args = append(args, "--output-format", "json")
		}
		return "claude", args, nil
	case "gemini":
		args := []string{"-p", prompt}
		if req.JSONOutput {
			args = append(args, "--json")
		}
		return "gemini", args, nil
	case "codex":
		return "codex", []string{"exec", prompt}, nil
	default:
		return "", nil, fmt.Errorf("unsupported headless target %q", req.Target)
	}
}

func (a *App) resolveHeadlessRunner() HeadlessRunner {
	if a.headlessRunner != nil {
		return a.headlessRunner
	}
	return CLIHeadlessRunner{}
}
