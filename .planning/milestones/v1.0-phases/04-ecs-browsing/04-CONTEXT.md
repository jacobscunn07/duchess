# Phase 4: ECS Browsing - Context

**Gathered:** 2026-03-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Full ECS navigation — cluster list → service list → task list → task detail pane — with live refresh, breadcrumb header, per-container CloudWatch Logs link (`L` key), and per-level refresh tickers.

Delivers:
- ECS client with ListClusters, ListServices, ListTasks, DescribeTasks (fully paginated)
- Four-state panel: ClusterList → ServiceList → TaskList → TaskDetail
- Breadcrumb header matching the S3 panel pattern (same component slot in rootModel)
- Vim-style navigation: j/k scroll, g/G top/bottom, Enter descend, Esc ascend
- Per-container `L` key in TaskDetail to open CloudWatch Logs in system browser
- Per-level refresh tickers (clusters/services: 10s, tasks: 5s)
- No filter (not in ECS-01 through ECS-06 requirements)

Profile and region switching is Phase 5. S3 browsing is Phase 3 (complete).

</domain>

<decisions>
## Implementation Decisions

### Navigation depth (4 states)

**Decision: `ClusterList → ServiceList → TaskList → TaskDetail pane`**

State machine constants:
```go
panelClusterList panelState = iota
panelServiceList
panelTaskList
panelTaskDetail
```

Rationale:
- ECS tasks have multiple containers (ECS-05). A TaskDetail pane is the only clean place to list them with j/k navigation.
- ECS-06 requires `L` key per container — need a navigable container list within TaskDetail to know which container's logs to open.
- Mirrors S3's `BucketList → PrefixList → ObjectDetail` depth (3 list states + 1 detail pane).
- Alternatives rejected:
  - "TaskList is the bottom, L opens first container logs" — ambiguous UX for multi-container tasks
  - "TaskList → ContainerList as separate full panel" — adds 5th breadcrumb level and extra state machine complexity

### Row layout (bubbles/list + custom delegate)

**Decision: Same `bubbles/list` + custom delegate pattern as S3. No `bubbles/table`.**

Custom delegate type: `ecsDelegate` with item kinds:
```go
kindCluster  ecsItemKind = iota
kindService
kindTask
```

Rationale:
- Consistent with established codebase pattern (S3 uses `list` + `s3Delegate`)
- `bubbles/table` introduces a new component pattern without sufficient benefit
- Key interception approach identical (Esc, Enter, j/k all work the same way)
- Filter key (`/`) is NOT wired — no filter for ECS in Phase 4

### Service row format

**Decision: Name on left + `running/desired` count on right. Launch type shown as a tag in the name area.**

```
my-api-service  [FARGATE]       3/3
long-svc-name   [EC2]           1/2 (1 pending)
```

- Right column: `running/desired` counts (most operationally useful at a glance)
- Left: service name with `[FARGATE]` or `[EC2]` launch type tag
- If `pending > 0`: append `(N pending)` after the count
- Pending count and task definition shown in TaskList header/breadcrumb area, not in the service row

### Task row format

**Decision: Task ID (short, 8 chars) on left + status + started-at on right.**

```
a1b2c3d4  RUNNING   2h ago
e5f6a7b8  STOPPED   1d ago
```

- Task IDs are 32-char UUIDs; display first 8 chars (same convention as git short SHA)
- Status in the middle: RUNNING, STOPPED, PENDING
- Started-at: relative time via `go-humanize` (matches S3 object last-modified format)

### TaskDetail pane (container list)

**Decision: Task metadata at top + navigable container list below. `L` key on highlighted container opens CloudWatch Logs.**

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

- Containers are a navigable list within the pane (separate cursor index, not bubbles/list)
- `j`/`k` moves the container cursor
- `L` key opens CloudWatch Logs URL for highlighted container
- `Esc` returns to TaskList
- No further navigation (TaskDetail is the leaf)
- Container rows: name on left, image + status on right
- Divider line rendered with `strings.Repeat("─", width-4)` or equivalent

### CloudWatch Logs URL construction

**Decision: Construct awslogs URL from `DescribeTasks` container `logConfiguration`. Open via `os/exec` with `open` (macOS) or `xdg-open` (Linux).**

URL pattern:
```
https://{region}.console.aws.amazon.com/cloudwatch/home?region={region}#logsV2:log-groups/log-group/{logGroup}/log-events/{logStream}
```

- Log group: from `logConfiguration.options["awslogs-group"]`
- Log stream: from `logConfiguration.options["awslogs-stream-prefix"]` + task short ID
- URL-encode the log group and stream path components (replace `/` with `$252F` for CloudWatch deep-link format)
- If container has no `awslogs` log driver: show inline message `"No CloudWatch logs configured for this container"`
- If log stream not yet available (container just started, no stream yet): show inline message `"Log stream not yet available"`
- `open`/`xdg-open` invocation: `exec.Command("open", url)` on darwin, `exec.Command("xdg-open", url)` on linux — detect via `runtime.GOOS`

### Refresh intervals

**Decision: Clusters 10s, Services 10s, Tasks 5s — per-level distinct ticker message types.**

Message types:
```go
ecsClusterRefreshTickMsg  // 10s
ecsServiceRefreshTickMsg  // 10s
ecsTaskRefreshTickMsg     // 5s
```

- Mirrors `s3RefreshTickMsg` pattern from `internal/ui/s3/messages.go`
- Refresh only re-fetches the current visible level (not the full hierarchy)
- Each tick message schedules the next tick (same self-scheduling pattern as S3)
- ECS task data changes frequently (RUNNING → STOPPED transitions in ~5s resolution)
- Clusters and services are more stable — 10s is sufficient

### Filter

**Decision: No filter for ECS in Phase 4.**

- Filter is not in ECS requirements (ECS-01 through ECS-06)
- S3 filter (`/` key) was explicitly required (S3-05); ECS has no equivalent requirement
- ECS cluster/service/task counts are typically small enough that filtering is not critical
- `/` key is NOT wired in the ECS panel; can be added in a future phase if needed

### Error handling

**Decision: Same inline neutral-color pattern as S3 and Phase 2.**

- API errors show in content area as plain text (no lipgloss color, no crash)
- `ClassifyCredentialError` from `internal/aws/session.go` reused for ECS API errors
- Per-level errors don't clear other levels (services error doesn't clear cluster list)
- "No clusters found" / "No services found" / "No tasks found" shows empty list with inline message (not an error)

### Claude's Discretion

- Exact lipgloss styling for TaskDetail pane (container list highlight color, divider line style)
- CloudWatch Logs URL encoding details (exact encoding of `/` in log group names)
- Short-circuit behavior when `DescribeTasks` returns no `logConfiguration` or nil containers
- Exact TaskDetail pane layout (padding between metadata section and container list)
- Refresh ticker start ordering (whether to batch all three tickers in Init or start lazily per-level)
- Pagination chunk size for `ListTasks` (ECS max is 100 per page — paginate fully like S3)
- Whether to use `DescribeServices` for per-service detail or defer to TaskList header display

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets

| Asset | File | How used |
|-------|------|----------|
| `s3panel.Model` pattern | `internal/ui/s3/model.go` | Template for `internal/ui/ecs/model.go` — same state machine, value receivers, `(Model, tea.Cmd)` helpers |
| `s3Delegate` pattern | `internal/ui/s3/delegate.go` | Template for `ecsDelegate` — same two-column render, item kind types |
| `FetchXxxCmd` pattern | `internal/ui/s3/client.go` | Template for ECS async `tea.Cmd` constructors (FetchClustersCmd, FetchServicesCmd, FetchTasksCmd, FetchTaskDetailCmd) |
| `s3RefreshTickMsg` | `internal/ui/s3/messages.go` | Template for per-level ECS refresh ticker message types |
| `rootModel.s3Panel` | `internal/ui/model.go` | Add `ecsPanel ecs.Model` field alongside `s3Panel`; route messages to active panel |
| `ClassifyCredentialError` | `internal/aws/session.go` | Reuse for ECS API error messages |
| `go-humanize` | `internal/ui/s3/model.go` (imported) | Already vendored; reuse for task started-at relative timestamps |

### S3 Panel Structural Patterns (mirror in ECS)

- Value receivers on `Model` struct throughout — all mutations return `(Model, tea.Cmd)`
- `ascendLevel()` and `descendIntoSelected()` helpers return `(Model, tea.Cmd)` — same pattern needed for ECS
- Key interception before `m.list.Update(msg)` — intercept `q`, `ctrl+c`, `esc`, `enter` before list consumes them
- `list.DisableQuitKeybindings()` + `list.SetFilteringEnabled(false)` on list construction
- Breadcrumb via `renderBreadcrumb()` — left-truncate with `"..."` when path too long; spinner appended when loading
- `listHeight()` helper subtracts reserved lines (breadcrumb = 1 line) from total height
- `clamp()` utility for safe list index operations after item replacement

### ECS-Specific Additions

- `selectedCluster string` — equivalent to `selectedBucket`
- `selectedService string` — no S3 equivalent (extra navigation level)
- `selectedTask string` — full task ARN (used for DescribeTasks call)
- `selectedTaskDetail *ecsTask` — equivalent to `selectedObj *s3Item`
- `containerCursor int` — index within the container list in TaskDetail pane (not a bubbles/list)
- Navigation stack: two selected strings (`selectedCluster`, `selectedService`) rather than S3's `prefixStack []string`

### New Package Layout

```
internal/ui/ecs/
  model.go      — Model struct, Init/Update/View, state machine helpers
  delegate.go   — ecsDelegate, ecsItem, ecsItemKind
  client.go     — FetchClustersCmd, FetchServicesCmd, FetchTasksCmd, FetchTaskDetailCmd
  messages.go   — all typed message structs (loaded/err/tick variants)
```

### rootModel Integration

- Add `ecsPanel ecs.Model` field to `rootModel` in `internal/ui/model.go`
- Active panel routing: `rootModel` forwards messages to whichever panel is active (tab-based in Phase 5; for Phase 4, ECS panel coexists with S3 panel)
- `rootModel.Init()` starts both `s3Panel.Init()` and `ecsPanel.Init()` — or lazily on panel switch (Phase 5 concern; Phase 4 may just hard-code one active panel)

</code_context>

<specifics>
## Specific Ideas

- Visual reference: k9s cluster/namespace/pod navigation — clean list transitions with a clear breadcrumb trail showing where you are in the hierarchy
- Task rows should feel like `ps aux` output — short ID, status, and age at a glance
- TaskDetail container list should feel like `docker ps` — name, image, status in aligned columns
- The `L` key for logs is the primary "action" key for ECS (analogous to pressing Enter on an object to see metadata in S3) — make it feel natural and obvious in the TaskDetail hint text

</specifics>

<deferred>
## Deferred Ideas

- **ECS filter** — `/` key for cluster/service/task filtering; not in Phase 4 requirements; can be added as Phase 4.1 if needed
- **Task log streaming** — inline log tailing within the TUI (vs browser redirect); much more complex; defer to a future phase
- **DescribeServices detail pane** — service-level detail (task definition, deployment status, events); not required by ECS-01 through ECS-06
- **EC2 launch type instance details** — showing underlying EC2 instance for tasks; out of scope for Phase 4
- **ECS Exec integration** — `exec` into a running container; out of scope
- **Multi-region ECS browsing** — Phase 5 concern (profile/region switching)

</deferred>

---

*Phase: 04-ecs-browsing*
*Context gathered: 2026-03-04*
