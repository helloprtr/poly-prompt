//go:build windows

package main

import "fmt"

func runDashboard(target, watchStatus, branch string) error {
	fmt.Println("prtr dashboard is not supported on Windows yet.")
	return nil
}
