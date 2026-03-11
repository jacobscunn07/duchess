# Phase 7: Header - Context

**Gathered:** 2026-03-11
**Status:** Ready for planning

<domain>
## Phase Boundary

Add a persistent ASCII art "duchess" logo header at the top of every screen with app metadata (version, profile, region, refresh interval) displayed alongside the logo. No changes to the footer/status bar — that is Phase 8's scope.

</domain>

<decisions>
## Implementation Decisions

### Logo style
- Font: "Big" figlet font — the specific art user provided:
  ```
       _            _
      | |          | |
    __| |_   _  ___| |__   ___  ___ ___
   / _` | | | |/ __| '_ \ / _ \/ __/ __|
  | (_| | |_| | (__| | | |  __/\__ \__ \
   \__,_|\__,_|\___|_| |_|\___||___/___/
  ```
- Trailing blank rows trimmed — headerHeight = 6 rows (the content rows only)
- No tagline or subtitle — just the word "duchess"
- Logo color: AWS Orange (`theme.DefaultTheme.Accent` / `#FF9900`)

### Metadata layout
- Stacked vertically — one item per line (version, profile, region, refresh interval)
- Bottom-aligned within the header height — metadata rows align with the bottom rows of the logo
- Right-aligned to terminal edge — logo far left, metadata far right
- Labels in Muted (`theme.DefaultTheme.Muted`), values in Accent (`theme.DefaultTheme.Accent`)
- Format: `profile: <value>`, `region: <value>`, `refresh: <value>`, `v<version>` (version on its own without a label key)

### Visual treatment
- Header background: Surface (`theme.DefaultTheme.Surface` / `#2D2D2D`) — same as the status bar, creating visual bookends
- No separator line between header and content — the color contrast between Surface and Background is sufficient
- Full-width, no horizontal padding — header fills the entire terminal width, logo starts at column 0

### Height constant
- `headerHeight` is a package-level constant derived from the actual ASCII art line count (after trailing trim), NOT a hardcoded integer
- This ensures layout math in `baseView()` updates automatically if the logo changes

### Claude's Discretion
- Exact vertical alignment implementation (lipgloss padding vs string manipulation)
- How to handle very narrow terminals (< logo width) — truncate or wrap gracefully
- File organization (`internal/ui/header.go` following status.go pattern, or separate package)

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `theme.DefaultTheme` — all color tokens available; `Surface`, `Accent`, `Muted`, `TextPrimary` all needed for header
- `status.go` / `renderStatusBar()` — structural pattern to follow: package-level style vars, single render function, width-aware, returns string
- `internal/ui/model.go` `baseView()` — integration point; currently does `height - statusBar height`; Phase 7 adds `- headerHeight` to this calculation

### Established Patterns
- Package-level lipgloss style vars defined once (not recreated on every render) — follow same pattern in header.go
- `lipgloss.CompleteColor` for all color tokens — no inline `lipgloss.Color()` strings
- All styles inherit from `theme.DefaultTheme.*` — no hardcoded hex strings in component files
- Width-aware rendering: final container uses `.Width(m.width)` to fill terminal

### Integration Points
- `baseView()` in `model.go`: currently `contentH = m.height - lipgloss.Height(statusBar)` — must become `contentH = m.height - headerHeight - lipgloss.Height(statusBar)` after Phase 7
- `View()` in `model.go`: currently returns `bg` = content + statusBar; Phase 7 prepends header: header + content + statusBar
- `contentView()` similarly needs headerHeight factored in for centering calculations
- `identityLoadedMsg` handler: recomputes `contentH` — must account for headerHeight there too
- `regionSelectedMsg` handler: same contentH recomputation

</code_context>

<specifics>
## Specific Ideas

- User provided the exact ASCII art (Big figlet font). The art should be used verbatim as a string literal in `header.go` — not generated at runtime.
- Visual reference from the mockup: logo fills ~40 cols on left, metadata stack occupies bottom ~4 rows on the right side. Similar to how a terminal dashboard separates branding from live status.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 07-header*
*Context gathered: 2026-03-11*
