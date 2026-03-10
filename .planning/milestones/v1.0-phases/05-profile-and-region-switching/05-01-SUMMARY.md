---
phase: 05-profile-and-region-switching
plan: "01"
subsystem: ui
tags: [overlay, profile-switching, context-cancellation, bubbletea-overlay, ini.v1, lipgloss]

# Dependency graph
requires:
  - phase: 02-ui-shell
    provides: rootModel, status bar rendering, stateLoading/stateReady/stateError lifecycle
  - phase: 04-ecs-browsing
    provides: dual-panel layout with s3Panel and ecsPanel, tab switching
provides:
  - ProfileOverlay tea.Model in internal/ui/overlay/profile.go
  - ListAWSProfiles() reading ~/.aws/config via ini.LooseLoad
  - profileSelectedMsg typed message for profile switch lifecycle
  - rootModel.sessionCtx / cancelSession for per-session goroutine lifecycle
  - p-key handler opening overlay in stateReady and stateError
  - btoverlay.Composite overlaying profile picker on baseView()
  - profileSelectedMsg handler: cancel session, new context, reset panels, re-fetch identity
affects: [05-02-region-switching, future-sso-reauthentication]

# Tech tracking
tech-stack:
  added:
    - gopkg.in/ini.v1 v1.67.1 — AWS config file parsing
    - github.com/rmhubbert/bubbletea-overlay v0.6.5 — overlay compositing
  patterns:
    - Per-session context cancellation: sessionCtx/cancelSession fields on rootModel, cancelled on profile switch
    - Overlay interception pattern: guard at top of tea.KeyMsg case returns early before panel forwarding fallthrough
    - baseView() helper extracts background rendering so View() can composite without recursion

key-files:
  created:
    - internal/ui/overlay/profile.go
  modified:
    - internal/ui/messages.go
    - internal/ui/model.go
    - go.mod
    - go.sum

key-decisions:
  - "profileDelegate.Render uses io.Writer (not *strings.Builder) — bubbles v1.0.0 API requires io.Writer"
  - "p-key opens overlay in both stateReady and stateError — user must be able to escape error state by switching profiles"
  - "m.cfg.Profile updated immediately on profileSelectedMsg before identity confirms — status bar shows new profile during loading"
  - "cancelSession called before creating new sessionCtx — all in-flight goroutines from previous session are cancelled atomically"
  - "baseView() extracted from View() — prevents recursive View() call when compositing overlay background"

patterns-established:
  - "Per-session context: rootModel.sessionCtx is the context passed to panels and fetchIdentityCmd; root ctx is only parent"
  - "Overlay key interception: guarded block at top of tea.KeyMsg handler returns early before existing key handling"

requirements-completed: [AUTH-01]

# Metrics
duration: 3min
completed: 2026-03-06
---

# Phase 5 Plan 1: Profile Overlay and Session Context Summary

**Centered profile picker overlay with per-session context cancellation — pressing p lists all ~/.aws/config profiles, Enter switches with full goroutine lifecycle management via context cancellation**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-06T17:56:16Z
- **Completed:** 2026-03-06T17:59:25Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Built ProfileOverlay using bubbles/list with profileDelegate (io.Writer Render API), ListAWSProfiles() via ini.LooseLoad, and btoverlay.Composite for centering
- Added sessionCtx/cancelSession to rootModel so profile switch cancels all in-flight S3/ECS goroutines before starting a new session
- Wired complete p-key -> overlay -> profileSelectedMsg -> stateLoading -> fetchIdentityCmd lifecycle in model.go with status bar immediately showing new profile

## Task Commits

Each task was committed atomically:

1. **Task 1: Add dependencies and create overlay/profile.go** - `4f9f722` (feat)
2. **Task 2: Wire overlay into rootModel with per-session context** - `cf42cf9` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified
- `internal/ui/overlay/profile.go` - ProfileOverlay, NewProfileOverlay, ListAWSProfiles, profileDelegate, profileItem
- `internal/ui/messages.go` - Added profileSelectedMsg struct
- `internal/ui/model.go` - sessionCtx/cancelSession fields, p-key handler, overlay key interception, profileSelectedMsg handler, baseView() helper, updated View() with btoverlay.Composite
- `go.mod` / `go.sum` - Added gopkg.in/ini.v1 v1.67.1, github.com/rmhubbert/bubbletea-overlay v0.6.5

## Decisions Made
- `profileDelegate.Render` uses `io.Writer` not `*strings.Builder` — bubbles v1.0.0 API requires this; plan spec had wrong type
- `p` key opens overlay in both `stateReady` and `stateError` so users can escape error state by switching profiles without restarting
- `m.cfg.Profile` updated immediately on `profileSelectedMsg` before `identityLoadedMsg` arrives — status bar shows new profile during the loading phase
- `cancelSession()` called before creating new `sessionCtx` — atomic cancellation of all in-flight goroutines

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed profileDelegate.Render signature**
- **Found during:** Task 1 (build verification)
- **Issue:** Plan specified `func (d profileDelegate) Render(w *strings.Builder, ...)` but bubbles v1.0.0 `list.ItemDelegate` interface requires `io.Writer`
- **Fix:** Changed `w *strings.Builder` to `w io.Writer` and `w.WriteString(...)` to `fmt.Fprint(w, ...)` — added `fmt` and `io` imports
- **Files modified:** internal/ui/overlay/profile.go
- **Verification:** `go build ./internal/ui/overlay/...` passes
- **Committed in:** 4f9f722 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — API signature mismatch)
**Impact on plan:** Required fix for compilation. No scope creep; purely a type correction.

## Issues Encountered
- Shell CWD differed from project root on first `go get` calls — re-ran `go get` and `go mod tidy` with absolute paths to ensure go.mod was updated correctly

## User Setup Required
None - no external service configuration required. Profile switching reads ~/.aws/config which is managed by the AWS CLI.

## Next Phase Readiness
- ProfileOverlay is complete and composited correctly; AUTH-01 (profile switching) delivered
- Phase 5 Plan 2 (region switching) can build on the same overlay pattern and sessionCtx cancellation approach
- SSO re-auth edge cases deferred to research spike before production hardening

---
*Phase: 05-profile-and-region-switching*
*Completed: 2026-03-06*
