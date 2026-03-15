package app

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
)

type platformSurface struct {
	Label     string
	Supported bool
}

func currentGOOS() string {
	return runtime.GOOS
}

func detectPlatformSurface(goos string, lookupEnv LookupEnv) platformSurface {
	switch goos {
	case "darwin":
		return platformSurface{Label: "macOS + Terminal.app", Supported: true}
	case "linux":
		if value, ok := lookupEnv("WAYLAND_DISPLAY"); ok && strings.TrimSpace(value) != "" {
			return platformSurface{Label: "Linux + Wayland", Supported: true}
		}
		if value, ok := lookupEnv("DISPLAY"); ok && strings.TrimSpace(value) != "" {
			return platformSurface{Label: "Linux + X11", Supported: true}
		}
		return platformSurface{Label: "unsupported platform", Supported: false}
	case "windows":
		if value, ok := lookupEnv("SESSIONNAME"); ok && strings.TrimSpace(value) != "" && !strings.EqualFold(strings.TrimSpace(value), "services") {
			return platformSurface{Label: "Windows interactive session", Supported: true}
		}
		return platformSurface{Label: "unsupported platform", Supported: false}
	default:
		return platformSurface{Label: "unsupported platform", Supported: false}
	}
}

func (a *App) runPlatform(jsonOutput bool) error {
	cfg, err := a.configLoader()
	if err != nil {
		return err
	}
	checks := a.buildPlatformMatrix(cfg)
	if jsonOutput {
		payload := make([]map[string]string, 0, len(checks))
		for _, check := range checks {
			item := map[string]string{
				"severity": string(check.Severity),
				"label":    check.Label,
			}
			if check.Detail != "" {
				item["detail"] = check.Detail
			}
			if check.Err != nil {
				item["error"] = check.Err.Error()
			}
			payload = append(payload, item)
		}
		data, err := json.MarshalIndent(map[string]any{"checks": payload}, "", "  ")
		if err != nil {
			return fmt.Errorf("encode platform json: %w", err)
		}
		_, _ = fmt.Fprintln(a.stdout, string(data))
		return nil
	}

	a.writeDoctorSection("Platform matrix", checks)
	return nil
}

func parsePlatformCommand(args []string) (bool, error) {
	for _, arg := range args {
		switch strings.TrimSpace(arg) {
		case "--json":
			return true, nil
		case "":
			continue
		default:
			return false, usageError{message: fmt.Sprintf("unknown platform flag %q", arg), helpText: platformHelpText()}
		}
	}
	return false, nil
}

func platformHelpText() string {
	return strings.Join([]string{
		"Show the current platform surface and handoff readiness.",
		"",
		"`prtr platform` exposes the platform matrix directly so you can see",
		"clipboard, launcher, paste, and submit readiness without reading the",
		"full doctor report.",
		"",
		"Usage:",
		"  prtr platform [--json]",
		"",
		"Examples:",
		"  prtr platform",
		"  prtr platform --json",
	}, "\n")
}
