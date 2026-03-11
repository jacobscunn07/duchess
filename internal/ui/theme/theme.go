package theme

import "github.com/charmbracelet/lipgloss"

// Theme holds all AWS Dark color tokens and pre-built border styles.
// Color fields are lipgloss.CompleteColor — three-tier fallback (TrueColor / ANSI256 / ANSI)
// for correct rendering in tmux, SSH, and 16-color terminals.
//
// Only the two border styles are pre-built on the struct. All other styles
// (selected row, text, etc.) are composed by consuming components using the color token fields.
type Theme struct {
	// Color tokens
	Background  lipgloss.CompleteColor // Squid Ink — main app background
	Surface     lipgloss.CompleteColor // Elevated panels: status bar, modals, overlays
	Border      lipgloss.CompleteColor // Panel border lines — dim gray, tunable from Surface
	Accent      lipgloss.CompleteColor // AWS Orange — selected items, active focus states
	TextPrimary lipgloss.CompleteColor // Primary readable text
	Muted       lipgloss.CompleteColor // Separators, dimmed/secondary text

	// Border shape and pre-built styles
	BorderShape         lipgloss.Border // lipgloss.RoundedBorder()
	InactiveBorderStyle lipgloss.Style  // Rounded border + Muted foreground
	ActiveBorderStyle   lipgloss.Style  // Rounded border + Accent foreground
}

// DarkTheme returns a Theme populated with the AWS Dark color palette.
//
// ANSI 16-color fallbacks (Claude's discretion per CONTEXT.md):
//
//	Background/Surface → "0" (black — darkest available; 16-color cannot distinguish these two)
//	Border/Muted       → "8" (bright black / dark gray — dim separator weight)
//	Accent             → "3" (yellow — nearest 16-color analog to orange)
//	TextPrimary        → "7" (light gray / white — readable on dark background)
//
// Note on initialization: var DefaultTheme = DarkTheme() runs at package init time,
// before main(). CompleteColor bypasses lipgloss profile auto-detection by design —
// the explicit TrueColor/ANSI256/ANSI fields are always used directly. This is safe.
func DarkTheme() Theme {
	accent := lipgloss.CompleteColor{
		TrueColor: "#FF9900",
		ANSI256:   "214",
		ANSI:      "3",
	}
	muted := lipgloss.CompleteColor{
		TrueColor: "#555555",
		ANSI256:   "240",
		ANSI:      "8",
	}
	border := lipgloss.RoundedBorder()

	return Theme{
		Background:  lipgloss.CompleteColor{TrueColor: "#232F3E", ANSI256: "235", ANSI: "0"},
		Surface:     lipgloss.CompleteColor{TrueColor: "#2D2D2D", ANSI256: "236", ANSI: "0"},
		Border:      lipgloss.CompleteColor{TrueColor: "#444444", ANSI256: "238", ANSI: "8"},
		Accent:      accent,
		TextPrimary: lipgloss.CompleteColor{TrueColor: "#D0D0D0", ANSI256: "248", ANSI: "7"},
		Muted:       muted,

		BorderShape: border,
		InactiveBorderStyle: lipgloss.NewStyle().
			Border(border).
			BorderForeground(muted),
		ActiveBorderStyle: lipgloss.NewStyle().
			Border(border).
			BorderForeground(accent),
	}
}

// DefaultTheme is a package-level convenience instance for callers that do not need
// theme selection. Avoids calling DarkTheme() at every call site in downstream phases.
// Safe to use at package-level var declaration time (CompleteColor bypasses profile detection).
var DefaultTheme = DarkTheme()
