package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func (a *App) runExec(ctx context.Context, args []string, stdin io.Reader, stdinPiped bool) error {
	command, err := parseExecCommand(args, a.builtInShortcutNames())
	if err != nil {
		return err
	}

	resolved, err := a.prepareExecRun(ctx, command, stdin, stdinPiped)
	if err != nil {
		return err
	}
	if command.dryRun {
		_, _ = fmt.Fprintln(a.stdout, resolved.finalPrompt)
		return nil
	}

	result, err := a.resolveHeadlessRunner().Run(ctx, HeadlessRequest{
		Target:     resolved.targetName,
		Prompt:     resolved.finalPrompt,
		JSONOutput: command.json,
	})
	if err != nil {
		if strings.TrimSpace(result.Stdout) != "" {
			_, _ = fmt.Fprintln(a.stdout, strings.TrimSpace(result.Stdout))
		}
		return err
	}

	resolved.deliveryMode = "exec"
	if err := a.appendHistory(resolved); err != nil {
		return err
	}

	if command.json {
		payload := map[string]any{
			"target":       resolved.targetName,
			"targetSource": resolved.targetSource,
			"targetReason": resolved.targetReason,
			"command":      result.Command,
			"args":         result.Args,
			"output":       strings.TrimSpace(result.Stdout),
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("encode exec json: %w", err)
		}
		_, _ = fmt.Fprintln(a.stdout, string(data))
		return nil
	}

	if strings.TrimSpace(result.Stdout) != "" {
		_, _ = fmt.Fprintln(a.stdout, strings.TrimSpace(result.Stdout))
	}
	_, _ = fmt.Fprintf(a.stderr, "-> exec | %s | %s | headless\n", resolved.targetName, blankDefault(command.mode, "ask"))
	return nil
}

func (a *App) prepareExecRun(ctx context.Context, command execCommandOptions, stdin io.Reader, stdinPiped bool) (resolvedRun, error) {
	text, inputSource, err := resolveSurfaceInput(command.prompt, stdin, stdinPiped, !command.noContext)
	if err != nil {
		return resolvedRun{}, err
	}

	repoSummary, repoSuffix, inputSource := a.collectRepoContext(ctx, inputSource, command.noContext)
	learned := a.resolveLearnedContext(command.mode, command.noContext)
	cfg, err := a.configLoader()
	if err != nil {
		return resolvedRun{}, err
	}
	decision := selectGoTarget(cfg, command.mode, command.app, text, repoSummary)

	opts := runOptions{
		target:               decision.Target,
		targetSource:         decision.Source,
		targetReason:         decision.Reason,
		noCopy:               true,
		launch:               false,
		paste:                false,
		surfaceMode:          blankDefault(command.mode, "ask"),
		surfaceInput:         inputSource,
		surfaceDelivery:      "exec",
		promptSuffix:         joinPromptSections(repoSuffix, learned.suffix),
		protectedTerms:       learned.protectedTerms,
		preferTargetTemplate: true,
		action:               "exec",
		sourceKind:           inputSource,
	}
	return a.prepareRun(ctx, opts, text, command.mode)
}

func parseExecCommand(args []string, builtInShortcuts map[string]bool) (execCommandOptions, error) {
	command := execCommandOptions{mode: "ask"}
	if len(args) > 0 && builtInShortcuts[args[0]] {
		command.mode = args[0]
		args = args[1:]
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--dry-run":
			command.dryRun = true
		case arg == "--json":
			command.json = true
		case arg == "--no-context":
			command.noContext = true
		case arg == "--to" || arg == "--app" || arg == "--target" || arg == "-t":
			i++
			if i >= len(args) {
				return execCommandOptions{}, usageError{message: fmt.Sprintf("%s requires a value", arg), helpText: execHelpText()}
			}
			command.app = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--to="):
			command.app = strings.TrimSpace(strings.TrimPrefix(arg, "--to="))
		case strings.HasPrefix(arg, "--app="):
			command.app = strings.TrimSpace(strings.TrimPrefix(arg, "--app="))
		case strings.HasPrefix(arg, "--target="):
			command.app = strings.TrimSpace(strings.TrimPrefix(arg, "--target="))
		case arg == "--":
			command.prompt = append(command.prompt, args[i+1:]...)
			return command, nil
		case strings.HasPrefix(arg, "-"):
			return execCommandOptions{}, usageError{message: fmt.Sprintf("unknown exec flag %q", arg), helpText: execHelpText()}
		default:
			command.prompt = append(command.prompt, arg)
		}
	}

	return command, nil
}

func execHelpText() string {
	return strings.Join([]string{
		"Run a headless request through Claude, Codex, or Gemini.",
		"",
		"`prtr exec` compiles the same mode-aware prompt as `prtr go`, but uses a",
		"non-interactive CLI invocation instead of open-copy delivery.",
		"",
		"Usage:",
		"  prtr exec [mode] [message...]",
		"",
		"Flags:",
		"      --to <app>        Choose the app: claude | codex | gemini",
		"      --json            Request structured JSON output when supported",
		"      --dry-run         Print the compiled prompt without running the app",
		"      --no-context      Do not attach repo or piped context automatically",
	}, "\n")
}
