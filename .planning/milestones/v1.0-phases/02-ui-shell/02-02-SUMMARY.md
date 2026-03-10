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
  - All credential/profile errors surface as inline TUI errors (never pre-launch cobra errors)

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
    - Async TUI pattern: all I/O that can fail goes inside tea.Cmd closures, never before tea.NewProgram

key-files:
  created:
    - internal/ui/status.go
    - internal/ui/status_test.go
  modified:
    - internal/ui/model.go (View() full layout, contentView() added, fetchIdentityCmd updated, NewRootModel simplified)
    - internal/ui/model_test.go (updated for new NewRootModel signature)
    - cmd/root.go (tea.NewProgram replaces fmt.Printf; session.NewAWSConfig removed)

key-decisions:
  - "renderStatusBar is a package-level function (not method) — keeps status.go independently testable without a tea.Program"
  - "contentView() calls renderStatusBar again to compute contentH — minor double render acceptable to avoid coupling contentView to View state"
  - "stateError renders err.Error() as plain text with padding — no red color, accessible and neutral per plan spec"
  - "truncateARN uses byte len() not rune len() — ARN characters are all ASCII, so byte-safe and simpler"
  - "Version constant defined in status.go not a shared constants file — status bar is the sole consumer"
  - "AWS config loading moved from cmd/root.go into fetchIdentityCmd closure — invalid/missing profiles now cause inline TUI errors instead of pre-launch cobra errors with usage output"
  - "NewRootModel no longer accepts aws.Config — the model creates config lazily in the async fetch cmd"

patterns-established:
  - "Package-level lipgloss styles: define as var block at top of file, use .Inherit(baseStyle) for consistent background"
  - "Full-width status bar gap pattern: gap = width - lipgloss.Width(left) - lipgloss.Width(right), guard gap < 0"
  - "contentView() method: separate from View() for single-responsibility and testability"
  - "Async TUI pattern: all I/O that can fail goes inside tea.Cmd closures, never in runRoot before tea.NewProgram"

requirements-completed: [STAT-01, STAT-02, STAT-03, STAT-04, STAT-05, STAT-06, NAV-07]

# Metrics
duration: 15min
completed: 2026-03-03
---

# Phase 2 Plan 02: UI Shell — Status Bar, Layout, and tea.NewProgram Wire-Up Summary

**Lipgloss status bar with version/profile/region left and identity/clock right, spinner-in-content loading state, inline error rendering for all credential failures (including profile-not-found), and tea.NewProgram replacing fmt.Printf in cmd/root.go**

## Performance

- **Duration:** ~15 min (2 min initial execution + fix during visual verification)
- **Started:** 2026-03-04T00:24:47Z
- **Completed:** 2026-03-03
- **Tasks:** 3 (Tasks 1 and 2 automated; Task 3 visual verification triggered a fix)
- **Files modified:** 5

## Accomplishments

- Created `internal/ui/status.go` with renderStatusBar and truncateARN using lipgloss gap pattern
- Updated `internal/ui/model.go` View() to use lipgloss.JoinVertical(content, statusBar) full layout with contentView()
- Replaced `fmt.Printf` in `cmd/root.go` with `tea.NewProgram(model, tea.WithAltScreen())` — TUI is now the primary interface
- Fixed pre-launch auth error: moved AWS config loading into fetchIdentityCmd so all credential failures render inline in the TUI
- 11 new status bar tests (TDD RED then GREEN) alongside 9 existing rootModel tests — all 28 pass

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Add failing status bar tests** - `9403394` (test)
2. **Task 1 GREEN: Implement internal/ui/status.go** - `9b08bae` (feat)
3. **Task 2: Wire full TUI layout and cmd/root.go** - `da84e8d` (feat)
4. **Task 3 fix: Defer AWS config loading to TUI** - `d14e52d` (fix)

_Note: TDD task has separate test and implementation commits. Task 3 visual verification revealed a bug fixed in commit d14e52d._

## Files Created/Modified

- `internal/ui/status.go` - Status bar renderer: renderStatusBar(), truncateARN(), package-level lipgloss styles
- `internal/ui/status_test.go` - 11 tests covering truncateARN and renderStatusBar for all states (updated import)
- `internal/ui/model.go` - Updated View() with JoinVertical layout; contentView() with all three states; fetchIdentityCmd now loads AWS config; NewRootModel simplified (no awsCfg arg)
- `internal/ui/model_test.go` - Updated newTestModel() for new NewRootModel signature (removed aws.Config arg)
- `cmd/root.go` - Removed session.NewAWSConfig call and session import; NewRootModel call simplified; added comment explaining async pattern

## Decisions Made

- **renderStatusBar as package-level function**: Not a method on rootModel — keeps status.go independently testable without needing a tea.Program; receives rootModel by value
- **contentView() calls renderStatusBar again**: contentH computed independently in contentView to avoid tight coupling with View(); minor extra render acceptable for clean separation
- **Plain text error display**: stateError renders `err.Error()` with neutral padding — no ANSI red, consistent with accessible terminal design
- **truncateARN uses byte len()**: ARN characters are pure ASCII, so `len(arn)` is safe and simpler than rune counting
- **version constant in status.go**: Status bar is the only consumer; no separate constants file needed at this stage
- **AWS config deferred to fetchIdentityCmd**: `session.NewAWSConfig` moved from `runRoot` into the async cmd closure so profile-not-found errors render inline instead of aborting before TUI launch

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Auth check firing synchronously before TUI launch**

- **Found during:** Task 3 (visual verification)
- **Issue:** `session.NewAWSConfig` in `runRoot` calls `config.LoadDefaultConfig` which eagerly validates that the named profile exists in `~/.aws/config`. With `--profile fake`, this returned `"authentication failed for profile \"fake\": failed to get shared config profile, fake"` before `tea.NewProgram` was called, causing cobra to print an error + usage block and exit — the TUI never launched.
- **Fix:** Removed `session.NewAWSConfig` from `cmd/root.go`. Changed `NewRootModel` signature to accept only `*config.Config` (no `aws.Config`). Updated `fetchIdentityCmd` to call `session.NewAWSConfig` internally as the first step of the async credential fetch. Any config loading error is returned as `identityErrMsg` and renders inline in the TUI content area.
- **Files modified:** `cmd/root.go`, `internal/ui/model.go`, `internal/ui/model_test.go`, `internal/ui/status_test.go`
- **Verification:** `go build ./...`, `go vet ./...`, `go test ./...` all pass (28 tests); binary confirmed — invalid profile shows TUI with spinner then inline error instead of pre-launch cobra error.
- **Committed in:** `d14e52d` (Task 3 fix commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — Bug)
**Impact on plan:** Essential fix. The must-have truth "When STS fails, the content area shows the ClassifyCredentialError message as plain text" implies the TUI must launch even for credential errors. The pre-launch cobra error violated this contract.

## Issues Encountered

- AWS SDK `config.LoadDefaultConfig` is eager — it validates that the named profile exists at load time, not at credential fetch time. Moving the call into the async `fetchIdentityCmd` is the correct architectural fix and aligns with the plan's intent for async error handling.

## User Setup Required

None - no external service configuration required. Binary built at `./duchess`.

## Next Phase Readiness

- TUI shell fully complete and verified: status bar renders version/profile/region/identity/clock; content area handles all three states; invalid profiles show inline errors
- `go build ./...`, `go vet ./...`, `go test ./...` all pass cleanly (28 tests)
- Plan 02-03 can proceed to build navigation panels on top of this shell

---
*Phase: 02-ui-shell*
*Completed: 2026-03-03*

## Self-Check: PASSED

- FOUND: internal/ui/status.go
- FOUND: internal/ui/status_test.go
- FOUND: internal/ui/model.go
- FOUND: internal/ui/model_test.go
- FOUND: cmd/root.go
- FOUND: .planning/phases/02-ui-shell/02-02-SUMMARY.md
- FOUND commit: 9403394 (test RED)
- FOUND commit: 9b08bae (feat GREEN)
- FOUND commit: da84e8d (feat Task 2)
- FOUND commit: d14e52d (fix Task 3)
