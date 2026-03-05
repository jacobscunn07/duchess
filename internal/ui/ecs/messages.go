package ecs

import ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"

// clustersLoadedMsg is sent when ListClusters + DescribeClusters succeeds.
type clustersLoadedMsg struct {
	clusters []ecstypes.Cluster
}

// clustersErrMsg is sent when cluster fetching fails.
type clustersErrMsg struct {
	err error
}

// servicesLoadedMsg is sent when ListServices + DescribeServices succeeds.
type servicesLoadedMsg struct {
	services []ecstypes.Service
}

// servicesErrMsg is sent when service fetching fails.
type servicesErrMsg struct {
	err error
}

// tasksLoadedMsg is sent when ListTasks + DescribeTasks succeeds.
type tasksLoadedMsg struct {
	tasks []ecstypes.Task
}

// tasksErrMsg is sent when task fetching fails.
type tasksErrMsg struct {
	err error
}

// taskDetailLoadedMsg is sent when FetchTaskDetailCmd succeeds.
// Carries both the task (from DescribeTasks) and the task definition (from
// DescribeTaskDefinition) — the task definition is needed for log configuration
// (ECS-06 CloudWatch log URLs). taskDef may be nil if DescribeTaskDefinition failed.
type taskDetailLoadedMsg struct {
	task    ecstypes.Task
	taskDef *ecstypes.TaskDefinition // nil if DescribeTaskDefinition failed or task has no def
}

// taskDetailErrMsg is sent when FetchTaskDetailCmd fails at the DescribeTasks step.
type taskDetailErrMsg struct {
	err error
}

// ecsClusterRefreshTickMsg is the 10-second cluster-level refresh ticker message.
// Distinct type so the model can guard by current navigation state.
type ecsClusterRefreshTickMsg struct{}

// ecsServiceRefreshTickMsg is the 10-second service-level refresh ticker message.
// Distinct type so the model can guard by current navigation state.
type ecsServiceRefreshTickMsg struct{}

// ecsTaskRefreshTickMsg is the 5-second task-level refresh ticker message.
// Distinct type so the model can guard by current navigation state.
type ecsTaskRefreshTickMsg struct{}
