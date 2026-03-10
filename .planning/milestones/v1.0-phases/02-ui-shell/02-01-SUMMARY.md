---
phase: 02-ui-shell
plan: 01
subsystem: ui
tags: [bubbletea, bubbles, lipgloss, tui, termenv, charmbracelet, go]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: config.Config, session.GetCallerIdentity, session.ClassifyCredentialError

provides:
  - bubbletea v1.3.10, bubbles v1.0.0, lipgloss v1.1.0 as direct go.mod dependencies
  - internal/ui package with rootModel struct implementing tea.Model (Init/Update/View)
  - internal/ui/messages.go with identityLoadedMsg, identityErrMsg, tickMsg types
  - rootModel with WindowSizeMsg, q/ctrl+c quit, NO_COLOR guard, zero-width View guard
  - fetchIdentityCmd and tickCmd package-level helpers

affects:
  - 02-02-ui-shell (status bar and identity display builds on rootModel)
  - All future TUI plans in phase 02

# Tech tracking
tech-stack:
  added:
    - github.com/charmbracelet/bubbletea v1.3.10 (Bubble Tea TUI framework)
    - github.com/charmbracelet/bubbles v1.0.0 (spinner component)
    - github.com/charmbracelet/lipgloss v1.1.0 (terminal styling)
    - github.com/muesli/termenv v0.16.0 (color profile control, NO_COLOR)
  patterns:
    - tea.Model interface pattern (Init/Update/View methods on value receiver)
    - TDD RED/GREEN workflow for Bubble Tea model logic
    - fetchIdentityCmd wraps async AWS calls as tea.Cmd closures
    - tickCmd fires tea.Tick every second for clock updates
    - Zero-dimension guard: View returns "" when width=0 (before first WindowSizeMsg)

key-files:
  created:
    - internal/ui/messages.go
    - internal/ui/model.go
    - internal/ui/model_test.go
  modified:
    - go.mod (added bubbletea, bubbles, lipgloss, termenv as direct deps)
    - go.sum

key-decisions:
  - "Charmbracelet v1.x stack (bubbletea v1.3.10, bubbles v1.0.0, lipgloss v1.1.0) — NOT v2 (charm.land/ paths have breaking API changes)"
  - "termenv.Ascii used for NO_COLOR guard via lipgloss.SetColorProfile — termenv is a direct dependency of lipgloss, available transitively"
  - "spinner.TickMsg swallowed when state != stateLoading — prevents wasted renders after identity loaded"
  - "rootModel uses value receivers for Init/Update/View — matches tea.Model convention, state mutations return new model"

patterns-established:
  - "tea.Model value receiver pattern: methods receive rootModel by value and return modified copy"
  - "Package-level cmd functions (fetchIdentityCmd, tickCmd) instead of methods — enables clean testing"
  - "State machine pattern: stateLoading -> stateReady | stateError transitions in Update switch"

requirements-completed: [NAV-04, NAV-06]

# Metrics
duration: 2min
completed: 2026-03-03
---

# Phase 2 Plan 01: UI Shell — Dependency Upgrade and rootModel Scaffold Summary

**Bubble Tea v1.x stack (bubbletea/bubbles/lipgloss) added to go.mod and rootModel TUI scaffold created with q-to-quit, WindowSizeMsg, NO_COLOR guard, and async identity fetch wiring**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-04T00:19:28Z
- **Completed:** 2026-03-04T00:22:09Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Upgraded charmbracelet stack to v1.x (bubbletea v1.3.10, bubbles v1.0.0, lipgloss v1.1.0) as direct go.mod dependencies
- Created `internal/ui` package with `rootModel` implementing `tea.Model` (Init/Update/View) with proper Bubble Tea wiring
- Established TDD workflow: 9 tests covering all rootModel behaviors written first (RED), then implemented (GREEN)

## Task Commits

Each task was committed atomically:

1. **Task 1: Upgrade Charmbracelet dependencies** - `bc18ee3` (chore)
2. **Task 2 RED: Add failing rootModel tests** - `9d5af1b` (test)
3. **Task 2 GREEN: Implement internal/ui package** - `55fec59` (feat)

_Note: TDD task has separate test and implementation commits._

## Files Created/Modified

- `internal/ui/messages.go` - Custom Bubble Tea message types: identityLoadedMsg, identityErrMsg, tickMsg
- `internal/ui/model.go` - rootModel struct, NewRootModel constructor, Init/Update/View, fetchIdentityCmd, tickCmd
- `internal/ui/model_test.go` - 9 tests covering all model behaviors
- `go.mod` - Added bubbletea v1.3.10, bubbles v1.0.0, lipgloss v1.1.0, termenv v0.16.0 as direct deps
- `go.sum` - Updated module checksums

## Decisions Made

- **v1.x not v2**: Used charmbracelet v1.x explicitly — v2 uses `charm.land/` import paths with breaking API changes incompatible with this phase's scope
- **termenv.Ascii for NO_COLOR**: `lipgloss.SetColorProfile(termenv.Ascii)` disables all ANSI codes when `NO_COLOR` env var is set; termenv is a transitive lipgloss dependency so no separate install required
- **Value receivers on rootModel**: All three `tea.Model` methods use value receivers and return modified copies — standard Bubble Tea convention, not pointer receivers
- **spinner.TickMsg swallowed after load**: When `m.state != stateLoading`, spinner tick messages are swallowed to prevent unnecessary re-renders

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `rootModel` scaffold is ready for Plan 02 to implement the status bar and display the identity fetched via `fetchIdentityCmd`
- All dependency versions pinned correctly for the full charmbracelet v1.x API surface
- `go build ./...` and `go vet ./...` pass cleanly
- No blockers or concerns

---
*Phase: 02-ui-shell*
*Completed: 2026-03-03*
