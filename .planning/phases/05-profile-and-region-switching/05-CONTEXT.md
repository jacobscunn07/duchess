# Phase 5: Profile and Region Switching - Context

**Gathered:** 2026-03-06
**Status:** Ready for planning

<domain>
## Phase Boundary

In-session AWS profile and region switching via modal overlays. Delivers AUTH-01 (profile switch) and AUTH-05 (region switch) without restarting the app.

Specifically:
- Profile picker overlay: lists profiles from ~/.aws/config, user selects one, credentials rebuild, status bar updates
- Region picker overlay: lists AWS regions, user selects one, ECS resets, S3 stays
- Expired/invalid credential surfacing on switch (SSO token expiry, role chain failure)
- Status bar updates immediately on switch (STAT-06)

Profile enumeration, credential error taxonomy, and S3-vs-ECS isolation are already specified in plan specs (05-01 and 05-02).

</domain>

<decisions>
## Implementation Decisions

### Keybindings
- `p` opens the profile picker overlay (from any stateReady screen)
- `r` opens the region picker overlay (from any stateReady screen)
- Both overlays dismissed with `Esc` (no switch — return to previous state)
- Both overlays navigate with j/k (scroll), g/G (top/bottom), Enter (confirm selection)
- Rationale: single-letter, memorable, matches k9s-style shortcuts; `p` = profile, `r` = region; consistent with existing vim-style nav throughout the app

### Overlay visual style
- Centered floating window rendered over the existing content
- Overlay has a border (lipgloss border style, consistent with existing panel styling)
- Content behind the overlay is still rendered but visually "behind" the overlay
- Overlay shows: title ("Switch Profile" / "Switch Region"), list of options with current selection highlighted
- Current active profile/region is pre-selected (cursor starts on the currently active item)
- Width: ~50% of terminal width, height: dynamic to content (capped at 80% terminal height)
- Rationale: standard TUI overlay pattern; avoids full-screen takeover which loses context of what you're switching from

### Panel state on profile switch
- Both S3 and ECS panels fully reset to top level (bucket list / cluster list) on profile switch
- All cached data cleared (no attempt to preserve position — new profile may not have access to the same resources)
- Loading state shown in both panels immediately after switch
- STS GetCallerIdentity re-fetched for new profile; status bar shows "loading..." until identity confirmed
- In-flight goroutines from previous session cancelled via context cancellation before panel rebuild
- Rationale: profile switch means new AWS identity — cached paths are likely invalid or inaccessible

### Panel state on region switch
- ECS panel resets to cluster list for the new region (clusters are regional)
- S3 panel does NOT reset — bucket list stays (S3 is global, accessible regardless of region)
- S3 per-bucket regional clients still work correctly (they use HeadBucket to detect per-bucket region)
- Active panel after region switch stays on whatever panel was active (no forced panel jump)
- Rationale: already specified in roadmap plan specs and aligned with the S3-global / ECS-regional architecture decision from Phase 3

### Credential error UX on switch
- Each panel independently shows its own inline error if credential check fails after switch
- Error message for expired SSO token: exact `aws sso login --profile <name>` command (consistent with Phase 1 error taxonomy via ClassifyCredentialError)
- Error message for role chain expiry: identifies the source_profile by name so user knows which chain broke
- Profile picker remains accessible (`p` key) even when panels are in error state — user can switch back
- Region picker also remains accessible (`r` key) when in error state
- Rationale: user must be able to escape error state without restarting; inline per-panel errors are the established pattern (Phase 2, Phase 3, Phase 4)

### Claude's Discretion
- Exact overlay border style (rounded, normal, thick — pick consistent with existing panel borders)
- Exact overlay sizing formula
- Whether to dim/fade non-overlay content (lipgloss opacity not well-supported in all terminals; optional)
- Animation/transition on overlay open (can be instant)
- How to handle extremely long profile names (truncate with "…")
- Region list source: hardcoded list of standard AWS commercial regions (not fetched dynamically — avoids bootstrapping problem of needing credentials to list regions)
- Exact set of regions to include in region picker

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `NewAWSConfig(ctx, profile, region string)` in `internal/aws/session.go` — already supports dynamic profile/region; call this on every switch
- `ClassifyCredentialError()` in `internal/aws/session.go` — reuse for error message after failed switch (SSO token expiry, role chain failure taxonomy already implemented)
- `fetchIdentityCmd()` in `internal/ui/model.go` — re-invoke after profile/region switch to refresh account + ARN in status bar
- `s3panel.NewModel()` and `ecspanel.NewModel()` — recreate these on profile switch (full reset)
- `bubbles/list` — use for the overlay item list (same j/k navigation, same pattern as all other list views)
- `lipgloss` border styles — use for overlay window border

### Established Patterns
- Value receiver Model with `(Model, tea.Cmd)` returns — overlay models follow same pattern
- Typed message structs for all state transitions — add `profileSelectedMsg{profile string}` and `regionSelectedMsg{region string}`
- `rootModel` owns all panel state — overlay open/close state lives in rootModel, not in panels
- Key interception before `list.Update()` — overlay intercepts Esc/Enter before its list model

### Integration Points
- `rootModel` in `internal/ui/model.go` — add overlay state fields (isProfileOverlayOpen bool, isRegionOverlayOpen bool, overlayModel tea.Model or separate typed fields)
- `rootModel.Update()` — intercept `p` and `r` keys in stateReady to open overlays; handle profileSelectedMsg and regionSelectedMsg to rebuild sessions
- `rootModel.View()` — render overlay on top of existing content when overlay is open
- Status bar already reads from `m.cfg.Profile` and `m.cfg.Region` — updating these fields auto-updates the status bar on next render

### Context Cancellation
- Currently: rootModel stores a single `ctx context.Context` with no cancel
- Phase 5 needs: per-session `context.WithCancel()` so in-flight goroutines from previous session are cancelled on switch
- Pattern: store `cancelSession context.CancelFunc` in rootModel; call it before rebuilding panels; create new cancelable context for the new session

</code_context>

<specifics>
## Specific Ideas

- The profile picker should feel like switching namespaces in k9s — fast, modal, stays in-terminal
- Current profile/region should be clearly indicated in the overlay (pre-selected + some visual marker) so user sees where they are before switching
- The `p` and `r` keys should be discoverable — add them to any keybinding hint display

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 05-profile-and-region-switching*
*Context gathered: 2026-03-06*
