package content

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/messages"
	"github.com/jacobscunn07/duchess/internal/style"
)

func New() Model {
	return Model{
		style: style.Border,
	}
}

type Model struct {
	style           lipgloss.Style
	availableWidth  int
	availableHeight int
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg interface{}) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case messages.AvailableWindowSizeMsg:
		m.updateAvailableWindowSize(msg.Width, msg.Height)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			"content",
		),
	)
}

func (m Model) ViewHeight() int {
	return lipgloss.Height(m.View())
}

func (m *Model) updateAvailableWindowSize(w, h int) (int, int) {
	frameW, frameH := m.style.GetFrameSize()

	m.availableWidth, m.availableHeight = w-frameW, h-frameH

	m.style = m.style.
		Height(m.availableHeight).
		Width(m.availableWidth)

	return m.availableWidth, m.availableHeight
}
