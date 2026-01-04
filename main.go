package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/ini.v1"
)

var (
	appStyle    = lipgloss.NewStyle().Padding(1, 2)
	bannerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#6fe7d2")).
			Foreground(lipgloss.Color("#000000")).
			PaddingLeft(1).
			PaddingRight(1)
)

const (
	homeView = iota
	profilesView
	regionsView
)

type model struct {
	profiles       list.Model
	regions        list.Model
	currentView    int
	currentARN     string
	currentProfile string
	currentRegion  string
	loading        bool
	spinner        spinner.Model
	err            error
	width          int
	height         int
}

type gotARNAndRegionMsg struct {
	arn    string
	region string
}
type gotProfilesMsg []list.Item
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func newModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Define styles for the list delegate.
	delegate := list.NewDefaultDelegate()

	// Style for normal (unselected) items.
	delegate.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#DDD"}).
		PaddingLeft(2) // Indent to align with selected item's border.

	// Style for selected items.
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Background(lipgloss.Color("#6fe7d2")). // Match banner background.
		Foreground(lipgloss.Color("#000000")). // Match banner foreground.
		PaddingLeft(1).                        // Indent one space.
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#000000")) // Match banner foreground for the border.

	// Since we don't use descriptions, we can clear their styles.
	// This prevents any lingering styling (like a border) from appearing.
	delegate.Styles.NormalDesc = lipgloss.NewStyle()
	delegate.Styles.SelectedDesc = lipgloss.NewStyle()

	l := list.New(nil, delegate, 0, 0)
	l.Title = "Select AWS Profile"

	// Style for the list title.
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("#6fe7d2")).
		Foreground(lipgloss.Color("#000000")).
		Padding(0, 1)

	r := list.New(getRegions(), delegate, 0, 0)
	r.Title = "Select AWS Region"
	r.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("#6fe7d2")).
		Foreground(lipgloss.Color("#000000")).
		Padding(0, 1)

	m := model{
		profiles:    l,
		regions:     r,
		currentView: homeView,
		spinner:     s,
		loading:     true,
	}

	m.currentProfile = os.Getenv("AWS_PROFILE")
	if m.currentProfile == "" {
		m.currentProfile = "default"
	}
	m.currentRegion = "us-east-1"

	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, getProfiles(), getARN(m.currentProfile, m.currentRegion))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := appStyle.GetFrameSize()
		listHeight := m.height - 1 - v
		listWidth := m.width - h
		m.profiles.SetSize(listWidth, listHeight)
		m.regions.SetSize(listWidth, listHeight)

	case gotProfilesMsg:
		m.profiles.SetItems(msg)
	case gotARNAndRegionMsg:
		m.loading = false
		m.currentARN = msg.arn
		m.currentRegion = msg.region
		m.err = nil // Clear any previous error
	case errMsg:
		m.err = msg
		m.loading = false
		return m, nil
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	switch m.currentView {
	case homeView:
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "p":
				m.currentView = profilesView
			case "r":
				m.currentView = regionsView
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		}
	case profilesView:
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "c":
				m.currentView = homeView
			case "enter":
				if i, ok := m.profiles.SelectedItem().(item); ok {
					m.err = nil
					m.currentProfile = string(i)
					m.loading = true
					m.currentView = homeView
					return m, getARN(m.currentProfile, m.currentRegion)
				}
			}
		}
		m.profiles, cmd = m.profiles.Update(msg)
	case regionsView:
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "c":
				m.currentView = homeView
			case "enter":
				if i, ok := m.regions.SelectedItem().(item); ok {
					m.currentRegion = string(i)
					m.loading = true
					m.currentView = homeView
					return m, getARN(m.currentProfile, m.currentRegion)
				}
			}
		}
		m.regions, cmd = m.regions.Update(msg)
	}

	return m, cmd
}

func (m model) View() string {
	var arnDisplay string
	if m.loading {
		arnDisplay = m.spinner.View() + " Fetching..."
	} else if m.err != nil {
		arnDisplay = "Error: " + m.err.Error()
	} else {
		arnDisplay = m.currentARN
	}

	bannerText := fmt.Sprintf("Profile: %s | ARN: %s | Region: %s", m.currentProfile, arnDisplay, m.currentRegion)
	banner := bannerStyle.Copy().Align(lipgloss.Left).Width(m.width).Render(bannerText)

	var mainContent string
	switch m.currentView {
	case profilesView:
		mainContent = appStyle.Render(m.profiles.View())
	case regionsView:
		mainContent = appStyle.Render(m.regions.View())
	default: // homeView
		mainContent = appStyle.Render("Hello World!")
	}

	return lipgloss.JoinVertical(lipgloss.Left, banner, mainContent)
}

func getARN(profile string, region string) tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(profile), config.WithRegion(region))
		if err != nil {
			return errMsg{err}
		}

		client := sts.NewFromConfig(cfg)
		identity, err := client.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
		if err != nil {
			return errMsg{err}
		}
		return gotARNAndRegionMsg{arn: *identity.Arn, region: cfg.Region}
	}
}

type item string

func (i item) Title() string       { return string(i) }
func (i item) Description() string { return "" }
func (i item) FilterValue() string { return string(i) }

func getRegions() []list.Item {
	regions := []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"af-south-1", "ap-east-1", "ap-south-1", "ap-northeast-1",
		"ap-northeast-2", "ap-northeast-3", "ap-southeast-1", "ap-southeast-2",
		"ca-central-1", "eu-central-1", "eu-west-1", "eu-west-2",
		"eu-west-3", "eu-north-1", "eu-south-1", "me-south-1",
		"sa-east-1",
	}
	items := make([]list.Item, len(regions))
	for i, r := range regions {
		items[i] = item(r)
	}
	return items
}

func getProfiles() tea.Cmd {
	return func() tea.Msg {
		usr, err := user.Current()
		if err != nil {
			return errMsg{err}
		}

		configFile := filepath.Join(usr.HomeDir, ".aws", "config")
		cfg, err := ini.Load(configFile)
		if err != nil {
			return errMsg{fmt.Errorf("failed to read AWS config file: %w", err)}
		}

		var profiles []string
		for _, section := range cfg.Sections() {
			if strings.HasPrefix(section.Name(), "profile ") {
				profiles = append(profiles, strings.TrimPrefix(section.Name(), "profile "))
			}
		}
		// Add default profile if it exists
		if _, err := cfg.GetSection("default"); err == nil {
			profiles = append(profiles, "default")
		}

		items := make([]list.Item, len(profiles))
		for i, p := range profiles {
			items[i] = item(p)
		}
		return gotProfilesMsg(items)
	}
}

func main() {
	if os.Getenv("DEBUG") != "" {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}

	p := tea.NewProgram(newModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
