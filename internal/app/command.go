package app

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func (a *App) Command(ctx context.Context, stdin io.Reader, stdinPiped bool) *cobra.Command {
	root := &cobra.Command{
		Use:                "prtr [message...]",
		Short:              "Beginner-first AI command layer for the next action.",
		Long:               rootHelpText(),
		SilenceErrors:      true,
		SilenceUsage:       true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
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
	root.AddCommand(a.newAgainCommand(ctx, stdin, stdinPiped))
	root.AddCommand(a.newSwapCommand(ctx, stdin, stdinPiped))
	root.AddCommand(a.newTakeCommand(ctx))
	root.AddCommand(a.newLearnCommand())
	root.AddCommand(a.newSyncCommand())
	root.AddCommand(a.newPlatformCommand())
	root.AddCommand(a.newExecCommand(ctx, stdin, stdinPiped))
	root.AddCommand(a.newServerCommand(ctx))
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
	root.AddCommand(a.newShortcutCommand(ctx, "review", stdin, stdinPiped))
	root.AddCommand(a.newShortcutCommand(ctx, "fix", stdin, stdinPiped))
	root.AddCommand(a.newShortcutCommand(ctx, "design", stdin, stdinPiped))

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
		Short:              "Send a translated, context-aware prompt to your AI app.",
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

func (a *App) newSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "sync [init|status]",
		Short:              "Sync canonical .prtr guidance into vendor files.",
		Long:               syncHelpText(),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runSync(args)
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) { _, _ = fmt.Fprintln(a.stdout, cmd.Long) })
	return cmd
}

func (a *App) newPlatformCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "platform [--json]",
		Short:              "Show the current platform surface and readiness.",
		Long:               platformHelpText(),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			jsonOutput, err := parsePlatformCommand(args)
			if err != nil {
				return err
			}
			return a.runPlatform(jsonOutput)
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) { _, _ = fmt.Fprintln(a.stdout, cmd.Long) })
	return cmd
}

func (a *App) newExecCommand(ctx context.Context, stdin io.Reader, stdinPiped bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "exec [mode] [message...]",
		Short:              "Run a headless request through a target app.",
		Long:               execHelpText(),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runExec(ctx, args, stdin, stdinPiped)
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) { _, _ = fmt.Fprintln(a.stdout, cmd.Long) })
	return cmd
}

func (a *App) newServerCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "server [--addr 127.0.0.1:8787]",
		Short:              "Start the alpha orchestration server.",
		Long:               serverHelpText(),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runServer(ctx, args)
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
