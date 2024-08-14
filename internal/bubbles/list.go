package bubbles

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/style"
)

type ListDefaultItem string

func (i ListDefaultItem) FilterValue() string { return "" }

type listDefaultItemDelegate struct{}

func (d listDefaultItemDelegate) Height() int                             { return 1 }
func (d listDefaultItemDelegate) Spacing() int                            { return 0 }
func (d listDefaultItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d listDefaultItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	var (
		itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
		selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(style.Green)
	)

	i, ok := listItem.(ListDefaultItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func NewList(options ...func(*list.Model)) list.Model {

	var (
		defaultItems           = []list.Item{}
		defaultItemDelegate    = listDefaultItemDelegate{}
		defaultWidth           = 0
		defaultHeight          = 0
		defaultTitleStyle      = style.BoldPrimary
		defaultPaginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
		defaultHelpStyle       = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
		defaultNoItemsStyle    = lipgloss.NewStyle().PaddingLeft(2)
	)

	DefaultStyles := func(l *list.Model) {
		l.Styles.Title = defaultTitleStyle
		l.Styles.PaginationStyle = defaultPaginationStyle
		l.Styles.HelpStyle = defaultHelpStyle
		l.Styles.NoItems = defaultNoItemsStyle
	}

	l := list.New(defaultItems, defaultItemDelegate, defaultWidth, defaultHeight)

	DefaultStyles(&l)

	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	for _, o := range options {
		o(&l)
	}

	return l
}

func WithStatusBarItemName(singular, plural string) func(*list.Model) {
	return func(lm *list.Model) {
		lm.SetStatusBarItemName(singular, plural)
	}
}

func WithTitleStyle(style lipgloss.Style) func(*list.Model) {
	return func(lm *list.Model) {
		lm.Styles.Title = style
	}
}

func WithPaginationStyle(style lipgloss.Style) func(*list.Model) {
	return func(lm *list.Model) {
		lm.Styles.PaginationStyle = style
	}
}

func WithHelpStyle(style lipgloss.Style) func(*list.Model) {
	return func(lm *list.Model) {
		lm.Styles.HelpStyle = style
	}
}

func WithNoItemsStyle(style lipgloss.Style) func(*list.Model) {
	return func(lm *list.Model) {
		lm.Styles.NoItems = style
	}
}

func WithTitle(title string) func(*list.Model) {
	return func(lm *list.Model) {
		lm.Title = title
	}
}

func WithItems(items []list.Item) func(*list.Model) {
	return func(lm *list.Model) {
		lm.SetItems(items)
	}
}

func WithItemDelegate(delegate list.ItemDelegate) func(*list.Model) {
	return func(lm *list.Model) {
		lm.SetDelegate(delegate)
	}
}
