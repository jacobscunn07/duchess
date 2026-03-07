---
status: diagnosed
phase: 05-profile-and-region-switching
source: 05-01-SUMMARY.md, 05-02-SUMMARY.md
started: 2026-03-06T00:00:00Z
updated: 2026-03-06T00:01:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Open Profile Overlay
expected: Press 'p' while the app is running. A centered overlay appears listing all profiles from your ~/.aws/config file. The overlay has a rounded border with a "Switch Profile" title.
result: issue
reported: "It works, but it only shows one profile on each page. I have two profiles in my aws config and there are two pages. There should be enough room for them to both fit on one page."
severity: minor

### 2. Switch Profile
expected: With the profile overlay open, select a different profile and press Enter. The overlay closes, the status bar immediately shows the new profile name while loading, then confirms identity under the new profile.
result: pass

### 3. Escape Closes Profile Overlay
expected: With the profile overlay open, press Esc. The overlay closes and the app returns to its previous state — no profile switch occurs.
result: pass

### 4. Open Profile Overlay in Error State
expected: When the app is showing an error (e.g., credentials expired), press 'p'. The profile overlay opens. Selecting a new profile dismisses the error and starts a fresh loading cycle under the new profile.
result: issue
reported: "I opened with a profile that does not have access to s3, but the profile selector did not open."
severity: major

### 5. Open Region Overlay
expected: Press 'r' while the app is running. A centered overlay appears listing all 30 AWS commercial regions (e.g., us-east-1, us-west-2, eu-west-1, etc.) with the current region pre-selected/highlighted.
result: pass

### 6. Switch Region
expected: With the region overlay open, select a different region and press Enter. The overlay closes, the status bar immediately shows the new region, and the ECS panel reloads for the new region. The S3 panel is NOT reloaded (bucket list is global).
result: pass

### 7. Escape Closes Region Overlay
expected: With the region overlay open, press Esc. The overlay closes and the app returns to its previous state — no region switch occurs.
result: pass

## Summary

total: 7
passed: 5
issues: 2
pending: 0
skipped: 0

## Gaps

- truth: "Profile overlay shows all profiles on one page without pagination"
  status: failed
  reason: "User reported: It works, but it only shows one profile on each page. I have two profiles in my aws config and there are two pages. There should be enough room for them to both fit on one page."
  severity: minor
  test: 1
  root_cause: "NewProfileOverlay does not call l.SetShowPagination(false) or l.SetShowHelp(false); the 3 chrome rows (help=2, pagination=1) consume all available height leaving PerPage=1 for 2 profiles"
  artifacts:
    - path: "internal/ui/overlay/profile.go"
      issue: "NewProfileOverlay() missing l.SetShowPagination(false) and l.SetShowHelp(false) calls after list creation"
  missing:
    - "Add l.SetShowPagination(false) and l.SetShowHelp(false) in NewProfileOverlay()"
  debug_session: ".planning/debug/profile-overlay-pagination.md"

- truth: "'p' key opens profile overlay in error state so user can escape by switching profiles"
  status: failed
  reason: "User reported: I opened with a profile that does not have access to s3, but the profile selector did not open."
  severity: major
  test: 4
  root_cause: "S3 access-denied errors are trapped in s3panel.Model.err and do not set rootModel.state to stateError — rootModel stays in stateReady. The 'p' guard covers stateReady so key routing is not the issue. The silent guard (lines 176-179) returns nil with no feedback if ListAWSProfiles() fails or returns no profiles, swallowing the keypress invisibly."
  artifacts:
    - path: "internal/ui/model.go"
      issue: "Silent return m, nil at lines 176-179 when ListAWSProfiles() errors or returns empty — no user feedback explaining why overlay did not open"
  missing:
    - "Replace silent return with visible error feedback (status bar message or m.err) when ListAWSProfiles fails"
    - "Investigate why ListAWSProfiles() may fail or return empty in the error-profile scenario"
  debug_session: ".planning/debug/p-key-error-state.md"
