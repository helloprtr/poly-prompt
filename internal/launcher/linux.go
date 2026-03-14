package launcher

import (
	"fmt"
	"strings"
)

type linuxTerminalBackend struct {
	name    string
	command string
	args    func(commandLine string) []string
}

var linuxTerminalBackends = []linuxTerminalBackend{
	{
		name:    "x-terminal-emulator",
		command: "x-terminal-emulator",
		args: func(commandLine string) []string {
			return []string{"-e", "/bin/sh", "-lc", commandLine}
		},
	},
	{
		name:    "gnome-terminal",
		command: "gnome-terminal",
		args: func(commandLine string) []string {
			return []string{"--", "/bin/sh", "-lc", commandLine}
		},
	},
	{
		name:    "konsole",
		command: "konsole",
		args: func(commandLine string) []string {
			return []string{"-e", "/bin/sh", "-lc", commandLine}
		},
	},
	{
		name:    "kitty",
		command: "kitty",
		args: func(commandLine string) []string {
			return []string{"/bin/sh", "-lc", commandLine}
		},
	},
	{
		name:    "wezterm",
		command: "wezterm",
		args: func(commandLine string) []string {
			return []string{"start", "--", "/bin/sh", "-lc", commandLine}
		},
	},
}

func (l *TerminalLauncher) selectLinuxBackend() (linuxTerminalBackend, error) {
	for _, backend := range linuxTerminalBackends {
		if _, err := l.lookPath(backend.command); err == nil {
			return backend, nil
		}
	}

	supported := make([]string, 0, len(linuxTerminalBackends))
	for _, backend := range linuxTerminalBackends {
		supported = append(supported, backend.name)
	}
	return linuxTerminalBackend{}, fmt.Errorf("no supported Linux terminal backend was found; install %s", strings.Join(supported, ", "))
}
