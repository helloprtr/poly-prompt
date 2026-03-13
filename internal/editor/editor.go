package editor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

var ErrCanceled = errors.New("interactive edit canceled")

type Request struct {
	Initial string
	Status  string
}

type Editor interface {
	Edit(ctx context.Context, req Request) (string, error)
}

type BubbleEditor struct {
	output io.Writer
}

func New(output io.Writer) *BubbleEditor {
	return &BubbleEditor{output: output}
}

func (e *BubbleEditor) Edit(ctx context.Context, req Request) (string, error) {
	initialModel := newModel(req)
	program := tea.NewProgram(
		initialModel,
		tea.WithContext(ctx),
		tea.WithOutput(e.output),
		tea.WithInputTTY(),
	)

	finalModel, err := program.Run()
	if err != nil {
		return "", fmt.Errorf("run interactive editor: %w", err)
	}

	result, ok := finalModel.(model)
	if !ok {
		return "", errors.New("interactive editor returned an unexpected model")
	}

	if result.canceled {
		return "", ErrCanceled
	}

	return result.value, nil
}

type model struct {
	textarea textarea.Model
	value    string
	canceled bool
	width    int
	height   int
	status   string
}

func newModel(req Request) model {
	field := textarea.New()
	field.SetValue(req.Initial)
	field.Focus()
	field.ShowLineNumbers = false
	field.Prompt = ""
	field.CharLimit = 0

	return model{
		textarea: field,
		value:    strings.TrimSpace(req.Initial),
		status:   strings.TrimSpace(req.Status),
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(maxInt(20, msg.Width-2))
		m.textarea.SetHeight(maxInt(6, msg.Height-5))
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.canceled = true
			return m, tea.Quit
		case "ctrl+s":
			m.value = m.textarea.Value()
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m model) View() string {
	help := "Ctrl+S save and exit | Ctrl+C cancel"
	if m.status != "" {
		help = m.status + "\n" + help
	}

	if m.height > 0 {
		return m.textarea.View() + "\n" + help
	}
	return help + "\n\n" + m.textarea.View()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
