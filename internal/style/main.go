package style

import "github.com/charmbracelet/lipgloss"

const (
	Green  = lipgloss.Color("#6fe7d2")
	Purple = lipgloss.Color("63")
)

var (
	Base   = lipgloss.NewStyle()
	Bold   = Base.Bold(true)
	Border = Base.
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(Green)
)
