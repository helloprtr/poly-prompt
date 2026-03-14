package launcher

import (
	"fmt"
	"strings"
)

type windowsTerminalBackend struct {
	name    string
	command string
	args    func(req Request) []string
}

var windowsTerminalBackends = []windowsTerminalBackend{
	{
		name:    "wt.exe",
		command: "wt.exe",
		args: func(req Request) []string {
			args := []string{"new-tab", req.Command}
			return append(args, req.Args...)
		},
	},
	{
		name:    "pwsh.exe",
		command: "pwsh.exe",
		args: func(req Request) []string {
			return []string{"-NoExit", "-Command", powerShellCommandLine(req)}
		},
	},
	{
		name:    "powershell.exe",
		command: "powershell.exe",
		args: func(req Request) []string {
			return []string{"-NoExit", "-Command", powerShellCommandLine(req)}
		},
	},
	{
		name:    "cmd.exe",
		command: "cmd.exe",
		args: func(req Request) []string {
			return []string{"/k", cmdCommandLine(req)}
		},
	},
}

func (l *TerminalLauncher) selectWindowsBackend() (windowsTerminalBackend, error) {
	for _, backend := range windowsTerminalBackends {
		if _, err := l.lookPath(backend.command); err == nil {
			return backend, nil
		}
	}

	supported := make([]string, 0, len(windowsTerminalBackends))
	for _, backend := range windowsTerminalBackends {
		supported = append(supported, backend.name)
	}
	return windowsTerminalBackend{}, fmt.Errorf("no supported Windows terminal backend was found; install Windows Terminal or ensure one of %s is on PATH", strings.Join(supported, ", "))
}

func powerShellCommandLine(req Request) string {
	parts := []string{"&", powerShellQuote(req.Command)}
	for _, arg := range req.Args {
		parts = append(parts, powerShellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func powerShellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func cmdCommandLine(req Request) string {
	parts := []string{cmdQuote(req.Command)}
	for _, arg := range req.Args {
		parts = append(parts, cmdQuote(arg))
	}
	return strings.Join(parts, " ")
}

func cmdQuote(value string) string {
	if value == "" {
		return `""`
	}
	escaped := strings.ReplaceAll(value, `"`, `\"`)
	return `"` + escaped + `"`
}
