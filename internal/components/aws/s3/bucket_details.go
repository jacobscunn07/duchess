package s3

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/components"
	"github.com/jacobscunn07/duchess/internal/style"
)

func NewBucketDetailsModel(bucket string) components.Model {
	return BucketDetailsModel{
		bucket: bucket,
	}
}

type BucketDetailsModel struct {
	bucket string
}

func (m BucketDetailsModel) Init() tea.Cmd {
	return nil
}

func (m BucketDetailsModel) Update(msg interface{}) (components.Model, tea.Cmd) {
	return m, nil
}

func (m BucketDetailsModel) View() string {
	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		activeTab.Render("Details"),
		tab.Render("Objects"),
	)
	gap := tabGap.Render(strings.Repeat(" ", max(0, lipgloss.Width(row))))
	row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		row,
		lipgloss.NewStyle().MarginLeft(2).Render(m.bucket))
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
