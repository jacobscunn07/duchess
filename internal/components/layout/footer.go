package layout

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/charmbracelet/bubbletea/messages/aws/sts"
	"github.com/jacobscunn07/duchess/internal/components"
	"github.com/jacobscunn07/duchess/internal/style"
	"github.com/jacobscunn07/duchess/internal/utils"
)

func NewFooter() *FooterModel {
	return &FooterModel{
		containerStyle: lipgloss.NewStyle().
			Background(style.Green).
			Padding(0).
			Margin(0),
	}
}

type FooterModel struct {
	containerStyle lipgloss.Style
	time           time.Time
	profile        string
	region         string
	accountid      string
}

func (m FooterModel) Init() tea.Cmd {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	api := sts.NewGetCallerIdentityAPI(cfg)

	return tea.Batch(
		sts.GetCallerIdentity(context.TODO(), api),
	)
}

func (m FooterModel) Update(msg interface{}) (components.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case sts.GetCallerIdentityMessage:
		m.accountid = msg.AccountId
		m.region = msg.Region
		m.profile = msg.Profile
	case utils.RefreshCommandMessage:
		m.time = msg.Time
	}

	return m, tea.Batch(cmds...)
}

func (m FooterModel) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().
				Background(style.Green).
				Foreground(style.Black).
				Padding(0).
				Margin(0).
				PaddingLeft(1).
				PaddingRight(1).
				Render(m.profile),
			lipgloss.NewStyle().
				Padding(0).
				Margin(0).
				PaddingLeft(1).
				PaddingRight(1).
				Render(m.region),
			lipgloss.NewStyle().
				Background(style.Green).
				Foreground(style.Black).
				Padding(0).
				Margin(0).
				PaddingLeft(1).
				PaddingRight(1).
				Render(m.accountid),
			lipgloss.NewStyle().
				Padding(0).
				Margin(0).
				PaddingLeft(1).
				PaddingRight(1).
				Render("", fmt.Sprint(m.time.Format("03:04:05PM"))),
		),
	)
}

func (m FooterModel) ViewHeight() int {
	return lipgloss.Height(m.View())
}

func (m FooterModel) SetSize(width, height int) components.Model {
	w, h := m.containerStyle.GetFrameSize()

	containerWidth, _ := width-w, height-h

	m.containerStyle = m.containerStyle.Width(containerWidth)

	return m
}

func (m FooterModel) GetBreadcrumb() []string {
	return []string{}
}
