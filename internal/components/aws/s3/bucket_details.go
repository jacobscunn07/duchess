package s3

import (
	"context"
	"fmt"
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
	"github.com/jacobscunn07/duchess/internal/messages"

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
		style: lipgloss.NewStyle(),
	}

	for _, o := range options {
		o(m)
	}

	w, h := m.style.GetFrameSize()
	m.list.SetSize(m.availableWidth-w, m.availableHeight-h)

	return *m
}

func BucketDetailsModelWithHeight(height int) func(m *BucketDetailsModel) {
	return func(m *BucketDetailsModel) {
		m.availableHeight = height
	}
}

func BucketDetailsModelWithWidth(width int) func(m *BucketDetailsModel) {
	return func(m *BucketDetailsModel) {
		m.availableWidth = width
	}
}

type BucketDetailsModel struct {
	bucket          string
	objects         []string
	selectedObject  string
	list            list.Model
	availableWidth  int
	availableHeight int
	style           lipgloss.Style
	tabContent      string
	viewport        viewport.Model
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

	case messages.AvailableWindowSizeMsg:
		// m.updateAvailableWindowSize(msg.Width, msg.Height)
		m.availableHeight = msg.Height
		m.availableWidth = msg.Width

		w, h := m.style.GetFrameSize()
		m.list.SetSize(msg.Width-w, msg.Height-h)

		m.viewport.SetWidth(msg.Width - w)
		m.viewport.SetHeight(msg.Height - h) // What is 4?

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

		w, h := m.style.GetFrameSize()

		m.viewport = viewport.NewViewport(
			m.availableWidth-w+2,
			m.availableHeight-h-4, // What is 4?
			// m.availableWidth-w,
			// m.availableHeight-h, // What is 4?
			viewport.WithTitle(m.selectedObject))

		// m.viewport.SetContent(strings.Repeat(fmt.Sprintf("%v\n", m.tabContent), 300))
		m.viewport.SetContent(fmt.Sprintf("availw:%v,\navailh:%v\nw:%v\nh:%v", m.availableWidth, m.availableHeight, w, h))
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
	} else {
		contents = m.viewport.View()
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		contents,
	)
}

func (m BucketDetailsModel) ViewHeight() int {
	return lipgloss.Height(m.View())
}

// func (m *BucketDetailsModel) updateAvailableWindowSize(w, h int) (int, int) {
// 	frameW, frameH := m.style.GetFrameSize()

// 	m.availableWidth, m.availableHeight = w-frameW, h-frameH

// 	m.style = m.style.
// 		Height(m.availableHeight).
// 		Width(m.availableWidth)
// 		// Height(h).Width(w)

// 	return m.availableWidth, m.availableHeight
// }
