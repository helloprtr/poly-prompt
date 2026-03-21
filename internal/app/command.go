package app

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/helloprtr/poly-prompt/internal/session"
)

func (a *App) Command(ctx context.Context, stdin io.Reader, stdinPiped bool) *cobra.Command {
	root := &cobra.Command{
		Use:                "prtr [message...]",
		Short:              "The command layer for AI work.",
		Long:               rootHelpText(),
		SilenceErrors:      true,
		SilenceUsage:       true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			if len(args) == 0 {
				return a.runBare(ctx, stdin)
			}
			if strings.HasPrefix(args[0], "@") {
				model := strings.TrimPrefix(args[0], "@")
				return a.runHandoff(ctx, model)
			}
			return a.runMain(ctx, args, stdin, stdinPiped, "")
		},
	}
	root.SetOut(a.stdout)
	root.SetErr(a.stderr)
	root.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		_, _ = fmt.Fprintln(a.stdout, cmd.Long)
	})

	root.AddCommand(a.newStartCommand(ctx, stdin, stdinPiped))
	root.AddCommand(a.newGoCommand(ctx, stdin, stdinPiped))
	root.AddCommand(a.newDemoCommand(ctx))
	root.AddCommand(a.newAgainCommand(ctx, stdin, stdinPiped))
	root.AddCommand(a.newSwapCommand(ctx, stdin, stdinPiped))
	root.AddCommand(a.newTakeCommand(ctx))
	root.AddCommand(a.newLearnCommand())
	root.AddCommand(a.newInspectCommand(ctx, stdin, stdinPiped))
	root.AddCommand(a.newHistoryCommand())
	root.AddCommand(a.newSetupCommand(stdin))
	root.AddCommand(a.newDoctorCommand(ctx))
	root.AddCommand(a.newVersionCommand())
	root.AddCommand(a.newTemplatesCommand())
	root.AddCommand(a.newProfilesCommand())
	root.AddCommand(a.newRerunCommand(ctx))
	root.AddCommand(a.newPinCommand())
	root.AddCommand(a.newFavoriteCommand())
	root.AddCommand(a.newLangCommand(stdin))
	root.AddCommand(a.newInitCommand())
	root.AddCommand(a.newShortcutCommand(ctx, "ask", stdin, stdinPiped))
	root.AddCommand(a.newModeCommand(ctx, "review", session.ModeReview, stdin, stdinPiped))
	root.AddCommand(a.newModeCommand(ctx, "edit", session.ModeEdit, stdin, stdinPiped))
	root.AddCommand(a.newModeCommand(ctx, "fix", session.ModeFix, stdin, stdinPiped))
	root.AddCommand(a.newModeCommand(ctx, "design", session.ModeDesign, stdin, stdinPiped))
	root.AddCommand(&cobra.Command{
		Use:   "checkpoint [note]",
		Short: "진행 상황 메모 (핸드오프 품질 향상)",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return a.runCheckpoint(ctx, args[0]) },
	})
	root.AddCommand(&cobra.Command{
		Use:   "done",
		Short: "세션 완료 처리",
		RunE:  func(cmd *cobra.Command, args []string) error { return a.runDone(ctx) },
	})
	root.AddCommand(&cobra.Command{
		Use:   "sessions",
		Short: "과거 세션 목록",
		RunE:  func(cmd *cobra.Command, args []string) error { return a.runSessions(ctx) },
	})

	return root
}

func (a *App) newStartCommand(ctx context.Context, stdin io.Reader, stdinPiped bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "start [message...]",
		Short:              "Run the beginner-first first-send flow.",
		Long:               startHelpText(),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runStart(ctx, args, stdin, stdinPiped)
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) { _, _ = fmt.Fprintln(a.stdout, cmd.Long) })
	return cmd
}

func (a *App) newGoCommand(ctx context.Context, stdin io.Reader, stdinPiped bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "go [mode] [message...]",
		Short:              "Turn intent into the next AI action in Claude, Codex, or Gemini.",
		Long:               goHelpText(),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runGo(ctx, args, stdin, stdinPiped)
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) { _, _ = fmt.Fprintln(a.stdout, cmd.Long) })
	return cmd
}

func (a *App) newDemoCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "demo",
		Short: "Preview prtr's core loop without a DeepL key.",
		Long:  demoHelpText(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runDemo(ctx)
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) { _, _ = fmt.Fprintln(a.stdout, cmd.Long) })
	return cmd
}

func (a *App) newAgainCommand(ctx context.Context, stdin io.Reader, stdinPiped bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "again [message...]",
		Short:              "Run the latest prompt flow again.",
		Long:               againHelpText(),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runAgain(ctx, args, stdin, stdinPiped)
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) { _, _ = fmt.Fprintln(a.stdout, cmd.Long) })
	return cmd
}

func (a *App) newSwapCommand(ctx context.Context, stdin io.Reader, stdinPiped bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "swap <app> [message...]",
		Short:              "Send the latest prompt to another app.",
		Long:               swapHelpText(),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runSwap(ctx, args, stdin, stdinPiped)
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) { _, _ = fmt.Fprintln(a.stdout, cmd.Long) })
	return cmd
}

func (a *App) newTakeCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "take <action>",
		Short:              "Turn the latest answer or clipboard text into the next action.",
		Long:               takeHelpText(),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runTake(ctx, args)
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) { _, _ = fmt.Fprintln(a.stdout, cmd.Long) })
	return cmd
}

func (a *App) newLearnCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "learn [paths...]",
		Short:              "Teach prtr your repo terms and style.",
		Long:               learnHelpText(),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runLearn(args)
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) { _, _ = fmt.Fprintln(a.stdout, cmd.Long) })
	return cmd
}

func (a *App) newInspectCommand(ctx context.Context, stdin io.Reader, stdinPiped bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "inspect [flags] [message...]",
		Short:              "Inspect the compiled prompt and resolved config without sending it.",
		Long:               inspectHelpText(),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runInspect(ctx, args, stdin, stdinPiped)
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) { _, _ = fmt.Fprintln(a.stdout, cmd.Long) })
	return cmd
}

func (a *App) newHistoryCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "history [search <query>]",
		Short:              "Show recent runs or search history.",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runHistory(args)
		},
	}
}

func (a *App) newSetupCommand(stdin io.Reader) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Run advanced guided setup for prtr defaults.",
		Long:  setupHelpText(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runSetup(stdin)
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) { _, _ = fmt.Fprintln(a.stdout, cmd.Long) })
	return cmd
}

func (a *App) newDoctorCommand(ctx context.Context) *cobra.Command {
	var fix bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run environment and configuration diagnostics.",
		Long:  doctorHelpText(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runDoctor(ctx, fix)
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "Apply safe automatic fixes when possible.")
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) { _, _ = fmt.Fprintln(a.stdout, cmd.Long) })
	return cmd
}

func (a *App) newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the prtr version.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runVersion()
		},
	}
}

func (a *App) newTemplatesCommand() *cobra.Command {
	parent := &cobra.Command{
		Use:   "templates",
		Short: "Inspect template presets.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runTemplates(args)
		},
	}
	parent.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List template presets.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runTemplates([]string{"list"})
		},
	})
	parent.AddCommand(&cobra.Command{
		Use:   "show <name>",
		Short: "Show a template preset.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runTemplates(append([]string{"show"}, args...))
		},
	})
	return parent
}

func (a *App) newProfilesCommand() *cobra.Command {
	parent := &cobra.Command{
		Use:   "profiles",
		Short: "Inspect or apply saved profiles.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runProfiles(args)
		},
	}
	parent.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List profiles.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runProfiles([]string{"list"})
		},
	})
	parent.AddCommand(&cobra.Command{
		Use:   "show <name>",
		Short: "Show a profile.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runProfiles(append([]string{"show"}, args...))
		},
	})
	parent.AddCommand(&cobra.Command{
		Use:   "use <name>",
		Short: "Apply a profile as defaults.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runProfiles(append([]string{"use"}, args...))
		},
	})
	return parent
}

func (a *App) newRerunCommand(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:                "rerun <id> [flags]",
		Short:              "Rerun a stored history entry.",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runRerun(ctx, args)
		},
	}
}

func (a *App) newPinCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pin <id>",
		Short: "Pin or unpin a history entry.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runPin(args)
		},
	}
}

func (a *App) newFavoriteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "favorite <id>",
		Short: "Favorite or unfavorite a history entry.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runFavorite(args)
		},
	}
}

func (a *App) newLangCommand(stdin io.Reader) *cobra.Command {
	return &cobra.Command{
		Use:   "lang",
		Short: "Update default language settings.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runLang(stdin)
		},
	}
}

func (a *App) newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a starter config.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runInit()
		},
	}
}

func (a *App) newShortcutCommand(ctx context.Context, name string, stdin io.Reader, stdinPiped bool) *cobra.Command {
	return &cobra.Command{
		Use:                name + " [message...]",
		Short:              "Compatibility alias for `prtr go " + name + "`.",
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return a.newGoCommand(ctx, stdin, stdinPiped).Help()
			}
			return a.runShortcut(ctx, name, args, stdin, stdinPiped)
		},
	}
}

func (a *App) newModeCommand(ctx context.Context, name string, mode session.Mode, stdin io.Reader, stdinPiped bool) *cobra.Command {
	return &cobra.Command{
		Use:           name + " [files...]",
		Short:         modeShort(mode),
		Args:          cobra.ArbitraryArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runSessionMode(ctx, mode, args, stdin)
		},
	}
}

func modeShort(m session.Mode) string {
	switch m {
	case session.ModeReview:
		return "코드 리뷰 세션 시작"
	case session.ModeEdit:
		return "코드 수정 세션 시작"
	case session.ModeFix:
		return "버그 수정 세션 시작"
	case session.ModeDesign:
		return "설계 세션 시작"
	default:
		return "세션 시작"
	}
}
