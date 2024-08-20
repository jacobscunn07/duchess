package style

import "github.com/charmbracelet/lipgloss"

const (
	Green  = lipgloss.Color("#6fe7d2")
	Purple = lipgloss.Color("63")
	Black  = lipgloss.Color("#000000")
)

var (
	Base        = lipgloss.NewStyle()
	Bold        = Base.Bold(true)
	BoldPrimary = Bold.Foreground(Green)
	Border      = Base.
			Border(lipgloss.NormalBorder(), true).
			BorderForeground(Green)
)
