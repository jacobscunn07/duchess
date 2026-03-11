---
phase: 07-header
plan: 02
subsystem: ui
tags: [tui, lipgloss, header, layout, bubbletea]

# Dependency graph
requires:
  - phase: 07-01
    provides: renderHeader() and headerHeight=6 from header.go
  - phase: 06-darktheme-package
    provides: theme.DefaultTheme consumed by renderHeader

provides:
  - Header rendered at top of every TUI screen via baseView()
  - All four contentH calculation sites subtract headerHeight
  - Panel heights correctly reduced so content does not overflow terminal

affects:
  - 08-footer (footerHeight analogous to headerHeight — same integration pattern)
  - 09-two-panel-layout (bodyHeight math uses headerHeight + footerHeight)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "baseView() assembles full layout via JoinVertical(Left, header, content, statusBar)"
    - "All contentH calculations follow: m.height - headerHeight - lipgloss.Height(statusBar)"

key-files:
  created: []
  modified:
    - internal/ui/model.go

key-decisions:
  - "No guard needed in baseView() for contentH < 1: existing < 0 guard is sufficient since header is always present"
  - "contentView() retains <= 0 guard (not < 1) for spinner fallback — guard semantics unchanged from original"

patterns-established:
  - "Any new zone height constant (footerHeight, navWidth) must be subtracted from contentH at ALL four sites in model.go"

requirements-completed:
  - LAYOUT-01
  - LAYOUT-02
  - LAYOUT-03

# Metrics
duration: 5min
completed: 2026-03-11
---

# Phase 7 Plan 02: Header Integration Summary

**renderHeader() wired into model.go baseView() and all four contentH sites subtract headerHeight so panels never overflow the terminal**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-03-11T18:50:00Z
- **Completed:** 2026-03-11T18:55:00Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments

- baseView() now calls renderHeader(m, m.width) and prepends header in JoinVertical(Left, header, content, statusBar)
- All four contentH calculation sites in model.go subtract headerHeight — no orphaned old pattern remains
- go build ./... exits 0; go test ./... passes (all 4 test packages pass)

## Task Commits

Each task was committed atomically:

1. **Task 1: Update baseView() to render and prepend header** - `3d56dd4` (feat)
2. **Task 2: Update contentView() and both message handler contentH calculations** - `4e6a9d3` (feat)

**Plan metadata:** committed with docs commit (07-02 complete)

## Files Created/Modified

- `internal/ui/model.go` — Four targeted edits: baseView() + contentView() + identityLoadedMsg handler + regionSelectedMsg handler

## Decisions Made

- Retained the existing `< 0` guard in baseView() unchanged — no need to tighten to `< 1` since the header is always rendered above content and the guard prevents negative dimensions.
- Retained the `<= 0` guard in contentView() unchanged — semantics match original (spin fallback fires on zero-height too).

## Deviations from Plan

None — plan executed exactly as written. All four edits matched the specified before/after diffs precisely.

## Integration Verification (Post-completion)

| Check | Expected | Actual |
|-------|----------|--------|
| `grep -c "renderHeader" internal/ui/model.go` | 1 | 1 |
| `grep -c "headerHeight" internal/ui/model.go` | 4 | 4 |
| Old pattern `m.height - lipgloss.Height(statusBar)` | 0 matches | 0 matches |
| `go build ./...` | exit 0 | exit 0 |
| `go test ./...` | all pass | all pass |

## Phase 7 Requirements Coverage

- **LAYOUT-01:** Every screen shows the persistent header at top — satisfied (baseView prepends header)
- **LAYOUT-02:** Panel heights reduced by headerHeight — satisfied (all four contentH sites subtract headerHeight)
- **LAYOUT-03:** No content overflows terminal — satisfied (contentH guard prevents negative/zero heights)

## Issues Encountered

None.

## Next Phase Readiness

- Phase 8 (Footer) can now proceed — footerHeight integration will follow the identical pattern (subtract from contentH at the same four sites, render in baseView JoinVertical)
- Phase 9 (Two-Panel Layout) needs both headerHeight and footerHeight before bodyHeight() math is correct

---
*Phase: 07-header*
*Completed: 2026-03-11*
