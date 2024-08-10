package content

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/charmbracelet/bubbletea/messages/aws/s3"
	"github.com/jacobscunn07/duchess/internal/messages"
	"github.com/jacobscunn07/duchess/internal/style"
)

func New() Model {
	return Model{
		style:   style.Border,
		buckets: []string{},
	}
}

type Model struct {
	style           lipgloss.Style
	availableWidth  int
	availableHeight int
	buckets         []string
}

func (m Model) Init() tea.Cmd {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	api := s3.NewListBucketsAPI(cfg)

	return tea.Batch(
		s3.ListBucketsQuery(context.TODO(), api),
	)
}

func (m Model) Update(msg interface{}) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case messages.AvailableWindowSizeMsg:
		m.updateAvailableWindowSize(msg.Width, msg.Height)

	case s3.ListBucketsQueryMessage:
		for _, b := range msg.Buckets {
			m.buckets = append(m.buckets, b.Name)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.buckets...,
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
