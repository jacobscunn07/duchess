# Phase 3: S3 Browsing - Context

**Gathered:** 2026-03-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Full S3 navigation — bucket list to prefix hierarchy to object metadata — with live refresh, breadcrumb header, prefix filter, and paginated API calls that never truncate results.

Delivers:
- S3 client with ListBuckets, per-bucket regional client via HeadBucket, and ListObjectsV2 (fully paginated)
- Three-state panel: BucketList → PrefixList → ObjectDetail
- Breadcrumb header at the top of the content area (first use of the top header reserved in Phase 2)
- Vim-style navigation throughout: j/k scroll, g/G top/bottom, Enter descend, Esc ascend, / filter
- Live refresh ticker (30s default for S3 — slower than the global 2s default since S3 buckets don't change often)

Profile and region switching is Phase 5. ECS is Phase 4.

</domain>

<decisions>
## Implementation Decisions

### Panel architecture
- Three states: BucketList → PrefixList → ObjectDetail
- Each state is a distinct view in the rootModel's content area (same Bubble Tea pattern as Phase 2's stateLoading/stateReady/stateError)
- BucketList and PrefixList use `bubbles/list` component (scrollable, keyboard-driven)
- ObjectDetail is a read-only metadata pane that replaces the list — Enter on an object shows the detail pane, Esc returns to the prefix list
- No side-panel split; full-width content area throughout (consistent with k9s approach)

### Breadcrumb header
- Single line at the very top of the content area (below nothing — leaves status bar at bottom per Phase 2)
- Format: `S3 > bucket-name > prefix/subprefix/` — plain text with ` > ` separator
- At bucket list level: just `S3`
- Updates at every navigation level; truncates from left if path is long (show `... > prefix/` rather than wrapping)
- Breadcrumb is part of the content area, not a separate lipgloss fixed header — subtract its height from available list height

### List columns
- **Bucket list**: Name only (one column) — S3 doesn't expose bucket region in ListBuckets; HeadBucket is called lazily for regional client construction, not for display
- **Prefix list**: Two-column display
  - Prefixes (folders): name with trailing `/`, no size/date (they're virtual, not real objects)
  - Objects: key name + size (human-readable: KB/MB/GB) + last modified date (relative: "2d ago" or absolute if >1 week)
- **Object detail pane**: Full metadata — key, size (exact bytes + human-readable), last modified (full ISO timestamp), storage class, ETag

### Filter behavior
- `/` key enters filter mode — a one-line input bar appears at the bottom of the content area (above the status bar), similar to vim's `/` search bar
- Filter uses server-side ListObjectsV2 prefix: appends the typed string to the current prefix and re-fetches
- Filter updates on Enter (not as-you-type) — avoids excessive API calls
- Esc cancels filter and returns to unfiltered view; the previously fetched results are shown
- Filter bar shows placeholder text: `Filter: ` with cursor

### Navigation keybindings
- `j`/`k` — scroll up/down in list (NAV-01)
- `g` — jump to top (NAV-02)
- `G` — jump to bottom (NAV-02)
- `Enter` — descend into bucket or prefix, or open object detail
- `Esc` — ascend one level (NAV-03); from bucket list, does nothing (already at root)
- `/` — enter filter mode
- `q` — quit from any view (carried from Phase 2)

### Refresh behavior
- S3 refresh ticker: 30s default (not the global 2s — S3 bucket lists are stable; over-polling is wasteful)
- The global `--refresh-interval` config flag does NOT override S3's 30s — S3 has its own cadence
- Refresh only re-fetches the current level (not the full path) — navigating to a prefix does not re-fetch the bucket list
- Refresh restarts when the user navigates to a new level
- A subtle loading indicator (spinner in the breadcrumb line or top-right corner) shows when a refresh is in progress without clearing the visible list

### Error handling
- S3 API errors display inline in the content area (same neutral-color plain text pattern from Phase 2)
- Per-bucket errors (e.g., access denied on a specific bucket) show as an inline error row in the bucket list — not a full-screen error
- ListBuckets failure (can't fetch any buckets) replaces the list with the inline error view

### Claude's Discretion
- Exact lipgloss styling for the breadcrumb line (color, padding)
- Spinner placement during background refresh
- Relative time formatting thresholds (when to switch from "Xd ago" to full date)
- HeadBucket retry/fallback strategy when per-bucket regional client construction fails
- Exact column widths and padding in the prefix list
- How to handle very long object keys in the list (truncation strategy)

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/aws/session.go`: `NewAWSConfig(ctx, profile, region)` — used to construct per-bucket regional AWS clients (same function, different region parameter from HeadBucket response)
- `internal/aws/session.go`: `ClassifyCredentialError(err, profile)` — reuse for S3 API errors in the panel
- `internal/ui/model.go`: `rootModel` state machine (stateLoading/stateReady/stateError) and `contentView()` — S3 panel states extend this pattern
- `internal/ui/model.go`: `fetchIdentityCmd` pattern — async `tea.Cmd` returning a typed message — use the same pattern for S3 API calls
- `internal/ui/messages.go`: Message type pattern — define typed messages for S3 events (bucketsLoadedMsg, prefixesLoadedMsg, objectDetailLoadedMsg)
- `internal/ui/model.go`: `tickCmd()` — same pattern for S3 refresh ticker, just 30s interval instead of 1s

### Established Patterns
- Value receivers on model structs (rootModel uses value receivers — standard Bubble Tea convention)
- Async tea.Cmd for all API calls: never block the Update loop
- Inline error rendering: `lipgloss.NewStyle().Width(m.width).Padding(1, 2).Render(err.Error())` — neutral color, no red
- NO_COLOR guard already in place at app startup — all new lipgloss styles automatically respect it
- Package naming: keep `internal/aws` package named "session" (imported with alias) — do not rename

### Integration Points
- `rootModel.contentView()` in `internal/ui/model.go`: currently returns `""` in stateReady — Phase 3 replaces this with the S3 panel render
- New `internal/ui/s3/` (or `internal/s3/`) package for S3-specific model, messages, and client wrapper
- `rootModel` needs a new field for the active S3 model and a way to dispatch S3 messages to it — consider embedding the S3 panel as a child model via Bubble Tea's component pattern
- The existing `tea.Batch` in `Init()` will need S3's initial fetch command added when state transitions to stateReady

</code_context>

<specifics>
## Specific Ideas

- k9s as visual reference: clean table/list with no decorative chrome; breadcrumb feels like a terminal path
- Object list should feel like `ls -lh` output — names on the left, size and date on the right, aligned in columns
- Filter mode should feel like vim's `/` — the list narrows as a result of the filter, not a separate search view

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 03-s3-browsing*
*Context gathered: 2026-03-04*
