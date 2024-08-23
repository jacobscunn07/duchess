package header

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/charmbracelet/bubbletea/messages/aws/sts"
	"github.com/jacobscunn07/duchess/internal/components"
	"github.com/jacobscunn07/duchess/internal/style"
)

func New() *Model {
	return &Model{
		containerStyle: lipgloss.NewStyle().
			Background(style.Green).
			Foreground(style.Black).
			Padding(0).
			Margin(0).
			PaddingLeft(1).
			PaddingRight(1),
	}
}

type Model struct {
	containerStyle lipgloss.Style
	principal      string
}

func (m Model) Init() tea.Cmd {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	api := sts.NewGetCallerIdentityAPI(cfg)

	return tea.Batch(
		sts.GetCallerIdentity(context.TODO(), api),
	)
}

func (m Model) Update(msg interface{}) (components.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case sts.GetCallerIdentityMessage:
		m.principal = msg.Arn
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			m.containerStyle.
				Background(style.Green).
				Foreground(style.Black).
				Padding(0).
				Margin(0).
				PaddingLeft(1).
				PaddingRight(1).
				Render(m.principal),
		),
	)
}

func (m Model) ViewHeight() int {
	return lipgloss.Height(m.View())
}

func (m Model) SetSize(width, height int) components.Model {
	w, h := m.containerStyle.GetFrameSize()

	containerWidth, _ := width-w, height-h

	m.containerStyle = m.containerStyle.Width(containerWidth)

	return m
}

func (m Model) GetBreadcrumb() []string {
	return []string{}
}
