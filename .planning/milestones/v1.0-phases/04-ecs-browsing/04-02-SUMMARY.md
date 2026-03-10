---
phase: 04-ecs-browsing
plan: 02
subsystem: ui
tags: [ecs, bubbletea, lipgloss, tui, four-state-machine, panel-toggle, cloudwatch]

# Dependency graph
requires:
  - phase: 04-ecs-browsing
    provides: internal/ui/ecs/messages.go + delegate.go + client.go — all ECS tea.Cmd constructors and typed messages consumed by this plan's model.go
  - phase: 03-s3-browsing
    provides: s3/model.go value-receiver pattern, ascendLevel/descendIntoSelected convention, breadcrumb style — mirrored exactly by ecs/model.go
provides:
  - internal/ui/ecs/model.go — ECS panel Model: four-state machine, Init/Update/View, ascend/descend helpers, TaskDetail pane with container navigation, CloudWatch log URL opener
  - internal/ui/model.go — rootModel updated with ecsPanel field, Tab key toggle, dual-panel message forwarding
  - internal/ui/status.go — status bar shows [S3] or [ECS] panel indicator
affects: [04-03-ecs-integration, 05-ecs-overlays]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Four-state ECS panel machine (panelClusterList -> panelServiceList -> panelTaskList -> panelTaskDetail) mirroring S3 three-state pattern
    - Tab key activePanel toggle in rootModel — simple int/const toggle routing Update/View to active panel
    - Dual-panel message forwarding — rootModel forwards all messages to both panels; unexported typed messages are panel-local so cross-panel interference is impossible
    - Per-level ticker lifecycle — cluster ticker started in Init only; service/task tickers started on descent to avoid running all three unconditionally
    - taskDef embedded in selectedTask update path — taskDetailLoadedMsg updates m.selectedTask.task to refresh stale ecsItem on load

key-files:
  created:
    - internal/ui/ecs/model.go
  modified:
    - internal/ui/model.go
    - internal/ui/status.go

key-decisions:
  - "Tab key toggle uses simple activePanel int constant in rootModel — no complex routing needed since all messages are forwarded to both panels"
  - "Both panels receive ALL messages in stateReady — unexported typed messages (clustersLoadedMsg etc.) are package-local so cross-panel interference is structurally impossible"
  - "taskDetailLoadedMsg updates m.selectedTask.task to the refreshed task from the response — the ecsItem set during descendIntoSelected may be stale"
  - "Service ticker and task ticker started on descent (not in Init) — avoids three concurrent refresh tickers when user is only looking at cluster list"
  - "renderBreadcrumb appends task def short name on panelTaskList via strings.LastIndex on TaskDefinition ARN — satisfies ECS-03"
  - "Status bar shows [S3]/[ECS] panel indicator via m.activePanel field added to rootModel"

patterns-established:
  - "Multi-panel TUI: rootModel holds multiple panel models, routes messages to all, renders only the active one"
  - "Tab key as panel switcher: simple const toggle, stateless beyond activePanel field"

requirements-completed: [ECS-01, ECS-02, ECS-03, ECS-04, ECS-05, ECS-06]

# Metrics
duration: 3min
completed: 2026-03-05
---

# Phase 4 Plan 02: ECS Panel Model Summary

**Four-state ECS panel (cluster -> service -> task list -> task detail) with CloudWatch log opener, wired into rootModel via Tab key toggle alongside existing S3 panel**

## Performance

- **Duration:** ~25 min (including human checkpoint pause)
- **Started:** 2026-03-05T02:10:43Z
- **Completed:** 2026-03-05T21:54:00Z
- **Tasks:** 3 of 3 complete
- **Files modified:** 3

## Accomplishments

- Created `internal/ui/ecs/model.go` — full four-state ECS panel Model mirroring s3/model.go value-receiver pattern; ascendLevel/descendIntoSelected return (Model, tea.Cmd); refresh tick handlers guard by current state; renderBreadcrumb shows task def short name when viewing task list (ECS-03); TaskDetail pane renders container list with j/k navigation and L key for CloudWatch logs (ECS-06)
- Updated `internal/ui/model.go` — rootModel now has ecsPanel + activePanel fields; Tab key toggles between S3 and ECS panels; both panels initialized in identityLoadedMsg handler; all messages (WindowSizeMsg, spinner.TickMsg, and catch-all) forwarded to both panels
- Updated `internal/ui/status.go` — status bar appends [S3] or [ECS] indicator when in stateReady to show active panel

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement internal/ui/ecs/model.go — four-state ECS panel** - `453752c` (feat)
2. **Task 2: Wire ecsPanel into rootModel with tab-key panel switching** - `f37c19c` (feat)

3. **Task 3: Human verification checkpoint** — partial verification accepted (see Issues Encountered)

**Plan metadata:** (final docs commit follows)

## Files Created/Modified

- `internal/ui/ecs/model.go` — ECS panel Model struct (12 fields), NewModel, Init, Update (all message handlers), View, renderBreadcrumb, renderTaskDetail, ascendLevel, descendIntoSelected, taskContainers, listHeight, clamp
- `internal/ui/model.go` — rootModel updated: ecsPanel + activePanel fields, Tab key handler, dual-panel WindowSizeMsg/spinner.TickMsg/catch-all forwarding, contentView renders active panel
- `internal/ui/status.go` — renderStatusBar updated to show [S3]/[ECS] panel indicator in right group

## Decisions Made

- Tab key toggle uses a simple `activePanel` int constant — no complex routing needed since all messages are forwarded to both panels regardless
- Both panels receive ALL messages in stateReady — the unexported typed ECS messages (clustersLoadedMsg, etc.) are package-local to `internal/ui/ecs`, making cross-panel interference structurally impossible
- taskDetailLoadedMsg handler updates `m.selectedTask.task` to refresh the ecsItem set during `descendIntoSelected` (which may be stale after network load)
- Service and task refresh tickers are started on descent (not in Init) — prevents three concurrent 10s/5s tickers when user has only descended to cluster list

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

Human verification was partial: user confirmed app launch, S3 default panel, Tab toggle to ECS (showing empty cluster list), Tab back to S3, and q exit cleanly. Deep ECS navigation (cluster -> service -> task -> TaskDetail) could not be verified because no live ECS cluster was available in the user's AWS environment. The structural navigation and panel switching is fully implemented per plan spec; ECS data loading behaviors are contingent on having AWS ECS resources.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `go build ./...` and `go vet ./...` both pass with all three new/modified files
- ECS panel implementation complete: Tab to switch, Enter to descend, Esc to ascend, breadcrumb updates at each level, L for CloudWatch logs
- Deep ECS navigation will be exercisable once user has a live ECS cluster in their AWS environment
- Phase 4 plan 02 of 03 complete; ready for next plan

## Self-Check: PASSED

- FOUND: internal/ui/ecs/model.go
- FOUND: internal/ui/model.go
- FOUND: internal/ui/status.go
- FOUND: .planning/phases/04-ecs-browsing/04-02-SUMMARY.md
- FOUND: commit 453752c (Task 1: ECS panel model.go)
- FOUND: commit f37c19c (Task 2: wire ecsPanel into rootModel)
- Human checkpoint (Task 3): partial verification accepted — core panel switching confirmed, deep ECS nav deferred (no live cluster available)

---
*Phase: 04-ecs-browsing*
*Completed: 2026-03-05*
