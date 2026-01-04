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

type model struct {
	profiles       list.Model
	currentARN     string
	currentProfile string
	loading        bool
	spinner        spinner.Model
	err            error
	width          int
	height         int
}

type gotARNMsg string
type gotProfilesMsg []list.Item
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func newModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select AWS Profile"

	m := model{
		profiles: l,
		spinner:  s,
		loading:  true,
	}

	m.currentProfile = os.Getenv("AWS_PROFILE")
	if m.currentProfile == "" {
		m.currentProfile = "default"
	}

	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, getProfiles, getARN(m.currentProfile))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Error is now cleared only on 'enter'
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if i, ok := m.profiles.SelectedItem().(item); ok {
				m.err = nil // Clear previous error on new selection
				m.currentProfile = string(i)
				m.loading = true
				return m, getARN(m.currentProfile)
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := appStyle.GetFrameSize()
		
		// The banner is assumed to take 1 line of vertical space.
		// appStyle has 1 unit of padding top and 1 unit of padding bottom, so v is 2.
		// So total vertical space consumed by banner + appStyle padding is 1 + v.
		// The remaining height is for the profiles list.
		m.profiles.SetSize(m.width-h, m.height - 1 - v) // Subtract 1 for banner height
	case gotProfilesMsg:
		m.profiles.SetItems(msg)
	case gotARNMsg:
		m.loading = false
		m.currentARN = string(msg)
		m.err = nil // Clear any previous error
	case errMsg:
		m.err = msg
		m.loading = false
		return m, nil
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	m.profiles, cmd = m.profiles.Update(msg)
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

	banner := bannerStyle.Render(fmt.Sprintf("Profile: %s | ARN: %s", m.currentProfile, arnDisplay))

	mainContent := appStyle.Render(m.profiles.View())

	return lipgloss.JoinVertical(lipgloss.Left, banner, mainContent)
}

func getARN(profile string) tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(profile))
		if err != nil {
			return errMsg{err}
		}

		client := sts.NewFromConfig(cfg)
		identity, err := client.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
		if err != nil {
			return errMsg{err}
		}
		return gotARNMsg(*identity.Arn)
	}
}

type item string

func (i item) Title() string       { return string(i) }
func (i item) Description() string { return "" }
func (i item) FilterValue() string { return string(i) }

func getProfiles() tea.Msg {
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
