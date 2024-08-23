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
)

func NewListBucketModel() components.Model {
	return ListBucketModel{
		containerStyle: lipgloss.NewStyle().
			Margin(0).
			Padding(1),
		list: bubbles.NewList(
			bubbles.WithTitle("Buckets"),
			bubbles.WithStatusBarItemName("bucket", "buckets"),
		),
	}
}

type ListBucketModel struct {
	containerStyle lipgloss.Style
	list           list.Model
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
					BucketDetailsModelWithHeight(m.containerStyle.GetHeight()),
					BucketDetailsModelWithWidth(m.containerStyle.GetWidth()),
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
	return m.containerStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			// lipgloss.NewStyle().Bold(true).Render("S3 / Buckets"),
			// "",
			m.list.View(),
		),
	)
}

func (m ListBucketModel) ViewHeight() int {
	return lipgloss.Height(m.View())
}

func (m ListBucketModel) SetSize(width, height int) components.Model {
	_, frameH := m.containerStyle.GetFrameSize()
	w, h := m.containerStyle.GetHorizontalMargins(), m.containerStyle.GetVerticalMargins()

	containerWidth, containerHeight := width-w, height-h

	m.containerStyle = m.containerStyle.Width(containerWidth)
	m.containerStyle = m.containerStyle.Height(containerHeight)

	m.list.SetHeight(containerHeight - frameH - 2)

	return m
}

func (m ListBucketModel) GetBreadcrumb() []string {
	return []string{"Buckets"}
}
