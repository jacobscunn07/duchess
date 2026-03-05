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
	ecspanel "github.com/jacobscunn07/duchess/internal/ui/ecs"
	s3panel "github.com/jacobscunn07/duchess/internal/ui/s3"
)

// state represents the loading state of the root model.
type state int

const (
	stateLoading state = iota
	stateReady
	stateError
)

// activePanel represents which panel is currently shown.
type activePanel int

const (
	panelS3  activePanel = iota
	panelECS
)

// rootModel is the top-level Bubble Tea model for the duchess TUI.
type rootModel struct {
	ctx         context.Context
	cfg         *config.Config
	awsCfg      aws.Config
	width       int
	height      int
	state       state
	spinner     spinner.Model
	now         time.Time
	account     string
	arn         string
	err         error
	s3Panel     s3panel.Model
	ecsPanel    ecspanel.Model // ECS browser panel
	activePanel activePanel   // defaults to panelS3 (zero value)
}

// NewRootModel constructs a rootModel with safe default dimensions.
// It applies NO_COLOR guard if the env var is set.
// The AWS config is NOT loaded here — it is loaded lazily inside fetchIdentityCmd
// so that any profile/credential errors surface as inline TUI errors rather than
// pre-launch cobra errors.
func NewRootModel(ctx context.Context, cfg *config.Config) rootModel {
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
		fetchIdentityCmd(m.ctx, m.cfg.Profile, m.cfg.Region),
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
		if m.state == stateReady {
			var s3Cmd, ecsCmd tea.Cmd
			m.s3Panel, s3Cmd = m.s3Panel.Update(msg)
			m.ecsPanel, ecsCmd = m.ecsPanel.Update(msg)
			return m, tea.Batch(s3Cmd, ecsCmd)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab":
			if m.state == stateReady {
				if m.activePanel == panelS3 {
					m.activePanel = panelECS
				} else {
					m.activePanel = panelS3
				}
				return m, nil
			}
		}

	case identityLoadedMsg:
		m.account = msg.account
		m.arn = msg.arn
		m.awsCfg = msg.cfg
		m.state = stateReady
		// Compute content height (excluding status bar) for panel sizing
		statusBar := renderStatusBar(m)
		contentH := m.height - lipgloss.Height(statusBar)
		if contentH < 1 {
			contentH = 1
		}
		m.s3Panel = s3panel.NewModel(m.ctx, m.awsCfg, m.width, contentH)
		m.ecsPanel = ecspanel.NewModel(m.ctx, m.awsCfg, m.width, contentH)
		return m, tea.Batch(m.s3Panel.Init(), m.ecsPanel.Init())

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
		// In stateReady, forward spinner ticks to both panels (for refresh spinners)
		if m.state == stateReady {
			var s3Cmd, ecsCmd tea.Cmd
			m.s3Panel, s3Cmd = m.s3Panel.Update(msg)
			m.ecsPanel, ecsCmd = m.ecsPanel.Update(msg)
			return m, tea.Batch(s3Cmd, ecsCmd)
		}
		return m, nil
	}

	// Forward all remaining messages to both panels when in stateReady.
	// Each panel only responds to its own typed messages — unexported types
	// like clustersLoadedMsg won't match s3Panel's switch, and vice versa.
	if m.state == stateReady {
		var s3Cmd, ecsCmd tea.Cmd
		m.s3Panel, s3Cmd = m.s3Panel.Update(msg)
		m.ecsPanel, ecsCmd = m.ecsPanel.Update(msg)
		return m, tea.Batch(s3Cmd, ecsCmd)
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
		switch m.activePanel {
		case panelECS:
			return m.ecsPanel.View()
		default: // panelS3
			return m.s3Panel.View()
		}
	}
}

// fetchIdentityCmd loads the AWS config and calls STS GetCallerIdentity asynchronously.
// Loading the AWS config here (instead of in cmd/root.go) ensures that profile-not-found
// and other config errors surface as inline TUI errors rather than pre-launch cobra errors.
// Always returns a closure that sends either identityLoadedMsg or identityErrMsg.
func fetchIdentityCmd(ctx context.Context, profile, region string) tea.Cmd {
	return func() tea.Msg {
		awsCfg, err := session.NewAWSConfig(ctx, profile, region)
		if err != nil {
			return identityErrMsg{err: err}
		}
		account, arn, err := session.GetCallerIdentity(ctx, awsCfg)
		if err != nil {
			return identityErrMsg{err: session.ClassifyCredentialError(err, profile)}
		}
		return identityLoadedMsg{account: account, arn: arn, cfg: awsCfg}
	}
}

// tickCmd returns a command that fires every second.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
