package ecs

import (
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"

	"github.com/jacobscunn07/duchess/internal/ui/theme"
)

// ecsItemKind distinguishes the three kinds of ECS items in the list.
type ecsItemKind int

const (
	kindCluster ecsItemKind = iota
	kindService
	kindTask
)

// ecsItem implements list.Item. Carries the data needed to render and navigate
// a single row in the ECS browser. Stores the raw SDK types for access in
// descendIntoSelected.
type ecsItem struct {
	kind    ecsItemKind
	left    string // display string for left column
	right   string // display string for right column
	// Navigation fields — only the relevant kind is populated:
	cluster *ecstypes.Cluster // set for kindCluster
	service *ecstypes.Service // set for kindService
	task    *ecstypes.Task    // set for kindTask
}

// FilterValue implements list.Item.
func (i ecsItem) FilterValue() string { return i.left }

// ecsDelegate implements list.ItemDelegate for two-column ECS rows.
type ecsDelegate struct {
	width int
}

var (
	normalStyle   = lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle = lipgloss.NewStyle().PaddingLeft(1).Foreground(theme.DefaultTheme.Accent).Bold(true)
)

func (d ecsDelegate) Height() int                               { return 1 }
func (d ecsDelegate) Spacing() int                              { return 0 }
func (d ecsDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

// Render implements list.ItemDelegate. Renders a two-column row: left=name/identifier,
// right=metadata. Gap between columns is filled with spaces, mirroring s3Delegate.Render.
func (d ecsDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(ecsItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	left := i.left
	right := i.right

	// Truncate very long left values to avoid overflow.
	maxLeft := d.width - lipgloss.Width(right) - 4
	if lipgloss.Width(left) > maxLeft && maxLeft > 3 {
		left = left[:maxLeft-3] + "..."
	}

	gap := d.width - lipgloss.Width(left) - lipgloss.Width(right) - 2 // -2 for left padding
	if gap < 1 {
		gap = 1
	}

	row := left + strings.Repeat(" ", gap) + right

	var style lipgloss.Style
	if isSelected {
		style = selectedStyle
	} else {
		style = normalStyle
	}

	fmt.Fprint(w, style.Render(row))
}

// clusterToItem converts an ECS Cluster to an ecsItem for the list.
// Left column: cluster name. Right column: cluster status (e.g. "ACTIVE").
func clusterToItem(c ecstypes.Cluster) ecsItem {
	name := aws.ToString(c.ClusterName)
	status := aws.ToString(c.Status)
	return ecsItem{
		kind:    kindCluster,
		left:    name,
		right:   status,
		cluster: &c,
	}
}

// serviceToItem converts an ECS Service to an ecsItem.
// Left column: "{name}  [{FARGATE|EC2}]". Right column: "{running}/{desired}" with optional " ({N} pending)".
func serviceToItem(svc ecstypes.Service) ecsItem {
	name := aws.ToString(svc.ServiceName)
	tag := ""
	switch svc.LaunchType {
	case ecstypes.LaunchTypeFargate:
		tag = "[FARGATE]"
	case ecstypes.LaunchTypeEc2:
		tag = "[EC2]"
	}
	left := name
	if tag != "" {
		left = name + "  " + tag
	}

	right := fmt.Sprintf("%d/%d", svc.RunningCount, svc.DesiredCount)
	if svc.PendingCount > 0 {
		right += fmt.Sprintf(" (%d pending)", svc.PendingCount)
	}

	return ecsItem{
		kind:    kindService,
		left:    left,
		right:   right,
		service: &svc,
	}
}

// taskToItem converts an ECS Task to an ecsItem.
// Left column: short task ID (first 8 chars of UUID portion of ARN).
// Right column: "{STATUS}  {relativeTime}" (e.g. "RUNNING  2 hours ago").
func taskToItem(t ecstypes.Task) ecsItem {
	shortID := taskShortDisplay(aws.ToString(t.TaskArn))
	status := aws.ToString(t.LastStatus)

	age := ""
	if t.StartedAt != nil {
		age = humanize.Time(*t.StartedAt)
	}

	right := status
	if age != "" {
		right = status + "  " + age
	}

	return ecsItem{
		kind:  kindTask,
		left:  shortID,
		right: right,
		task:  &t,
	}
}

// clustersToItems converts a slice of ECS Cluster types to a list.Item slice.
func clustersToItems(clusters []ecstypes.Cluster) []list.Item {
	items := make([]list.Item, len(clusters))
	for idx, c := range clusters {
		items[idx] = clusterToItem(c)
	}
	return items
}

// servicesToItems converts a slice of ECS Service types to a list.Item slice.
func servicesToItems(services []ecstypes.Service) []list.Item {
	items := make([]list.Item, len(services))
	for idx, svc := range services {
		items[idx] = serviceToItem(svc)
	}
	return items
}

// tasksToItems converts a slice of ECS Task types to a list.Item slice.
func tasksToItems(tasks []ecstypes.Task) []list.Item {
	items := make([]list.Item, len(tasks))
	for idx, t := range tasks {
		items[idx] = taskToItem(t)
	}
	return items
}
