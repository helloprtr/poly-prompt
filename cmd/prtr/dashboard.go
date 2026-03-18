//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type keyChoice struct {
	Key     string
	Command string
	Desc    string
}

type dashboardModel struct {
	target  string
	watch   string
	branch  string
	choices []keyChoice
	chosen  string
}

func newDashboardModel(target, watchStatus, branch string) dashboardModel {
	return dashboardModel{
		target: target,
		watch:  watchStatus,
		branch: branch,
		choices: []keyChoice{
			{"g", "go", "send a prompt"},
			{"t", "take", "next action from clipboard"},
			{"s", "swap", "change AI target"},
			{"h", "history", "recent runs"},
			{"q", "", "quit"},
		},
	}
}

func (m dashboardModel) KeyChoices() []keyChoice { return m.choices }

func (m dashboardModel) Init() tea.Cmd { return nil }

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" || key == "q" {
			return m, tea.Quit
		}
		for _, c := range m.choices {
			if c.Key == key && c.Command != "" {
				m.chosen = c.Command
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

var (
	dashTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#58a6ff"))
	dashLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("#8b949e")).Width(8)
	dashValue   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e6edf3"))
	dashKey     = lipgloss.NewStyle().Foreground(lipgloss.Color("#58a6ff")).Width(3)
	dashCmdName = lipgloss.NewStyle().Foreground(lipgloss.Color("#e6edf3")).Width(10)
	dashCmdDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#484f58"))
)

func (m dashboardModel) View() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(dashTitle.Render("⚡ prtr") + "\n\n")

	sb.WriteString(dashLabel.Render("CONTEXT") + "\n")
	sb.WriteString("  " + dashLabel.Render("branch:") + dashValue.Render(m.branch) + "\n")
	sb.WriteString("  " + dashLabel.Render("target:") + dashValue.Render(m.target) + "\n")
	sb.WriteString("  " + dashLabel.Render("watch:") + dashValue.Render(m.watch) + "\n")

	sb.WriteString("\n" + dashLabel.Render("QUICK ACTIONS") + "\n")
	for _, c := range m.choices {
		sb.WriteString("  " + dashKey.Render(c.Key) + dashCmdName.Render(c.Command) + dashCmdDesc.Render(c.Desc) + "\n")
	}
	return sb.String()
}

// runDashboard launches the dashboard TUI and exec-replaces if a command was chosen.
func runDashboard(target, watchStatus, branch string) error {
	m := newDashboardModel(target, watchStatus, branch)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return err
	}

	final, ok := result.(dashboardModel)
	if !ok || final.chosen == "" {
		return nil
	}

	// Exec-replace with chosen command
	binary, err := os.Executable()
	if err != nil {
		binary, err = exec.LookPath("prtr")
		if err != nil {
			return fmt.Errorf("could not find prtr binary: %w", err)
		}
	}
	return syscall.Exec(binary, []string{"prtr", final.chosen}, os.Environ())
}
