package s3

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/charmbracelet/bubbletea/messages/aws/s3"
	"github.com/jacobscunn07/duchess/internal/components"
	"github.com/jacobscunn07/duchess/internal/style"

	s3_data "github.com/jacobscunn07/duchess/internal/data/s3"
)

func NewBucketDetailsModel(bucket string) components.Model {
	return BucketDetailsModel{
		bucket:  bucket,
		objects: []string{},
	}
}

type BucketDetailsModel struct {
	bucket  string
	objects []string
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
	case s3.ListBucketObjectsQueryMessage:
		m.objects = msg.Objects
	}
	return m, tea.Batch(cmds...)
}

func (m BucketDetailsModel) View() string {
	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		activeTab.Render("Objects"),
		tab.Render("Permissions"),
	)
	space := tabGap.Render(strings.Repeat(" ", 0))
	gap := tabGap.Render(strings.Repeat(" ", max(0, lipgloss.Width(row))))
	row = lipgloss.JoinHorizontal(lipgloss.Bottom, space, row, gap)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		style.BoldPrimary.MarginLeft(2).MarginTop(1).MarginBottom(1).Render(fmt.Sprintf("s3 / buckets / %v", m.bucket)),
		row,
		lipgloss.NewStyle().MarginLeft(2).Render(m.objects...))
}

func (m BucketDetailsModel) ViewHeight() int {
	return lipgloss.Height(m.View())
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
