package s3

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/components"
)

func NewServiceShell() *ServiceShellModel {
	return &ServiceShellModel{
		Navigation: nil,
		Content:    NewListBucketModel(),
	}
}

type ServiceShellModel struct {
	Navigation components.Model
	Content    components.Model
}

func (m ServiceShellModel) Init() tea.Cmd {
	return tea.Batch(
		m.Content.Init(),
	)
}

func (m ServiceShellModel) Update(msg interface{}) (components.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.Content, cmd = m.Content.Update(msg)
	return m, cmd
}

func (m ServiceShellModel) View() string {
	return m.Content.View()
}

func (m ServiceShellModel) SetSize(width, height int) components.Model {
	m.Content = m.Content.SetSize(width, height)

	return m
}

func (m ServiceShellModel) ViewHeight() int {
	return lipgloss.Height(m.View())
}

func (m ServiceShellModel) GetBreadcrumb() []string {
	return m.Content.GetBreadcrumb()
}
