package content

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/charmbracelet/bubbletea/messages/aws/s3"
	"github.com/jacobscunn07/duchess/internal/components"
	"github.com/jacobscunn07/duchess/internal/style"
)

func New() *Model {
	return &Model{
		containerStyle: lipgloss.NewStyle().
			Margin(0).
			Padding(1),
		list: NewList(),
	}
}

type Model struct {
	containerStyle lipgloss.Style
	list           list.Model
	choice         string
	quitting       bool
}

func (m Model) Init() tea.Cmd {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	api := s3.NewListBucketsAPI(cfg)

	return tea.Batch(
		s3.ListBucketsQuery(context.TODO(), api),
	)
}

func (m Model) Update(msg interface{}) (components.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case s3.ListBucketsQueryMessage:
		buckets := []list.Item{}
		for _, b := range msg.Buckets {
			buckets = append(buckets, item(b.Name))
		}

		m.list.SetItems(buckets)
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
			}

			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.choice != "" {
		return quitTextStyle.Render(fmt.Sprintf("%s? Sounds good to me.", m.choice))
	}
	if m.quitting {
		return quitTextStyle.Render("Not hungry? That’s cool.")
	}

	return m.containerStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			// lipgloss.NewStyle().Bold(true).Render("S3 / Buckets"),
			// "",
			m.list.View(),
		),
	)
}

func (m Model) ViewHeight() int {
	return lipgloss.Height(m.View())
}

func (m Model) SetSize(width, height int) components.Model {
	_, frameH := m.containerStyle.GetFrameSize()
	w, h := m.containerStyle.GetHorizontalMargins(), m.containerStyle.GetVerticalMargins()

	containerWidth, containerHeight := width-w, height-h

	m.containerStyle = m.containerStyle.Width(containerWidth)
	m.containerStyle = m.containerStyle.Height(containerHeight)

	m.list.SetHeight(containerHeight - frameH)

	return m
}

func (m Model) GetBreadcrumb() []string {
	return []string{}
}

var (
	titleStyle        = style.BoldPrimary
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(style.Green)
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
	noItemsStyle      = lipgloss.NewStyle().PaddingLeft(2)
)

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
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

func NewList() list.Model {
	l := list.New([]list.Item{}, itemDelegate{}, 0, 0)

	l.Title = "Buckets"
	l.SetStatusBarItemName("bucket", "buckets")
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	l.Styles.NoItems = noItemsStyle

	return l
}
