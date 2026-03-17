//go:build !windows

package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/helloprtr/poly-prompt/internal/watcher"
	"github.com/spf13/cobra"
)

func (a *App) newWatchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Start background event watcher (zsh/bash only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			off, _ := cmd.Flags().GetBool("off")
			status, _ := cmd.Flags().GetBool("status")

			pidPath, err := watcher.PIDPath()
			if err != nil {
				return err
			}

			if off {
				return stopWatcher(pidPath)
			}
			if status {
				return printWatcherStatus(pidPath)
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

func stopWatcher(pidPath string) error {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("prtr watch: not running")
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
	_ = os.Remove(pidPath)
	fmt.Println("prtr watch: stopped")
	return nil
}

func printWatcherStatus(pidPath string) error {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		fmt.Println("prtr watch: inactive")
		return nil
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	proc, err := os.FindProcess(pid)
	if err != nil || proc.Signal(syscall.Signal(0)) != nil {
		fmt.Println("prtr watch: inactive (stale PID)")
		return nil
	}
	fmt.Printf("prtr watch: active (PID %d)\n", pid)
	return nil
}
