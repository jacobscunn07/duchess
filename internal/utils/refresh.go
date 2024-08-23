package utils

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func RefreshCommand() tea.Cmd {
	return tea.Every(2*time.Second, func(t time.Time) tea.Msg {
		return RefreshCommandMessage{
			Time: t,
		}
	})
}

type RefreshCommandMessage struct {
	Time time.Time
}
