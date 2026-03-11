package overlay

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jacobscunn07/duchess/internal/ui/theme"
)

// awsRegions is the hardcoded list of standard AWS commercial regions.
var awsRegions = []string{
	"us-east-1", "us-east-2", "us-west-1", "us-west-2",
	"ca-central-1", "ca-west-1", "mx-central-1",
	"sa-east-1",
	"eu-west-1", "eu-west-2", "eu-west-3",
	"eu-central-1", "eu-central-2", "eu-north-1", "eu-south-1", "eu-south-2",
	"me-south-1", "me-central-1", "il-central-1",
	"af-south-1",
	"ap-northeast-1", "ap-northeast-2", "ap-northeast-3",
	"ap-southeast-1", "ap-southeast-2", "ap-southeast-3", "ap-southeast-4", "ap-southeast-5",
	"ap-south-1", "ap-south-2",
}

// regionItem implements list.Item for a single AWS region name.
type regionItem struct {
	name string
}

func (i regionItem) FilterValue() string { return i.name }
func (i regionItem) Title() string       { return i.name }
func (i regionItem) Description() string { return "" }

// regionDelegate is a minimal single-line list.ItemDelegate.
// The selected item is rendered bold with color 62 (matches the overlay border);
// unselected items are plain text.
type regionDelegate struct{}

func (d regionDelegate) Height() int                             { return 1 }
func (d regionDelegate) Spacing() int                            { return 0 }
func (d regionDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d regionDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ri, ok := item.(regionItem)
	if !ok {
		return
	}
	if index == m.Index() {
		style := lipgloss.NewStyle().Bold(true).Foreground(theme.DefaultTheme.Accent)
		fmt.Fprint(w, style.Render(ri.name))
	} else {
		fmt.Fprint(w, ri.name)
	}
}

// RegionOverlay is a centered modal that lists AWS regions for selection.
type RegionOverlay struct {
	list        list.Model
	borderStyle lipgloss.Style
	width       int
	height      int
}

// NewRegionOverlay builds a RegionOverlay centred within a terminal of the
// given dimensions. currentRegion is pre-selected in the list.
func NewRegionOverlay(currentRegion string, termWidth, termHeight int) RegionOverlay {
	overlayW := termWidth / 2
	if overlayW < 30 {
		overlayW = 30
	}
	overlayH := len(awsRegions) + 6
	maxH := termHeight * 4 / 5
	if overlayH > maxH {
		overlayH = maxH
	}

	items := make([]list.Item, len(awsRegions))
	selectedIdx := 0
	for i, r := range awsRegions {
		items[i] = regionItem{name: r}
		if r == currentRegion {
			selectedIdx = i
		}
	}

	l := list.New(items, regionDelegate{}, overlayW-4, overlayH-4)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	l.Select(selectedIdx)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.DefaultTheme.Accent).
		Width(overlayW - 2).
		Padding(0, 1)

	return RegionOverlay{
		list:        l,
		borderStyle: borderStyle,
		width:       overlayW,
		height:      overlayH,
	}
}

// Init implements tea.Model.
func (m RegionOverlay) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. Forwards all messages to the inner list.
func (m RegionOverlay) Update(msg tea.Msg) (RegionOverlay, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View implements tea.Model. Returns a bordered overlay with title and list.
func (m RegionOverlay) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.DefaultTheme.TextPrimary).
		Render("Switch Region")
	content := lipgloss.JoinVertical(lipgloss.Left, title, m.list.View())
	return m.borderStyle.Render(content)
}

// SelectedRegion returns the name of the currently highlighted region.
// Returns "" if no item is selected (e.g., empty list).
func (m RegionOverlay) SelectedRegion() string {
	item := m.list.SelectedItem()
	if item == nil {
		return ""
	}
	ri, ok := item.(regionItem)
	if !ok {
		return ""
	}
	return ri.name
}
