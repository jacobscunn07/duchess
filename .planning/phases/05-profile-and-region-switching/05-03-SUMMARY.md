---
phase: 05-profile-and-region-switching
plan: 03
subsystem: ui/overlay
tags: [gap-closure, uat, profile-overlay, p-key, chrome, pagination]
dependency_graph:
  requires: []
  provides: [UAT-Test1-fix, UAT-Test4-fix]
  affects: [internal/ui/overlay/profile.go, internal/ui/model.go]
tech_stack:
  added: []
  patterns: [bubbles/list chrome flags, explicit error branching]
key_files:
  created: []
  modified:
    - internal/ui/overlay/profile.go
    - internal/ui/model.go
decisions:
  - "SetShowPagination(false) + SetShowHelp(false) in NewProfileOverlay eliminates 3 chrome rows that forced PerPage=1 with only 2 profiles"
  - "'p' key failure modes split into two explicit branches (err != nil vs len==0) each with fmt.Errorf messages rendered in stateError content area"
metrics:
  duration: 1 min
  completed_date: "2026-03-07"
  tasks_completed: 2
  files_modified: 2
---

# Phase 05 Plan 03: Gap Closure — Profile Overlay Chrome and p-Key Error Visibility Summary

**One-liner:** Disabled bubbles/list pagination and help chrome in ProfileOverlay and split the p-key guard into two explicit fmt.Errorf branches to eliminate a silent no-op.

## What Was Built

Two targeted one-or-two-line edits closing UAT gaps from phase 05:

**Task 1 (UAT Test 1 — Minor):** `NewProfileOverlay` left `showPagination` and `showHelp` enabled. bubbles/list.updatePagination() deducts chrome rows (help=2, pagination placeholder=1) from the available list height before computing PerPage. With a list height of 4 and 3 chrome rows consumed, PerPage was forced to 1 — pushing 2 profiles onto 2 pages. Adding `l.SetShowPagination(false)` and `l.SetShowHelp(false)` (grouped with the existing chrome-disable calls, before `l.Select`) eliminates all chrome overhead so all profiles fit on one page.

**Task 2 (UAT Test 4 — Major):** The `case "p":` handler had a single-branch guard `if err != nil || len(profiles) == 0 { return m, nil }` that silently swallowed both failure modes. The user pressing 'p' in an error state saw no feedback and had no escape path. The guard was replaced with two explicit `if` blocks: one for `err != nil` (sets `m.err = fmt.Errorf("could not read ~/.aws/config: %w", err)`) and one for `len(profiles) == 0` (sets `m.err = fmt.Errorf("no profiles found in ~/.aws/config — add a [profile ...] section")`). Both set `m.state = stateError` so the message renders in the existing stateError content view with no new rendering paths required. `fmt` import added to model.go.

## Commits

| Task | Commit  | Message |
|------|---------|---------|
| 1    | 0d9097e | fix(05-03): disable list chrome in NewProfileOverlay |
| 2    | 091a120 | fix(05-03): surface visible error when ListAWSProfiles fails on 'p' key |

## Deviations from Plan

None - plan executed exactly as written.

## Decisions Made

- `SetShowPagination(false)` and `SetShowHelp(false)` placed after existing chrome-disable calls, before `l.Select(selectedIdx)` — grouped for readability, no functional ordering dependency
- Two separate `if` branches (not `else if`) for err and empty-slice — each is an independent failure mode with its own human-readable message
- `fmt` import added alphabetically in the import block

## Self-Check: PASSED

- FOUND: internal/ui/overlay/profile.go
- FOUND: internal/ui/model.go
- FOUND commit: 0d9097e (Task 1)
- FOUND commit: 091a120 (Task 2)
