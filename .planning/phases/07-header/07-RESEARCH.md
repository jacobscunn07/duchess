# Phase 7: Header - Research

**Researched:** 2026-03-11
**Domain:** lipgloss TUI layout — persistent header bar with ASCII art and metadata stack
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Logo style**
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

**Metadata layout**
- Stacked vertically — one item per line (version, profile, region, refresh interval)
- Bottom-aligned within the header height — metadata rows align with the bottom rows of the logo
- Right-aligned to terminal edge — logo far left, metadata far right
- Labels in Muted (`theme.DefaultTheme.Muted`), values in Accent (`theme.DefaultTheme.Accent`)
- Format: `profile: <value>`, `region: <value>`, `refresh: <value>`, `v<version>` (version on its own without a label key)

**Visual treatment**
- Header background: Surface (`theme.DefaultTheme.Surface` / `#2D2D2D`) — same as the status bar
- No separator line between header and content — color contrast is sufficient
- Full-width, no horizontal padding — header fills the entire terminal width, logo starts at column 0

**Height constant**
- `headerHeight` is a package-level constant derived from the actual ASCII art line count (after trailing trim), NOT a hardcoded integer
- This ensures layout math in `baseView()` updates automatically if the logo changes

### Claude's Discretion
- Exact vertical alignment implementation (lipgloss padding vs string manipulation)
- How to handle very narrow terminals (< logo width) — truncate or wrap gracefully
- File organization (`internal/ui/header.go` following status.go pattern, or separate package)

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LAYOUT-01 | App displays a persistent header at the top of every screen | `baseView()` integration pattern identified; header prepended to content+statusBar join |
| LAYOUT-02 | Header shows ASCII art "duchess" logo on the left | ASCII art is verbatim string literal in header.go; lipgloss Left-join positions it |
| LAYOUT-03 | Header shows app version, AWS profile, region, and refresh interval to the right of the logo | Metadata stack rendered as right column; gap-fill pattern mirrors status.go approach |
</phase_requirements>

---

## Summary

Phase 7 adds a persistent header bar above all screens. The implementation is a new file `internal/ui/header.go` in the existing `ui` package, following the exact pattern already established by `status.go`. The ASCII art is embedded as a raw string literal — no runtime font generation. A package-level `const headerHeight` is derived from `strings.Count(duchessLogo, "\n") + 1` (or equivalent), which is computable at compile time from the string.

The primary challenge is two-column layout within a fixed height: logo occupies the left at its natural width, metadata stack occupies the right, and the metadata must be bottom-aligned (padded at the top) to sit flush with the logo's bottom rows. This is achievable with lipgloss `PlaceVertical` on the metadata column, or with explicit newline padding before the metadata lines. The `lipgloss.JoinHorizontal` + `lipgloss.PlaceVertical` approach is idiomatic for this codebase.

Integration requires three changes to `model.go`: `baseView()` subtracts `headerHeight` from `contentH`, `View()` prepends the header string, and the two `contentH` calculations in the message handlers (`identityLoadedMsg`, `regionSelectedMsg`) also subtract `headerHeight` so panels receive correct height.

**Primary recommendation:** Implement `header.go` mirroring `status.go`'s package-level style var pattern. Compute `headerHeight` as a `const` using `len(strings.Split(duchessLogo, "\n"))` evaluated at init, or declare `headerHeight = 6` as a named constant with a comment tying it to the ASCII art. Use `lipgloss.PlaceVertical` for bottom-aligning the metadata column.

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/charmbracelet/lipgloss` | v1.1.0 | Style, layout, color rendering | Already in go.mod; all existing UI uses it |
| `github.com/jacobscunn07/duchess/internal/ui/theme` | — | Color tokens via `DefaultTheme` | Established project pattern — no inline colors |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `strings` (stdlib) | — | Split logo string to compute line count | Used to derive `headerHeight` from ASCII art |
| `fmt` (stdlib) | — | Format metadata values (`fmt.Sprintf`) | Profile, region, refresh interval formatting |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Verbatim string literal for ASCII art | `github.com/common-nighthawk/go-figure` (figlet) | Runtime generation adds dependency; user already has the exact art — embed it directly |
| `lipgloss.PlaceVertical` for bottom-align | Manual `strings.Repeat("\n", padLines)` prefix | Both work; PlaceVertical is more explicit about intent; manual padding is simpler and testable |

**Installation:** No new dependencies needed.

---

## Architecture Patterns

### Recommended Project Structure

```
internal/ui/
├── header.go          # NEW — renderHeader(), headerHeight const, package-level styles
├── model.go           # MODIFIED — baseView(), View(), identityLoadedMsg, regionSelectedMsg handlers
├── status.go          # EXISTING — reference pattern for header.go
└── status_test.go     # EXISTING — reference pattern for header_test.go
```

### Pattern 1: Package-Level Style Vars (matches status.go exactly)

**What:** Declare all `lipgloss.NewStyle()` instances as package-level `var`s, initialized once at startup, not inside render functions.

**When to use:** All style declarations in this codebase. Avoids repeated allocation on every frame.

**Example (from status.go — the direct template):**
```go
// Source: internal/ui/status.go
var (
    statusBarStyle = lipgloss.NewStyle().Background(theme.DefaultTheme.Surface)

    profileStyle = lipgloss.NewStyle().
        Foreground(theme.DefaultTheme.Accent).
        Inherit(statusBarStyle)
)
```

Follow this exactly for `header.go`:
```go
var (
    headerStyle = lipgloss.NewStyle().Background(theme.DefaultTheme.Surface)

    logoStyle = lipgloss.NewStyle().
        Foreground(theme.DefaultTheme.Accent).
        Inherit(headerStyle)

    metaLabelStyle = lipgloss.NewStyle().
        Foreground(theme.DefaultTheme.Muted).
        Inherit(headerStyle)

    metaValueStyle = lipgloss.NewStyle().
        Foreground(theme.DefaultTheme.Accent).
        Inherit(headerStyle)
)
```

### Pattern 2: headerHeight as Package-Level Constant

**What:** `headerHeight` must be a value derived from the ASCII art line count — not a bare integer literal.

**Two valid approaches:**

Option A — `var` computed at init time (most robust, auto-tracks art changes):
```go
// duchessLogo is the ASCII art string (verbatim, no leading/trailing blank lines).
const duchessLogo = `     _            _
    | |          | |
  __| |_   _  ___| |__   ___  ___ ___
 / _` + "`" + `| | | |/ __| '_ \ / _ \/ __/ __|
| (_| | |_| | (__| | | |  __/\__ \__ \
 \__,_|\__,_|\___|_| |_|\___||___/___/`

// headerHeight is the number of display rows the header occupies.
// Derived from the logo line count so layout math is always in sync.
var headerHeight = strings.Count(duchessLogo, "\n") + 1  // = 6
```

Option B — named `const` with explanatory comment:
```go
// headerHeight is the number of display rows occupied by the header.
// This matches the line count of duchessLogo (6 rows, trailing blank trimmed).
// Update this constant if the logo art changes.
const headerHeight = 6
```

**Recommendation:** Option A (`var` derived from `strings.Count`) because the CONTEXT.md success criterion explicitly requires the height to be "derived from the actual ASCII art line count — not a hardcoded integer."

### Pattern 3: Two-Column Header Layout

**What:** Left column = logo, right column = metadata stack, side-by-side filling terminal width.

**Approach (lipgloss.JoinHorizontal + PlaceVertical):**
```go
func renderHeader(m rootModel, width int) string {
    if width == 0 {
        return ""
    }

    // Left: logo styled
    logoStr := logoStyle.Render(duchessLogo)
    logoW := lipgloss.Width(logoStr)

    // Right: metadata lines
    meta := strings.Join([]string{
        metaLabelStyle.Render("profile: ") + metaValueStyle.Render(m.cfg.Profile),
        metaLabelStyle.Render("region: ")  + metaValueStyle.Render(m.cfg.Region),
        metaLabelStyle.Render("refresh: ") + metaValueStyle.Render(fmt.Sprintf("%ds", m.cfg.RefreshInterval)),
        metaValueStyle.Render("v" + version),
    }, "\n")

    // Bottom-align metadata within headerHeight rows
    metaAligned := lipgloss.PlaceVertical(headerHeight, lipgloss.Bottom, meta)

    // Right-align metadata column: fill gap between logo and right edge
    metaW := lipgloss.Width(metaAligned)  // natural column width
    gapW := width - logoW - metaW
    if gapW < 0 {
        gapW = 0
    }
    gap := strings.Repeat(" ", gapW)

    row := logoStr + gap + metaAligned
    return headerStyle.Width(width).Render(row)
}
```

**Note on narrow terminals:** When `gapW < 0`, clamp to 0 (overlap). No wrapping — terminal users who resize to sub-logo width accept visual clipping. This is the simplest defensible behavior.

### Pattern 4: baseView() Integration

**Current (model.go lines 303–311):**
```go
func (m rootModel) baseView() string {
    statusBar := renderStatusBar(m)
    contentH := m.height - lipgloss.Height(statusBar)
    if contentH < 0 {
        contentH = 0
    }
    content := lipgloss.NewStyle().Width(m.width).Height(contentH).Render(m.contentView())
    return lipgloss.JoinVertical(lipgloss.Left, content, statusBar)
}
```

**After Phase 7:**
```go
func (m rootModel) baseView() string {
    header := renderHeader(m, m.width)
    statusBar := renderStatusBar(m)
    contentH := m.height - headerHeight - lipgloss.Height(statusBar)
    if contentH < 0 {
        contentH = 0
    }
    content := lipgloss.NewStyle().Width(m.width).Height(contentH).Render(m.contentView())
    return lipgloss.JoinVertical(lipgloss.Left, header, content, statusBar)
}
```

**View() integration** — prepend header in the returned string. Currently `View()` returns `bg` which already includes header via `baseView()` — no additional change needed there once `baseView()` is updated.

**contentView() — note:** `contentView()` recomputes `contentH` independently. It must also subtract `headerHeight`:
```go
// In contentView():
contentH := m.height - headerHeight - lipgloss.Height(statusBar)
```

**Message handlers:** `identityLoadedMsg` and `regionSelectedMsg` both compute `contentH` to size the panels:
```go
// Current (identityLoadedMsg handler):
statusBar := renderStatusBar(m)
contentH := m.height - lipgloss.Height(statusBar)

// After Phase 7:
contentH := m.height - headerHeight - lipgloss.Height(statusBar)
```

Same fix applies to `regionSelectedMsg`.

### Anti-Patterns to Avoid

- **Inline `lipgloss.Color()` in header.go:** All colors must flow through `theme.DefaultTheme.*` — the Phase 6 migration explicitly prohibited inline colors.
- **Recreating styles inside renderHeader:** All styles are package-level vars — never `lipgloss.NewStyle()` inside a render function.
- **Hardcoded `6` without derivation:** The success criterion requires derivation from actual line count. Use Option A (`strings.Count`) to satisfy this unambiguously.
- **Modifying `version` constant:** `version` is already defined in `status.go` as a package-level const. `header.go` shares the same `ui` package — reference it directly, do not redeclare.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Bottom-align metadata | Manual line counting + string padding | `lipgloss.PlaceVertical(h, lipgloss.Bottom, str)` | lipgloss handles ANSI-aware height counting |
| Terminal-width filling | Character counting loops | `headerStyle.Width(width).Render(...)` | lipgloss fills and clips correctly with ANSI awareness |
| Logo line count | Manual integer constant | `strings.Count(duchessLogo, "\n") + 1` | Stays correct if logo is ever edited |

**Key insight:** lipgloss's `PlaceVertical`, `Width()`, and `JoinVertical` handle all ANSI-aware sizing. Never count raw bytes — always use `lipgloss.Height()` and `lipgloss.Width()` for rendered string measurements.

---

## Common Pitfalls

### Pitfall 1: Using lipgloss.Height() on the un-rendered logo string
**What goes wrong:** `duchessLogo` is a raw string with literal `\n` chars; `lipgloss.Height(duchessLogo)` returns the correct line count but ONLY if the string has not been Render()ed yet (ANSI codes can interfere with line splitting in edge cases).
**Why it happens:** After `logoStyle.Render(duchessLogo)`, ANSI sequences are interspersed. `lipgloss.Height` on rendered strings is generally safe in lipgloss v1, but computing `headerHeight` from the raw const before any Render is most reliable.
**How to avoid:** Derive `headerHeight` from the raw `duchessLogo` const string before styling: `var headerHeight = strings.Count(duchessLogo, "\n") + 1`.

### Pitfall 2: contentH going negative in tiny terminals
**What goes wrong:** `m.height - headerHeight - lipgloss.Height(statusBar)` can be negative on a very short terminal (< 8 rows). `lipgloss.NewStyle().Height(negative)` panics or produces undefined output.
**Why it happens:** `baseView()` already guards with `if contentH < 0 { contentH = 0 }` but `contentView()` has its own guard at `if contentH <= 0`. Both must be updated consistently.
**How to avoid:** Ensure all four `contentH` computation sites (baseView, contentView, identityLoadedMsg handler, regionSelectedMsg handler) subtract `headerHeight` and have the `< 1` or `<= 0` guard.

### Pitfall 3: version const redeclaration
**What goes wrong:** `version` is declared in `status.go` as `const version = "0.1.0"`. If `header.go` redeclares it, Go produces a compile error ("version redeclared in this block").
**Why it happens:** Both files are in `package ui` — all top-level names share the same package scope.
**How to avoid:** `header.go` uses `version` directly without redeclaring it. Only one declaration exists in the package.

### Pitfall 4: PlaceVertical adds a trailing newline
**What goes wrong:** `lipgloss.PlaceVertical` may add a trailing newline to reach the target height, which can cause the header to render one row taller than `headerHeight`.
**Why it happens:** lipgloss counts rows by newline delimiters; padding to N rows means inserting N-1 newlines before the content, which may append a terminal newline.
**How to avoid:** After `PlaceVertical`, check `lipgloss.Height(result) == headerHeight`. Typically this works correctly — verify with a unit test asserting `lipgloss.Height(renderHeader(...))` equals `headerHeight`.

### Pitfall 5: Background style not covering full width
**What goes wrong:** If the metadata column is rendered separately from the `headerStyle.Width(width).Render(...)` call, the background only fills the styled region, leaving the gap between logo and metadata unstyled (visible as terminal default background instead of Surface).
**Why it happens:** Each `lipgloss.Render()` call applies background to its own content only.
**How to avoid:** Apply `headerStyle.Width(width).Render(entireRow)` to the assembled row string at the very end — one outer Render call, not separate renders per segment joined afterward.

---

## Code Examples

### Deriving headerHeight from ASCII art (verified pattern)

```go
// Source: stdlib strings package — COUNT newlines then +1 for last line
const duchessLogo = `     _            _
    | |          | |
  __| |_   _  ___| |__   ___  ___ ___
 / _` + "`" + `| | | |/ __| '_ \ / _ \/ __/ __|
| (_| | |_| | (__| | | |  __/\__ \__ \
 \__,_|\__,_|\___|_| |_|\___||___/___/`

// headerHeight is derived from the logo — never a bare integer.
var headerHeight = strings.Count(duchessLogo, "\n") + 1
```

### Bottom-aligned metadata column (lipgloss PlaceVertical)

```go
// Source: lipgloss v1.1 docs — PlaceVertical(height, position, str)
metaLines := strings.Join([]string{
    metaLabelStyle.Render("profile: ") + metaValueStyle.Render(m.cfg.Profile),
    metaLabelStyle.Render("region: ")  + metaValueStyle.Render(m.cfg.Region),
    metaLabelStyle.Render("refresh: ") + metaValueStyle.Render(fmt.Sprintf("%ds", m.cfg.RefreshInterval)),
    metaValueStyle.Render("v" + version),
}, "\n")

metaCol := lipgloss.PlaceVertical(headerHeight, lipgloss.Bottom, metaLines)
```

### Full-width header assembly (mirrors status.go gap-fill)

```go
// Source: internal/ui/status.go gap-fill pattern
logoStr    := logoStyle.Render(duchessLogo)
gapW       := width - lipgloss.Width(logoStr) - lipgloss.Width(metaCol)
if gapW < 0 { gapW = 0 }
row := logoStr + strings.Repeat(" ", gapW) + metaCol
return headerStyle.Width(width).Render(row)
```

### model.go integration diff (baseView)

```go
// BEFORE:
func (m rootModel) baseView() string {
    statusBar := renderStatusBar(m)
    contentH := m.height - lipgloss.Height(statusBar)
    ...
    return lipgloss.JoinVertical(lipgloss.Left, content, statusBar)
}

// AFTER:
func (m rootModel) baseView() string {
    header    := renderHeader(m, m.width)
    statusBar := renderStatusBar(m)
    contentH  := m.height - headerHeight - lipgloss.Height(statusBar)
    ...
    return lipgloss.JoinVertical(lipgloss.Left, header, content, statusBar)
}
```

---

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|-----------------|--------|
| Inline `lipgloss.Color("#FF9900")` in components | `theme.DefaultTheme.Accent` (CompleteColor) | Phase 6 completed this migration — header.go must follow the new standard from day one |
| Hardcoded height integers | Named constant derived from content | Phase 7 introduces this pattern; Phase 8 (footer) will follow suit with `footerHeight` |

**Deprecated/outdated:**
- Direct `lipgloss.Color()` in UI component files: prohibited since Phase 6; header.go must not introduce any.

---

## Open Questions

1. **Backtick in logo string literal**
   - What we know: The logo contains backtick characters (`` ` ``) in the figlet "Big" art — specifically in `` / _`| ``. Go raw string literals (`...`) cannot contain backticks.
   - What's unclear: Whether the logo as provided actually contains backticks, or if the display above uses them as visual artifacts.
   - Recommendation: Use string concatenation to embed backtick: `const duchessLogo = "..." + "`" + "..."`, or use a regular (non-raw) string with `\`` escape (not valid in Go) — actually Go does not support backtick escapes in regular strings either. The correct Go approach is string concatenation: split the raw string at each backtick occurrence and concatenate. Verify the actual logo character before implementation.

2. **`lipgloss.PlaceVertical` behavior with ANSI-styled strings**
   - What we know: `PlaceVertical` is documented to work with styled strings in lipgloss v1.
   - What's unclear: Whether background inheritance is preserved through PlaceVertical's padding lines.
   - Recommendation: Apply background style to the outer container (`headerStyle.Width(width).Render(...)`) rather than relying on PlaceVertical to carry the background. This is the safe pattern.

3. **`contentView()` double-computation of statusBar**
   - What we know: `contentView()` recomputes `statusBar := renderStatusBar(m)` independently of `baseView()`. It also computes `contentH` locally.
   - What's unclear: Whether the intent is intentional (defensive isolation) or accidental duplication.
   - Recommendation: Follow the existing pattern — update `contentView()`'s local `contentH` computation to subtract `headerHeight` as well. Do not refactor the structure.

---

## Sources

### Primary (HIGH confidence)
- `internal/ui/status.go` — authoritative reference pattern for header.go structure (package-level styles, gap-fill layout, Width render)
- `internal/ui/model.go` — exact integration points (baseView, contentView, identityLoadedMsg handler, regionSelectedMsg handler)
- `internal/ui/theme/theme.go` — confirmed token names: `Surface`, `Accent`, `Muted`, `TextPrimary`
- `go.mod` — confirmed lipgloss v1.1.0 in use; no new dependencies needed
- `internal/config/config.go` — confirmed `Config.Profile`, `Config.Region`, `Config.RefreshInterval` field names
- `.planning/phases/07-header/07-CONTEXT.md` — locked user decisions (logo art, metadata format, color tokens, height derivation requirement)

### Secondary (MEDIUM confidence)
- lipgloss v1.1 API behavior for `PlaceVertical`, `JoinVertical`, `Width()` — based on training knowledge of the lipgloss library; patterns verified against existing codebase usage

### Tertiary (LOW confidence)
- Backtick embedding approach in Go — standard Go knowledge, but the actual presence of backticks in the logo art needs to be verified against the literal characters in the art

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; same libraries as all existing UI
- Architecture: HIGH — mirrors status.go exactly; integration points are explicit in model.go source
- Pitfalls: HIGH — derived from direct code inspection of the integration points and Go language rules

**Research date:** 2026-03-11
**Valid until:** 2026-04-10 (lipgloss v1.1.0 is stable; no fast-moving dependencies)
