package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jacobscunn07/duchess/internal/ui/theme"
)

// duchessLogo is the verbatim ASCII art for the application name.
// Font: "Big" figlet. Trailing blank lines trimmed.
// NOTE: The fourth line contains a backtick character — string concatenation used to embed it.
const duchessLogo = "     _            _\n" +
	"    | |          | |\n" +
	"  __| |_   _  ___| |__   ___  ___ ___\n" +
	" / _`| | | |/ __| '_ \\ / _ \\/ __/ __|\n" +
	"| (_| | |_| | (__| | | |  __/\\__ \\__ \\\n" +
	" \\__,_|\\__,_|\\___|_| |_|\\___||___/___/"

// logoHeight is the number of rows the ASCII logo occupies.
var logoHeight = strings.Count(duchessLogo, "\n") + 1

// headerHeight is the number of terminal rows the header occupies.
// One extra row below the logo is reserved for the centered version string.
var headerHeight = logoHeight + 1

const (
	headerPadLeft = 2 // cols of Surface background before the logo
	headerLogoGap = 4 // cols of Surface background between logo and metadata
)

// Package-level lipgloss styles defined once to avoid recreation on every render.
// All colors use theme.DefaultTheme fields — no inline lipgloss.Color() calls.
var (
	headerStyle = lipgloss.NewStyle().Background(theme.DefaultTheme.Surface)

	logoStyle = lipgloss.NewStyle().
			Foreground(theme.DefaultTheme.Accent).
			Inherit(headerStyle)

	metaLabelStyle = lipgloss.NewStyle().
			Foreground(theme.DefaultTheme.Muted).
			Inherit(headerStyle)

	metaValueStyle = lipgloss.NewStyle().
			Foreground(theme.DefaultTheme.Accent).
			Inherit(headerStyle)
)

// renderHeader returns the header string for the current model state.
// The result is exactly width display columns wide and headerHeight rows tall.
// Returns "" when width == 0 (guard against zero-dimension first render).
func renderHeader(m rootModel, width int) string {
	if width == 0 {
		return ""
	}

	// Left column: ASCII logo followed by a centered version string on the next line.
	logoStr := logoStyle.Render(duchessLogo)
	logoW := lipgloss.Width(logoStr)
	// Version line: centered within logo width with full Surface background.
	versionLine := headerStyle.Width(logoW).Align(lipgloss.Center).Render("v" + version)
	logoCol := lipgloss.JoinVertical(lipgloss.Left, logoStr, versionLine)

	// Right column: metadata directly to the right of the logo, vertically centered.
	metaLines := strings.Join([]string{
		metaLabelStyle.Render("profile: ") + metaValueStyle.Render(m.cfg.Profile),
		metaLabelStyle.Render("region:  ") + metaValueStyle.Render(m.cfg.Region),
		metaLabelStyle.Render("refresh: ") + metaValueStyle.Render(fmt.Sprintf("%ds", m.cfg.RefreshInterval)),
	}, "\n")

	// Vertically center metadata within the full header height.
	metaColRaw := lipgloss.PlaceVertical(headerHeight, lipgloss.Center, metaLines)
	metaCol    := headerStyle.Render(metaColRaw)
	metaW      := lipgloss.Width(metaCol)

	// Spacer columns with Surface background.
	spacer := strings.Repeat("\n", headerHeight-1)
	leftPadCol := headerStyle.Width(headerPadLeft).Render(spacer)
	gapCol     := headerStyle.Width(headerLogoGap).Render(spacer)

	// Fill remaining width to the right with Surface background.
	rightW := width - headerPadLeft - logoW - headerLogoGap - metaW
	if rightW < 0 {
		rightW = 0
	}
	rightFill := headerStyle.Width(rightW).Render(spacer)

	row := lipgloss.JoinHorizontal(lipgloss.Top, leftPadCol, logoCol, gapCol, metaCol, rightFill)
	return headerStyle.Width(width).Render(row)
}
