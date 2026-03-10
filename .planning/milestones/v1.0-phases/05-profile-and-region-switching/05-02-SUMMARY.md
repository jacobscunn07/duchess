---
phase: 05-profile-and-region-switching
plan: 02
subsystem: ui-region-switching
tags: [region, overlay, context-cancellation, ecs, s3, bubbles]
dependency_graph:
  requires: [05-01]
  provides: [region-picker-overlay, region-switching, context-canceled-guards]
  affects: [internal/ui/overlay, internal/ui/model.go, internal/ui/s3/model.go, internal/ui/ecs/model.go]
tech_stack:
  added: []
  patterns: [bubbles-list-overlay, context-cancellation-guard, tea-composite-overlay]
key_files:
  created:
    - internal/ui/overlay/region.go
  modified:
    - internal/ui/messages.go
    - internal/ui/model.go
    - internal/ui/s3/model.go
    - internal/ui/ecs/model.go
decisions:
  - "r key opens region overlay in both stateReady and stateError — consistent with p key behavior for profile switching"
  - "S3 panel NOT reset on region switch — bucket list is global, ECS is region-specific"
  - "m.awsCfg.Region updated before ecspanel.NewModel call — critical ordering to avoid wrong-region ECS client"
  - "context.Canceled guard placed FIRST in each error handler before any field mutations — prevents stale state"
  - "regionDelegate uses color 62 (same as border) for selected item — visual cohesion"
metrics:
  duration: 2 minutes
  completed_date: "2026-03-06"
  tasks_completed: 2
  files_changed: 5
---

# Phase 05 Plan 02: Region Picker Overlay and context.Canceled Guards Summary

**One-liner:** Region picker overlay with bubbles/list using hardcoded 30-region list, r-key wiring into rootModel cancelling in-flight goroutines and rebuilding only ECS panel, plus context.Canceled silently dropped in all 7 panel error handlers.

## Tasks Completed

| Task | Description | Commit | Files |
|------|-------------|--------|-------|
| 1 | Create internal/ui/overlay/region.go | 6cc0419 | internal/ui/overlay/region.go |
| 2 | Wire region overlay into rootModel; add context.Canceled guards | ba65fa2 | messages.go, model.go, s3/model.go, ecs/model.go |

## What Was Built

### Task 1: RegionOverlay model

Created `internal/ui/overlay/region.go` in package `overlay`, mirroring `profile.go` exactly:

- `awsRegions` package-level var with all 30 standard AWS commercial regions
- `regionItem` struct implementing `list.Item` (FilterValue, Title, Description)
- `regionDelegate` single-line delegate rendering selected item bold with color 62
- `RegionOverlay` struct with `list.Model`, `borderStyle`, `width`, `height`
- `NewRegionOverlay(currentRegion string, termWidth, termHeight int) RegionOverlay` — pre-selects current region, overlayW = termWidth/2 (min 30), overlayH = regions+6 (capped at 4/5 terminal height)
- `Init()`, `Update()`, `View()` (rounded border, bold "Switch Region" title), `SelectedRegion()` (nil-guarded)

### Task 2: rootModel wiring + context.Canceled guards

**messages.go:** Added `regionSelectedMsg{region string}`.

**model.go:**
- Added `isRegionOverlayOpen bool` and `regionOverlay overlay.RegionOverlay` to `rootModel`
- Added region overlay key intercept block after profile overlay block (esc/enter/default, all return before panel fallthrough)
- Added `r` key handler opening `RegionOverlay` with `m.cfg.Region` as current region in stateReady or stateError
- Added `regionSelectedMsg` handler: cancels session, creates new sessionCtx, updates `m.cfg.Region` and `m.awsCfg.Region`, rebuilds only ECS panel (S3 intentionally unchanged)
- Updated `View()` to composite `regionOverlay.View()` over `baseView()` when `isRegionOverlayOpen`

**s3/model.go:** Added `errors` import; added `errors.Is(msg.err, context.Canceled)` guard (returns early) at top of:
- `bucketsErrMsg`
- `bucketClientErrMsg`
- `prefixesErrMsg`

**ecs/model.go:** Added `errors` import; added `errors.Is(msg.err, context.Canceled)` guard (returns early) at top of:
- `clustersErrMsg`
- `servicesErrMsg`
- `tasksErrMsg`
- `taskDetailErrMsg`

## Verification

```
go build ./...        → PASS
go test ./internal/ui/... -count=1 → PASS (all existing tests)
```

Success criteria confirmed:
- [x] `go build ./...` passes
- [x] `go test ./internal/ui/... -count=1` passes
- [x] r key opens region overlay in stateReady and stateError
- [x] Esc closes without switching region
- [x] Enter fires regionSelectedMsg; rebuilds only ECS panel (not S3)
- [x] m.cfg.Region and m.awsCfg.Region both updated before ecspanel.NewModel call
- [x] status bar shows new region immediately after regionSelectedMsg
- [x] context.Canceled guard present in all 7 panel error handlers
- [x] 30 AWS commercial regions in awsRegions list
- [x] AUTH-05 requirement satisfied

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

Files confirmed present:
- internal/ui/overlay/region.go: FOUND
- internal/ui/messages.go: FOUND (regionSelectedMsg added)
- internal/ui/model.go: FOUND (isRegionOverlayOpen, regionSelectedMsg handler, View() update)
- internal/ui/s3/model.go: FOUND (3 context.Canceled guards)
- internal/ui/ecs/model.go: FOUND (4 context.Canceled guards)

Commits confirmed:
- 6cc0419: feat(05-02): create region picker overlay model
- ba65fa2: feat(05-02): wire region overlay into rootModel; add context.Canceled guards
