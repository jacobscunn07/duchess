package header

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/charmbracelet/bubbletea/messages/aws/sts"
	"github.com/jacobscunn07/duchess/internal/components"
	"github.com/jacobscunn07/duchess/internal/messages"
	"github.com/jacobscunn07/duchess/internal/style"
)

func New() components.Model {
	return Model{
		style: style.Border,
	}
}

type Model struct {
	style           lipgloss.Style
	availableWidth  int
	availableHeight int
	accountId       string
	region          string
	principal       string
}

func (m Model) Init() tea.Cmd {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	api := sts.NewGetCallerIdentityAPI(cfg)

	return tea.Batch(
		sts.GetCallerIdentity(context.TODO(), api),
	)
}

func (m Model) Update(msg interface{}) (components.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case messages.AvailableWindowSizeMsg:
		m.updateAvailableWindowSize(msg.Width, msg.Height)
	case sts.GetCallerIdentityMessage:
		m.accountId = msg.AccountId
		m.principal = msg.Arn
		m.region = msg.Region
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			fmt.Sprintf("region: %s", m.region),
			fmt.Sprintf("account id: %s", m.accountId),
			fmt.Sprintf("principal: %s", m.principal),
		),
	)
}

func (m Model) ViewHeight() int {
	return lipgloss.Height(m.View())
}

func (m *Model) updateAvailableWindowSize(w, h int) (int, int) {
	frameW, frameH := m.style.GetFrameSize()

	m.availableWidth, m.availableHeight = w-frameW, h-frameH

	m.style = m.style.
		Width(m.availableWidth)

	return m.availableWidth, m.availableHeight
}
