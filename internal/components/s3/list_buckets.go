package s3

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

func NewListBucketModel() ListBucketsModel {
	return ListBucketsModel{
		style: style.Border,
	}
}

type ListBucketsModel struct {
	style           lipgloss.Style
	availableWidth  int
	availableHeight int
}

func (m ListBucketsModel) Init() tea.Cmd {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	api := s3.NewListBucketsAPI(cfg)

	return tea.Batch(
		s3.ListBucketsQuery(context.TODO(), api),
	)
}

func (m ListBucketsModel) Update(msg interface{}) (ListBucketsModel, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case messages.AvailableWindowSizeMsg:
		m.updateAvailableWindowSize(msg.Width, msg.Height)
	}

	return m, tea.Batch(cmds...)
}

func (m ListBucketsModel) View() string {
	return m.style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			"buckets",
		),
	)
}

func (m ListBucketsModel) ViewHeight() int {
	return lipgloss.Height(m.View())
}

func (m *ListBucketsModel) updateAvailableWindowSize(w, h int) (int, int) {
	frameW, frameH := m.style.GetFrameSize()

	m.availableWidth, m.availableHeight = w-frameW, h-frameH

	m.style = m.style.
		Width(m.availableWidth)

	return m.availableWidth, m.availableHeight
}
