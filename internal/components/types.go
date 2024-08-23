package components

import tea "github.com/charmbracelet/bubbletea"

type Model interface {
	Init() tea.Cmd
	Update(interface{}) (Model, tea.Cmd)
	View() string
	SetSize(width, height int) Model
	ViewHeight() int
	GetBreadcrumb() []string
}
