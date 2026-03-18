//go:build !windows

package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/helloprtr/poly-prompt/internal/watcher"
	"github.com/spf13/cobra"
)

func (a *App) newWatchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Start foreground event watcher (zsh/bash only — run in a separate terminal or with &)",
		RunE: func(cmd *cobra.Command, args []string) error {
			off, _ := cmd.Flags().GetBool("off")
			status, _ := cmd.Flags().GetBool("status")

			pidPath, err := watcher.PIDPath()
			if err != nil {
				return err
			}

			if off {
				return stopWatcher(pidPath, cmd.OutOrStdout())
			}
			if status {
				return printWatcherStatus(pidPath, cmd.OutOrStdout())
			}

			shellConfig := watcher.DetectShellConfig()
			if err := watcher.InstallHook(shellConfig); err != nil {
				fmt.Fprintf(a.stderr, "warning: could not install shell hook: %v\n", err)
			} else {
				fmt.Fprintf(a.stdout, "⚡ prtr watch: hook installed in %s\n", shellConfig)
				fmt.Fprintln(a.stdout, "  Restart your shell or run: source "+shellConfig)
			}

			fmt.Fprintln(a.stdout, "⚡ prtr watch: starting...")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				c := make(chan os.Signal, 1)
				signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
				<-c
				cancel()
			}()

			return watcher.Run(ctx)
		},
	}
	cmd.Flags().Bool("off", false, "stop the watcher")
	cmd.Flags().Bool("status", false, "show watcher status")
	return cmd
}

func stopWatcher(pidPath string, w io.Writer) error {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(w, "prtr watch: not running")
			return nil
		}
		return err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return fmt.Errorf("invalid PID file: %w", err)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		_ = os.Remove(pidPath)
		return nil
	}
	_ = proc.Signal(syscall.SIGTERM)

	// Wait up to 2 s for the process to exit before removing the PID file.
	// Without this wait, a rapid `prtr watch` restart can race against the
	// still-bound Unix socket and fail with "address already in use".
	const maxWait = 20
	for i := 0; i < maxWait; i++ {
		if proc.Signal(syscall.Signal(0)) != nil {
			break // process has exited
		}
		time.Sleep(100 * time.Millisecond)
	}

	_ = os.Remove(pidPath)
	fmt.Fprintln(w, "prtr watch: stopped")
	return nil
}

func printWatcherStatus(pidPath string, w io.Writer) error {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		fmt.Fprintln(w, "prtr watch: inactive")
		return nil
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	proc, err := os.FindProcess(pid)
	if err != nil || proc.Signal(syscall.Signal(0)) != nil {
		fmt.Fprintln(w, "prtr watch: inactive (stale PID)")
		return nil
	}
	fmt.Fprintf(w, "prtr watch: active (PID %d)\n", pid)
	return nil
}
