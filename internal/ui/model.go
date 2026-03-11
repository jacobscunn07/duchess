package ui

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	btoverlay "github.com/rmhubbert/bubbletea-overlay"

	"github.com/jacobscunn07/duchess/internal/config"
	session "github.com/jacobscunn07/duchess/internal/aws"
	overlay "github.com/jacobscunn07/duchess/internal/ui/overlay"
	ecspanel "github.com/jacobscunn07/duchess/internal/ui/ecs"
	s3panel "github.com/jacobscunn07/duchess/internal/ui/s3"
	"github.com/jacobscunn07/duchess/internal/ui/theme"
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
	ctx                  context.Context    // root context (parent of session contexts)
	sessionCtx           context.Context    // per-session cancellable context
	cancelSession        context.CancelFunc // cancels sessionCtx and all its children
	cfg                  *config.Config
	awsCfg               aws.Config
	width                int
	height               int
	state                state
	spinner              spinner.Model
	now                  time.Time
	account              string
	arn                  string
	err                  error
	s3Panel              s3panel.Model
	ecsPanel             ecspanel.Model      // ECS browser panel
	activePanel          activePanel         // defaults to panelS3 (zero value)
	isProfileOverlayOpen bool
	profileOverlay       overlay.ProfileOverlay
	isRegionOverlayOpen  bool
	regionOverlay        overlay.RegionOverlay
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
		spinner.WithStyle(lipgloss.NewStyle().Foreground(theme.DefaultTheme.Accent)),
	)

	sessionCtx, cancelSession := context.WithCancel(ctx)

	return rootModel{
		ctx:           ctx,
		sessionCtx:    sessionCtx,
		cancelSession: cancelSession,
		cfg:           cfg,
		width:         80,
		height:        24,
		state:         stateLoading,
		spinner:       s,
		now:           time.Now(),
	}
}

// Init implements tea.Model. Returns a batch of startup commands.
func (m rootModel) Init() tea.Cmd {
	return tea.Batch(
		fetchIdentityCmd(m.sessionCtx, m.cfg.Profile, m.cfg.Region),
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
		// Overlay key interception — must return before reaching panel fallthrough.
		if m.isProfileOverlayOpen {
			switch msg.String() {
			case "esc":
				m.isProfileOverlayOpen = false
				return m, nil
			case "enter":
				selected := m.profileOverlay.SelectedProfile()
				if selected != "" {
					m.isProfileOverlayOpen = false
					return m, func() tea.Msg { return profileSelectedMsg{profile: selected} }
				}
				m.isProfileOverlayOpen = false
				return m, nil
			default:
				var cmd tea.Cmd
				m.profileOverlay, cmd = m.profileOverlay.Update(msg)
				return m, cmd
			}
		}

		if m.isRegionOverlayOpen {
			switch msg.String() {
			case "esc":
				m.isRegionOverlayOpen = false
				return m, nil
			case "enter":
				selected := m.regionOverlay.SelectedRegion()
				if selected != "" {
					m.isRegionOverlayOpen = false
					return m, func() tea.Msg { return regionSelectedMsg{region: selected} }
				}
				m.isRegionOverlayOpen = false
				return m, nil
			default:
				var cmd tea.Cmd
				m.regionOverlay, cmd = m.regionOverlay.Update(msg)
				return m, cmd
			}
		}

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

		case "p":
			if m.state == stateReady || m.state == stateError {
				profiles, err := overlay.ListAWSProfiles()
				if err != nil {
					m.err = fmt.Errorf("could not read ~/.aws/config: %w", err)
					m.state = stateError
					return m, nil
				}
				if len(profiles) == 0 {
					m.err = fmt.Errorf("no profiles found in ~/.aws/config — add a [profile ...] section")
					m.state = stateError
					return m, nil
				}
				m.isProfileOverlayOpen = true
				m.profileOverlay = overlay.NewProfileOverlay(m.cfg.Profile, m.width, m.height, profiles)
				return m, m.profileOverlay.Init()
			}

		case "r":
			if m.state == stateReady || m.state == stateError {
				m.isRegionOverlayOpen = true
				m.regionOverlay = overlay.NewRegionOverlay(m.cfg.Region, m.width, m.height)
				return m, m.regionOverlay.Init()
			}
		}

	case identityLoadedMsg:
		m.account = msg.account
		m.arn = msg.arn
		m.awsCfg = msg.cfg
		m.state = stateReady
		// Compute content height (excluding header and status bar) for panel sizing
		statusBar := renderStatusBar(m)
		contentH := m.height - headerHeight - lipgloss.Height(statusBar)
		if contentH < 1 {
			contentH = 1
		}
		refreshInterval := time.Duration(m.cfg.RefreshInterval) * time.Second
		m.s3Panel = s3panel.NewModel(m.sessionCtx, m.awsCfg, m.width, contentH, refreshInterval)
		m.ecsPanel = ecspanel.NewModel(m.sessionCtx, m.awsCfg, m.width, contentH, refreshInterval)
		return m, tea.Batch(m.s3Panel.Init(), m.ecsPanel.Init())

	case identityErrMsg:
		m.err = msg.err
		m.state = stateError
		return m, nil

	case profileSelectedMsg:
		// Cancel in-flight goroutines from the current session.
		if m.cancelSession != nil {
			m.cancelSession()
		}
		// Create a new cancellable session context.
		sessionCtx, cancel := context.WithCancel(m.ctx)
		m.sessionCtx = sessionCtx
		m.cancelSession = cancel
		// Update config immediately so the status bar shows the new profile name
		// before identity confirms.
		m.cfg.Profile = msg.profile
		// Reset to loading state; clear identity fields.
		m.state = stateLoading
		m.account = ""
		m.arn = ""
		m.err = nil
		// Re-fetch identity with new profile using the new session context.
		return m, tea.Batch(
			fetchIdentityCmd(m.sessionCtx, m.cfg.Profile, m.cfg.Region),
			m.spinner.Tick,
		)

	case regionSelectedMsg:
		if m.cancelSession != nil {
			m.cancelSession()
		}
		sessionCtx, cancel := context.WithCancel(m.ctx)
		m.sessionCtx = sessionCtx
		m.cancelSession = cancel
		// Update status bar immediately — cfg.Region is read by renderStatusBar
		m.cfg.Region = msg.region
		m.isRegionOverlayOpen = false
		// Update awsCfg region — CRITICAL: must set before passing to NewModel
		m.awsCfg.Region = msg.region
		// Reset ECS panel only — S3 is global (bucket list is not region-specific)
		statusBar := renderStatusBar(m)
		contentH := m.height - headerHeight - lipgloss.Height(statusBar)
		if contentH < 1 {
			contentH = 1
		}
		refreshInterval := time.Duration(m.cfg.RefreshInterval) * time.Second
		m.ecsPanel = ecspanel.NewModel(m.sessionCtx, m.awsCfg, m.width, contentH, refreshInterval)
		// S3 panel intentionally NOT reset (S3 is global)
		return m, m.ecsPanel.Init()

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

// baseView returns the full panel + status bar view without any overlay.
func (m rootModel) baseView() string {
	header := renderHeader(m, m.width)
	statusBar := renderStatusBar(m)
	contentH := m.height - headerHeight - lipgloss.Height(statusBar)
	if contentH < 0 {
		contentH = 0
	}
	content := lipgloss.NewStyle().Width(m.width).Height(contentH).Render(m.contentView())
	return lipgloss.JoinVertical(lipgloss.Left, header, content, statusBar)
}

// View implements tea.Model. Returns the current string representation.
// Returns "" when width=0 to guard against zero-dimension first render.
// When the profile overlay is open, composites it over baseView().
func (m rootModel) View() string {
	if m.width == 0 {
		return ""
	}
	bg := m.baseView()
	if m.isProfileOverlayOpen {
		return btoverlay.Composite(m.profileOverlay.View(), bg, btoverlay.Center, btoverlay.Center, 0, 0)
	}
	if m.isRegionOverlayOpen {
		return btoverlay.Composite(m.regionOverlay.View(), bg, btoverlay.Center, btoverlay.Center, 0, 0)
	}
	return bg
}

// contentView returns the main content area string based on the current model state.
func (m rootModel) contentView() string {
	statusBar := renderStatusBar(m)
	contentH := m.height - headerHeight - lipgloss.Height(statusBar)
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
