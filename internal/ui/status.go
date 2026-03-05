package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// version is the current application version shown in the status bar.
const version = "0.1.0"

// Package-level lipgloss styles defined once to avoid recreation on every render.
var (
	statusBarStyle = lipgloss.NewStyle().Background(lipgloss.Color("236"))

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("248")).
			Bold(true).
			Inherit(statusBarStyle)

	profileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Inherit(statusBarStyle)

	regionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Inherit(statusBarStyle)

	identityStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("248")).
			Inherit(statusBarStyle)

	clockStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("248")).
			Inherit(statusBarStyle)

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Inherit(statusBarStyle)
)

// separator is the rendered separator string with dimmed foreground.
const separatorValue = " | "

// truncateARN trims arn so it fits within maxLen display characters.
// If len(arn) <= maxLen the arn is returned unchanged.
// Otherwise returns arn[:maxLen-3] + "..." so the result is exactly maxLen characters.
func truncateARN(arn string, maxLen int) string {
	if len(arn) <= maxLen {
		return arn
	}
	return arn[:maxLen-3] + "..."
}

// renderStatusBar returns the status bar string for the current model state.
// The result is exactly m.width display columns wide (lipgloss.Width == m.width).
// Returns "" when m.width == 0.
func renderStatusBar(m rootModel) string {
	if m.width == 0 {
		return ""
	}

	sep := separatorStyle.Render(separatorValue)

	// Build left group: version | profile | region
	leftStr := versionStyle.Render(version) +
		sep +
		profileStyle.Render(m.cfg.Profile) +
		sep +
		regionStyle.Render(m.cfg.Region)

	// Build identity string based on current state.
	var identityStr string
	switch m.state {
	case stateLoading:
		identityStr = m.spinner.View() + " loading..."
	case stateError:
		identityStr = "error"
	case stateReady:
		identityStr = m.account + " " + truncateARN(m.arn, 40)
	}

	// Build active panel indicator shown when ready.
	panelIndicator := ""
	if m.state == stateReady {
		switch m.activePanel {
		case panelECS:
			panelIndicator = "[ECS]"
		default: // panelS3
			panelIndicator = "[S3]"
		}
	}

	// Build right group: identity | panel indicator | clock
	clock := m.now.Format("15:04:05")
	rightStr := identityStyle.Render(identityStr)
	if panelIndicator != "" {
		rightStr += sep + versionStyle.Render(panelIndicator)
	}
	rightStr += sep + clockStyle.Render(clock)

	// Calculate gap to fill full width.
	gap := m.width - lipgloss.Width(leftStr) - lipgloss.Width(rightStr)
	if gap < 0 {
		gap = 0
	}

	return statusBarStyle.Width(m.width).Render(leftStr + strings.Repeat(" ", gap) + rightStr)
}
