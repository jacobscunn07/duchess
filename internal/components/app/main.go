package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/style"
)

func New() *Model {
	return &Model{
		style: style.Border,
	}
}

type Model struct {
	style  lipgloss.Style
	height int
	width  int
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	w, h := m.style.GetFrameSize()

	m.style = m.style.
		Height(m.height - h).
		Width(m.width - w)

	return m.style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			"hi",
			"hola",
		),
	)
}
