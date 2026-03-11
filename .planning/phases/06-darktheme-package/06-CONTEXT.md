# Phase 6: DarkTheme Package - Context

**Gathered:** 2026-03-10
**Status:** Ready for planning

<domain>
## Phase Boundary

Build the centralized theme struct in `internal/ui/theme/theme.go`. All AWS Dark color tokens and border styles flow from this one object. No rendering changes — this phase is purely the theme package. Every downstream phase (7–11) imports and consumes it.

</domain>

<decisions>
## Implementation Decisions

### Token set
- **Background**: Squid Ink (`#232F3E`) — deepest layer, main app background
- **Surface**: `#2D2D2D` — elevated panels (status bar, modals, overlays); distinct from Background for depth hierarchy
- **Border**: separate color token for panel border lines (dim gray, independently tunable from Surface)
- **Accent**: AWS Orange (`#FF9900`) — selected items, active focus states, highlighted text; no separate Selected token (selected items = accent)
- **TextPrimary**: current `Color("248")` equivalent — primary readable text
- **Muted**: current `Color("240")` equivalent — covers both separators and dimmed/secondary text (single token, same visual weight)

### Border styles
- Theme struct holds a `lipgloss.Border` field (rounded border — `lipgloss.RoundedBorder()`)
- Theme pre-builds `InactiveBorderStyle` and `ActiveBorderStyle` as `lipgloss.Style` — phase 9 references these directly, no per-component color math
- Active border = Accent color; inactive border = Muted color

### Struct design
- Exported fields — `theme.Accent`, `theme.Background`, etc. (not getter methods)
- Only pre-built styles are the two border variants; all other styles built by consuming components from color tokens

### Color fallbacks
- All tokens use `lipgloss.CompleteColor` (TrueColor + ANSI256 + ANSI) — consistent degradation across tmux, SSH, and 16-color terminals
- ANSI (16-color) fallback values: Claude's discretion (semantic nearest: Background → `"0"`, TextPrimary → `"7"`, Accent → `"3"`, etc.)

### Claude's Discretion
- Exact ANSI 16-color fallback values for each token
- Exact Border color hex/ANSI256 value (dim gray)
- Package-level `var DefaultTheme = DarkTheme()` convenience or just the constructor — Claude decides

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- No existing theme or style package — this creates the first one
- `lipgloss` already imported in all UI files — no new dependency needed

### Established Patterns
- Inline colors currently defined as package-level `var` blocks in each file (e.g., `status.go` lines 15–44, `s3/delegate.go` lines 71–73, `ecs/delegate.go` lines 47–48)
- All current colors are ANSI256 string codes (`Color("205")`, `Color("33")`, etc.) — no hex values currently in use
- `Color("205")` (hot pink) is the current selected-item color in both delegates — will become Accent (Orange)
- `Color("33")` (blue) is the current profile/region accent — will become Accent (Orange)

### Integration Points
- `internal/ui/theme/theme.go` — new package; can be imported by any `internal/ui/*` package without circular dependency
- Phase 8 (Footer) migrates `internal/ui/status.go` inline colors → theme tokens
- Phase 9 (Two-Panel Layout) uses `InactiveBorderStyle` / `ActiveBorderStyle` from theme
- Delegates (`s3/delegate.go`, `ecs/delegate.go`) will replace `selectedStyle` with `theme.Accent`

</code_context>

<specifics>
## Specific Ideas

No specific references beyond the requirements — open to standard Go struct patterns for the theme implementation.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 06-darktheme-package*
*Context gathered: 2026-03-10*
