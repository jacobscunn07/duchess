package s3

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/components"
)

func NewListBucketModel() components.Model {
	return ListBucketModel{}
}

type ListBucketModel struct{}

func (m ListBucketModel) Init() tea.Cmd {
	return nil
}

func (m ListBucketModel) Update(interface{}) (components.Model, tea.Cmd) {
	return m, nil
}

func (m ListBucketModel) View() string {
	return "hello from list bucket model"
}

func (m ListBucketModel) ViewHeight() int {
	return lipgloss.Height(m.View())
}
