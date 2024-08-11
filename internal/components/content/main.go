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
	"github.com/jacobscunn07/duchess/internal/messages"
	"github.com/jacobscunn07/duchess/internal/style"
)

const listHeight = 14
const defaultWidth = 20

var (
	titleStyle        = style.BoldPrimary // lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(style.Green)
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

// type item struct {
// 	title, desc string
// }

// func (i item) Title() string       { return i.title }
// func (i item) Description() string { return i.desc }
// func (i item) FilterValue() string { return i.title }

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

func New() Model {

	l := list.New([]list.Item{}, itemDelegate{}, defaultWidth, listHeight)

	l.Title = "Buckets"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	return Model{
		style:   style.Border,
		buckets: []string{},
		list:    l,
	}
}

type Model struct {
	style           lipgloss.Style
	availableWidth  int
	availableHeight int
	buckets         []string
	list            list.Model
	choice          string
	quitting        bool
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

func (m Model) Update(msg interface{}) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case messages.AvailableWindowSizeMsg:
		m.updateAvailableWindowSize(msg.Width, msg.Height)

		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case s3.ListBucketsQueryMessage:
		buckets := []list.Item{}
		for _, b := range msg.Buckets {
			// m.buckets = append(m.buckets, b.Name)
			// m.list = list.New()
			buckets = append(buckets, item(b.Name))
		}

		// m.list = list.New(buckets, list.NewDefaultDelegate(), 0, 0)
		m.list.SetItems(buckets)
		// m.list.Title = "Buckets"
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
	// return m.style.Render(
	// 	lipgloss.JoinVertical(
	// 		lipgloss.Left,
	// 		// m.buckets...,
	// 		docStyle.Render(m.list.View()),
	// 	),
	// )

	if m.choice != "" {
		return quitTextStyle.Render(fmt.Sprintf("%s? Sounds good to me.", m.choice))
	}
	if m.quitting {
		return quitTextStyle.Render("Not hungry? That’s cool.")
	}
	return "\n" + m.list.View()
}

func (m Model) ViewHeight() int {
	return lipgloss.Height(m.View())
}

func (m *Model) updateAvailableWindowSize(w, h int) (int, int) {
	frameW, frameH := m.style.GetFrameSize()

	m.availableWidth, m.availableHeight = w-frameW, h-frameH

	m.style = m.style.
		Height(m.availableHeight).
		Width(m.availableWidth)

	return m.availableWidth, m.availableHeight
}
