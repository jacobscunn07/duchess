package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/components"
	"github.com/jacobscunn07/duchess/internal/components/aws/s3"
	"github.com/jacobscunn07/duchess/internal/components/footer"
	"github.com/jacobscunn07/duchess/internal/components/header"
	"github.com/jacobscunn07/duchess/internal/style"
	"github.com/jacobscunn07/duchess/internal/utils"
)

func New() *Model {
	return &Model{
		containerStyle: lipgloss.NewStyle().Margin(0).Padding(0),
		header:         header.New(),
		content:        s3.NewListBucketModel(),
		footer:         footer.New(),
	}
}

type Model struct {
	containerStyle lipgloss.Style
	header         components.Model
	content        components.Model
	footer         components.Model
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.header.Init(),
		m.content.Init(),
		m.footer.Init(),
		utils.RefreshCommand(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		w, h := m.containerStyle.GetFrameSize()
		width, height := msg.Width-w, msg.Height-h

		m.containerStyle = m.containerStyle.Width(width)
		m.containerStyle = m.containerStyle.Height(height)

		m.header = m.header.SetSize(width, 5)
		m.content = m.content.SetSize(width, height-m.header.ViewHeight()-lipgloss.Height(m.GetBreadcrumbsView())-m.footer.ViewHeight())
		m.footer = m.footer.SetSize(width, 5)
	case utils.RefreshCommandMessage:
		cmd = utils.RefreshCommand()
		cmds = append(cmds, cmd)
	}

	m.header, cmd = m.header.Update(msg)
	cmds = append(cmds, cmd)

	m.content, cmd = m.content.Update(msg)
	cmds = append(cmds, cmd)

	m.footer, cmd = m.footer.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.containerStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.header.View(),
			m.GetBreadcrumbsView(),
			m.content.View(),
			m.footer.View(),
		),
	)
}

func (m Model) ViewHeight() int {
	return lipgloss.Height(m.View())
}

func (m Model) GetBreadcrumb() []string {
	return []string{}
}

func (m Model) GetBreadcrumbsView() string {
	style := lipgloss.NewStyle().
		Padding(0).
		Margin(1, 0, 0, 1).
		Bold(true).
		Foreground(style.Green)
	return style.Render(strings.Join(m.content.GetBreadcrumb(), " / "))
}
