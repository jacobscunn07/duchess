package s3

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/bubbles"
	"github.com/jacobscunn07/duchess/internal/charmbracelet/bubbletea/messages/aws/s3"
	"github.com/jacobscunn07/duchess/internal/components"
	"github.com/jacobscunn07/duchess/internal/messages"
	"github.com/jacobscunn07/duchess/internal/style"
)

func NewListBucketModel() components.Model {
	return ListBucketModel{
		style: style.Border,
		list: bubbles.NewList(
			bubbles.WithTitle("Buckets"),
			bubbles.WithStatusBarItemName("bucket", "buckets"),
		),
	}
}

type ListBucketModel struct {
	style           lipgloss.Style
	availableWidth  int
	availableHeight int
	list            list.Model
}

func (m ListBucketModel) Init() tea.Cmd {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	api := s3.NewListBucketsAPI(cfg)

	return tea.Batch(
		s3.ListBucketsQuery(context.TODO(), api),
	)
}

func (m ListBucketModel) Update(msg interface{}) (components.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case messages.AvailableWindowSizeMsg:
		m.updateAvailableWindowSize(msg.Width, msg.Height)

		w, h := m.style.GetFrameSize()
		m.list.SetSize(msg.Width-w, msg.Height-h)

	case s3.ListBucketsQueryMessage:
		buckets := []list.Item{}
		for _, b := range msg.Buckets {
			buckets = append(buckets, bubbles.ListDefaultItem(b.Name))
		}

		m.list.SetItems(buckets)
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "enter":
			if i, ok := m.list.SelectedItem().(bubbles.ListDefaultItem); ok {
				bm := NewBucketDetailsModel(
					string(i),
					BucketDetailsModelWithHeight(m.availableHeight),
					BucketDetailsModelWithWidth(m.availableWidth),
				)

				cmd := bm.Init()
				cmds = append(cmds, cmd)
				return bm, tea.Batch(cmds...)
			}

			return m, tea.Batch(cmds...)
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m ListBucketModel) View() string {
	return m.list.View()
}

func (m ListBucketModel) ViewHeight() int {
	return lipgloss.Height(m.View())
}

func (m *ListBucketModel) updateAvailableWindowSize(w, h int) (int, int) {
	frameW, frameH := m.style.GetFrameSize()

	m.availableWidth, m.availableHeight = w-frameW, h-frameH

	m.style = m.style.
		Height(m.availableHeight).
		Width(m.availableWidth)

	return m.availableWidth, m.availableHeight
}
