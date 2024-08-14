package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/components"
	"github.com/jacobscunn07/duchess/internal/components/aws/s3"
	"github.com/jacobscunn07/duchess/internal/components/header"
	"github.com/jacobscunn07/duchess/internal/messages"
	"github.com/jacobscunn07/duchess/internal/style"
)

func New() *Model {
	return &Model{
		style:   style.Border,
		header:  header.New(),
		content: s3.NewListBucketModel(),
	}
}

type Model struct {
	style           lipgloss.Style
	availableHeight int
	availableWidth  int
	header          components.Model
	content         components.Model
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.header.Init(),
		m.content.Init(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		w, h := m.updateAvailableWindowSize(msg.Width, msg.Height)

		m.header, cmd = m.header.Update(messages.AvailableWindowSizeMsg{
			Height: h,
			Width:  w,
		})
		cmds = append(cmds, cmd)

		m.content, cmd = m.content.Update(messages.AvailableWindowSizeMsg{
			Height: h - m.header.ViewHeight(),
			Width:  w,
		})
		cmds = append(cmds, cmd)
	}

	m.header, cmd = m.header.Update(msg)
	cmds = append(cmds, cmd)

	m.content, cmd = m.content.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.header.View(),
			m.content.View(),
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
