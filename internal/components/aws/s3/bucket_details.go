package s3

import (
	"context"
	"io"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/bubbles"
	"github.com/jacobscunn07/duchess/internal/bubbles/viewport"
	"github.com/jacobscunn07/duchess/internal/charmbracelet/bubbletea/messages/aws/s3"
	"github.com/jacobscunn07/duchess/internal/components"

	s3_sdk "github.com/aws/aws-sdk-go-v2/service/s3"
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

	m.viewport = viewport.NewViewport(
		m.containerStyle.GetWidth()-w-m.containerStyle.GetHorizontalMargins(),
		m.containerStyle.GetHeight()-h-m.containerStyle.GetVerticalMargins())

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
	selectedObject string
	list           list.Model
	containerStyle lipgloss.Style
	tabContent     string
	viewport       viewport.Model
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
		case "enter":
			if i, ok := m.list.SelectedItem().(bubbles.ListDefaultItem); ok {
				m.selectedObject = string(i)
				cfg, err := config.LoadDefaultConfig(context.TODO())
				if err != nil {
					log.Fatalf("unable to load SDK config, %v", err)
				}

				client := s3_sdk.NewFromConfig(cfg)

				api := s3_data.NewApi(client)
				cmd := s3.GetObjectQuery(context.TODO(), *api, m.bucket, m.selectedObject)
				cmds = append(cmds, cmd)
			}
		}

	case s3.ListBucketObjectsQueryMessage:
		m.objects = msg.Objects

		objects := []list.Item{}
		for _, o := range msg.Objects {
			objects = append(objects, bubbles.ListDefaultItem(o))
		}

		m.list.SetItems(objects)
	case s3.GetObjectQueryMessage:
		defer msg.Contents.Close()

		tabContent, _ := io.ReadAll(msg.Contents)
		m.tabContent = string(tabContent)

		m.viewport.SetContent(m.tabContent)
		m.viewport.SetTitle(m.selectedObject)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m BucketDetailsModel) View() string {
	var contents string

	if m.tabContent == "" {
		contents = m.list.View()

		// contents = lipgloss.JoinVertical(
		// 	lipgloss.Left,
		// 	"S3 / Objects",
		// 	"",
		// 	m.list.View(),
		// )
	} else {
		contents = m.viewport.View()
	}

	return m.containerStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			contents,
		),
	)
}

func (m BucketDetailsModel) ViewHeight() int {
	return lipgloss.Height(m.View())
}

func (m BucketDetailsModel) SetSize(width, height int) components.Model {
	frameW, frameH := m.containerStyle.GetFrameSize()
	w, h := m.containerStyle.GetHorizontalMargins(), m.containerStyle.GetVerticalMargins()

	containerWidth, containerHeight := width-w, height-h

	m.containerStyle = m.containerStyle.Width(containerWidth)
	m.containerStyle = m.containerStyle.Height(containerHeight)

	m.viewport.SetHeight(containerHeight - frameH)
	m.viewport.SetWidth(containerWidth - frameW)

	return m
}
