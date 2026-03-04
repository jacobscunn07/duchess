package ui

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/jacobscunn07/duchess/internal/config"
	session "github.com/jacobscunn07/duchess/internal/aws"
)

// state represents the loading state of the root model.
type state int

const (
	stateLoading state = iota
	stateReady
	stateError
)

// rootModel is the top-level Bubble Tea model for the duchess TUI.
type rootModel struct {
	ctx     context.Context
	awsCfg  aws.Config
	cfg     *config.Config
	width   int
	height  int
	state   state
	spinner spinner.Model
	now     time.Time
	account string
	arn     string
	err     error
}

// NewRootModel constructs a rootModel with safe default dimensions.
// It applies NO_COLOR guard if the env var is set.
func NewRootModel(ctx context.Context, cfg *config.Config, awsCfg aws.Config) rootModel {
	// NO_COLOR guard: disable ANSI color codes when env var is set.
	if os.Getenv("NO_COLOR") != "" {
		lipgloss.SetColorProfile(termenv.Ascii)
	}

	s := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("205"))),
	)

	return rootModel{
		ctx:     ctx,
		awsCfg:  awsCfg,
		cfg:     cfg,
		width:   80,
		height:  24,
		state:   stateLoading,
		spinner: s,
		now:     time.Now(),
	}
}

// Init implements tea.Model. Returns a batch of startup commands.
func (m rootModel) Init() tea.Cmd {
	return tea.Batch(
		fetchIdentityCmd(m.ctx, m.awsCfg, m.cfg.Profile),
		m.spinner.Tick,
		tickCmd(),
	)
}

// Update implements tea.Model. Handles all incoming messages.
func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case identityLoadedMsg:
		m.account = msg.account
		m.arn = msg.arn
		m.state = stateReady
		return m, nil

	case identityErrMsg:
		m.err = msg.err
		m.state = stateError
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		return m, tickCmd()

	case spinner.TickMsg:
		// Only propagate spinner ticks when loading; swallow otherwise.
		if m.state == stateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

// View implements tea.Model. Returns the current string representation.
// Returns "" when width=0 to guard against zero-dimension first render.
func (m rootModel) View() string {
	if m.width == 0 {
		return ""
	}
	statusBar := renderStatusBar(m)
	contentH := m.height - lipgloss.Height(statusBar)
	if contentH < 0 {
		contentH = 0
	}
	content := lipgloss.NewStyle().
		Width(m.width).
		Height(contentH).
		Render(m.contentView())
	return lipgloss.JoinVertical(lipgloss.Left, content, statusBar)
}

// contentView returns the main content area string based on the current model state.
func (m rootModel) contentView() string {
	statusBar := renderStatusBar(m)
	contentH := m.height - lipgloss.Height(statusBar)
	if contentH <= 0 {
		return m.spinner.View() + " Connecting to AWS..."
	}
	switch m.state {
	case stateLoading:
		return lipgloss.Place(m.width, contentH, lipgloss.Center, lipgloss.Center,
			m.spinner.View()+" Connecting to AWS...")
	case stateError:
		return lipgloss.NewStyle().Width(m.width).Padding(1, 2).Render(m.err.Error())
	default: // stateReady
		return lipgloss.NewStyle().Width(m.width).Render("")
	}
}

// fetchIdentityCmd wraps session.GetCallerIdentity asynchronously.
// Always returns a closure that sends either identityLoadedMsg or identityErrMsg.
func fetchIdentityCmd(ctx context.Context, awsCfg aws.Config, profile string) tea.Cmd {
	return func() tea.Msg {
		account, arn, err := session.GetCallerIdentity(ctx, awsCfg)
		if err != nil {
			return identityErrMsg{err: session.ClassifyCredentialError(err, profile)}
		}
		return identityLoadedMsg{account: account, arn: arn}
	}
}

// tickCmd returns a command that fires every second.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
