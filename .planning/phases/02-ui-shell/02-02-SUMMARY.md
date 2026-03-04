---
phase: 02-ui-shell
plan: 02
subsystem: ui
tags: [bubbletea, lipgloss, tui, status-bar, spinner, charmbracelet, go]

# Dependency graph
requires:
  - phase: 02-ui-shell/02-01
    provides: rootModel struct, spinner.Model, stateLoading/stateReady/stateError, fetchIdentityCmd, tickCmd, messages

provides:
  - internal/ui/status.go with renderStatusBar(m rootModel) string using lipgloss left/right gap pattern
  - truncateARN(arn string, maxLen int) string for display-safe ARN truncation
  - Updated View() with lipgloss.JoinVertical layout (content area + status bar)
  - contentView() with stateLoading spinner (centered), stateError (plain text), stateReady (empty)
  - cmd/root.go wired to tea.NewProgram(model, tea.WithAltScreen()) — TUI replaces fmt.Printf

affects:
  - 02-03 and all future TUI plans that build panels on top of rootModel

# Tech tracking
tech-stack:
  added: []
  patterns:
    - lipgloss left/right gap pattern for full-width status bar (Width - leftWidth - rightWidth)
    - Package-level lipgloss styles with Inherit(statusBarStyle) for consistent background
    - lipgloss.JoinVertical for stacking content area and status bar
    - lipgloss.Place for centering spinner in content area during loading
    - tea.WithAltScreen() for clean terminal restore on exit
    - contentView() as separate method from View() for layout decomposition

key-files:
  created:
    - internal/ui/status.go
    - internal/ui/status_test.go
  modified:
    - internal/ui/model.go (View() full layout, contentView() added)
    - cmd/root.go (tea.NewProgram replaces fmt.Printf)

key-decisions:
  - "renderStatusBar is a package-level function (not method) — keeps status.go independently testable without a tea.Program"
  - "contentView() calls renderStatusBar again to compute contentH — minor double render acceptable to avoid coupling contentView to View state"
  - "stateError renders err.Error() as plain text with padding — no red color, accessible and neutral per plan spec"
  - "truncateARN uses byte len() not rune len() — ARN characters are all ASCII, so byte-safe and simpler"
  - "Version constant defined in status.go not a shared constants file — status bar is the sole consumer"

patterns-established:
  - "Package-level lipgloss styles: define as var block at top of file, use .Inherit(baseStyle) for consistent background"
  - "Full-width status bar gap pattern: gap = width - lipgloss.Width(left) - lipgloss.Width(right), guard gap < 0"
  - "contentView() method: separate from View() for single-responsibility and testability"

requirements-completed: [STAT-01, STAT-02, STAT-03, STAT-04, STAT-05, STAT-06, NAV-07]

# Metrics
duration: 2min
completed: 2026-03-03
---

# Phase 2 Plan 02: UI Shell — Status Bar, Layout, and tea.NewProgram Wire-Up Summary

**Lipgloss status bar with version/profile/region left and identity/clock right, spinner-in-content loading state, inline error rendering, and tea.NewProgram replacing fmt.Printf in cmd/root.go**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-04T00:24:47Z
- **Completed:** 2026-03-04T00:26:47Z
- **Tasks:** 2 automated (Task 3 is human-verify checkpoint)
- **Files modified:** 4

## Accomplishments

- Created `internal/ui/status.go` with renderStatusBar and truncateARN using lipgloss gap pattern
- Updated `internal/ui/model.go` View() to use lipgloss.JoinVertical(content, statusBar) full layout with contentView()
- Replaced `fmt.Printf` in `cmd/root.go` with `tea.NewProgram(model, tea.WithAltScreen())` — TUI is now the primary interface
- 11 new status bar tests (TDD RED then GREEN) added alongside 9 existing rootModel tests — all 29 pass

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Add failing status bar tests** - `9403394` (test)
2. **Task 1 GREEN: Implement internal/ui/status.go** - `9b08bae` (feat)
3. **Task 2: Wire full TUI layout and cmd/root.go** - `da84e8d` (feat)

_Note: TDD task has separate test and implementation commits. Task 3 is a human-verify checkpoint — awaiting human verification._

## Files Created/Modified

- `internal/ui/status.go` - Status bar renderer: renderStatusBar(), truncateARN(), package-level lipgloss styles
- `internal/ui/status_test.go` - 11 tests covering truncateARN and renderStatusBar for all states
- `internal/ui/model.go` - Updated View() with JoinVertical layout; added contentView() with all three states
- `cmd/root.go` - Removed fmt.Printf + direct STS call; added tea.NewProgram(model, tea.WithAltScreen())

## Decisions Made

- **renderStatusBar as package-level function**: Not a method on rootModel — keeps status.go independently testable without needing a tea.Program; receives rootModel by value
- **contentView() calls renderStatusBar again**: contentH computed independently in contentView to avoid tight coupling with View(); minor extra render acceptable for clean separation
- **Plain text error display**: stateError renders `err.Error()` with neutral padding — no ANSI red, consistent with accessible terminal design
- **truncateARN uses byte len()**: ARN characters are pure ASCII, so `len(arn)` is safe and simpler than rune counting
- **version constant in status.go**: Status bar is the only consumer; no separate constants file needed at this stage

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. Binary built at `./duchess`.

## Next Phase Readiness

- TUI shell complete: status bar renders version/profile/region/identity/clock; content area handles all three states
- `go build ./...`, `go vet ./...`, `go test ./...` all pass cleanly (29 tests)
- Human verification (Task 3 checkpoint) pending — user must run `./duchess --profile PROFILE --region us-east-1` to confirm visual behavior
- After human verification: plan 02-02 is fully complete and plan 02-03 can proceed

---
*Phase: 02-ui-shell*
*Completed: 2026-03-03*

## Self-Check: PASSED

- FOUND: internal/ui/status.go
- FOUND: internal/ui/status_test.go
- FOUND: internal/ui/model.go
- FOUND: cmd/root.go
- FOUND: .planning/phases/02-ui-shell/02-02-SUMMARY.md
- FOUND commit: 9403394 (test RED)
- FOUND commit: 9b08bae (feat GREEN)
- FOUND commit: da84e8d (feat Task 2)
