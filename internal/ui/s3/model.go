package s3

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

// panelState represents the three navigation states of the S3 panel.
type panelState int

const (
	panelBucketList  panelState = iota
	panelPrefixList             // viewing prefixes/objects within a bucket
	panelObjectDetail           // viewing metadata for a single object
)

// Model is the Bubble Tea child model for the S3 browser panel.
// It manages the three-state S3 navigation: bucket list -> prefix list -> object detail.
type Model struct {
	ctx            context.Context
	awsCfg         aws.Config     // base AWS config (needed for per-bucket regional clients)
	baseClient     *awss3.Client  // global/default-region client for ListBuckets + HeadBucket
	bucketClient   *awss3.Client  // per-bucket regional client (set after bucketClientReadyMsg)
	state          panelState
	list           list.Model
	prefixStack    []string  // navigation path stack; empty = at bucket root
	selectedBucket string
	selectedObj    *s3Item   // set when entering panelObjectDetail
	filterMode     bool
	filterInput    textinput.Model
	filterValue    string    // last applied filter value; "" = no filter
	savedItems     []list.Item // items before filter; restored on Esc
	refreshSpinner spinner.Model
	refreshing     bool // true while 30s background refresh is in-flight
	loading        bool // true during initial or navigation fetch
	err            error
	width          int
	height         int
}

// NewModel constructs a new S3 panel Model and initializes the list, filter input, and spinner.
func NewModel(ctx context.Context, awsCfg aws.Config, width, height int) Model {
	baseClient := awss3.NewFromConfig(awsCfg)

	d := s3Delegate{width: width}
	// Reserve 1 line for breadcrumb; filter bar adjusts dynamically via listHeight()
	listH := height - 1
	if listH < 1 {
		listH = 1
	}
	l := list.New([]list.Item{}, d, width, listH)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(false)  // we handle "/" ourselves with textinput
	l.DisableQuitKeybindings()    // prevent list from consuming q/ctrl+c

	fi := textinput.New()
	fi.Placeholder = "prefix filter"
	fi.Prompt = "Filter: "

	rs := spinner.New(spinner.WithSpinner(spinner.MiniDot))

	return Model{
		ctx:            ctx,
		awsCfg:         awsCfg,
		baseClient:     baseClient,
		state:          panelBucketList,
		list:           l,
		filterInput:    fi,
		refreshSpinner: rs,
		loading:        true,
		width:          width,
		height:         height,
	}
}

// Init implements tea.Model. Fires the initial bucket fetch, starts the refresh ticker,
// and starts the refresh spinner.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		FetchBucketsCmd(m.ctx, m.baseClient),
		S3RefreshTickCmd(),
		m.refreshSpinner.Tick,
	)
}

// Update implements tea.Model. Handles all incoming messages for the S3 panel.
// Key messages are intercepted before forwarding to the bubbles/list component
// to prevent the list from consuming q, Esc, and / for its own purposes.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, m.listHeight())
		m.list.SetDelegate(s3Delegate{width: msg.Width})
		return m, nil

	case tea.KeyMsg:
		// INTERCEPT before list sees it — list consumes q, Esc, / for its own purposes
		switch msg.String() {
		case "q", "ctrl+c":
			// Let rootModel handle quit
			return m, tea.Quit

		case "esc":
			if m.filterMode {
				// Cancel filter; restore previous items.
				m.filterMode = false
				m.filterInput.SetValue("")
				m.filterValue = ""
				if m.savedItems != nil {
					savedIdx := m.list.Index()
					cmd := m.list.SetItems(m.savedItems)
					m.list.Select(clamp(savedIdx, 0, len(m.savedItems)-1))
					m.savedItems = nil
					return m, cmd
				}
				return m, nil
			}
			// ascendLevel returns (Model, tea.Cmd); captures updated model
			m, cmd := m.ascendLevel()
			return m, cmd

		case "enter":
			if m.filterMode {
				// Apply filter: re-fetch with currentPrefix + filterInput.Value()
				m.filterValue = m.filterInput.Value()
				m.filterMode = false
				m.loading = true
				filterPrefix := m.currentPrefix() + m.filterValue
				return m, FetchPrefixCmd(m.ctx, m.bucketClient, m.selectedBucket, filterPrefix)
			}
			// descendIntoSelected returns (Model, tea.Cmd); captures updated model
			m, cmd := m.descendIntoSelected()
			return m, cmd

		case "/":
			if !m.filterMode && m.state == panelPrefixList {
				m.filterMode = true
				m.savedItems = m.list.Items() // save for Esc restore
				m.filterInput.SetValue("")
				m.list.SetSize(m.width, m.listHeight())
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
				m.filterInput.Focus()
				return m, cmd
			}
		}

		// In filter mode: route to textinput (j/k insert literal chars — correct UX)
		if m.filterMode {
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			return m, cmd
		}

	case bucketsLoadedMsg:
		m.loading = false
		m.err = nil
		items := bucketsToItems(msg.buckets)
		savedIdx := m.list.Index()
		cmd := m.list.SetItems(items)
		m.list.Select(clamp(savedIdx, 0, len(items)-1))
		return m, cmd

	case bucketsErrMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case bucketClientReadyMsg:
		// Per-bucket client ready — now construct the regional client and fetch root prefix
		m.bucketClient = awss3.NewFromConfig(m.awsCfg, func(o *awss3.Options) {
			o.Region = msg.region
		})
		m.state = panelPrefixList
		m.loading = true
		return m, FetchPrefixCmd(m.ctx, m.bucketClient, m.selectedBucket, "")

	case bucketClientErrMsg:
		m.loading = false
		m.err = fmt.Errorf("failed to open bucket %q: %w", msg.bucket, msg.err)
		return m, nil

	case prefixesLoadedMsg:
		m.loading = false
		m.refreshing = false
		m.err = nil
		items := prefixesToItems(msg.prefixes, msg.objects)
		savedIdx := m.list.Index()
		cmd := m.list.SetItems(items)
		m.list.Select(clamp(savedIdx, 0, len(items)-1))
		m.list.SetSize(m.width, m.listHeight())
		return m, cmd

	case prefixesErrMsg:
		m.loading = false
		m.refreshing = false
		m.err = msg.err
		return m, nil

	case s3RefreshTickMsg:
		// Refresh current level; schedule next tick
		m.refreshing = true
		var refreshCmd tea.Cmd
		switch m.state {
		case panelBucketList:
			refreshCmd = FetchBucketsCmd(m.ctx, m.baseClient)
		case panelPrefixList:
			if m.bucketClient != nil {
				prefix := m.currentPrefix()
				if m.filterValue != "" {
					prefix = m.currentPrefix() + m.filterValue
				}
				refreshCmd = FetchPrefixCmd(m.ctx, m.bucketClient, m.selectedBucket, prefix)
			}
		case panelObjectDetail:
			// No refresh needed for object detail — metadata is static
			m.refreshing = false
		}
		return m, tea.Batch(refreshCmd, S3RefreshTickCmd())

	case spinner.TickMsg:
		if m.refreshing || m.loading {
			var cmd tea.Cmd
			m.refreshSpinner, cmd = m.refreshSpinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Forward remaining messages to list (handles j/k/g/G/pagination internally)
	if m.state != panelObjectDetail {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View implements tea.Model. Renders the breadcrumb header plus the appropriate
// content for the current panel state.
func (m Model) View() string {
	breadcrumb := m.renderBreadcrumb()

	switch m.state {
	case panelObjectDetail:
		if m.selectedObj != nil {
			return lipgloss.JoinVertical(lipgloss.Left, breadcrumb, m.renderObjectDetail())
		}

	case panelBucketList, panelPrefixList:
		if m.err != nil && !m.loading {
			errView := lipgloss.NewStyle().Width(m.width).Padding(1, 2).Render(m.err.Error())
			return lipgloss.JoinVertical(lipgloss.Left, breadcrumb, errView)
		}
		if m.filterMode {
			filterBar := lipgloss.NewStyle().Width(m.width).PaddingLeft(2).Render(m.filterInput.View())
			return lipgloss.JoinVertical(lipgloss.Left, breadcrumb, m.list.View(), filterBar)
		}
		return lipgloss.JoinVertical(lipgloss.Left, breadcrumb, m.list.View())
	}

	return breadcrumb
}

// renderBreadcrumb builds the breadcrumb header line with optional refresh spinner.
// Format: "S3 > bucket-name > prefix/" — left-truncated with "... >" when path is too long.
func (m Model) renderBreadcrumb() string {
	parts := []string{"S3"}
	if m.selectedBucket != "" {
		parts = append(parts, m.selectedBucket)
	}
	for _, p := range m.prefixStack {
		parts = append(parts, p)
	}

	crumb := strings.Join(parts, " > ")

	// Left-truncate if wider than terminal
	maxWidth := m.width - 8 // leave margin for spinner
	for lipgloss.Width(crumb) > maxWidth && len(parts) > 2 {
		parts = append([]string{"..."}, parts[2:]...)
		crumb = strings.Join(parts, " > ")
	}

	// Append refresh spinner on the right if refreshing or loading
	suffix := ""
	if m.refreshing || m.loading {
		suffix = "  " + m.refreshSpinner.View()
	}

	style := lipgloss.NewStyle().
		Width(m.width).
		Foreground(lipgloss.Color("248")).
		PaddingLeft(1)

	return style.Render(crumb + suffix)
}

// renderObjectDetail renders the object metadata pane shown in panelObjectDetail state.
func (m Model) renderObjectDetail() string {
	if m.selectedObj == nil {
		return ""
	}
	obj := m.selectedObj
	lines := []string{
		fmt.Sprintf("  Key:           %s", obj.fullKey),
	}
	if obj.size != nil {
		lines = append(lines, fmt.Sprintf("  Size:          %s (%d bytes)", humanize.IBytes(uint64(*obj.size)), *obj.size))
	}
	if obj.lastModified != nil {
		lines = append(lines, fmt.Sprintf("  Last Modified: %s", obj.lastModified.Format(time.RFC3339)))
	}
	if obj.storageClass != "" {
		lines = append(lines, fmt.Sprintf("  Storage Class: %s", obj.storageClass))
	}
	if obj.etag != "" {
		lines = append(lines, fmt.Sprintf("  ETag:          %s", obj.etag))
	}
	lines = append(lines, "")
	lines = append(lines, "  Press Esc to return")

	return strings.Join(lines, "\n")
}

// listHeight computes the list height accounting for the breadcrumb line and optional filter bar.
func (m Model) listHeight() int {
	h := m.height - 1 // breadcrumb always takes 1 line
	if m.filterMode {
		h-- // filter bar takes 1 additional line
	}
	if h < 1 {
		h = 1
	}
	return h
}

// currentPrefix returns the current prefix path (joined prefix stack).
func (m Model) currentPrefix() string {
	if len(m.prefixStack) == 0 {
		return ""
	}
	return strings.Join(m.prefixStack, "") // prefixes already have trailing /
}

// ascendLevel handles Esc navigation: pops prefix stack or returns to bucket list.
// Returns (Model, tea.Cmd) — all field mutations on the local copy are returned to caller.
func (m Model) ascendLevel() (Model, tea.Cmd) {
	switch m.state {
	case panelObjectDetail:
		m.state = panelPrefixList
		m.selectedObj = nil
		return m, nil

	case panelPrefixList:
		if len(m.prefixStack) > 0 {
			m.prefixStack = m.prefixStack[:len(m.prefixStack)-1]
			m.loading = true
			if len(m.prefixStack) == 0 {
				// Back to bucket root
				return m, FetchPrefixCmd(m.ctx, m.bucketClient, m.selectedBucket, "")
			}
			return m, FetchPrefixCmd(m.ctx, m.bucketClient, m.selectedBucket, m.currentPrefix())
		}
		// No more prefixes — go back to bucket list
		m.state = panelBucketList
		m.selectedBucket = ""
		m.bucketClient = nil
		m.filterValue = ""
		m.loading = true
		cmd := m.list.SetItems([]list.Item{})
		return m, tea.Batch(cmd, FetchBucketsCmd(m.ctx, m.baseClient))

	case panelBucketList:
		// Already at root — Esc does nothing
		return m, nil
	}
	return m, nil
}

// descendIntoSelected handles Enter: descend into selected bucket/prefix or open object detail.
// Returns (Model, tea.Cmd) — all field mutations on the local copy are returned to caller.
func (m Model) descendIntoSelected() (Model, tea.Cmd) {
	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}
	item, ok := selected.(s3Item)
	if !ok {
		return m, nil
	}

	switch item.kind {
	case kindBucket:
		m.selectedBucket = item.name
		m.prefixStack = nil
		m.bucketClient = nil
		m.filterValue = ""
		m.loading = true
		_ = m.list.SetItems([]list.Item{})
		return m, FetchBucketClientCmd(m.ctx, m.awsCfg, m.baseClient, item.name)

	case kindPrefix:
		// Push the suffix of the prefix (relative to current prefix) onto the stack
		currentPrefix := m.currentPrefix()
		suffix := item.name
		if strings.HasPrefix(item.name, currentPrefix) {
			suffix = item.name[len(currentPrefix):]
		}
		m.prefixStack = append(m.prefixStack, suffix)
		m.loading = true
		m.filterValue = ""
		_ = m.list.SetItems([]list.Item{})
		return m, FetchPrefixCmd(m.ctx, m.bucketClient, m.selectedBucket, m.currentPrefix())

	case kindObject:
		// Show object detail pane
		m.state = panelObjectDetail
		m.selectedObj = &item
		return m, nil
	}
	return m, nil
}

// clamp returns v clamped to the range [lo, hi].
func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
