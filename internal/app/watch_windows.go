//go:build windows

package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (a *App) newWatchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "watch",
		Short: "Start background event watcher (not supported on Windows)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "prtr watch is not supported on Windows yet.")
			return nil
		},
	}
}
