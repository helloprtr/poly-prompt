package automation

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrInteractiveDesktopRequired = errors.New("automation requires an interactive Windows desktop session")

type WindowsAutomator struct {
	goos      string
	lookPath  func(string) (string, error)
	run       runCommandFunc
	lookupEnv func(string) string
}

func NewWindowsForTesting(goos string, lookPath func(string) (string, error), run runCommandFunc, lookupEnv func(string) string) *WindowsAutomator {
	return &WindowsAutomator{
		goos:      goos,
		lookPath:  lookPath,
		run:       run,
		lookupEnv: lookupEnv,
	}
}

func (a *WindowsAutomator) Diagnose(req Request) error {
	_, _, err := a.describeBackend(req)
	return err
}

func (a *WindowsAutomator) Describe(req Request) (string, error) {
	name, shell, err := a.describeBackend(req)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s via %s", name, shell), nil
}

func (a *WindowsAutomator) Paste(ctx context.Context, req Request) error {
	backendName, shellName, err := a.describeBackend(req)
	if err != nil {
		return err
	}
	if err := waitForDelay(ctx, req.PasteDelay); err != nil {
		return err
	}

	psShell, psArgs := a.powerShellInvocation(buildWindowsPasteScript(req.Target))
	if _, err := a.run(ctx, psShell, psArgs...); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "terminal-not-frontmost") {
			return ErrTerminalNotFrontmost
		}
		return fmt.Errorf("paste into terminal via %s (%s): %w", backendName, shellName, err)
	}
	return nil
}

func (a *WindowsAutomator) Submit(context.Context, Request) error {
	return ErrUnsupportedSubmitMode
}

func (a *WindowsAutomator) describeBackend(req Request) (string, string, error) {
	if a.goos != "windows" {
		return "", "", ErrUnsupportedPlatform
	}

	sessionName := strings.TrimSpace(a.lookupEnv("SESSIONNAME"))
	if strings.EqualFold(sessionName, "services") {
		return "", "", ErrInteractiveDesktopRequired
	}

	if _, err := a.lookPath("powershell.exe"); err == nil {
		return "sendkeys", "powershell.exe", nil
	}
	if _, err := a.lookPath("pwsh.exe"); err == nil {
		return "sendkeys", "pwsh.exe", nil
	}

	_ = req
	return "", "", errors.New("automation requires powershell.exe or pwsh.exe")
}

func (a *WindowsAutomator) powerShellInvocation(script string) (string, []string) {
	if _, err := a.lookPath("powershell.exe"); err == nil {
		return "powershell.exe", []string{"-NoProfile", "-NonInteractive", "-Command", script}
	}
	return "pwsh.exe", []string{"-NoProfile", "-NonInteractive", "-Command", script}
}

func buildWindowsPasteScript(target string) string {
	pattern := "terminal|powershell|pwsh|cmd|claude|codex|gemini"
	if strings.TrimSpace(target) != "" {
		pattern += "|" + regexpEscape(target)
	}

	return strings.Join([]string{
		"Add-Type -AssemblyName System.Windows.Forms",
		`Add-Type @"`,
		"using System;",
		"using System.Runtime.InteropServices;",
		"using System.Text;",
		"public static class PrtrWin32 {",
		`  [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();`,
		`  [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern int GetWindowText(IntPtr hWnd, StringBuilder text, int count);`,
		"}",
		`"@`,
		"$hwnd = [PrtrWin32]::GetForegroundWindow()",
		"$sb = New-Object System.Text.StringBuilder 512",
		"[void][PrtrWin32]::GetWindowText($hwnd, $sb, $sb.Capacity)",
		"$title = $sb.ToString()",
		`if ([string]::IsNullOrWhiteSpace($title)) { throw "terminal-not-frontmost" }`,
		fmt.Sprintf(`if ($title.ToLowerInvariant() -notmatch '%s') { throw "terminal-not-frontmost" }`, pattern),
		`[System.Windows.Forms.SendKeys]::SendWait("^v")`,
	}, "\n")
}

func regexpEscape(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`.`, `\.`,
		`+`, `\+`,
		`*`, `\*`,
		`?`, `\?`,
		`(`, `\(`,
		`)`, `\)`,
		`[`, `\[`,
		`]`, `\]`,
		`{`, `\{`,
		`}`, `\}`,
		`^`, `\^`,
		`$`, `\$`,
		`|`, `\|`,
	)
	return replacer.Replace(strings.ToLower(value))
}
