package app

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/helloprtr/poly-prompt/internal/memory"
	"github.com/helloprtr/poly-prompt/internal/termbook"
)

type syncCommandOptions struct {
	subcommand string
	dryRun     bool
	write      []string
}

type syncTarget struct {
	key     string
	path    string
	content []byte
	state   string
}

func (a *App) runSync(args []string) error {
	command, err := parseSyncCommand(args)
	if err != nil {
		return err
	}

	repoRoot, err := a.resolveRepoRoot()
	if err != nil {
		return err
	}

	switch command.subcommand {
	case "init":
		return a.runSyncInit(repoRoot, command.dryRun)
	case "status":
		return a.runSyncStatus(repoRoot, command.write)
	default:
		return a.runSyncWrite(repoRoot, command.write, command.dryRun)
	}
}

func parseSyncCommand(args []string) (syncCommandOptions, error) {
	command := syncCommandOptions{subcommand: "write"}

	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch {
		case arg == "init" || arg == "status":
			if command.subcommand != "write" {
				return syncCommandOptions{}, usageError{message: "sync accepts only one subcommand", helpText: syncHelpText()}
			}
			command.subcommand = arg
		case arg == "--dry-run":
			command.dryRun = true
		case arg == "--write":
			i++
			if i >= len(args) {
				return syncCommandOptions{}, usageError{message: "--write requires a value", helpText: syncHelpText()}
			}
			command.write = parseSyncWriteList(args[i])
		case strings.HasPrefix(arg, "--write="):
			command.write = parseSyncWriteList(strings.TrimPrefix(arg, "--write="))
		case strings.HasPrefix(arg, "-"):
			return syncCommandOptions{}, usageError{message: fmt.Sprintf("unknown sync flag %q", arg), helpText: syncHelpText()}
		default:
			return syncCommandOptions{}, usageError{message: fmt.Sprintf("unexpected sync argument %q", arg), helpText: syncHelpText()}
		}
	}

	return command, nil
}

func parseSyncWriteList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func (a *App) runSyncInit(repoRoot string, dryRun bool) error {
	prtrDir := filepath.Join(repoRoot, ".prtr")
	files := []struct {
		path    string
		content []byte
	}{
		{
			path: filepath.Join(prtrDir, "guide.md"),
			content: []byte(strings.Join([]string{
				"# Project Guide",
				"",
				"Describe repository-specific terminology, workflows, and guardrails here.",
				"Keep this file human-edited; `prtr sync` will render vendor files from it.",
				"",
				"## Team Preferences",
				"",
				"- Preserve project identifiers exactly.",
				"- Prefer concrete next actions over generic summaries.",
			}, "\n")),
		},
	}

	termbookBytes, err := termbook.Encode(termbook.Book{GeneratedAt: time.Now().UTC()})
	if err != nil {
		return err
	}
	files = append(files, struct {
		path    string
		content []byte
	}{path: filepath.Join(prtrDir, "termbook.toml"), content: termbookBytes})

	memoryBytes, err := memory.Encode(memory.Book{GeneratedAt: time.Now().UTC()})
	if err != nil {
		return err
	}
	files = append(files, struct {
		path    string
		content []byte
	}{path: filepath.Join(prtrDir, "memory.toml"), content: memoryBytes})

	for _, file := range files {
		if _, statErr := os.Stat(file.path); statErr == nil {
			_, _ = fmt.Fprintf(a.stdout, "SKIP %s already exists\n", file.path)
			continue
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
		if dryRun {
			_, _ = fmt.Fprintf(a.stdout, "CREATE %s\n", file.path)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(file.path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(file.path, file.content, 0o644); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(a.stdout, "CREATED %s\n", file.path)
	}

	return nil
}

func (a *App) runSyncStatus(repoRoot string, write []string) error {
	targets, err := a.renderSyncTargets(repoRoot, write)
	if err != nil {
		return err
	}
	for _, target := range targets {
		_, _ = fmt.Fprintf(a.stdout, "%s %s\n", strings.ToUpper(target.state), target.path)
	}
	return nil
}

func (a *App) runSyncWrite(repoRoot string, write []string, dryRun bool) error {
	targets, err := a.renderSyncTargets(repoRoot, write)
	if err != nil {
		return err
	}
	for _, target := range targets {
		switch target.state {
		case "synced":
			_, _ = fmt.Fprintf(a.stdout, "OK   %s already in sync\n", target.path)
			continue
		case "missing", "drifted":
			if dryRun {
				_, _ = fmt.Fprintf(a.stdout, "WRITE %s (%s)\n", target.path, target.state)
				continue
			}
			if err := os.WriteFile(target.path, target.content, 0o644); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(a.stdout, "WROTE %s\n", target.path)
		}
	}
	return nil
}

func (a *App) renderSyncTargets(repoRoot string, selected []string) ([]syncTarget, error) {
	selection := map[string]bool{}
	for _, item := range selected {
		selection[item] = true
	}
	include := func(name string) bool {
		return len(selection) == 0 || selection[name]
	}

	guidePath := filepath.Join(repoRoot, ".prtr", "guide.md")
	guideData, err := os.ReadFile(guidePath)
	if err != nil {
		return nil, fmt.Errorf("read guide: %w", err)
	}
	termbookBook, err := termbook.Load(repoRoot)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	memoryBook, err := memory.Load(repoRoot)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	type spec struct {
		key      string
		filename string
		title    string
	}
	specs := []spec{
		{key: "claude", filename: filepath.Join(repoRoot, "CLAUDE.md"), title: "Claude"},
		{key: "gemini", filename: filepath.Join(repoRoot, "GEMINI.md"), title: "Gemini"},
		{key: "codex", filename: filepath.Join(repoRoot, "AGENTS.md"), title: "Codex"},
	}

	targets := make([]syncTarget, 0, len(specs))
	for _, spec := range specs {
		if !include(spec.key) {
			continue
		}
		content := a.renderVendorGuide(spec.title, guideData, termbookBook, memoryBook)
		state := "missing"
		if existing, err := os.ReadFile(spec.filename); err == nil {
			if bytes.Equal(existing, content) {
				state = "synced"
			} else {
				state = "drifted"
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		targets = append(targets, syncTarget{
			key:     spec.key,
			path:    spec.filename,
			content: content,
			state:   state,
		})
	}
	return targets, nil
}

func (a *App) renderVendorGuide(title string, guide []byte, termbookBook termbook.Book, memoryBook memory.Book) []byte {
	lines := []string{
		"<!-- Generated by prtr sync. Edit .prtr/guide.md, .prtr/termbook.toml, and .prtr/memory.toml instead. -->",
		"",
		"# " + title + " Project Guidance",
		"",
		strings.TrimSpace(string(guide)),
	}

	if len(termbookBook.ProtectedTerms) > 0 {
		lines = append(lines, "", "## Protected Terms", "")
		for _, term := range termbookBook.ProtectedTerms {
			lines = append(lines, "- "+term)
		}
	}

	if strings.TrimSpace(memoryBook.RepoSummary) != "" {
		lines = append(lines, "", "## Repo Memory", "", memoryBook.RepoSummary)
	}
	appendBullets := func(title string, values []string) {
		if len(values) == 0 {
			return
		}
		lines = append(lines, "", "## "+title, "")
		for _, value := range values {
			lines = append(lines, "- "+value)
		}
	}
	appendBullets("Guidance", memoryBook.Guidance)
	appendBullets("Coding Norms", memoryBook.CodingNorms)
	appendBullets("Testing Norms", memoryBook.TestingNorms)

	content := strings.TrimSpace(strings.Join(lines, "\n"))
	return []byte(content + "\n")
}
