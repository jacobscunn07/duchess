# Phase 3: S3 Browsing - Research

**Researched:** 2026-03-04
**Domain:** AWS S3 API + Bubble Tea list component + TUI state machine composition
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Panel architecture**
- Three states: BucketList → PrefixList → ObjectDetail
- Each state is a distinct view in the rootModel's content area (same Bubble Tea pattern as Phase 2's stateLoading/stateReady/stateError)
- BucketList and PrefixList use `bubbles/list` component (scrollable, keyboard-driven)
- ObjectDetail is a read-only metadata pane that replaces the list — Enter on an object shows the detail pane, Esc returns to the prefix list
- No side-panel split; full-width content area throughout (consistent with k9s approach)

**Breadcrumb header**
- Single line at the very top of the content area (below nothing — leaves status bar at bottom per Phase 2)
- Format: `S3 > bucket-name > prefix/subprefix/` — plain text with ` > ` separator
- At bucket list level: just `S3`
- Updates at every navigation level; truncates from left if path is long (show `... > prefix/` rather than wrapping)
- Breadcrumb is part of the content area, not a separate lipgloss fixed header — subtract its height from available list height

**List columns**
- **Bucket list**: Name only (one column) — S3 doesn't expose bucket region in ListBuckets; HeadBucket is called lazily for regional client construction, not for display
- **Prefix list**: Two-column display
  - Prefixes (folders): name with trailing `/`, no size/date (they're virtual, not real objects)
  - Objects: key name + size (human-readable: KB/MB/GB) + last modified date (relative: "2d ago" or absolute if >1 week)
- **Object detail pane**: Full metadata — key, size (exact bytes + human-readable), last modified (full ISO timestamp), storage class, ETag

**Filter behavior**
- `/` key enters filter mode — a one-line input bar appears at the bottom of the content area (above the status bar), similar to vim's `/` search bar
- Filter uses server-side ListObjectsV2 prefix: appends the typed string to the current prefix and re-fetches
- Filter updates on Enter (not as-you-type) — avoids excessive API calls
- Esc cancels filter and returns to unfiltered view; the previously fetched results are shown
- Filter bar shows placeholder text: `Filter: ` with cursor

**Navigation keybindings**
- `j`/`k` — scroll up/down in list (NAV-01)
- `g` — jump to top (NAV-02)
- `G` — jump to bottom (NAV-02)
- `Enter` — descend into bucket or prefix, or open object detail
- `Esc` — ascend one level (NAV-03); from bucket list, does nothing (already at root)
- `/` — enter filter mode
- `q` — quit from any view (carried from Phase 2)

**Refresh behavior**
- S3 refresh ticker: 30s default (not the global 2s — S3 bucket lists are stable; over-polling is wasteful)
- The global `--refresh-interval` config flag does NOT override S3's 30s — S3 has its own cadence
- Refresh only re-fetches the current level (not the full path) — navigating to a prefix does not re-fetch the bucket list
- Refresh restarts when the user navigates to a new level
- A subtle loading indicator (spinner in the breadcrumb line or top-right corner) shows when a refresh is in progress without clearing the visible list

**Error handling**
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

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| S3-01 | User can view a list of all S3 buckets accessible by the current profile | ListBuckets API with paginator; `s3.NewListBucketsPaginator`; global endpoint returns all regions |
| S3-02 | User can navigate into a bucket to browse object prefixes (Enter to descend) | ListObjectsV2 with Delimiter="/"; CommonPrefixes = virtual folders; per-bucket regional S3 client via HeadBucket BucketRegion |
| S3-03 | User can navigate into prefixes recursively (Esc to ascend) | Prefix stack in s3Model; Esc pops prefix; BucketList→PrefixList→ObjectDetail panel state machine |
| S3-04 | User can view object metadata (key, size, last modified, storage class) | ListObjectsV2 Contents[].{Size, LastModified, StorageClass, ETag, Key}; ObjectDetail panel renders these fields |
| S3-05 | User can filter objects within current prefix by typing (/ key, server-side prefix) | `/` enters custom filter mode with textinput; Enter re-fetches with extended prefix; list.SetFilteringEnabled(false) disables built-in filter |
| NAV-01 | All list views support vim-style scrolling (j/k) | bubbles/list v1 DefaultKeyMap has j/k for CursorUp/CursorDown; DisableQuitKeybindings() called to prevent list consuming q |
| NAV-02 | User can jump to top with g and bottom with G | bubbles/list v1 DefaultKeyMap has g/G for GoToStart/GoToEnd — already built in |
| NAV-03 | User can navigate back one level with Esc | Esc handled in parent Update before forwarding to list (intercept pattern); Esc in filter mode cancels filter first |
| NAV-05 | Each view displays a breadcrumb header showing current location | Breadcrumb string rendered above list; height subtracted from list SetSize; left-truncated with "... >" if too long |
</phase_requirements>

---

## Summary

Phase 3 builds full S3 navigation on top of the Bubble Tea shell from Phase 2. The domain has three interconnected concerns: the AWS S3 API layer (ListBuckets + ListObjectsV2 pagination with delimiter-based prefix hierarchy), the bubbles/list component integration (custom delegates for two-column display, key interception before forwarding to list.Update), and the s3Model state machine (BucketList/PrefixList/ObjectDetail panel states embedded in rootModel).

The most important architectural decision confirmed by research: the `bubbles/list` component already provides `j`/`k`, `g`/`G`, and `/` as default key bindings. This is a gift — navigation is free. The challenge is properly intercepting `q` (must go to rootModel quit, not list quit) and `Esc`/`Enter` (must drive panel navigation) before the list component consumes them. The pattern is: handle app-specific keys in the parent `s3Model.Update` first, then forward remaining messages to `m.list.Update(msg)`.

The AWS S3 API layer has one critical gotcha: `HeadBucket` is the correct way to get a bucket's region (via `BucketRegion *string` field in `HeadBucketOutput`). `GetBucketLocation` has a well-documented bug where us-east-1 buckets return `null` instead of the region string. Per-bucket regional clients are created via `s3.NewFromConfig(cfg, func(o *s3.Options) { o.Region = region })`. For the filter feature: disable the list's built-in filter (`SetFilteringEnabled(false)`) and implement a custom `textinput` that re-fetches with the extended prefix on Enter.

**Primary recommendation:** Build `internal/ui/s3/` as a standalone package containing s3Model (with embedded list.Model), s3Messages, and an S3 client wrapper. Wire into rootModel's `contentView()` by embedding `s3Model` as a field and forwarding all messages to it in stateReady. Use `s3.NewFromConfig` with a functional option for per-bucket regional overrides.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/aws/aws-sdk-go-v2/service/s3` | v1.96.3 (latest) | S3 API calls: ListBuckets, ListObjectsV2, HeadBucket | Official AWS SDK v2; already in go.mod ecosystem |
| `github.com/charmbracelet/bubbles/list` | v1.0.0 (already in go.mod) | Scrollable, keyboard-driven list with j/k/g/G/filter built in | Already used in project; provides vim keys for free |
| `github.com/charmbracelet/bubbles/textinput` | v1.0.0 (already in go.mod) | Filter input bar at bottom of content area | Already in bubbles v1; same package |
| `github.com/dustin/go-humanize` | v1.0.1 (already transitive dep) | Human-readable bytes (IBytes) and relative time (Time) | Already present in go.sum; zero new dependency cost |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/aws/aws-sdk-go-v2/service/s3/types` | (bundled with s3) | S3 types: Bucket, Object, CommonPrefix, StorageClass | Always — required for response field access |
| `github.com/charmbracelet/bubbles/spinner` | v1.0.0 (already in go.mod) | Background refresh indicator in breadcrumb | During 30s refresh cycle while existing items remain visible |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `bubbles/list` | Custom list rendering | Custom gives more control over column layout but loses j/k/g/G/pagination for free; use list with custom delegate instead |
| `go-humanize` | Custom byte formatter | No reason to hand-roll — go-humanize is already in the dependency graph |
| `HeadBucket` for region | `GetBucketLocation` | GetBucketLocation returns null for us-east-1 (documented bug); HeadBucket.BucketRegion is the correct v2 approach |

**Installation:**
```bash
go get github.com/aws/aws-sdk-go-v2/service/s3@latest
# go-humanize and bubbles already in go.mod
go mod tidy
```

---

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── aws/
│   └── session.go          # existing — NewAWSConfig, ClassifyCredentialError (unchanged)
├── ui/
│   ├── model.go            # existing — add s3Model field to rootModel, wire in contentView()
│   ├── messages.go         # existing — unchanged
│   ├── status.go           # existing — unchanged
│   └── s3/
│       ├── model.go        # s3Model: state machine + list.Model embedding
│       ├── messages.go     # S3-specific message types
│       ├── client.go       # S3 client wrapper: ListBuckets, ListObjectsV2, HeadBucket
│       └── delegate.go     # Custom ItemDelegate for two-column bucket/prefix/object rows
```

### Pattern 1: Child Model Embedded in Parent (Bubble Tea Component Pattern)

**What:** `rootModel` embeds `s3Model` as a field. In `stateReady`, `rootModel.Update` forwards all messages to `s3Model.Update`. `rootModel.contentView()` calls `s3Model.View()`.

**When to use:** Always — this is the established pattern from Phase 2's spinner child model.

**Example:**
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/bubbletea
// internal/ui/model.go — extend rootModel

type rootModel struct {
    // ... existing fields ...
    s3Panel s3.Model  // new field — zero value is safe
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    // ... existing cases ...
    case identityLoadedMsg:
        m.account = msg.account
        m.arn = msg.arn
        m.state = stateReady
        // Initialize S3 panel and fire first fetch
        m.s3Panel = s3.NewModel(m.ctx, m.cfg, m.width, m.height)
        return m, tea.Batch(m.s3Panel.Init(), /* existing cmds */)
    }

    if m.state == stateReady {
        var cmd tea.Cmd
        m.s3Panel, cmd = m.s3Panel.Update(msg)
        return m, cmd
    }
    return m, nil
}

func (m rootModel) contentView() string {
    // ... (height calculation same as before) ...
    switch m.state {
    case stateReady:
        return m.s3Panel.View()
    // ... other cases ...
    }
}
```

### Pattern 2: s3Model Three-State Panel Machine

**What:** `s3Model` has its own `panelState` enum (panelBucketList / panelPrefixList / panelObjectDetail). Each state holds the data needed for that view. Navigation transitions between states.

**When to use:** Always for panel navigation — mirrors how rootModel uses stateLoading/stateReady/stateError.

**Example:**
```go
// Source: https://donderom.com/posts/managing-nested-models-with-bubble-tea/
// internal/ui/s3/model.go

type panelState int
const (
    panelBucketList  panelState = iota
    panelPrefixList
    panelObjectDetail
)

type Model struct {
    ctx         context.Context
    cfg         *config.Config
    state       panelState
    list        list.Model    // shared list component, reconfigured per state
    prefixStack []string      // current navigation path; empty = bucket list
    selectedBucket string
    selectedObj    *types.Object
    s3Client    *s3.Client    // base client (us-east-1 or default region)
    bucketClient *s3.Client   // per-bucket regional client (set after HeadBucket)
    filterMode  bool
    filterInput textinput.Model
    loading     bool
    err         error
    width, height int
}
```

### Pattern 3: Key Interception Before list.Update

**What:** Parent `s3Model.Update` intercepts navigation keys (`Enter`, `Esc`, `q`, `/`) before passing the message to `m.list.Update(msg)`. This prevents the list component from consuming keys meant for panel navigation.

**When to use:** Always — the list's built-in `q` and `Esc` bindings conflict with app semantics.

**Example:**
```go
// Source: verified from bubbles/list key handling analysis
// internal/ui/s3/model.go

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Intercept before list.Update sees it
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit  // or send to rootModel via message

        case "esc":
            if m.filterMode {
                m.filterMode = false  // cancel filter first
                return m, nil
            }
            return m, m.ascendLevel()  // pop prefix stack

        case "enter":
            if m.filterMode {
                return m, m.applyFilter()  // re-fetch with filter prefix
            }
            return m, m.descendIntoSelected()

        case "/":
            if !m.filterMode && m.state != panelObjectDetail {
                m.filterMode = true
                m.filterInput.Focus()
                return m, nil
            }
        }

        // Forward remaining keys to list (j/k/g/G handled here automatically)
        if m.filterMode {
            var cmd tea.Cmd
            m.filterInput, cmd = m.filterInput.Update(msg)
            return m, cmd
        }
    }

    // Forward all non-intercepted messages to list
    var cmd tea.Cmd
    m.list, cmd = m.list.Update(msg)
    return m, cmd
}
```

### Pattern 4: Custom ItemDelegate for Two-Column Rows

**What:** Implement `list.ItemDelegate` with a custom `Render` method that formats bucket name, or prefix/object with size and date columns. Height=1 for single-line rows.

**When to use:** Always — `DefaultDelegate` is two-line (Title + Description) and we need precise column control.

**Example:**
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/bubbles/list
// internal/ui/s3/delegate.go

type s3Delegate struct {
    width int
}

func (d s3Delegate) Height() int  { return 1 }
func (d s3Delegate) Spacing() int { return 0 }
func (d s3Delegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d s3Delegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
    i, ok := item.(S3Item)
    if !ok { return }

    isSelected := index == m.Index()
    style := normalRowStyle
    if isSelected {
        style = selectedRowStyle
    }

    // Two-column: name left, size+date right, aligned to d.width
    left := i.DisplayName()
    right := i.SizeAndDate()
    gap := d.width - lipgloss.Width(left) - lipgloss.Width(right)
    if gap < 1 { gap = 1 }

    row := left + strings.Repeat(" ", gap) + right
    fmt.Fprint(w, style.Render(row))
}
```

### Pattern 5: ListObjectsV2 Paginator with Delimiter

**What:** Use `s3.NewListObjectsV2Paginator` with `Delimiter: aws.String("/")` and `Prefix: aws.String(currentPrefix)`. This returns `CommonPrefixes` (virtual folders) and `Contents` (actual objects) separately.

**When to use:** Always for prefix browsing. Always paginate — never assume one page is enough.

**Example:**
```go
// Source: https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html
// internal/ui/s3/client.go

func ListPrefix(ctx context.Context, client *s3.Client, bucket, prefix string) (prefixes []string, objects []types.Object, err error) {
    paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
        Bucket:    aws.String(bucket),
        Prefix:    aws.String(prefix),   // e.g., "logs/" or "" for root
        Delimiter: aws.String("/"),
    })

    for paginator.HasMorePages() {
        page, err := paginator.NextPage(ctx)
        if err != nil {
            return nil, nil, err
        }
        for _, cp := range page.CommonPrefixes {
            prefixes = append(prefixes, aws.ToString(cp.Prefix))
        }
        objects = append(objects, page.Contents...)
    }
    return prefixes, objects, nil
}
```

### Pattern 6: Per-Bucket Regional S3 Client via HeadBucket

**What:** After selecting a bucket, call HeadBucket with the base client. The response contains `BucketRegion *string`. Create a new `s3.Client` with that region override for all subsequent bucket operations.

**When to use:** Always — S3 operations require the correct regional endpoint. Using the wrong region returns 301 redirects or endpoint errors.

**Example:**
```go
// Source: https://github.com/aws/aws-sdk-go-v2/blob/main/service/s3/api_op_HeadBucket.go
// internal/ui/s3/client.go

func NewBucketClient(ctx context.Context, baseClient *s3.Client, baseCfg aws.Config, bucket string) (*s3.Client, error) {
    out, err := baseClient.HeadBucket(ctx, &s3.HeadBucketInput{
        Bucket: aws.String(bucket),
    })
    if err != nil {
        return nil, err
    }

    region := aws.ToString(out.BucketRegion)
    if region == "" {
        region = "us-east-1"  // HeadBucket returns "" for us-east-1 (same as GetBucketLocation null behavior)
    }

    return s3.NewFromConfig(baseCfg, func(o *s3.Options) {
        o.Region = region
    }), nil
}
```

### Pattern 7: Breadcrumb with Left-Truncation

**What:** Build breadcrumb string from `prefixStack`, measure it against `m.width`, and truncate from the left if too long.

**When to use:** Always — breadcrumb must fit on one line regardless of depth.

**Example:**
```go
// Source: established pattern; lipgloss.Width for display measurement
// internal/ui/s3/model.go

func (m Model) renderBreadcrumb() string {
    parts := []string{"S3"}
    if m.selectedBucket != "" {
        parts = append(parts, m.selectedBucket)
    }
    parts = append(parts, m.prefixStack...)

    crumb := strings.Join(parts, " > ")

    // Left-truncate if too wide
    maxWidth := m.width - 4  // leave margin
    for lipgloss.Width(crumb) > maxWidth && len(parts) > 2 {
        parts = append([]string{"..."}, parts[2:]...)  // drop second element, keep "..."
        crumb = strings.Join(parts, " > ")
    }

    return breadcrumbStyle.Width(m.width).Render(crumb)
}
```

### Pattern 8: Server-Side Filter with Custom textinput

**What:** Disable list's built-in filter. On `/`, set `filterMode=true` and focus `textinput`. On Enter, re-fetch `ListObjectsV2` with `Prefix = currentPrefix + filterInput.Value()`. On Esc, clear filter and restore previous results.

**When to use:** For `/` filter behavior — this implements S3-05.

**Example:**
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/bubbles/textinput
// internal/ui/s3/model.go — in Init()

fi := textinput.New()
fi.Placeholder = "prefix filter"
fi.Prompt = "Filter: "
m.filterInput = fi

// Disable list's built-in "/" filter entirely
m.list.SetFilteringEnabled(false)
```

### Anti-Patterns to Avoid

- **Passing all messages to list.Update without interception first:** The list consumes `q`, `Esc`, and `/` for its own purposes. Always check app-specific keys first.
- **Using GetBucketLocation for bucket region:** Returns `null` (empty LocationConstraint) for us-east-1 — a known AWS quirk. Use `HeadBucket.BucketRegion` instead.
- **Fetching all objects without delimiter:** Without `Delimiter="/"`, ListObjectsV2 returns every object recursively — potentially millions. Always use delimiter for prefix browsing.
- **Blocking the Update loop with API calls:** All S3 API calls must be wrapped in `tea.Cmd` closures. Never call SDK methods directly inside `Update`.
- **Recreating the list.Model on every state transition:** Preserve scroll position by updating items with `m.list.SetItems()` rather than creating a new list. Only resize with `m.list.SetSize()` on window size changes.
- **Ignoring IsTruncated/HasMorePages:** S3 defaults to 1000 objects per page. Always use the paginator, not a single API call.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Vim-style j/k/g/G scrolling | Custom keyboard handler | `bubbles/list` DefaultKeyMap | j/k/g/G/Home/End are built into list v1 — zero code |
| Human-readable file sizes | Custom byte formatter | `go-humanize.IBytes(uint64(size))` | Already in go.sum; handles edge cases (0B, 1023B) |
| Relative time display | Custom time diff | `go-humanize.Time(lastModified)` | Handles "X minutes ago" / "X days ago" formatting |
| S3 pagination | Manual ContinuationToken loop | `s3.NewListObjectsV2Paginator` | Paginator handles ContinuationToken automatically |
| Fuzzy list filtering | Custom filter algorithm | `bubbles/list` FilterFunc (or disable entirely for server-side) | Built into list; or disable and use textinput for server-side |
| Breadcrumb string truncation | None — hand-roll is fine | Custom left-truncate function | lipgloss has no TruncateLeft; a simple loop is ~10 lines and fully owned |

**Key insight:** The `bubbles/list` component handles the hardest parts of S3 browsing UX for free — keyboard navigation, cursor tracking, and pagination. The work is in the glue: custom delegate for column layout, key interception for navigation, and the S3 API calls wrapped as tea.Cmd.

---

## Common Pitfalls

### Pitfall 1: list consumes `q` and `Esc` before parent sees them
**What goes wrong:** The list component handles `q` (quit) and `Esc` (clear filter) in its own `Update`. If you forward `tea.KeyMsg` to `m.list.Update(msg)` without intercepting first, panel navigation never fires.
**Why it happens:** `bubbles/list.Update` processes key messages and returns early — the message never reaches your code.
**How to avoid:** In `s3Model.Update`, handle all app-specific keys (`q`, `Esc`, `Enter`, `/`) with a type switch on `tea.KeyMsg` BEFORE calling `m.list.Update(msg)`. Call `m.list.DisableQuitKeybindings()` in the constructor.
**Warning signs:** Pressing `q` inside a list view quits (correct) but also triggers list's quit behavior (redundant). Pressing `Esc` clears a non-existent filter instead of ascending.

### Pitfall 2: GetBucketLocation returns null for us-east-1 buckets
**What goes wrong:** Calling `GetBucketLocation` on a us-east-1 bucket returns an empty `LocationConstraint`. Code that treats empty string as "region unknown" will fail to create a regional client.
**Why it happens:** AWS S3 API design quirk — us-east-1 is the original region and predates the LocationConstraint field.
**How to avoid:** Use `HeadBucket` instead. `HeadBucketOutput.BucketRegion` is a `*string` that contains the region. If it's empty after `aws.ToString`, default to `"us-east-1"`.
**Warning signs:** Per-bucket operations fail with `301 Moved Permanently` or `IllegalLocationConstraintException` for us-east-1 buckets.

### Pitfall 3: List height not accounting for breadcrumb
**What goes wrong:** Breadcrumb line renders on top of the list, overlapping the first item. Or the list overflows into the status bar.
**Why it happens:** `list.SetSize(width, height)` must receive the height of the CONTENT area minus breadcrumb height minus filter bar height.
**How to avoid:** Calculate content height as: `contentH = totalHeight - statusBarH`. Then list height = `contentH - breadcrumbH - (filterMode ? 1 : 0)`. Re-call `m.list.SetSize` whenever `filterMode` toggles or window resizes.
**Warning signs:** First list item is hidden, or status bar overlaps list content.

### Pitfall 4: S3 API calls not per-bucket regional
**What goes wrong:** ListObjectsV2 calls succeed for buckets in the configured region but fail with `PermanentRedirect` or `AuthorizationHeaderMalformed` for buckets in other regions.
**Why it happens:** S3 is region-specific for object operations. A client configured for `us-east-1` cannot directly list objects in `eu-west-1` buckets.
**How to avoid:** Always call `HeadBucket` when descending into a bucket to get the bucket's region. Cache the per-bucket `*s3.Client`. Reuse cached client for all prefix/object operations in that bucket.
**Warning signs:** Works for some buckets, fails for others with cryptic redirect errors.

### Pitfall 5: Fetching objects without Delimiter collapses the entire bucket
**What goes wrong:** ListObjectsV2 without `Delimiter` returns every object in the bucket recursively — potentially millions of items. TUI hangs or OOMs.
**Why it happens:** Without a delimiter, S3 has no concept of "current level" — everything is returned flat.
**How to avoid:** Always set `Delimiter: aws.String("/")`. For the root of a bucket, `Prefix` is empty string `""`. For a prefix, it's the current path e.g. `"logs/"`. CommonPrefixes gives virtual folders; Contents gives real objects at this level only.
**Warning signs:** Navigating into a large bucket causes a long hang before showing a flat list of thousands of objects.

### Pitfall 6: list.SetItems resets scroll position
**What goes wrong:** After a background refresh, the user's scroll position jumps to the top.
**Why it happens:** `list.SetItems` resets the cursor to position 0.
**How to avoid:** Save the current `m.list.Index()` before `SetItems`. After `SetItems`, call `m.list.Select(savedIndex)`. Or only update items if the list changed (compare lengths).
**Warning signs:** User scrolls halfway down the bucket list; 30s refresh fires; cursor jumps to top.

### Pitfall 7: Filter input captures j/k when in filter mode
**What goes wrong:** While filter mode is active, pressing `j`/`k` inserts literal "j"/"k" into the filter input instead of scrolling.
**Why it happens:** When `filterMode=true`, all key messages route to `m.filterInput.Update(msg)`. This is correct behavior — the textinput is active.
**How to avoid:** This is the intended behavior. Document it: filter mode is an exclusive input state. Only `Enter` (apply filter) and `Esc` (cancel filter) exit filter mode. The textinput's own backspace/ctrl+u handling clears input.
**Warning signs:** None — this is correct. But document the UX so it's not reported as a bug.

---

## Code Examples

Verified patterns from official sources:

### ListBuckets with Paginator
```go
// Source: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3
// internal/ui/s3/client.go

func ListAllBuckets(ctx context.Context, client *s3.Client) ([]types.Bucket, error) {
    paginator := s3.NewListBucketsPaginator(client, &s3.ListBucketsInput{})
    var buckets []types.Bucket
    for paginator.HasMorePages() {
        page, err := paginator.NextPage(ctx)
        if err != nil {
            return nil, err
        }
        buckets = append(buckets, page.Buckets...)
    }
    return buckets, nil
}
```

### HeadBucket for Regional Client Construction
```go
// Source: https://github.com/aws/aws-sdk-go-v2/blob/main/service/s3/api_op_HeadBucket.go
// HeadBucketOutput.BucketRegion is *string

func BucketRegion(ctx context.Context, client *s3.Client, bucket string) (string, error) {
    out, err := client.HeadBucket(ctx, &s3.HeadBucketInput{
        Bucket: aws.String(bucket),
    })
    if err != nil {
        return "", err
    }
    region := aws.ToString(out.BucketRegion)
    if region == "" {
        return "us-east-1", nil  // AWS quirk: empty = us-east-1
    }
    return region, nil
}
```

### ListObjectsV2 with Prefix + Delimiter
```go
// Source: https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html
// CommonPrefixes = virtual folders; Contents = real objects at this level

paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
    Bucket:    aws.String(bucket),
    Prefix:    aws.String(currentPrefix),  // "" for root, "logs/" for subdirectory
    Delimiter: aws.String("/"),
})
for paginator.HasMorePages() {
    page, _ := paginator.NextPage(ctx)
    for _, cp := range page.CommonPrefixes {
        // cp.Prefix is a virtual folder like "logs/2024/"
    }
    for _, obj := range page.Contents {
        // obj.Key, obj.Size, obj.LastModified, obj.StorageClass, obj.ETag
    }
}
```

### server-side Filter: Extended Prefix
```go
// Source: CONTEXT.md decision — filter appends typed string to current prefix
// "logs/" + "2024" → ListObjectsV2 with Prefix="logs/2024"

filterPrefix := currentPrefix + m.filterInput.Value()
paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
    Bucket:    aws.String(bucket),
    Prefix:    aws.String(filterPrefix),
    Delimiter: aws.String("/"),
})
```

### go-humanize for Size and Time
```go
// Source: https://pkg.go.dev/github.com/dustin/go-humanize
// Already in go.sum as transitive dep — import directly

import "github.com/dustin/go-humanize"

// Bytes display
humanize.IBytes(uint64(obj.Size))      // "1.5 MiB"
humanize.Bytes(uint64(obj.Size))       // "1.6 MB" (SI)

// Relative time
humanize.Time(*obj.LastModified)        // "3 days ago", "2 hours ago"
```

### list.Model Init and Configuration
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/bubbles/list

d := s3Delegate{width: width}
l := list.New([]list.Item{}, d, width, listHeight)
l.SetShowTitle(false)       // breadcrumb replaces list title
l.SetShowStatusBar(false)   // no "X items" bar needed
l.SetShowPagination(true)   // keep page indicator
l.SetFilteringEnabled(false) // disable built-in "/" filter — we handle it ourselves
l.DisableQuitKeybindings()  // prevent list from consuming q/ctrl+c
```

### tea.Cmd Pattern for Async S3 Calls
```go
// Source: internal/ui/model.go — fetchIdentityCmd pattern (established in Phase 2)
// internal/ui/s3/messages.go + model.go

// Message types
type bucketsLoadedMsg struct { buckets []types.Bucket }
type bucketsErrMsg    struct { err error }
type prefixesLoadedMsg struct {
    prefixes []string
    objects  []types.Object
}

// Async command
func fetchBucketsCmd(ctx context.Context, client *s3.Client) tea.Cmd {
    return func() tea.Msg {
        buckets, err := ListAllBuckets(ctx, client)
        if err != nil {
            return bucketsErrMsg{err: err}
        }
        return bucketsLoadedMsg{buckets: buckets}
    }
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `GetBucketLocation` for bucket region | `HeadBucket` + `BucketRegion` field | S3 API evolution (HeadBucket got BucketRegion field) | GetBucketLocation returns null for us-east-1 — use HeadBucket |
| `ListObjects` (v1 API) | `ListObjectsV2` | 2016 (S3 API v2) | ListObjectsV2 has ContinuationToken vs older Marker; always use v2 |
| Manual pagination loop | `NewListObjectsV2Paginator` | aws-sdk-go-v2 initial release | Paginator handles token management automatically |
| `bubbles` v0.x (no g/G) | `bubbles` v1.0.0 | Phase 2 upgrade | g/G for top/bottom now built into DefaultKeyMap |

**Deprecated/outdated:**
- `GetBucketLocation`: Returns null LocationConstraint for us-east-1 — superseded by `HeadBucket.BucketRegion`
- `ListObjects` (v1 API): Deprecated in favor of `ListObjectsV2`; `NewListObjectsPaginator` still works but prefer v2

---

## Open Questions

1. **HeadBucket returns empty BucketRegion for some bucket types?**
   - What we know: `BucketRegion *string` is in `HeadBucketOutput`. Empty string convention same as GetBucketLocation null → us-east-1.
   - What's unclear: Whether directory buckets (S3 Express One Zone) behave differently.
   - Recommendation: Treat empty `BucketRegion` as us-east-1 per the documented null convention. Flag in code with a comment.

2. **ListBuckets pagination: does it matter for typical accounts?**
   - What we know: AWS recommends always using paginated ListBuckets. Accounts with >10,000 buckets must paginate. Default quota is 10,000.
   - What's unclear: Whether `NewListBucketsPaginator` is available in the current S3 SDK or whether manual token handling is needed.
   - Recommendation: Use paginator if available (check in Wave 1); fall back to manual ContinuationToken loop. Most accounts hit <100 buckets — single page is fine for v1 but use paginator to be correct.

3. **Spinner placement during background 30s refresh**
   - What we know: The existing `bubbles/spinner` is already in the codebase. CONTEXT.md marks spinner placement as Claude's Discretion.
   - What's unclear: Whether to show spinner in breadcrumb line (right-aligned) or in list status bar area.
   - Recommendation: Add a small spinner to the right end of the breadcrumb line using the existing spinner component pattern from Phase 2. Start spinner on refresh tick, stop on data loaded.

---

## Sources

### Primary (HIGH confidence)
- `pkg.go.dev/github.com/charmbracelet/bubbles/list` — Full API: New(), SetFilteringEnabled(), DisableQuitKeybindings(), Item/ItemDelegate interfaces, DefaultKeyMap (j/k/g/G//)
- `github.com/aws/aws-sdk-go-v2/blob/main/service/s3/api_op_HeadBucket.go` — HeadBucketOutput.BucketRegion field confirmed as *string
- `docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html` — Delimiter/Prefix/CommonPrefixes semantics, pagination with ContinuationToken
- `pkg.go.dev/github.com/charmbracelet/bubbles/textinput` — Full textinput API including Focus(), Value(), SetValue(), Prompt
- `pkg.go.dev/github.com/dustin/go-humanize` — IBytes(), Bytes(), Time() function signatures confirmed

### Secondary (MEDIUM confidence)
- `docs.aws.amazon.com/AmazonS3/latest/userguide/list-buckets.html` — ListBuckets pagination behavior; accounts >10,000 buckets must paginate
- `pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3` — ListBuckets, ListObjectsV2, NewListObjectsV2Paginator signatures
- `donderom.com/posts/managing-nested-models-with-bubble-tea/` — Child model embedding pattern in rootModel.Update
- `github.com/aws/aws-sdk-go-v2/issues/1894` — GetBucketLocation null for us-east-1 confirmed as expected behavior

### Tertiary (LOW confidence)
- WebSearch for "per-bucket regional S3 client" — s3.NewFromConfig with functional option `func(o *s3.Options) { o.Region = region }` consistent across multiple sources but not verified against official docs page directly

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — bubbles v1 and aws-sdk-go-v2 both verified via pkg.go.dev; go-humanize confirmed in go.sum
- Architecture: HIGH — child model pattern matches existing Phase 2 code; key interception pattern confirmed from list source
- Pitfalls: HIGH — GetBucketLocation/us-east-1 bug confirmed via GitHub issues and AWS docs; list key consumption confirmed from source
- S3 API semantics: HIGH — ListObjectsV2 Delimiter/CommonPrefixes from official AWS API docs

**Research date:** 2026-03-04
**Valid until:** 2026-04-04 (stable APIs — bubbles v1 and aws-sdk-go-v2 service packages are stable)
