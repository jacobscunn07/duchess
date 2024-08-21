package viewport

import (
	"fmt"
	"strings"

	vp "github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/style"
)

func NewViewport(width, height int, options ...func(m *Model)) Model {
	var (
		defaultViewPortTitleStyle = func() lipgloss.Style {
			b := lipgloss.NormalBorder()
			b.Right = "├"
			return lipgloss.NewStyle().
				BorderStyle(b).
				Padding(0, 1).
				BorderForeground(style.Green).
				Foreground(style.Green)
		}()

		defaultViewPortInfoStyle = func() lipgloss.Style {
			b := lipgloss.NormalBorder()
			b.Left = "┤"
			return defaultViewPortTitleStyle.
				BorderStyle(b).
				BorderForeground(style.Green).
				Foreground(style.Green)
		}()
	)

	m := Model{
		model:              vp.New(width, height-6), //6 = height of header and footer
		title:              "Title",
		viewPortTitleStyle: defaultViewPortTitleStyle,
		viewPortInfoStyle:  defaultViewPortInfoStyle,
	}

	m.model.YPosition = lipgloss.Height(m.headerView())
	m.model.HighPerformanceRendering = false

	for _, o := range options {
		o(&m)
	}

	return m
}

func WithTitle(title string) func(*Model) {
	return func(m *Model) {
		m.title = title
	}
}

type Model struct {
	model              vp.Model
	viewPortTitleStyle lipgloss.Style
	viewPortInfoStyle  lipgloss.Style
	title              string
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	updatedModel, cmd := m.model.Update(msg)

	m.model = updatedModel

	return m, cmd
}

func (m Model) View() string {
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.model.View(), m.footerView())
}

func (m Model) headerView() string {
	title := m.viewPortTitleStyle.Render(m.title)
	line := lipgloss.NewStyle().Foreground(style.Green).Render(strings.Repeat("─", max(0, m.model.Width-lipgloss.Width(title))))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m Model) footerView() string {
	info := m.viewPortInfoStyle.Render(fmt.Sprintf("%3.f%%", m.model.ScrollPercent()*100))
	line := lipgloss.NewStyle().Foreground(style.Green).Render(strings.Repeat("─", max(0, m.model.Width-lipgloss.Width(info))))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (m *Model) SetContent(content string) {
	m.model.SetContent(
		lipgloss.
			NewStyle().
			PaddingLeft(2).
			Render(content),
	)
}

func (m *Model) SetHeight(height int) {
	m.model.Height = height - lipgloss.Height(m.headerView()) - lipgloss.Height(m.footerView())
}

func (m *Model) SetWidth(width int) {
	m.model.Width = width
}

func (m *Model) SetTitle(title string) {
	m.title = title
}
