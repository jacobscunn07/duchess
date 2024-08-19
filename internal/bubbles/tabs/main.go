package tabs

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/style"
)

type Tab interface {
	tea.Model
	Title() string
}

func NewModel(options ...func(*Model)) *Model {
	m := &Model{
		tabs: map[string]Tab{},
	}

	for _, o := range options {
		o(m)
	}

	return m
}

type Model struct {
	tabs      map[string]Tab
	activetab string
}

func (m *Model) AddTab(key string, tab Tab) {
	m.tabs[key] = tab
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return nil, nil
}

func (m Model) View() string {
	// row := lipgloss.JoinHorizontal(
	// 	lipgloss.Top,
	// 	activeTab.Render("Objects"),
	// 	tab.Render("TBD"),
	// )
	// space := tabGap.Render(strings.Repeat(" ", 0))
	// gap := tabGap.Render(strings.Repeat(" ", max(0, lipgloss.Width(row))))
	// row = lipgloss.JoinHorizontal(lipgloss.Bottom, space, row, gap)

	// str := strings.Ne
	str := []string{}

	for k := range m.tabs {
		str = append(str, k)
	}

	row := lipgloss.JoinHorizontal(
		lipgloss.Bottom,
		str...,
	)

	return row
}

var (

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
