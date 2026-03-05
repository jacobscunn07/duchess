package ecs

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsecs "github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	tea "github.com/charmbracelet/bubbletea"
)

// ListAllClusters returns all ECS clusters in the current region using pagination.
// It uses the ListClusters paginator to collect all cluster ARNs, then calls
// DescribeClusters in batches of 100 to get full cluster details.
func ListAllClusters(ctx context.Context, client *awsecs.Client) ([]ecstypes.Cluster, error) {
	paginator := awsecs.NewListClustersPaginator(client, &awsecs.ListClustersInput{})
	var arns []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing clusters: %w", err)
		}
		arns = append(arns, page.ClusterArns...)
	}
	if len(arns) == 0 {
		return nil, nil
	}

	// DescribeClusters in batches of 100 (safe batch size).
	const batchSize = 100
	var clusters []ecstypes.Cluster
	for i := 0; i < len(arns); i += batchSize {
		end := i + batchSize
		if end > len(arns) {
			end = len(arns)
		}
		out, err := client.DescribeClusters(ctx, &awsecs.DescribeClustersInput{
			Clusters: arns[i:end],
		})
		if err != nil {
			return nil, fmt.Errorf("describing clusters: %w", err)
		}
		clusters = append(clusters, out.Clusters...)
	}
	return clusters, nil
}

// ListAllServices returns all ECS services for the given cluster ARN using pagination.
// It uses the ListServices paginator to collect all service ARNs, then calls
// DescribeServices in batches of 10 (hard API limit).
func ListAllServices(ctx context.Context, client *awsecs.Client, clusterArn string) ([]ecstypes.Service, error) {
	paginator := awsecs.NewListServicesPaginator(client, &awsecs.ListServicesInput{
		Cluster: aws.String(clusterArn),
	})
	var arns []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing services: %w", err)
		}
		arns = append(arns, page.ServiceArns...)
	}
	if len(arns) == 0 {
		return nil, nil
	}

	// DescribeServices hard limit is 10 per call.
	const batchSize = 10
	var services []ecstypes.Service
	for i := 0; i < len(arns); i += batchSize {
		end := i + batchSize
		if end > len(arns) {
			end = len(arns)
		}
		out, err := client.DescribeServices(ctx, &awsecs.DescribeServicesInput{
			Cluster:  aws.String(clusterArn),
			Services: arns[i:end],
		})
		if err != nil {
			return nil, fmt.Errorf("describing services: %w", err)
		}
		services = append(services, out.Services...)
	}
	return services, nil
}

// ListAndDescribeTasks returns all ECS tasks for the given service within a cluster.
// ListTasks requires the short service name (not the ARN) — extracted from serviceArn
// using strings.LastIndex. DescribeTasks is called in batches of 100 (hard API limit).
func ListAndDescribeTasks(ctx context.Context, client *awsecs.Client, clusterArn, serviceArn string) ([]ecstypes.Task, error) {
	// Extract the short service name from the ARN.
	// Service ARN format: arn:aws:ecs:region:account:service/cluster-name/service-name
	shortName := serviceArn[strings.LastIndex(serviceArn, "/")+1:]

	paginator := awsecs.NewListTasksPaginator(client, &awsecs.ListTasksInput{
		Cluster:     aws.String(clusterArn),
		ServiceName: aws.String(shortName),
	})
	var arns []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing tasks: %w", err)
		}
		arns = append(arns, page.TaskArns...)
	}
	if len(arns) == 0 {
		return nil, nil
	}

	// DescribeTasks hard limit is 100 per call.
	const batchSize = 100
	var tasks []ecstypes.Task
	for i := 0; i < len(arns); i += batchSize {
		end := i + batchSize
		if end > len(arns) {
			end = len(arns)
		}
		out, err := client.DescribeTasks(ctx, &awsecs.DescribeTasksInput{
			Cluster: aws.String(clusterArn),
			Tasks:   arns[i:end],
		})
		if err != nil {
			return nil, fmt.Errorf("describing tasks: %w", err)
		}
		tasks = append(tasks, out.Tasks...)
	}
	return tasks, nil
}

// FetchClustersCmd returns a tea.Cmd that lists all ECS clusters asynchronously.
// On success it returns clustersLoadedMsg; on failure it returns clustersErrMsg.
func FetchClustersCmd(ctx context.Context, client *awsecs.Client) tea.Cmd {
	return func() tea.Msg {
		clusters, err := ListAllClusters(ctx, client)
		if err != nil {
			return clustersErrMsg{err: err}
		}
		return clustersLoadedMsg{clusters: clusters}
	}
}

// FetchServicesCmd returns a tea.Cmd that lists all ECS services for the given
// cluster asynchronously. On success it returns servicesLoadedMsg; on failure
// it returns servicesErrMsg.
func FetchServicesCmd(ctx context.Context, client *awsecs.Client, clusterArn string) tea.Cmd {
	return func() tea.Msg {
		services, err := ListAllServices(ctx, client, clusterArn)
		if err != nil {
			return servicesErrMsg{err: err}
		}
		return servicesLoadedMsg{services: services}
	}
}

// FetchTasksCmd returns a tea.Cmd that lists all ECS tasks for the given service
// within a cluster asynchronously. On success it returns tasksLoadedMsg; on
// failure it returns tasksErrMsg.
func FetchTasksCmd(ctx context.Context, client *awsecs.Client, clusterArn, serviceArn string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := ListAndDescribeTasks(ctx, client, clusterArn, serviceArn)
		if err != nil {
			return tasksErrMsg{err: err}
		}
		return tasksLoadedMsg{tasks: tasks}
	}
}

// FetchTaskDetailCmd returns a tea.Cmd that fetches the full detail for a single
// ECS task, including its task definition (for log configuration). It calls
// DescribeTasks to get the task, then DescribeTaskDefinition for log config.
// If DescribeTaskDefinition fails, taskDef is nil and the task detail is still
// returned (non-fatal — log config will simply be unavailable).
func FetchTaskDetailCmd(ctx context.Context, client *awsecs.Client, clusterArn, taskArn string) tea.Cmd {
	return func() tea.Msg {
		// Step 1: Describe the single task.
		out, err := client.DescribeTasks(ctx, &awsecs.DescribeTasksInput{
			Cluster: aws.String(clusterArn),
			Tasks:   []string{taskArn},
		})
		if err != nil {
			return taskDetailErrMsg{err: fmt.Errorf("describing task: %w", err)}
		}
		if len(out.Tasks) == 0 {
			return taskDetailErrMsg{err: fmt.Errorf("task %s not found", taskArn)}
		}
		task := out.Tasks[0]

		// Step 2: Describe the task definition for log configuration.
		// Non-fatal if this fails — taskDef will be nil.
		var taskDef *ecstypes.TaskDefinition
		if task.TaskDefinitionArn != nil {
			defOut, defErr := client.DescribeTaskDefinition(ctx, &awsecs.DescribeTaskDefinitionInput{
				TaskDefinition: task.TaskDefinitionArn,
			})
			if defErr == nil && defOut != nil {
				taskDef = defOut.TaskDefinition
			}
		}

		return taskDetailLoadedMsg{task: task, taskDef: taskDef}
	}
}

// ECSClusterRefreshTickCmd returns a tea.Cmd that fires ecsClusterRefreshTickMsg
// after 10 seconds. Used to refresh cluster list data.
func ECSClusterRefreshTickCmd() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return ecsClusterRefreshTickMsg{}
	})
}

// ECSServiceRefreshTickCmd returns a tea.Cmd that fires ecsServiceRefreshTickMsg
// after 10 seconds. Used to refresh service list data.
func ECSServiceRefreshTickCmd() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return ecsServiceRefreshTickMsg{}
	})
}

// ECSTaskRefreshTickCmd returns a tea.Cmd that fires ecsTaskRefreshTickMsg
// after 5 seconds. Used to refresh task list data (faster than cluster/service
// because task status changes more frequently).
func ECSTaskRefreshTickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return ecsTaskRefreshTickMsg{}
	})
}

// taskShortID extracts the UUID portion from a task ARN.
// Task ARN formats:
//   - arn:aws:ecs:region:account:task/cluster-name/uuid  (new format)
//   - arn:aws:ecs:region:account:task/uuid               (old format)
func taskShortID(taskArn string) string {
	idx := strings.LastIndex(taskArn, "/")
	if idx < 0 || idx >= len(taskArn)-1 {
		return taskArn
	}
	return taskArn[idx+1:]
}

// taskShortDisplay returns the first 8 chars of the task short UUID,
// mirroring the git short SHA convention for compact display.
func taskShortDisplay(taskArn string) string {
	id := taskShortID(taskArn)
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// buildCloudWatchURL constructs the CloudWatch Logs console deep-link URL for
// a specific log group and log stream.
//
// CloudWatch deep-links use a non-standard encoding where "/" becomes "$252F":
//   - url.PathEscape converts "/" to "%2F"
//   - Replace "%" with "$25" converts "%2F" to "$252F"
func buildCloudWatchURL(region, logGroup, logStream string) string {
	cwEncode := func(s string) string {
		escaped := url.PathEscape(s)
		return strings.ReplaceAll(escaped, "%", "$25")
	}
	return fmt.Sprintf(
		"https://%s.console.aws.amazon.com/cloudwatch/home?region=%s#logsV2:log-groups/log-group/%s/log-events/%s",
		region, region,
		cwEncode(logGroup),
		cwEncode(logStream),
	)
}

// openURL opens a URL in the system default browser.
// Uses "open" on macOS (darwin) and "xdg-open" on Linux.
func openURL(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}

// logURLForContainer builds the CloudWatch Logs URL for a specific container
// within a task. logConfig must come from the ContainerDefinition (task definition),
// NOT the Container struct from DescribeTasks — log configuration lives on the
// definition, not the running instance.
//
// Log stream format: {prefix}/{containerName}/{taskShortID}
//
// Returns ("", errMsg) if no CloudWatch Logs driver is configured.
func logURLForContainer(region, taskArn, containerName string, logConfig *ecstypes.LogConfiguration) (string, string) {
	if logConfig == nil || logConfig.LogDriver != ecstypes.LogDriverAwslogs {
		return "", "No CloudWatch logs configured for this container"
	}
	logGroup := logConfig.Options["awslogs-group"]
	prefix := logConfig.Options["awslogs-stream-prefix"]
	if logGroup == "" {
		return "", "No CloudWatch logs configured for this container"
	}
	shortID := taskShortID(taskArn)
	logStream := prefix + "/" + containerName + "/" + shortID
	return buildCloudWatchURL(region, logGroup, logStream), ""
}
