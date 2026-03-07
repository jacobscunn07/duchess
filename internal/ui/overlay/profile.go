package overlay

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	ini "gopkg.in/ini.v1"
)

// ListAWSProfiles reads ~/.aws/config and returns a sorted slice of profile names.
// It uses ini.LooseLoad so that a missing config file returns an empty list rather
// than an error. The "default" section is included as-is; all "profile <name>"
// sections are stripped of the "profile " prefix.
func ListAWSProfiles() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".aws", "config")

	cfg, err := ini.LooseLoad(path)
	if err != nil {
		return nil, err
	}

	var profiles []string
	for _, section := range cfg.Sections() {
		name := section.Name()
		if name == "" || name == ini.DefaultSection {
			continue
		}
		if name == "default" {
			profiles = append(profiles, "default")
		} else {
			profiles = append(profiles, strings.TrimPrefix(name, "profile "))
		}
	}

	sort.Strings(profiles)
	return profiles, nil
}

// profileItem implements list.Item for a single AWS profile name.
type profileItem struct {
	name string
}

func (i profileItem) FilterValue() string { return i.name }
func (i profileItem) Title() string       { return i.name }
func (i profileItem) Description() string { return "" }

// profileDelegate is a minimal single-line list.ItemDelegate.
// The selected item is rendered bold with color 33 (matches the project's
// profileStyle in the status bar); unselected items are plain text.
type profileDelegate struct{}

func (d profileDelegate) Height() int                             { return 1 }
func (d profileDelegate) Spacing() int                            { return 0 }
func (d profileDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d profileDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	pi, ok := item.(profileItem)
	if !ok {
		return
	}
	if index == m.Index() {
		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33"))
		fmt.Fprint(w, style.Render(pi.name))
	} else {
		fmt.Fprint(w, pi.name)
	}
}

// ProfileOverlay is a centered modal that lists AWS profiles for selection.
type ProfileOverlay struct {
	list        list.Model
	borderStyle lipgloss.Style
	width       int
	height      int
}

// NewProfileOverlay builds a ProfileOverlay centred within a terminal of the
// given dimensions. currentProfile is pre-selected in the list.
func NewProfileOverlay(currentProfile string, termWidth, termHeight int, profiles []string) ProfileOverlay {
	overlayW := termWidth / 2
	if overlayW < 30 {
		overlayW = 30
	}
	overlayH := len(profiles) + 6
	maxH := termHeight * 4 / 5
	if overlayH > maxH {
		overlayH = maxH
	}

	items := make([]list.Item, len(profiles))
	selectedIdx := 0
	for i, p := range profiles {
		items[i] = profileItem{name: p}
		if p == currentProfile {
			selectedIdx = i
		}
	}

	l := list.New(items, profileDelegate{}, overlayW-4, overlayH-4)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.Select(selectedIdx)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(overlayW - 2).
		Padding(0, 1)

	return ProfileOverlay{
		list:        l,
		borderStyle: borderStyle,
		width:       overlayW,
		height:      overlayH,
	}
}

// Init implements tea.Model.
func (m ProfileOverlay) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. Forwards all messages to the inner list.
func (m ProfileOverlay) Update(msg tea.Msg) (ProfileOverlay, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View implements tea.Model. Returns a bordered overlay with title and list.
func (m ProfileOverlay) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("248")).
		Render("Switch Profile")
	content := lipgloss.JoinVertical(lipgloss.Left, title, m.list.View())
	return m.borderStyle.Render(content)
}

// SelectedProfile returns the name of the currently highlighted profile.
// Returns "" if no item is selected (e.g., empty list).
func (m ProfileOverlay) SelectedProfile() string {
	item := m.list.SelectedItem()
	if item == nil {
		return ""
	}
	pi, ok := item.(profileItem)
	if !ok {
		return ""
	}
	return pi.name
}
