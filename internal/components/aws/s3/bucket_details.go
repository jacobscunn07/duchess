package s3

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/bubbles"
	"github.com/jacobscunn07/duchess/internal/charmbracelet/bubbletea/messages/aws/s3"
	"github.com/jacobscunn07/duchess/internal/components"
	"github.com/jacobscunn07/duchess/internal/messages"
	"github.com/jacobscunn07/duchess/internal/style"

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
		tabs: func() string {
			row := lipgloss.JoinHorizontal(
				lipgloss.Top,
				activeTab.Render("Objects"),
				tab.Render("TBD"),
			)
			space := tabGap.Render(strings.Repeat(" ", 0))
			gap := tabGap.Render(strings.Repeat(" ", max(0, lipgloss.Width(row))))
			row = lipgloss.JoinHorizontal(lipgloss.Bottom, space, row, gap)

			return row
		}(),
	}

	for _, o := range options {
		o(m)
	}

	w, h := m.style.GetFrameSize()
	m.list.SetSize(m.availableWidth-w, m.availableHeight-h-lipgloss.Height(m.tabs))

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
	tabs            string
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
		m.updateAvailableWindowSize(msg.Width, msg.Height)

		w, h := m.style.GetFrameSize()
		m.list.SetSize(msg.Width-w, msg.Height-h-lipgloss.Height(m.tabs))

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
		// m.viewport.SetContent(m.tabContent)

		m.viewport = viewport.New(80, 5)
		m.viewport.YPosition = lipgloss.Height(m.headerView())
		m.viewport.HighPerformanceRendering = false
		m.viewport.SetContent(m.tabContent)
		// m.ready = true
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
		// contents = m.tabContent
		contents = fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.tabs,
		contents,
	)
}

func (m BucketDetailsModel) ViewHeight() int {
	return lipgloss.Height(m.View())
}

func (m *BucketDetailsModel) updateAvailableWindowSize(w, h int) (int, int) {
	frameW, frameH := m.style.GetFrameSize()

	m.availableWidth, m.availableHeight = w-frameW, h-frameH

	m.style = m.style.
		Height(m.availableHeight).
		Width(m.availableWidth)

	return m.availableWidth, m.availableHeight
}
func (m BucketDetailsModel) headerView() string {
	title := viewPortTitleStyle.Render("Mr. Pager")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m BucketDetailsModel) footerView() string {
	info := viewPortInfoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

var (

	// General.

	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	divider = lipgloss.NewStyle().
		SetString("•").
		Padding(0, 1).
		Foreground(subtle).
		String()

	url = lipgloss.NewStyle().Foreground(special).Render

	// Tabs.

	activeTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}

	tabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┴",
		BottomRight: "┴",
	}

	tab = lipgloss.NewStyle().
		Border(tabBorder, true).
		BorderForeground(style.Green).
		Padding(0, 1)

	activeTab = tab.Border(activeTabBorder, true)

	tabGap = tab.
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false)
)

var (
	viewPortTitleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	viewPortInfoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return viewPortTitleStyle.BorderStyle(b)
	}()
)
