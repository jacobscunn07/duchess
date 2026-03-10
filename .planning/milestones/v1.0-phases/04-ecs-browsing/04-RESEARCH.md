# Phase 4: ECS Browsing - Research

**Researched:** 2026-03-04
**Domain:** AWS ECS SDK v2, Bubble Tea TUI, ECS hierarchy navigation
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Navigation depth (4 states)
`ClusterList → ServiceList → TaskList → TaskDetail pane`

State machine constants:
```go
panelClusterList panelState = iota
panelServiceList
panelTaskList
panelTaskDetail
```

#### Row layout
Same `bubbles/list` + custom delegate pattern as S3. No `bubbles/table`.

Custom delegate type: `ecsDelegate` with item kinds:
```go
kindCluster  ecsItemKind = iota
kindService
kindTask
```

Filter key (`/`) is NOT wired — no filter for ECS in Phase 4.

#### Service row format
```
my-api-service  [FARGATE]       3/3
long-svc-name   [EC2]           1/2 (1 pending)
```

#### Task row format
```
a1b2c3d4  RUNNING   2h ago
e5f6a7b8  STOPPED   1d ago
```
Task IDs are 32-char UUIDs; display first 8 chars.
Status: RUNNING, STOPPED, PENDING.
Started-at: relative time via `go-humanize`.

#### TaskDetail pane (container list)
Task metadata at top + navigable container list below. `L` key on highlighted container opens CloudWatch Logs.

Layout:
```
ECS > my-cluster > my-service > task a1b2c3d4

  Task ID:    a1b2c3d4-full-uuid-here
  Status:     RUNNING
  Started:    2h ago

  Containers  (j/k to navigate, L to open logs)
  ─────────────────────────────────────────────
> my-app    nginx:1.25   RUNNING
  sidecar   envoy:1.28   RUNNING
```

- Containers are a navigable list within the pane (separate `containerCursor int`, not bubbles/list)
- `L` key opens CloudWatch Logs URL for highlighted container
- `Esc` returns to TaskList
- Container rows: name on left, image + status on right
- Divider line rendered with `strings.Repeat("─", width-4)` or equivalent

#### CloudWatch Logs URL construction
Construct awslogs URL from `DescribeTaskDefinition` container definition `logConfiguration`. Open via `os/exec` with `open` (macOS) or `xdg-open` (Linux).

URL pattern:
```
https://{region}.console.aws.amazon.com/cloudwatch/home?region={region}#logsV2:log-groups/log-group/{logGroup}/log-events/{logStream}
```

- Log group: from `logConfiguration.options["awslogs-group"]`
- Log stream: `{prefix}/{container-name}/{task-id}` where task-id is the SHORT task ID (last segment of task ARN after final `/`)
- URL-encode the log group and stream path components (replace `/` with `$252F` for CloudWatch deep-link format)
- If container has no `awslogs` log driver: show inline message `"No CloudWatch logs configured for this container"`
- If log stream not yet available: show inline message `"Log stream not yet available"`
- `open`/`xdg-open` invocation: `exec.Command("open", url)` on darwin, `exec.Command("xdg-open", url)` on linux — detect via `runtime.GOOS`

#### Refresh intervals
```go
ecsClusterRefreshTickMsg  // 10s
ecsServiceRefreshTickMsg  // 10s
ecsTaskRefreshTickMsg     // 5s
```
Mirrors `s3RefreshTickMsg` pattern. Self-scheduling (each tick schedules the next).
Refresh only re-fetches the current visible level.

#### Filter
No filter for ECS in Phase 4. `/` key is NOT wired.

#### Error handling
Same inline neutral-color pattern as S3 and Phase 2.
- `ClassifyCredentialError` from `internal/aws/session.go` reused for ECS API errors
- Per-level errors don't clear other levels

### Claude's Discretion

- Exact lipgloss styling for TaskDetail pane (container list highlight color, divider line style)
- CloudWatch Logs URL encoding details (exact encoding of `/` in log group names)
- Short-circuit behavior when `DescribeTasks` returns no `logConfiguration` or nil containers
- Exact TaskDetail pane layout (padding between metadata section and container list)
- Refresh ticker start ordering (whether to batch all three tickers in Init or start lazily per-level)
- Pagination chunk size for `ListTasks` (ECS max is 100 per page — paginate fully like S3)
- Whether to use `DescribeServices` for per-service detail or defer to TaskList header display

### Deferred Ideas (OUT OF SCOPE)

- **ECS filter** — `/` key for cluster/service/task filtering; not in Phase 4 requirements
- **Task log streaming** — inline log tailing within the TUI (vs browser redirect)
- **DescribeServices detail pane** — service-level detail pane
- **EC2 launch type instance details**
- **ECS Exec integration**
- **Multi-region ECS browsing** — Phase 5 concern
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| ECS-01 | User can view a list of ECS clusters in the current region | `ListClusters` + `ListClustersPaginator` (max 100/page); returns cluster ARNs; need `DescribeClusters` for name/status |
| ECS-02 | User can navigate into a cluster to view its services | `ListServices` + `ListServicesPaginator` (max 100/page); `ServiceName` field from `DescribeServices` |
| ECS-03 | User can view service details (desired count, running count, pending count, launch type, task definition) | `DescribeServices` returns `Service` struct with `DesiredCount`, `RunningCount`, `PendingCount`, `LaunchType`, `TaskDefinition` fields |
| ECS-04 | User can navigate into a service to view its running tasks | `ListTasks` with `ServiceName` filter; `ListTasksPaginator`; then `DescribeTasks` to get status/startedAt |
| ECS-05 | User can view task details (task ID, status, started at, containers with name, image, and status) | `DescribeTasks` returns `Task.Containers []Container`; each `Container` has `Name`, `Image`, `LastStatus`; task has `TaskArn`, `LastStatus`, `StartedAt` |
| ECS-06 | User can open the CloudWatch Logs console URL for a selected task's containers in their browser (L key) | `LogConfiguration` lives on `ContainerDefinition` (task definition), NOT on `Container` (task); requires `DescribeTaskDefinition(task.TaskDefinitionArn)` then match container by name |
</phase_requirements>

## Summary

Phase 4 adds a full ECS navigation hierarchy mirroring the S3 panel architecture: a four-state panel (`ClusterList → ServiceList → TaskList → TaskDetail`) with `bubbles/list` + custom delegate, per-level refresh tickers, and a leaf TaskDetail pane with an embedded navigable container list. The implementation is structurally identical to the S3 panel — same value-receiver `Model`, same async `tea.Cmd` constructors in `client.go`, same typed message structs in `messages.go`, same breadcrumb rendering pattern — differing mainly in the API surface, the four-level (vs. three-level) hierarchy, and the TaskDetail pane substituting for S3's ObjectDetail pane.

The most important discovery: **`LogConfiguration` does NOT exist on the `Container` struct returned by `DescribeTasks`.** It lives on `ContainerDefinition` within the task definition. To get log configuration for ECS-06, the implementation must call `DescribeTaskDefinition(task.TaskDefinitionArn)` and match container definitions by name. This adds a `FetchTaskDefinitionCmd` to `client.go` and a `taskDefinitionLoadedMsg` to `messages.go`. The CONTEXT.md describes this correctly (it says "from `logConfiguration.options["awslogs-group"]`") but does not call out the extra API call explicitly — planners must wire it.

The ECS SDK (`github.com/aws/aws-sdk-go-v2/service/ecs v1.73.1`) is already in `go.mod` as `// indirect`. The first plan must promote it to a direct dependency with `go get github.com/aws/aws-sdk-go-v2/service/ecs`.

**Primary recommendation:** Mirror the S3 panel architecture exactly, add `DescribeTaskDefinition` for ECS-06 log URLs, and integrate the `ecsPanel` into `rootModel` alongside `s3Panel`.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/aws/aws-sdk-go-v2/service/ecs` | v1.73.1 (already in go.mod indirect) | ECS API: ListClusters, ListServices, ListTasks, DescribeTasks, DescribeTaskDefinition | Only official AWS Go v2 ECS client |
| `github.com/charmbracelet/bubbles` | v1.0.0 (already in go.mod) | `list.Model` for ClusterList/ServiceList/TaskList panels | Established project pattern; same as S3 panel |
| `github.com/dustin/go-humanize` | v1.0.1 (already in go.mod) | Relative time for task started-at timestamps | Already vendored; used in S3 panel |
| `github.com/charmbracelet/lipgloss` | v1.1.0 (already in go.mod) | Styling, breadcrumb rendering, TaskDetail divider | Established project pattern |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `os/exec` (stdlib) | Go 1.24.2 | Open CloudWatch URL in system browser | ECS-06: `exec.Command("open", url)` / `exec.Command("xdg-open", url)` |
| `runtime` (stdlib) | Go 1.24.2 | `runtime.GOOS` to detect darwin vs linux for URL open | ECS-06 cross-platform browser open |
| `strings` (stdlib) | Go 1.24.2 | Task ARN parsing (`strings.LastIndex`), divider lines | Throughout model/view code |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `bubbles/list` + `ecsDelegate` | `bubbles/table` | `bubbles/table` rejected (CONTEXT.md locked decision); breaks consistency with S3 panel pattern |
| `DescribeTaskDefinition` for log config | Store log config from task creation event | No event sourcing in this app; DescribeTaskDefinition is the only way |

**Installation (promote from indirect to direct):**
```bash
cd /Users/jacobcunningham/_CODE/duchess
go get github.com/aws/aws-sdk-go-v2/service/ecs
```

## Architecture Patterns

### Recommended Project Structure

```
internal/ui/ecs/
  model.go      — Model struct, Init/Update/View, state machine, ascend/descend helpers
  delegate.go   — ecsDelegate, ecsItem, ecsItemKind (cluster/service/task)
  client.go     — FetchClustersCmd, FetchServicesCmd, FetchTasksCmd, FetchTaskDetailCmd, FetchTaskDefinitionCmd
  messages.go   — all typed message structs (loaded/err/tick variants for each level)
internal/ui/model.go — add ecsPanel ecs.Model field, forward messages in Update
```

### Pattern 1: Value-Receiver State Machine (mirror S3 exactly)

**What:** All `Model` methods use value receivers. State mutations return `(Model, tea.Cmd)`. The Go copy semantics mean every state change is explicit and returned to the caller.

**When to use:** Always — this is the locked project convention.

```go
// Source: internal/ui/s3/model.go (established pattern)
func (m Model) ascendLevel() (Model, tea.Cmd) {
    switch m.state {
    case panelTaskDetail:
        m.state = panelTaskList
        m.selectedTaskDetail = nil
        m.containerCursor = 0
        return m, nil
    case panelTaskList:
        m.state = panelServiceList
        m.selectedService = ""
        m.loading = true
        cmd := m.list.SetItems([]list.Item{})
        return m, tea.Batch(cmd, FetchServicesCmd(m.ctx, m.client, m.selectedCluster))
    case panelServiceList:
        m.state = panelClusterList
        m.selectedCluster = ""
        m.loading = true
        cmd := m.list.SetItems([]list.Item{})
        return m, tea.Batch(cmd, FetchClustersCmd(m.ctx, m.client))
    case panelClusterList:
        return m, nil // already at root
    }
    return m, nil
}
```

### Pattern 2: Async tea.Cmd Constructors (ECS version)

**What:** Each API fetch is a `tea.Cmd` closure. The panel model never imports the ECS SDK directly for API calls — it fires `FetchXxxCmd` functions that close over `ctx` and `client`.

**When to use:** Every ECS API call.

```go
// Source: internal/ui/s3/client.go (template pattern)
// ECS equivalent:

func FetchClustersCmd(ctx context.Context, client *ecs.Client) tea.Cmd {
    return func() tea.Msg {
        clusters, err := ListAllClusters(ctx, client)
        if err != nil {
            return clustersErrMsg{err: err}
        }
        return clustersLoadedMsg{clusters: clusters}
    }
}

func FetchServicesCmd(ctx context.Context, client *ecs.Client, clusterArn string) tea.Cmd {
    return func() tea.Msg {
        services, err := ListAllServices(ctx, client, clusterArn)
        if err != nil {
            return servicesErrMsg{err: err}
        }
        return servicesLoadedMsg{services: services}
    }
}

// ECS-06: Separate cmd for task definition (log configuration)
func FetchTaskDefinitionCmd(ctx context.Context, client *ecs.Client, taskDefArn string) tea.Cmd {
    return func() tea.Msg {
        out, err := client.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
            TaskDefinition: aws.String(taskDefArn),
        })
        if err != nil {
            return taskDefinitionErrMsg{err: err}
        }
        return taskDefinitionLoadedMsg{taskDef: out.TaskDefinition}
    }
}
```

### Pattern 3: Pagination with SDK Paginators

**What:** ECS provides `ListClustersPaginator`, `ListServicesPaginator`, `ListTasksPaginator`. Use them the same way S3 uses `ListObjectsV2Paginator`.

**When to use:** All ECS list calls.

```go
// Source: verified from ~/go/pkg/mod/github.com/aws/aws-sdk-go-v2/service/ecs@v1.73.1
func ListAllClusters(ctx context.Context, client *ecs.Client) ([]string, error) {
    paginator := ecs.NewListClustersPaginator(client, &ecs.ListClustersInput{})
    var arns []string
    for paginator.HasMorePages() {
        page, err := paginator.NextPage(ctx)
        if err != nil {
            return nil, err
        }
        arns = append(arns, page.ClusterArns...)
    }
    return arns, nil
}

func ListAllServices(ctx context.Context, client *ecs.Client, clusterArn string) ([]ecstypes.Service, error) {
    // ListServices returns ARNs only; follow up with DescribeServices (max 10 per call)
    paginator := ecs.NewListServicesPaginator(client, &ecs.ListServicesInput{
        Cluster: aws.String(clusterArn),
    })
    var serviceArns []string
    for paginator.HasMorePages() {
        page, err := paginator.NextPage(ctx)
        if err != nil {
            return nil, err
        }
        serviceArns = append(serviceArns, page.ServiceArns...)
    }
    // Describe in batches of 10 (DescribeServices limit)
    var services []ecstypes.Service
    for i := 0; i < len(serviceArns); i += 10 {
        end := i + 10
        if end > len(serviceArns) {
            end = len(serviceArns)
        }
        out, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
            Cluster:  aws.String(clusterArn),
            Services: serviceArns[i:end],
        })
        if err != nil {
            return nil, err
        }
        services = append(services, out.Services...)
    }
    return services, nil
}

func ListAndDescribeTasks(ctx context.Context, client *ecs.Client, clusterArn, serviceArn string) ([]ecstypes.Task, error) {
    // Extract service name from ARN for ListTasks filter
    serviceName := serviceArn[strings.LastIndex(serviceArn, "/")+1:]
    paginator := ecs.NewListTasksPaginator(client, &ecs.ListTasksInput{
        Cluster:     aws.String(clusterArn),
        ServiceName: aws.String(serviceName),
    })
    var taskArns []string
    for paginator.HasMorePages() {
        page, err := paginator.NextPage(ctx)
        if err != nil {
            return nil, err
        }
        taskArns = append(taskArns, page.TaskArns...)
    }
    if len(taskArns) == 0 {
        return nil, nil
    }
    // DescribeTasks: max 100 per call
    var tasks []ecstypes.Task
    for i := 0; i < len(taskArns); i += 100 {
        end := i + 100
        if end > len(taskArns) {
            end = len(taskArns)
        }
        out, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
            Cluster: aws.String(clusterArn),
            Tasks:   taskArns[i:end],
        })
        if err != nil {
            return nil, err
        }
        tasks = append(tasks, out.Tasks...)
    }
    return tasks, nil
}
```

### Pattern 4: Per-Level Refresh Tickers (self-scheduling)

**What:** Each level has its own typed tick message. Each tick handler re-fetches current level data AND schedules the next tick via `tea.Batch`.

**When to use:** All three tick message types.

```go
// Source: internal/ui/s3/messages.go + client.go (template)
// messages.go
type ecsClusterRefreshTickMsg struct{}
type ecsServiceRefreshTickMsg struct{}
type ecsTaskRefreshTickMsg struct{}

// client.go
func ECSClusterRefreshTickCmd() tea.Cmd {
    return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
        return ecsClusterRefreshTickMsg{}
    })
}
func ECSServiceRefreshTickCmd() tea.Cmd {
    return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
        return ecsServiceRefreshTickMsg{}
    })
}
func ECSTaskRefreshTickCmd() tea.Cmd {
    return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
        return ecsTaskRefreshTickMsg{}
    })
}

// model.go Update handler
case ecsClusterRefreshTickMsg:
    m.refreshing = true
    return m, tea.Batch(FetchClustersCmd(m.ctx, m.client), ECSClusterRefreshTickCmd())
case ecsServiceRefreshTickMsg:
    if m.state == panelServiceList && m.selectedCluster != "" {
        m.refreshing = true
        return m, tea.Batch(FetchServicesCmd(m.ctx, m.client, m.selectedCluster), ECSServiceRefreshTickCmd())
    }
    return m, ECSServiceRefreshTickCmd()
case ecsTaskRefreshTickMsg:
    if m.state == panelTaskList && m.selectedCluster != "" && m.selectedService != "" {
        m.refreshing = true
        return m, tea.Batch(FetchTasksCmd(m.ctx, m.client, m.selectedCluster, m.selectedService), ECSTaskRefreshTickCmd())
    }
    return m, ECSTaskRefreshTickCmd()
```

### Pattern 5: CloudWatch Logs URL Construction (ECS-06)

**What:** Construct the CloudWatch deep-link URL from `ContainerDefinition.LogConfiguration`. The CloudWatch URL format uses a non-standard encoding where `/` in log group/stream paths is encoded as `$252F`.

**When to use:** `L` key handler in TaskDetail pane.

```go
// Source: CONTEXT.md + verified against AWS docs
// Log stream format: {prefix}/{container-name}/{task-short-id}
// where task-short-id is the last path segment of the task ARN (the UUID after the last /)

func cloudWatchLogsURL(region, logGroup, logStream string) string {
    // CloudWatch deep-link format: / must be encoded as $252F (double-encoded)
    encodeForCW := func(s string) string {
        return strings.ReplaceAll(url.PathEscape(s), "%", "$")
    }
    encodedGroup := encodeForCW(logGroup)
    encodedStream := encodeForCW(logStream)
    return fmt.Sprintf(
        "https://%s.console.aws.amazon.com/cloudwatch/home?region=%s#logsV2:log-groups/log-group/%s/log-events/%s",
        region, region, encodedGroup, encodedStream,
    )
}

func taskShortID(taskArn string) string {
    // Task ARN: arn:aws:ecs:region:account:task/cluster-name/uuid OR arn:aws:ecs:...:task/uuid
    idx := strings.LastIndex(taskArn, "/")
    if idx < 0 {
        return taskArn
    }
    return taskArn[idx+1:]
}

func openURL(url string) error {
    var cmd *exec.Cmd
    switch runtime.GOOS {
    case "darwin":
        cmd = exec.Command("open", url)
    default: // linux
        cmd = exec.Command("xdg-open", url)
    }
    return cmd.Start()
}
```

### Pattern 6: rootModel Integration

**What:** Add `ecsPanel ecs.Model` field to `rootModel`. For Phase 4, both panels exist but the active one is determined by a panel selector (the CONTEXT.md notes this is "Phase 5 concern" for tab-based switching; for Phase 4, hard-code which is active OR add a minimal toggle).

**When to use:** `internal/ui/model.go`

```go
// Add to rootModel struct:
ecsPanel  ecspanel.Model
activePanel panelSelector // or just hard-code ecsPanel as visible in Phase 4

// In identityLoadedMsg handler:
m.ecsPanel = ecspanel.NewModel(m.ctx, m.awsCfg, m.width, contentH)
return m, tea.Batch(m.s3Panel.Init(), m.ecsPanel.Init())

// Forward messages in Update:
if m.state == stateReady {
    // Forward to active panel; for Phase 4, determine which is active
    var cmd tea.Cmd
    m.ecsPanel, cmd = m.ecsPanel.Update(msg)
    return m, cmd
}
```

### Anti-Patterns to Avoid

- **Calling ListTasks without a ServiceName filter:** Will return ALL tasks in the cluster (potentially hundreds from other services). Always pass `ServiceName` to `ListTasksInput`.
- **Accessing LogConfiguration from Container struct:** The `Container` type returned by `DescribeTasks` has NO `LogConfiguration` field. Only `ContainerDefinition` (from `DescribeTaskDefinition`) has it. Match by container `Name` string.
- **Pointer mutation in value-receiver methods:** All fields must be mutated on the local copy `m` and returned. Mutations to `m.list` via methods like `SetItems` return `tea.Cmd` — capture it, don't discard.
- **Starting all 3 tickers in Init unconditionally:** Only the cluster ticker is relevant from Init. Service and task tickers should start when descending into those levels to avoid unnecessary API calls.
- **Batching DescribeServices calls without respecting the 10-item limit:** `DescribeServices` accepts a maximum of 10 service ARNs per call. Batch in chunks of 10.
- **Using `ListTasks` with service ARN instead of service name:** `ListTasksInput.ServiceName` expects the short service name, not the full ARN. Extract with `strings.LastIndex(serviceArn, "/")+1`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ECS API pagination | Manual NextToken loops | `NewListClustersPaginator`, `NewListServicesPaginator`, `NewListTasksPaginator` | Paginators handle `HasMorePages()` + `NextToken` automatically; verified in SDK source |
| Relative timestamps | Time formatting logic | `go-humanize` v1.0.1 (already vendored) | Already used in S3 panel; handles "2h ago", "1d ago" edge cases |
| List scrolling/pagination | Manual j/k handling | `bubbles/list v1.0.0` with `DisableQuitKeybindings()` + `SetFilteringEnabled(false)` | Handles j/k, g/G, page up/down, item wrapping — identical to S3 pattern |
| CloudWatch URL encoding | Custom `%` → `$` encoder | `net/url.PathEscape` + `strings.ReplaceAll("%", "$")` | The CloudWatch deep-link format requires exactly this two-step encoding |
| ANSI color width calculation | Manual string length | `lipgloss.Width()` | Accounts for ANSI escape sequences that `len()` does not |

**Key insight:** The ECS panel is a 90% structural copy of the S3 panel. Don't abstract prematurely — copy the S3 files as the starting template, then adapt for ECS specifics.

## Common Pitfalls

### Pitfall 1: LogConfiguration Not on Container struct
**What goes wrong:** Code tries to access `container.LogConfiguration` in the `DescribeTasks` response and gets a compile error — the field doesn't exist.
**Why it happens:** `Container` (runtime representation) and `ContainerDefinition` (task definition representation) are separate types. Only `ContainerDefinition` has `LogConfiguration`.
**How to avoid:** When entering `panelTaskDetail`, fire a `FetchTaskDefinitionCmd(task.TaskDefinitionArn)` alongside the task detail display. Match container definitions by `ContainerDefinition.Name == Container.Name` to look up log config.
**Warning signs:** Compile error "type Container has no field LogConfiguration".

### Pitfall 2: ListTasks Returns ARNs, Not Task Details
**What goes wrong:** After listing task ARNs via `ListTasks`, code displays ARNs directly rather than calling `DescribeTasks` for status/startedAt/containers.
**Why it happens:** `ListTasksOutput.TaskArns` is `[]string` (ARNs only). `DescribeTasks` is required for all meaningful task data.
**How to avoid:** `FetchTasksCmd` must always call both `ListTasks` (paginator) and `DescribeTasks` (batched, max 100). They can be composed in `ListAndDescribeTasks()`.
**Warning signs:** Task rows show only ARN strings, no status or timestamps.

### Pitfall 3: DescribeServices Limit (10 per call)
**What goes wrong:** Calling `DescribeServices` with more than 10 service ARNs returns an error: "Services list cannot have more than 10 items".
**Why it happens:** `DescribeServices` has a hard limit of 10 ARNs per API call (not paginated like List calls).
**How to avoid:** Chunk service ARNs into groups of 10 in `ListAllServices()`.
**Warning signs:** API error on clusters with >10 services.

### Pitfall 4: ListTasks ServiceName vs ServiceArn
**What goes wrong:** Passing the full service ARN to `ListTasksInput.ServiceName` returns 0 results or an error.
**Why it happens:** `ListTasksInput.ServiceName` expects the short service name (e.g., "my-service"), not the ARN.
**How to avoid:** Extract short name: `serviceName := serviceArn[strings.LastIndex(serviceArn, "/")+1:]`
**Warning signs:** ListTasks returns 0 tasks even when the service is running.

### Pitfall 5: CloudWatch URL Encoding
**What goes wrong:** Standard `url.QueryEscape` or `url.PathEscape` produces `%2F` for `/`, but CloudWatch console requires `$252F`.
**Why it happens:** CloudWatch uses a non-standard URL encoding where `%` itself is encoded as `$25`, so `%2F` (slash) becomes `$252F`.
**How to avoid:** Use `strings.ReplaceAll(url.PathEscape(s), "%", "$")` for log group and stream path components.
**Warning signs:** CloudWatch URL opens the console but shows "log group not found" or navigates to the wrong group.

### Pitfall 6: ECS Client is Not Per-Cluster Regional
**What goes wrong:** Unlike S3 (where buckets live in specific regions requiring per-bucket clients), ECS API calls use a single regional client built from `aws.Config` — the same region applies to all clusters in that region.
**Why it happens:** S3 buckets can live in any region; ECS clusters are always in the configured region.
**How to avoid:** ECS panel needs only ONE `*ecs.Client` built from `aws.Config`. No per-cluster client switching needed.
**Warning signs:** Unnecessary regional client construction logic.

### Pitfall 7: Tickers Fire for All Levels Even When on Different Level
**What goes wrong:** `ecsServiceRefreshTickMsg` fires a `FetchServicesCmd` even when the user is on `panelClusterList`, triggering errors because `selectedCluster` is empty.
**Why it happens:** Tickers self-schedule regardless of panel state.
**How to avoid:** In each tick handler, guard with current state check before issuing refresh command:
```go
case ecsServiceRefreshTickMsg:
    if m.state == panelServiceList && m.selectedCluster != "" {
        // fetch
    }
    return m, ECSServiceRefreshTickCmd() // always reschedule
```

### Pitfall 8: go.mod ECS Package is Indirect
**What goes wrong:** `go build` or import fails because ECS package is `// indirect` in go.mod.
**Why it happens:** `go get` added it as indirect because no Go file in the project directly imports it yet.
**How to avoid:** First plan task must run `go get github.com/aws/aws-sdk-go-v2/service/ecs` (without `// indirect`) after adding the import to a `.go` file, OR manually add direct dependency.
**Warning signs:** `go mod tidy` keeps marking it `// indirect`.

## Code Examples

Verified patterns from SDK source and established codebase:

### ECS Client Construction
```go
// Source: internal/ui/s3/client.go pattern + ecs SDK
import awsecs "github.com/aws/aws-sdk-go-v2/service/ecs"

func NewModel(ctx context.Context, awsCfg aws.Config, width, height int) Model {
    client := awsecs.NewFromConfig(awsCfg)
    // ... rest of NewModel
    return Model{ctx: ctx, client: client, ...}
}
```

### Task Short ID from ARN
```go
// Source: CONTEXT.md decision + common AWS pattern
// Task ARN format: arn:aws:ecs:region:account:task/cluster-name/task-uuid
//                  OR arn:aws:ecs:region:account:task/task-uuid (older format)
func taskShortID(taskArn string) string {
    idx := strings.LastIndex(taskArn, "/")
    if idx < 0 || idx >= len(taskArn)-1 {
        return taskArn
    }
    return taskArn[idx+1:]
}

// Display first 8 chars (like git short SHA)
shortDisplay := taskShortID(taskArn)
if len(shortDisplay) > 8 {
    shortDisplay = shortDisplay[:8]
}
```

### Service Row Rendering
```go
// Source: CONTEXT.md decision
func serviceRowRight(svc ecstypes.Service) string {
    running := svc.RunningCount
    desired := svc.DesiredCount
    pending := svc.PendingCount
    right := fmt.Sprintf("%d/%d", running, desired)
    if pending > 0 {
        right += fmt.Sprintf(" (%d pending)", pending)
    }
    return right
}

func serviceRowLeft(svc ecstypes.Service) string {
    name := aws.ToString(svc.ServiceName)
    tag := ""
    switch svc.LaunchType {
    case ecstypes.LaunchTypeFargate:
        tag = "[FARGATE]"
    case ecstypes.LaunchTypeEc2:
        tag = "[EC2]"
    }
    if tag != "" {
        name = name + "  " + tag
    }
    return name
}
```

### CloudWatch Logs URL (ECS-06)
```go
// Source: CONTEXT.md + AWS docs (verified encoding)
// net/url package for PathEscape
import "net/url"

func buildCloudWatchURL(region, logGroup, logStream string) string {
    // CloudWatch deep-link: / must appear as $252F
    // PathEscape converts / to %2F; then replace % with $ gives $2F... but we need $252F
    // Correct approach: encode twice or use the known substitution
    cwEncode := func(s string) string {
        escaped := url.PathEscape(s)                     // / -> %2F, etc.
        return strings.ReplaceAll(escaped, "%", "$25")  // %2F -> $252F
    }
    return fmt.Sprintf(
        "https://%s.console.aws.amazon.com/cloudwatch/home?region=%s#logsV2:log-groups/log-group/%s/log-events/%s",
        region, region,
        cwEncode(logGroup),
        cwEncode(logStream),
    )
}

// Log stream name: {prefix}/{containerName}/{taskShortID}
// prefix = logConfiguration.Options["awslogs-stream-prefix"]
// containerName = containerDef.Name
// taskShortID = last path segment of task ARN
```

### TaskDetail Container Cursor (not bubbles/list)
```go
// Source: CONTEXT.md decision — separate cursor int, not bubbles/list component
// j/k key handling in Update when state == panelTaskDetail:
case "j":
    if m.containerCursor < len(m.selectedTaskDetail.containers)-1 {
        m.containerCursor++
    }
    return m, nil
case "k":
    if m.containerCursor > 0 {
        m.containerCursor--
    }
    return m, nil
case "L":
    if m.state == panelTaskDetail && m.selectedTaskDetail != nil {
        container := m.selectedTaskDetail.containers[m.containerCursor]
        // Use taskDef container definition for log config
        url, err := buildLogsURL(m, container)
        // handle err, open url
    }
```

### rootModel Integration
```go
// Source: internal/ui/model.go (existing pattern, add ecsPanel)
import ecspanel "github.com/jacobscunn07/duchess/internal/ui/ecs"

type rootModel struct {
    // ... existing fields ...
    s3Panel  s3panel.Model
    ecsPanel ecspanel.Model   // NEW
}

// In identityLoadedMsg handler:
m.ecsPanel = ecspanel.NewModel(m.ctx, m.awsCfg, m.width, contentH)
// NOTE: For Phase 4, decide which panel is "active" — the rootModel
// currently shows s3Panel always. Phase 4 needs at minimum a way to
// switch to ecsPanel (keyboard shortcut or hardcoded active panel).
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| AWS SDK Go v1 (`aws-sdk-go`) | AWS SDK Go v2 (`aws-sdk-go-v2`) | 2021+ | v1 is EOL; project already on v2 |
| Manual pagination with `NextToken` | SDK paginators (`NewListXxxPaginator`) | v2 SDK | Paginators handle token management; use them |
| `bubbles v0.21.0` (old) | `bubbles v1.0.0` | Phase 2 (already upgraded) | v1.0.0 API; no further migration needed |

**Deprecated/outdated:**
- `aws-sdk-go` (v1): EOL, not used in this project
- `GetBucketLocation` for region detection: Known bug with us-east-1; irrelevant to ECS (no per-cluster region detection needed)

## Open Questions

1. **Active Panel Switching in Phase 4**
   - What we know: CONTEXT.md says "for Phase 4, ECS panel coexists with S3 panel"; tab-based switching is Phase 5
   - What's unclear: How does the user currently get to the ECS panel in Phase 4? The rootModel always renders `s3Panel.View()`. No mechanism exists yet.
   - Recommendation: Add a minimal panel toggle key (e.g., `tab` key toggles `activePanel` between `panelS3` and `panelECS`) to `rootModel` as part of Phase 4. This is minimal scope — just a string/int toggle and forward to the active panel's Update/View.

2. **DescribeTaskDefinition Caching**
   - What we know: Every time the user enters TaskDetail, a `DescribeTaskDefinition` API call is needed for ECS-06 log URLs. Task definitions don't change frequently.
   - What's unclear: Whether to cache the task definition in the `ecsTask` data struct (fetched once on entry, stored in `selectedTaskDetail`) or re-fetch on each `L` key press.
   - Recommendation: Cache in `selectedTaskDetail` struct as `taskDef *ecstypes.TaskDefinition`. Fetch once when entering `panelTaskDetail` via a `FetchTaskDetailCmd` that calls both `DescribeTasks` AND `DescribeTaskDefinition` concurrently (or sequentially in one goroutine).

3. **ListTasks ServiceName with Spaces/Special Chars**
   - What we know: ECS service names can contain hyphens but not special URL chars. `ListTasksInput.ServiceName` accepts the short name, not ARN.
   - What's unclear: Whether service names extracted from ARNs ever contain characters that need extra handling.
   - Recommendation: `strings.LastIndex(serviceArn, "/")` is sufficient for the ARN format. No additional encoding needed for the API call.

## Sources

### Primary (HIGH confidence)
- `~/go/pkg/mod/github.com/aws/aws-sdk-go-v2/service/ecs@v1.73.1/types/types.go` — Verified `Container`, `ContainerDefinition`, `Task`, `Service`, `LogConfiguration`, `TaskDefinition` struct fields
- `~/go/pkg/mod/github.com/aws/aws-sdk-go-v2/service/ecs@v1.73.1/api_op_ListTasks.go` — Verified `ListTasksInput.ServiceName` field, MaxResults=100
- `~/go/pkg/mod/github.com/aws/aws-sdk-go-v2/service/ecs@v1.73.1/api_op_DescribeTasks.go` — Verified "A list of up to 100 task IDs or full ARN entries"
- `~/go/pkg/mod/github.com/aws/aws-sdk-go-v2/service/ecs@v1.73.1/api_op_DescribeServices.go` — Verified "A list of services to describe. You may specify up to 10 services"
- `~/go/pkg/mod/github.com/aws/aws-sdk-go-v2/service/ecs@v1.73.1/api_op_DescribeTaskDefinition.go` — Verified `DescribeTaskDefinitionInput.TaskDefinition` field
- `internal/ui/s3/model.go`, `delegate.go`, `client.go`, `messages.go` — Established patterns to mirror
- `internal/ui/model.go` — rootModel integration point
- `go.mod` — ECS SDK v1.73.1 already present as indirect dependency
- https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_LogConfiguration.html — LogConfiguration struct fields and awslogs options map keys

### Secondary (MEDIUM confidence)
- WebSearch + AWS docs: Log stream name format = `{prefix}/{container-name}/{ecs-task-id}` where ecs-task-id is the task UUID short ID
- WebSearch: CloudWatch deep-link URL format `#logsV2:log-groups/log-group/{group}/log-events/{stream}` with `$252F` encoding for `/`

### Tertiary (LOW confidence)
- CloudWatch URL encoding: `strings.ReplaceAll(url.PathEscape(s), "%", "$25")` produces `$252F` for `/` — this follows the two-step encoding described in community sources; should be validated with a real CloudWatch URL before finalizing

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — verified from go.mod and SDK source files
- Architecture patterns: HIGH — verified from existing S3 panel code + ECS SDK types
- ECS API struct fields: HIGH — read directly from downloaded SDK source
- CloudWatch URL encoding: MEDIUM — confirmed pattern from community sources; validate with actual URL
- Pitfalls: HIGH — derived from actual struct definitions confirming Container lacks LogConfiguration

**Research date:** 2026-03-04
**Valid until:** 2026-04-04 (AWS SDK stable; ECS API stable; 30-day window)
