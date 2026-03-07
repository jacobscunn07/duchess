---
status: resolved
trigger: "Profile overlay shows only 1 profile per page instead of all profiles on one page"
created: 2026-03-06T00:00:00Z
updated: 2026-03-07T00:00:00Z
---

## Current Focus

hypothesis: list height passed to list.New is too small; bubbles/list internal chrome (help bar + pagination placeholder) consumes all available height leaving PerPage=1
test: traced updatePagination() arithmetic with overlayH=8, list height=4
expecting: confirmed — PerPage resolves to 1 with 2 items, producing 2 pages
next_action: DONE — root cause confirmed, diagnose-only mode

## Symptoms

expected: With 2 AWS profiles, both appear on one page in the profile overlay (no pagination)
actual: Only 1 profile shown per page; 2 pages created; user must scroll to see second profile
errors: none
reproduction: Press 'p' in the app with 2 profiles in ~/.aws/config
started: Discovered during UAT of phase 05

## Eliminated

- hypothesis: profileDelegate.Height() returns wrong value
  evidence: profileDelegate.Height() returns 1 (correct, confirmed in source)
  timestamp: 2026-03-06

- hypothesis: SetShowTitle(false) removes a row from the list but title is still rendered inside the list
  evidence: title is rendered OUTSIDE the list via lipgloss.JoinVertical in View(); showTitle=false correctly removes the list's internal title row
  timestamp: 2026-03-06

## Evidence

- timestamp: 2026-03-06
  checked: NewProfileOverlay() height calculation in internal/ui/overlay/profile.go lines 95-110
  found: overlayH = len(profiles) + 6 = 2 + 6 = 8; list height passed to list.New = overlayH - 4 = 4
  implication: the list gets height=4 for 2 items

- timestamp: 2026-03-06
  checked: bubbles/list list.go updatePagination() (line 780-813, v1.0.0)
  found: |
    availHeight starts at m.height (4)
    showTitle=false -> no deduction
    showStatusBar=false -> no deduction
    showPagination=true -> deducts lipgloss.Height(paginationView())
      paginationView() returns "" when TotalPages < 2 (line 1191)
      on construction TotalPages=0, so returns ""
      lipgloss.Height("") = 1 (empty string still counts as one line)
      availHeight = 4 - 1 = 3
    showHelp=true -> deducts lipgloss.Height(helpView())
      HelpStyle has Padding(1, 0, 0, 2) — padding-top of 1 (line 84 of style.go)
      help content is 1 line + 1 line padding = 2 lines total
      availHeight = 3 - 2 = 1
    PerPage = max(1, 1 / (1 + 0)) = 1
  implication: PerPage=1 with 2 items => 2 pages; pagination kicks in

- timestamp: 2026-03-06
  checked: which list chrome flags are disabled in NewProfileOverlay()
  found: SetShowTitle(false), SetShowStatusBar(false), SetFilteringEnabled(false), DisableQuitKeybindings() — but showPagination and showHelp remain at their defaults of TRUE
  implication: the two largest consumers of height (help bar at 2 rows, pagination placeholder at 1 row) are left enabled, eating 3 of the 4 available rows

## Resolution

root_cause: |
  NewProfileOverlay passes list height = overlayH - 4 = 4 (for 2 profiles).
  bubbles/list.updatePagination() deducts chrome height before computing PerPage:
    - showPagination=true: lipgloss.Height("") = 1 row (pagination placeholder even when empty)
    - showHelp=true: HelpStyle Padding(1,0,0,2) = 2 rows
  Total chrome = 3 rows. availHeight = 4 - 3 = 1. PerPage = max(1, 1/1) = 1.
  With PerPage=1 and 2 profiles, bubbles creates 2 pages.

fix: not applied (diagnose-only mode)
verification: not applied
files_changed: []
