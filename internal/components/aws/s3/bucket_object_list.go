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

	s3_data "github.com/jacobscunn07/duchess/internal/data/s3"
)

func NewBucketDetailsModel(bucket string, options ...func(*BucketDetailsModel)) components.Model {
	m := &BucketDetailsModel{
		bucket:  bucket,
		objects: []string{},
		list: bubbles.NewList(
			bubbles.WithTitle("Objects"),
			bubbles.WithStatusBarItemName("object", "objects"),
		),
		containerStyle: lipgloss.NewStyle().
			Margin(0).
			Padding(1),
	}

	for _, o := range options {
		o(m)
	}

	w, h := m.containerStyle.GetFrameSize()
	m.list.SetSize(m.containerStyle.GetWidth()-w, m.containerStyle.GetHeight()-h)

	return *m
}

func BucketDetailsModelWithHeight(height int) func(m *BucketDetailsModel) {
	return func(m *BucketDetailsModel) {
		m.containerStyle = m.containerStyle.Height(height)
	}
}

func BucketDetailsModelWithWidth(width int) func(m *BucketDetailsModel) {
	return func(m *BucketDetailsModel) {
		m.containerStyle = m.containerStyle.Width(width)
	}
}

type BucketDetailsModel struct {
	bucket         string
	objects        []string
	list           list.Model
	containerStyle lipgloss.Style
}

func (m BucketDetailsModel) Init() tea.Cmd {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	repo := s3_data.NewBucketObjectRepository(cfg, m.bucket)

	return tea.Batch(
		s3.ListBucketObjectsQuery(context.TODO(), repo),
	)
}

func (m BucketDetailsModel) Update(msg interface{}) (components.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "esc":
			blm := NewListBucketModel()
			blm = blm.SetSize(m.containerStyle.GetWidth(), m.containerStyle.GetHeight())
			cmd := blm.Init()
			cmds = append(cmds, cmd)
			return blm, tea.Batch(cmds...)
		case "enter":
			if i, ok := m.list.SelectedItem().(bubbles.ListDefaultItem); ok {
				bodm := NewBucketObjectDetailsModel(
					m.bucket,
					string(i),
					BucketObjectDetailsModelWithHeight(m.containerStyle.GetHeight()),
					BucketObjectDetailsModelWithWidth(m.containerStyle.GetWidth()),
				)

				cmd := bodm.Init()
				cmds = append(cmds, cmd)
				return bodm, tea.Batch(cmds...)
			}
		}

	case s3.ListBucketObjectsQueryMessage:
		m.objects = msg.Objects

		objects := []list.Item{}
		for _, o := range msg.Objects {
			objects = append(objects, bubbles.ListDefaultItem(o))
		}

		m.list.SetItems(objects)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m BucketDetailsModel) View() string {
	return m.containerStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.list.View(),
		),
	)
}

func (m BucketDetailsModel) ViewHeight() int {
	return lipgloss.Height(m.View())
}

func (m BucketDetailsModel) SetSize(width, height int) components.Model {
	w, h := m.containerStyle.GetHorizontalMargins(), m.containerStyle.GetVerticalMargins()

	containerWidth, containerHeight := width-w, height-h

	m.containerStyle = m.containerStyle.Width(containerWidth)
	m.containerStyle = m.containerStyle.Height(containerHeight)

	return m
}

func (m BucketDetailsModel) GetBreadcrumb() []string {
	return []string{"Buckets", m.bucket}
}
