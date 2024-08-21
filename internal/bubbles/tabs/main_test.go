package tabs

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModelView(t *testing.T) {
	// Arrange

	tabs := NewModel()
	// tabs.AddTab()
	// Act
	// Assert
	fmt.Println(tabs)
}

type TestTab struct {
	title string
}

func (t TestTab) Title() string {
	return t.title
}

func (t TestTab) Init() tea.Cmd {
	return nil
}

// func (t TestTab) Update

// tea.Model
// Title() string
