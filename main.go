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
	appStyle = lipgloss.NewStyle().Padding(1, 2)
)

type model struct {
	profiles       list.Model
	currentARN     string
	currentProfile string
	loading        bool
	switching      bool
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

	return model{
		profiles: l,
		spinner:  s,
		loading:  true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, getProfiles, getARN(""))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "x":
			m.switching = !m.switching
			return m, nil
		case "enter":
			if m.switching {
				if i, ok := m.profiles.SelectedItem().(item); ok {
					m.currentProfile = string(i)
					m.switching = false
					m.loading = true
					return m, getARN(m.currentProfile)
				}
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := appStyle.GetFrameSize()
		m.profiles.SetSize(m.width-h, m.height-v)
	case gotProfilesMsg:
		m.profiles.SetItems(msg)
	case gotARNMsg:
		m.loading = false
		m.currentARN = string(msg)
	case errMsg:
		m.err = msg
		m.loading = false
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	if m.switching {
		var cmd tea.Cmd
		m.profiles, cmd = m.profiles.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\nError: %v\n\n", m.err)
	}

	if m.switching {
		return appStyle.Render(m.profiles.View())
	}

	s := "AWS Profile Switcher\n\n"
	if m.loading {
		s += m.spinner.View() + " Fetching Principal ARN..."
	} else {
		s += fmt.Sprintf("Current Profile: %s\n", m.currentProfile)
		s += fmt.Sprintf("ARN: %s\n\n", m.currentARN)
	}

	s += "\nPress 'x' to switch profiles, 'q' to quit."
	return appStyle.Render(s)
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
