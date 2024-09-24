package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jacobscunn07/duchess/internal/components/layout"
)

func main() {
	duchess := layout.NewApp()

	if _, err := tea.NewProgram(duchess, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
