package deep

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	deeprun "github.com/helloprtr/poly-prompt/internal/deep/run"
)

// StageState represents a pipeline stage's current state.
type StageState int

const (
	StagePending StageState = iota
	StageRunning
	StageDone
	StageFailed
)

// Stage is a single pipeline step.
type Stage struct {
	Name  string
	State StageState
}

// PipelineModel is the bubbletea model for --deep visualization.
type PipelineModel struct {
	stages   []Stage
	progress progress.Model
	current  string
	elapsed  time.Duration
	start    time.Time
	done     bool
	ch       <-chan deeprun.Progress
}

type progressMsg deeprun.Progress
type tickMsg time.Time

// NewPipelineModel creates a model for the given stage names.
func NewPipelineModel(names []string) PipelineModel {
	stages := make([]Stage, len(names))
	for i, n := range names {
		stages[i] = Stage{Name: n, State: StagePending}
	}
	bar := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)
	return PipelineModel{stages: stages, progress: bar, start: time.Now()}
}

// Stages returns the current stage list (for testing).
func (m PipelineModel) Stages() []Stage { return m.stages }

// Advance marks a stage as running and returns a new model.
// Copies the stages slice to avoid mutating the backing array of the original.
func (m PipelineModel) Advance(name string) (PipelineModel, bool) {
	copied := make([]Stage, len(m.stages))
	copy(copied, m.stages)
	m.stages = copied
	for i, s := range m.stages {
		if s.Name == name {
			m.stages[i].State = StageRunning
			m.current = name
			return m, true
		}
	}
	return m, false
}

// Complete marks a stage as done and returns a new model.
// Copies the stages slice to avoid mutating the backing array of the original.
func (m PipelineModel) Complete(name string) (PipelineModel, bool) {
	copied := make([]Stage, len(m.stages))
	copy(copied, m.stages)
	m.stages = copied
	for i, s := range m.stages {
		if s.Name == name {
			m.stages[i].State = StageDone
			return m, true
		}
	}
	return m, false
}

// WithChannel attaches a Progress channel to the model.
func (m PipelineModel) WithChannel(ch <-chan deeprun.Progress) PipelineModel {
	m.ch = ch
	return m
}

func (m PipelineModel) Init() tea.Cmd {
	return tea.Batch(waitForProgress(m.ch), tickCmd())
}

func waitForProgress(ch <-chan deeprun.Progress) tea.Cmd {
	return func() tea.Msg {
		p, ok := <-ch
		if !ok {
			return tea.Quit()
		}
		return progressMsg(p)
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m PipelineModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case progressMsg:
		p := deeprun.Progress(msg)
		// Mark previous running stage as done.
		copied := make([]Stage, len(m.stages))
		copy(copied, m.stages)
		m.stages = copied
		for i, s := range m.stages {
			if s.State == StageRunning {
				m.stages[i].State = StageDone
			}
		}
		m, _ = m.Advance(p.Step)
		if p.Index >= p.Total {
			m.done = true
			return m, tea.Quit
		}
		return m, waitForProgress(m.ch)
	case tickMsg:
		m.elapsed = time.Since(m.start)
		return m, tickCmd()
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m PipelineModel) View() string {
	done := 0
	for _, s := range m.stages {
		if s.State == StageDone {
			done++
		}
	}

	total := len(m.stages)
	var pct float64
	if total > 0 {
		pct = float64(done) / float64(total)
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(m.progress.ViewAs(pct))
	sb.WriteString(fmt.Sprintf("  %d/%d\n", done, total))
	sb.WriteString(renderStages(m.stages))
	if m.current != "" && !m.done {
		sb.WriteString(fmt.Sprintf("\n%s: working... (%s)\n",
			m.current, m.elapsed.Round(time.Second)))
	}
	if m.done {
		sb.WriteString("\n done\n")
	}
	return sb.String()
}

var (
	styleDone    = lipgloss.NewStyle().Foreground(lipgloss.Color("#3fb950"))
	styleRunning = lipgloss.NewStyle().Foreground(lipgloss.Color("#e3b341"))
	stylePending = lipgloss.NewStyle().Foreground(lipgloss.Color("#484f58"))
)

func renderStages(stages []Stage) string {
	parts := make([]string, len(stages))
	for i, s := range stages {
		switch s.State {
		case StageDone:
			parts[i] = styleDone.Render(s.Name + " ✓")
		case StageRunning:
			parts[i] = styleRunning.Render(s.Name + " ⠼")
		case StageFailed:
			parts[i] = lipgloss.NewStyle().Foreground(lipgloss.Color("#f85149")).Render(s.Name + " ✕")
		default:
			parts[i] = stylePending.Render(s.Name + " ○")
		}
	}
	return strings.Join(parts, "  →  ")
}

// RunPipelineTUI starts the bubbletea program and returns when done.
func RunPipelineTUI(ch <-chan deeprun.Progress, stages []string) error {
	m := NewPipelineModel(stages).WithChannel(ch)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}
