package s3

import (
	"context"
	"io"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	s3_sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/bubbles/viewport"
	"github.com/jacobscunn07/duchess/internal/charmbracelet/bubbletea/messages/aws/s3"
	"github.com/jacobscunn07/duchess/internal/components"
	s3_data "github.com/jacobscunn07/duchess/internal/data/s3"
)

func NewBucketObjectDetailsModel(bucket, key string, options ...func(*BucketObjectDetailsModel)) components.Model {
	m := &BucketObjectDetailsModel{
		bucket: bucket,
		key:    key,
		containerStyle: lipgloss.NewStyle().
			Margin(0).
			Padding(1),
	}

	for _, o := range options {
		o(m)
	}

	w, h := m.containerStyle.GetFrameSize()

	m.viewport = viewport.NewViewport(
		m.containerStyle.GetWidth()-w-m.containerStyle.GetHorizontalMargins(),
		m.containerStyle.GetHeight()-h-m.containerStyle.GetVerticalMargins())

	return *m
}

func BucketObjectDetailsModelWithHeight(height int) func(m *BucketObjectDetailsModel) {
	return func(m *BucketObjectDetailsModel) {
		m.containerStyle = m.containerStyle.Height(height)
	}
}

func BucketObjectDetailsModelWithWidth(width int) func(m *BucketObjectDetailsModel) {
	return func(m *BucketObjectDetailsModel) {
		m.containerStyle = m.containerStyle.Width(width)
	}
}

type BucketObjectDetailsModel struct {
	bucket         string
	key            string
	containerStyle lipgloss.Style
	viewport       viewport.Model
}

func (m BucketObjectDetailsModel) Init() tea.Cmd {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	client := s3_sdk.NewFromConfig(cfg)
	api := s3_data.NewApi(client)

	return tea.Batch(
		s3.GetObjectQuery(context.TODO(), *api, m.bucket, m.key),
	)
}

func (m BucketObjectDetailsModel) Update(msg interface{}) (components.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case s3.GetObjectQueryMessage:
		defer msg.Contents.Close()

		contents, _ := io.ReadAll(msg.Contents)

		m.viewport.SetTitle(m.key)
		m.viewport.SetContent(string(contents))
	}

	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m BucketObjectDetailsModel) View() string {
	return m.containerStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.viewport.View(),
		),
	)
}

func (m BucketObjectDetailsModel) ViewHeight() int {
	return lipgloss.Height(m.View())
}

func (m BucketObjectDetailsModel) SetSize(width, height int) components.Model {
	frameW, frameH := m.containerStyle.GetFrameSize()
	w, h := m.containerStyle.GetHorizontalMargins(), m.containerStyle.GetVerticalMargins()

	containerWidth, containerHeight := width-w, height-h

	m.containerStyle = m.containerStyle.Width(containerWidth)
	m.containerStyle = m.containerStyle.Height(containerHeight)

	m.viewport.SetHeight(containerHeight - frameH)
	m.viewport.SetWidth(containerWidth - frameW)

	return m
}

func (m BucketObjectDetailsModel) GetBreadcrumb() []string {
	return []string{"Buckets", m.bucket, m.key}
}
