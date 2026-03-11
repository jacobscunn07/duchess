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

// headerHeight is the number of terminal rows the header occupies.
// Derived from the actual duchessLogo line count so layout math stays in sync
// if the logo ever changes. Never a bare integer.
var headerHeight = strings.Count(duchessLogo, "\n") + 1

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

	// Left column: ASCII logo styled with Accent foreground on Surface background.
	logoStr := logoStyle.Render(duchessLogo)
	logoW := lipgloss.Width(logoStr)

	// Right column: metadata stacked vertically — one item per line.
	// Order: version (no label), profile, region, refresh interval.
	// Labels in Muted, values in Accent.
	metaLines := strings.Join([]string{
		metaValueStyle.Render("v" + version),
		metaLabelStyle.Render("profile: ") + metaValueStyle.Render(m.cfg.Profile),
		metaLabelStyle.Render("region: ") + metaValueStyle.Render(m.cfg.Region),
		metaLabelStyle.Render("refresh: ") + metaValueStyle.Render(fmt.Sprintf("%ds", m.cfg.RefreshInterval)),
	}, "\n")

	// Bottom-align the metadata column within headerHeight rows.
	// This visually anchors the metadata to the bottom rows of the logo.
	metaCol := lipgloss.PlaceVertical(headerHeight, lipgloss.Bottom, metaLines)
	metaW := lipgloss.Width(metaCol)

	// Build a gap column to push metadata to the right edge.
	gapW := width - logoW - metaW
	if gapW < 0 {
		gapW = 0
	}
	// A gap column of the right height ensures JoinHorizontal aligns correctly.
	gapLines := strings.Repeat("\n", headerHeight-1)
	gapCol := lipgloss.NewStyle().Width(gapW).Render(gapLines)

	// Use JoinHorizontal so multi-line columns are stitched side-by-side correctly.
	// This avoids raw string concatenation which causes incorrect height measurement.
	row := lipgloss.JoinHorizontal(lipgloss.Top, logoStr, gapCol, metaCol)
	return headerStyle.Width(width).Render(row)
}
