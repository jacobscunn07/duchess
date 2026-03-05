package ecs

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsecs "github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

// panelState represents the four navigation states of the ECS panel.
type panelState int

const (
	panelClusterList panelState = iota
	panelServiceList
	panelTaskList
	panelTaskDetail
)

// Model is the Bubble Tea child model for the ECS browser panel.
// It manages a four-state ECS navigation: cluster list -> service list -> task list -> task detail.
type Model struct {
	ctx             context.Context
	awsCfg          aws.Config
	client          *awsecs.Client
	region          string                   // stored from awsCfg for CloudWatch URL building
	state           panelState
	list            list.Model
	selectedCluster string                   // cluster ARN
	selectedService *ecsItem                 // set when entering panelTaskList; stores full ecsItem so TaskDefinition ARN is accessible for breadcrumb
	selectedTask    *ecsItem                 // set when entering panelTaskDetail
	taskDef         *ecstypes.TaskDefinition // task definition for log config (ECS-06)
	containerCursor int                      // cursor within container list in TaskDetail pane
	logStatusMsg    string                   // inline message for L key result ("No CW logs" etc.)
	refreshSpinner  spinner.Model
	refreshing      bool
	loading         bool
	err             error
	width           int
	height          int
}

// NewModel constructs a new ECS panel Model and initializes the list and spinner.
// Mirrors s3.NewModel exactly in value-receiver pattern and initialization style.
func NewModel(ctx context.Context, awsCfg aws.Config, width, height int) Model {
	client := awsecs.NewFromConfig(awsCfg)
	region := awsCfg.Region

	d := ecsDelegate{width: width}
	listH := height - 1 // reserve 1 line for breadcrumb
	if listH < 1 {
		listH = 1
	}
	l := list.New([]list.Item{}, d, width, listH)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(false)  // no filter for ECS (per locked decision)
	l.DisableQuitKeybindings()    // prevent list from consuming q/ctrl+c

	rs := spinner.New(spinner.WithSpinner(spinner.MiniDot))

	return Model{
		ctx:            ctx,
		awsCfg:         awsCfg,
		client:         client,
		region:         region,
		state:          panelClusterList,
		list:           l,
		refreshSpinner: rs,
		loading:        true,
		width:          width,
		height:         height,
	}
}

// Init implements tea.Model. Fires the initial cluster fetch, starts the cluster
// refresh ticker, and starts the refresh spinner.
// Only cluster ticker is started here; service and task tickers start on descent
// to avoid running all 3 tickers unconditionally.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		FetchClustersCmd(m.ctx, m.client),
		ECSClusterRefreshTickCmd(),
		m.refreshSpinner.Tick,
	)
}

// Update implements tea.Model. Handles all incoming messages for the ECS panel.
// Key messages are intercepted before forwarding to the bubbles/list component.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, m.listHeight())
		m.list.SetDelegate(ecsDelegate{width: msg.Width})
		return m, nil

	case tea.KeyMsg:
		// INTERCEPT before list sees it
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "esc":
			m, cmd := m.ascendLevel()
			return m, cmd

		case "enter":
			m, cmd := m.descendIntoSelected()
			return m, cmd

		case "j":
			if m.state == panelTaskDetail {
				containers := m.taskContainers()
				if m.containerCursor < len(containers)-1 {
					m.containerCursor++
					m.logStatusMsg = ""
				}
				return m, nil
			}

		case "k":
			if m.state == panelTaskDetail {
				if m.containerCursor > 0 {
					m.containerCursor--
					m.logStatusMsg = ""
				}
				return m, nil
			}

		case "L":
			if m.state == panelTaskDetail && m.selectedTask != nil && m.taskDef != nil {
				containers := m.taskContainers()
				if len(containers) == 0 {
					m.logStatusMsg = "No containers available"
					return m, nil
				}
				if m.containerCursor >= len(containers) {
					m.containerCursor = 0
				}
				container := containers[m.containerCursor]
				containerName := aws.ToString(container.Name)
				taskArn := aws.ToString(m.selectedTask.task.TaskArn)

				// Find matching ContainerDefinition in task definition for log config
				var logConfig *ecstypes.LogConfiguration
				for _, cd := range m.taskDef.ContainerDefinitions {
					if aws.ToString(cd.Name) == containerName {
						logConfig = cd.LogConfiguration
						break
					}
				}

				cwURL, errMsg := logURLForContainer(m.region, taskArn, containerName, logConfig)
				if errMsg != "" {
					m.logStatusMsg = errMsg
					return m, nil
				}
				if err := openURL(cwURL); err != nil {
					m.logStatusMsg = "Failed to open browser: " + err.Error()
					return m, nil
				}
				m.logStatusMsg = "Opened in browser"
				return m, nil
			}
		}

	case clustersLoadedMsg:
		m.loading = false
		m.refreshing = false
		m.err = nil
		items := clustersToItems(msg.clusters)
		savedIdx := m.list.Index()
		cmd := m.list.SetItems(items)
		m.list.Select(clamp(savedIdx, 0, len(items)-1))
		return m, cmd

	case clustersErrMsg:
		m.loading = false
		m.refreshing = false
		m.err = msg.err
		return m, nil

	case servicesLoadedMsg:
		m.loading = false
		m.refreshing = false
		m.err = nil
		items := servicesToItems(msg.services)
		savedIdx := m.list.Index()
		cmd := m.list.SetItems(items)
		m.list.Select(clamp(savedIdx, 0, len(items)-1))
		return m, cmd

	case servicesErrMsg:
		m.loading = false
		m.refreshing = false
		m.err = msg.err
		return m, nil

	case tasksLoadedMsg:
		m.loading = false
		m.refreshing = false
		m.err = nil
		items := tasksToItems(msg.tasks)
		savedIdx := m.list.Index()
		cmd := m.list.SetItems(items)
		m.list.Select(clamp(savedIdx, 0, len(items)-1))
		return m, cmd

	case tasksErrMsg:
		m.loading = false
		m.refreshing = false
		m.err = msg.err
		return m, nil

	case taskDetailLoadedMsg:
		// Update the selected task with the refreshed data from the response
		if m.selectedTask != nil {
			m.selectedTask.task = &msg.task
		}
		m.taskDef = msg.taskDef
		m.loading = false
		m.containerCursor = 0
		return m, nil

	case taskDetailErrMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case ecsClusterRefreshTickMsg:
		m.refreshing = true
		var cmd tea.Cmd
		if m.state == panelClusterList {
			cmd = FetchClustersCmd(m.ctx, m.client)
		} else {
			m.refreshing = false
		}
		return m, tea.Batch(cmd, ECSClusterRefreshTickCmd())

	case ecsServiceRefreshTickMsg:
		if m.state == panelServiceList && m.selectedCluster != "" {
			m.refreshing = true
			return m, tea.Batch(FetchServicesCmd(m.ctx, m.client, m.selectedCluster), ECSServiceRefreshTickCmd())
		}
		return m, ECSServiceRefreshTickCmd()

	case ecsTaskRefreshTickMsg:
		if m.state == panelTaskList && m.selectedCluster != "" && m.selectedService != nil {
			serviceArn := aws.ToString(m.selectedService.service.ServiceArn)
			m.refreshing = true
			return m, tea.Batch(FetchTasksCmd(m.ctx, m.client, m.selectedCluster, serviceArn), ECSTaskRefreshTickCmd())
		}
		return m, ECSTaskRefreshTickCmd()

	case spinner.TickMsg:
		if m.refreshing || m.loading {
			var cmd tea.Cmd
			m.refreshSpinner, cmd = m.refreshSpinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Forward remaining messages to list when not in task detail pane
	if m.state != panelTaskDetail {
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
	case panelTaskDetail:
		return lipgloss.JoinVertical(lipgloss.Left, breadcrumb, m.renderTaskDetail())
	default: // panelClusterList, panelServiceList, panelTaskList
		if m.err != nil && !m.loading {
			errView := lipgloss.NewStyle().Width(m.width).Padding(1, 2).Render(m.err.Error())
			return lipgloss.JoinVertical(lipgloss.Left, breadcrumb, errView)
		}
		return lipgloss.JoinVertical(lipgloss.Left, breadcrumb, m.list.View())
	}
}

// renderBreadcrumb builds the breadcrumb header line with optional refresh spinner.
// Format: "ECS > cluster-name > service-name > task-shortid"
// When state == panelTaskList, appends the task definition short name from the
// selected service's TaskDefinition ARN (satisfies ECS-03).
func (m Model) renderBreadcrumb() string {
	parts := []string{"ECS"}

	if m.selectedCluster != "" {
		// Extract short cluster name: last path segment of the ARN
		idx := strings.LastIndex(m.selectedCluster, "/")
		clusterShort := m.selectedCluster
		if idx >= 0 && idx < len(m.selectedCluster)-1 {
			clusterShort = m.selectedCluster[idx+1:]
		}
		parts = append(parts, clusterShort)
	}

	if m.selectedService != nil && m.selectedService.service != nil {
		svcName := aws.ToString(m.selectedService.service.ServiceName)
		if svcName != "" {
			parts = append(parts, svcName)
		}
	}

	if m.state == panelTaskDetail && m.selectedTask != nil && m.selectedTask.task != nil {
		taskArn := aws.ToString(m.selectedTask.task.TaskArn)
		parts = append(parts, taskShortDisplay(taskArn))
	}

	crumb := strings.Join(parts, " > ")

	// When in task list, append task definition short name from selected service's TaskDefinition ARN
	taskDefLabel := ""
	if m.state == panelTaskList && m.selectedService != nil && m.selectedService.service != nil {
		taskDefArn := aws.ToString(m.selectedService.service.TaskDefinition)
		if taskDefArn != "" {
			idx := strings.LastIndex(taskDefArn, "/")
			taskDefShort := taskDefArn
			if idx >= 0 && idx < len(taskDefArn)-1 {
				taskDefShort = taskDefArn[idx+1:]
			}
			taskDefLabel = "  task def: " + taskDefShort
		}
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

	if taskDefLabel != "" {
		taskDefStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		return style.Render(crumb + taskDefStyle.Render(taskDefLabel) + suffix)
	}

	return style.Render(crumb + suffix)
}

// renderTaskDetail renders the task detail pane shown in panelTaskDetail state.
func (m Model) renderTaskDetail() string {
	if m.selectedTask == nil || m.selectedTask.task == nil {
		return ""
	}

	task := m.selectedTask.task

	// If task definition is still loading, show spinner
	if m.taskDef == nil && m.loading {
		return lipgloss.NewStyle().Padding(1, 2).Render(m.refreshSpinner.View() + " Loading task definition...")
	}

	// Task metadata section
	taskID := taskShortDisplay(aws.ToString(task.TaskArn))
	// Use full UUID portion (not truncated to 8 chars) for the detail view
	fullUUID := aws.ToString(task.TaskArn)
	if idx := strings.LastIndex(fullUUID, "/"); idx >= 0 && idx < len(fullUUID)-1 {
		fullUUID = fullUUID[idx+1:]
	}

	status := aws.ToString(task.LastStatus)
	startedAt := "unknown"
	if task.StartedAt != nil {
		startedAt = humanize.Time(*task.StartedAt)
	}

	lines := []string{
		fmt.Sprintf("  Task ID:    %s", fullUUID),
		fmt.Sprintf("  Status:     %s", status),
		fmt.Sprintf("  Started:    %s", startedAt),
		"",
		"  Containers  (j/k to navigate, L to open logs)",
		"  " + strings.Repeat("─", m.width-4),
	}

	containers := m.taskContainers()
	if len(containers) == 0 {
		lines = append(lines, "  (no containers)")
	} else {
		for i, c := range containers {
			name := aws.ToString(c.Name)
			image := aws.ToString(c.Image)
			cStatus := aws.ToString(c.LastStatus)

			// Truncate image to keep row manageable
			maxImageLen := m.width - len(name) - len(cStatus) - 10
			if maxImageLen > 0 && len(image) > maxImageLen {
				image = "..." + image[len(image)-maxImageLen+3:]
			}

			row := fmt.Sprintf("%s  %s  %s", name, image, cStatus)

			cursor := "  "
			if i == m.containerCursor {
				cursor = "> "
				highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
				lines = append(lines, cursor+highlightStyle.Render(row))
			} else {
				lines = append(lines, cursor+row)
			}
		}
	}

	// Show log status message if set
	if m.logStatusMsg != "" {
		lines = append(lines, "")
		lines = append(lines, "  "+m.logStatusMsg)
	}

	_ = taskID // taskID is the short form, fullUUID is used in detail view
	return strings.Join(lines, "\n")
}

// taskContainers returns the containers for the selected task, or nil if none.
func (m Model) taskContainers() []ecstypes.Container {
	if m.selectedTask == nil || m.selectedTask.task == nil {
		return nil
	}
	return m.selectedTask.task.Containers
}

// ascendLevel handles Esc navigation: moves up one level in the ECS hierarchy.
// Returns (Model, tea.Cmd) — all field mutations on the local copy are returned to caller.
func (m Model) ascendLevel() (Model, tea.Cmd) {
	switch m.state {
	case panelTaskDetail:
		// Return to task list — clear task detail state
		m.state = panelTaskList
		m.selectedTask = nil
		m.taskDef = nil
		m.containerCursor = 0
		m.logStatusMsg = ""
		return m, nil

	case panelTaskList:
		// Return to service list — re-fetch services
		m.state = panelServiceList
		m.selectedService = nil
		m.loading = true
		cmd := m.list.SetItems([]list.Item{})
		return m, tea.Batch(cmd, FetchServicesCmd(m.ctx, m.client, m.selectedCluster))

	case panelServiceList:
		// Return to cluster list — clear cluster selection and re-fetch clusters
		m.state = panelClusterList
		m.selectedCluster = ""
		m.loading = true
		cmd := m.list.SetItems([]list.Item{})
		return m, tea.Batch(cmd, FetchClustersCmd(m.ctx, m.client))

	case panelClusterList:
		// Already at root — Esc does nothing
		return m, nil
	}
	return m, nil
}

// descendIntoSelected handles Enter: descend into the selected ECS item.
// Returns (Model, tea.Cmd) — all field mutations on the local copy are returned to caller.
func (m Model) descendIntoSelected() (Model, tea.Cmd) {
	if m.state == panelTaskDetail {
		// Leaf — no further descent
		return m, nil
	}

	selected := m.list.SelectedItem()
	if selected == nil {
		return m, nil
	}
	item, ok := selected.(ecsItem)
	if !ok {
		return m, nil
	}

	switch m.state {
	case panelClusterList:
		if item.cluster == nil {
			return m, nil
		}
		m.selectedCluster = aws.ToString(item.cluster.ClusterArn)
		m.state = panelServiceList
		m.loading = true
		cmd := m.list.SetItems([]list.Item{})
		return m, tea.Batch(
			cmd,
			FetchServicesCmd(m.ctx, m.client, m.selectedCluster),
			ECSServiceRefreshTickCmd(),
		)

	case panelServiceList:
		if item.service == nil {
			return m, nil
		}
		itemCopy := item
		m.selectedService = &itemCopy
		serviceArn := aws.ToString(item.service.ServiceArn)
		m.state = panelTaskList
		m.loading = true
		cmd := m.list.SetItems([]list.Item{})
		return m, tea.Batch(
			cmd,
			FetchTasksCmd(m.ctx, m.client, m.selectedCluster, serviceArn),
			ECSTaskRefreshTickCmd(),
		)

	case panelTaskList:
		if item.task == nil {
			return m, nil
		}
		itemCopy := item
		m.selectedTask = &itemCopy
		taskArn := aws.ToString(item.task.TaskArn)
		m.state = panelTaskDetail
		m.loading = true
		m.containerCursor = 0
		return m, FetchTaskDetailCmd(m.ctx, m.client, m.selectedCluster, taskArn)
	}

	return m, nil
}

// listHeight computes the list height accounting for the breadcrumb line.
func (m Model) listHeight() int {
	h := m.height - 1 // breadcrumb always takes 1 line
	if h < 1 {
		h = 1
	}
	return h
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
