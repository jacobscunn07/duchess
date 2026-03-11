# Phase 6: DarkTheme Package - Research

**Researched:** 2026-03-10
**Domain:** Go struct design + lipgloss v1.1.0 color/style API
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Token set:**
- **Background**: Squid Ink (`#232F3E`) — deepest layer, main app background
- **Surface**: `#2D2D2D` — elevated panels (status bar, modals, overlays); distinct from Background for depth hierarchy
- **Border**: separate color token for panel border lines (dim gray, independently tunable from Surface)
- **Accent**: AWS Orange (`#FF9900`) — selected items, active focus states, highlighted text; no separate Selected token (selected items = accent)
- **TextPrimary**: current `Color("248")` equivalent — primary readable text
- **Muted**: current `Color("240")` equivalent — covers both separators and dimmed/secondary text (single token, same visual weight)

**Border styles:**
- Theme struct holds a `lipgloss.Border` field (rounded border — `lipgloss.RoundedBorder()`)
- Theme pre-builds `InactiveBorderStyle` and `ActiveBorderStyle` as `lipgloss.Style` — phase 9 references these directly, no per-component color math
- Active border = Accent color; inactive border = Muted color

**Struct design:**
- Exported fields — `theme.Accent`, `theme.Background`, etc. (not getter methods)
- Only pre-built styles are the two border variants; all other styles built by consuming components from color tokens

**Color fallbacks:**
- All tokens use `lipgloss.CompleteColor` (TrueColor + ANSI256 + ANSI) — consistent degradation across tmux, SSH, and 16-color terminals
- ANSI (16-color) fallback values: Claude's discretion (semantic nearest: Background → `"0"`, TextPrimary → `"7"`, Accent → `"3"`, etc.)

### Claude's Discretion
- Exact ANSI 16-color fallback values for each token
- Exact Border color hex/ANSI256 value (dim gray)
- Package-level `var DefaultTheme = DarkTheme()` convenience or just the constructor — Claude decides

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| THEME-02 | App uses AWS Dark color theme (AWS Orange `#FF9900` accent, Squid Ink `#232F3E` background, `#2D2D2D` surfaces) | `CompleteColor` struct with verified hex values; color token fields on struct |
| THEME-03 | Selected items, active borders, and focus states use AWS Orange as accent color | `InactiveBorderStyle`/`ActiveBorderStyle` pre-built on struct; Accent field referenced by delegates |
| THEME-04 | All colors and styles flow from a centralized `DarkTheme` object — no inline hardcoded colors in components | Single constructor `DarkTheme()` returning populated `Theme` struct; all current inline ANSI256 codes replaced by struct field references |
</phase_requirements>

---

## Summary

This phase creates a single Go file — `internal/ui/theme/theme.go` — that defines a `Theme` struct and a `DarkTheme()` constructor. No rendering code changes in this phase; it is purely the package that downstream phases 7–11 will import. The package has no imports from within `internal/ui/*`, so circular dependency is impossible.

The lipgloss v1.1.0 API (the version already in go.mod) provides everything needed: `CompleteColor` for three-tier color fallback, `lipgloss.RoundedBorder()` for the border shape, and `lipgloss.NewStyle().Border(...).BorderForeground(...)` for pre-building the two border styles. No new dependencies are required.

The main design risk is choosing the right ANSI 16-color fallbacks — these are rarely tested but matter in minimal SSH sessions. The choices recommended here follow semantic intent (dark → black, light text → white/gray, orange → yellow as nearest 16-color analog).

**Primary recommendation:** Create `internal/ui/theme/theme.go` with a `Theme` struct of exported `lipgloss.CompleteColor` fields, two exported `lipgloss.Style` border fields, and a `DarkTheme()` constructor that populates all fields. Add a package-level `var DefaultTheme = DarkTheme()` for convenience (zero cost, reduces boilerplate in callers).

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/charmbracelet/lipgloss` | v1.1.0 (already in go.mod) | Color types, Style, Border — all theme primitives | Already the project's styling library; no alternative needed |

### Supporting

None — this phase adds no new dependencies.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `lipgloss.CompleteColor` | `lipgloss.Color` (ANSI256 only) | `Color` auto-downgrades but gives no control over 16-color fallback; `CompleteColor` required per locked decision |
| `lipgloss.CompleteColor` | `lipgloss.AdaptiveColor` | `AdaptiveColor` switches on light/dark bg — irrelevant for a dark-only theme; adds runtime overhead for no benefit |

**Installation:** None required — lipgloss already in go.mod.

---

## Architecture Patterns

### Recommended Project Structure

```
internal/
└── ui/
    └── theme/
        └── theme.go    # Theme struct + DarkTheme() constructor
```

No subdirectories, no additional files. Single file, single package.

### Pattern 1: Flat exported struct with `CompleteColor` fields

**What:** All color tokens are exported struct fields of type `lipgloss.CompleteColor`. Consuming code reads `theme.Accent`, `theme.Background`, etc. directly.

**When to use:** Always for this phase — locked decision.

**Example:**
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/lipgloss@v1.1.0#CompleteColor
package theme

import "github.com/charmbracelet/lipgloss"

type Theme struct {
    Background lipgloss.CompleteColor
    Surface    lipgloss.CompleteColor
    Border     lipgloss.CompleteColor
    Accent     lipgloss.CompleteColor
    TextPrimary lipgloss.CompleteColor
    Muted      lipgloss.CompleteColor

    BorderShape        lipgloss.Border
    InactiveBorderStyle lipgloss.Style
    ActiveBorderStyle   lipgloss.Style
}

func DarkTheme() Theme {
    accent := lipgloss.CompleteColor{
        TrueColor: "#FF9900",
        ANSI256:   "214",
        ANSI:      "3",  // Yellow — nearest 16-color to orange
    }
    muted := lipgloss.CompleteColor{
        TrueColor: "#555555",
        ANSI256:   "240",
        ANSI:      "8",  // Bright black / dark gray
    }
    border := lipgloss.RoundedBorder()

    return Theme{
        Background:  lipgloss.CompleteColor{TrueColor: "#232F3E", ANSI256: "235", ANSI: "0"},
        Surface:     lipgloss.CompleteColor{TrueColor: "#2D2D2D", ANSI256: "236", ANSI: "0"},
        Border:      lipgloss.CompleteColor{TrueColor: "#444444", ANSI256: "238", ANSI: "8"},
        Accent:      accent,
        TextPrimary: lipgloss.CompleteColor{TrueColor: "#D0D0D0", ANSI256: "248", ANSI: "7"},
        Muted:       muted,

        BorderShape: border,
        InactiveBorderStyle: lipgloss.NewStyle().
            Border(border).
            BorderForeground(muted),
        ActiveBorderStyle: lipgloss.NewStyle().
            Border(border).
            BorderForeground(accent),
    }
}

// DefaultTheme is a package-level convenience for callers that don't need
// theme selection — avoids calling DarkTheme() at every call site.
var DefaultTheme = DarkTheme()
```

### Pattern 2: Import in consuming packages

**What:** Any `internal/ui/*` package imports `theme` and calls `theme.DefaultTheme` or `theme.DarkTheme()`.

**When to use:** Phases 7–11 consume the theme. Phase 6 only defines it.

**Example:**
```go
// In internal/ui/s3/delegate.go (Phase 7+ migration)
import "github.com/jacobscunn07/duchess/internal/ui/theme"

var selectedStyle = lipgloss.NewStyle().
    PaddingLeft(1).
    Foreground(theme.DefaultTheme.Accent).
    Bold(true)
```

### Anti-Patterns to Avoid

- **Getter methods instead of fields:** `theme.Accent()` instead of `theme.Accent`. Locked decision is exported fields — no methods.
- **Single `lipgloss.Color` per token:** Loses ANSI/ANSI256 control. All tokens must use `CompleteColor`.
- **Embedding lipgloss styles for all tokens:** Only the two border styles are pre-built on the struct. Other styles (text, selected row) are composed by consumers at their declaration sites.
- **Circular import:** `internal/ui/theme` must import ONLY `lipgloss` — no imports from `internal/ui` or its sub-packages.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Three-tier color fallback | Custom color wrapper struct | `lipgloss.CompleteColor` | Already does exactly this — TrueColor/ANSI256/ANSI fields, zero overhead |
| Rounded border shape | Manually build `lipgloss.Border` struct | `lipgloss.RoundedBorder()` | Factory function returns correct Unicode box-drawing chars for rounded corners |
| Border style pre-build | Inline `.Border(...).BorderForeground(...)` in every component | Pre-built `InactiveBorderStyle`/`ActiveBorderStyle` on `Theme` struct | Locked decision; avoids duplicated color math in Phase 9 |

**Key insight:** The entire phase is essentially: fill a struct with lipgloss primitives. No logic, no computation, no custom types beyond the struct itself.

---

## Common Pitfalls

### Pitfall 1: Wrong ANSI256 code for Squid Ink / Surface

**What goes wrong:** `#232F3E` doesn't map cleanly to a named xterm-256 color. Using `"235"` is closest (dark blue-gray), but visually it may read too blue vs. the hex.

**Why it happens:** Squid Ink is a custom AWS brand color — not aligned to the standard 256-color palette.

**How to avoid:** Accept that 256-color mode is a degraded fallback, not a pixel-perfect match. `"235"` is the nearest dark blue-gray. Verify visually in a 256-color terminal (e.g., `TERM=xterm-256color` without truecolor) and adjust if needed.

**Warning signs:** Background looks notably blue-shifted in 256-color SSH sessions.

### Pitfall 2: `DefaultTheme` var initialized before lipgloss color profile is set

**What goes wrong:** `model.go` calls `lipgloss.SetColorProfile(termenv.Ascii)` when `NO_COLOR` is set. If `var DefaultTheme = DarkTheme()` runs before `NewRootModel()`, the color profile is already set at init time — but `CompleteColor` bypasses auto-degradation anyway (that's the point), so this is NOT a problem for color selection. However, if `lipgloss.NewStyle()` calls inside `DarkTheme()` capture renderer state, rendering could be wrong.

**Why it happens:** Go package-level `var` initializes at program start, before `main()` runs, before the `NO_COLOR` check in `NewRootModel()`.

**How to avoid:** `CompleteColor` is immune — it does not use the color profile for selection. The pre-built `Style` values in `InactiveBorderStyle`/`ActiveBorderStyle` only store border shape and foreground color; the actual rendering respects the profile at render time, not at style-creation time. This is safe. No action required, but document the initialization order clearly.

**Warning signs:** None expected — CompleteColor is designed for exactly this use case.

### Pitfall 3: ANSI256 code `"214"` for orange — verify it's not clamped

**What goes wrong:** In very old terminals claiming 256-color support, color `214` (bright orange) may render differently than expected.

**Why it happens:** `214` is correct xterm-256 orange (`#ffaf00`), not `#FF9900` exactly, but it's the closest match.

**How to avoid:** Accept the approximation. The TrueColor path renders the exact hex. ANSI256 is the fallback path.

**Warning signs:** Orange looks more yellow than expected in 256-color terminals — this is expected and acceptable.

### Pitfall 4: Naming collision — `normalStyle` / `selectedStyle` in delegate packages

**What goes wrong:** Both `internal/ui/s3/delegate.go` and `internal/ui/ecs/delegate.go` declare package-level `var normalStyle` and `var selectedStyle`. When Phase 7+ migrates these to use theme tokens, the var block must be updated — not just the color value.

**Why it happens:** Both delegate packages use identical var names — not a conflict (they're in separate packages) but easy to miss one.

**How to avoid:** When migrating in downstream phases, grep for `selectedStyle` and `normalStyle` across the whole codebase to catch all instances.

**Warning signs:** A delegate still renders `Color("205")` (hot pink) after migration — means one file was missed.

---

## Code Examples

Verified patterns from official sources:

### CompleteColor declaration
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/lipgloss@v1.1.0#CompleteColor
accent := lipgloss.CompleteColor{
    TrueColor: "#FF9900",
    ANSI256:   "214",
    ANSI:      "3",
}
```

### Pre-building a border style
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/lipgloss@v1.1.0#Style.Border
activeStyle := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(accent)
```

### Consuming a theme field as a foreground color
```go
// CompleteColor implements lipgloss.TerminalColor — use directly in style methods
style := lipgloss.NewStyle().Foreground(theme.DefaultTheme.Accent).Bold(true)
```

### Package-level DefaultTheme convenience
```go
// Initializes once at program start. Safe because CompleteColor bypasses profile detection.
var DefaultTheme = DarkTheme()
```

---

## Recommended ANSI 16-Color Fallback Values

These are Claude's discretion (per CONTEXT.md). Reasoning:

| Token | TrueColor | ANSI256 | ANSI | Reasoning |
|-------|-----------|---------|------|-----------|
| Background | `#232F3E` | `"235"` | `"0"` | Black — darkest available; Squid Ink is a dark background |
| Surface | `#2D2D2D` | `"236"` | `"0"` | Also black — 16-color can't distinguish Surface from Background; acceptable |
| Border | `#444444` | `"238"` | `"8"` | Bright black (dark gray) — dim separator, same semantic weight |
| Accent | `#FF9900` | `"214"` | `"3"` | Yellow — nearest 16-color to orange; `"3"` is standard yellow |
| TextPrimary | `#D0D0D0` | `"248"` | `"7"` | Light gray / white — readable on dark background |
| Muted | `#555555` | `"240"` | `"8"` | Bright black — dim/secondary, same as Border in 16-color |

**DefaultTheme vs per-call:** Recommend `var DefaultTheme = DarkTheme()` at package level. Rationale: every downstream phase (7–11) needs the theme at var-declaration time for their own package-level style vars. Without a package-level var, they'd each call `DarkTheme()` in an `init()` or pass a theme instance through the call stack — unnecessary complexity for a single fixed theme.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Inline `Color("205")` / `Color("33")` per file | Centralized `theme.Accent` field | This phase (Phase 6) | Single change point for all accent color usage |
| No 16-color fallback (ANSI256 auto-downgrades) | `CompleteColor` with explicit 3-tier fallback | This phase | Correct orange rendering in tmux/SSH/16-color sessions |

**Current codebase color inventory (all to be replaced in downstream phases):**
- `Color("205")` — hot pink selected items (s3/delegate.go:72, ecs/delegate.go:48, model.go:77 spinner) → will become `theme.Accent`
- `Color("33")` — blue accent for profile/region/interval in status bar (status.go:24,27,30) and s3 prefixes (s3/delegate.go:73) → will become `theme.Accent`
- `Color("248")` — primary text (status.go:18, 34, 38; overlay/region.go:123) → will become `theme.TextPrimary`
- `Color("240")` — muted separator (status.go:43) → will become `theme.Muted`
- `Color("236")` — status bar background (status.go:15) → will become `theme.Surface`
- `Color("62")` — overlay border (overlay/region.go:49, 95) → will become `theme.Accent` or a border style in downstream phases
- `Color("205")` — spinner foreground (model.go:77) → will become `theme.Accent`

---

## Open Questions

1. **`#232F3E` vs actual AWS dark background**
   - What we know: `#232F3E` is the community-consensus Squid Ink value used across AWS-themed tools
   - What's unclear: AWS doesn't publish an official color spec for their console dark theme
   - Recommendation: Use `#232F3E` as decided; note in STATE.md that visual verification on first render is required (already documented as a blocker/concern)

2. **Border dim gray exact hex**
   - What we know: Must be "dim gray, independently tunable from Surface" per CONTEXT.md
   - What's unclear: No specific hex mandated
   - Recommendation: Use `#444444` / ANSI256 `"238"` / ANSI `"8"` — sits between Surface (`#2D2D2D`) and TextPrimary (`#D0D0D0`), creates visible but unobtrusive panel dividers

---

## Sources

### Primary (HIGH confidence)
- `https://pkg.go.dev/github.com/charmbracelet/lipgloss@v1.1.0#CompleteColor` — struct fields, usage, TerminalColor interface
- `https://pkg.go.dev/github.com/charmbracelet/lipgloss@v1.1.0` — Border types, RoundedBorder(), Style.Border(), Style.BorderForeground()
- `/Users/jacobcunningham/_CODE/duchess/go.mod` — confirmed lipgloss v1.1.0 in use
- Existing codebase (`status.go`, `s3/delegate.go`, `ecs/delegate.go`, `overlay/region.go`) — complete inventory of all inline colors to be replaced

### Secondary (MEDIUM confidence)
- ANSI256 `"214"` for orange: xterm-256 color chart consensus — nearest to `#FF9900`; ANSI256 `"235"` / `"236"` for Squid Ink / Surface dark backgrounds

### Tertiary (LOW confidence)
- `#232F3E` as canonical Squid Ink: community consensus, not official AWS spec (flagged in STATE.md)

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — lipgloss v1.1.0 confirmed in go.mod, CompleteColor API verified in official docs
- Architecture: HIGH — simple Go struct, no framework decisions, locked design from CONTEXT.md
- Pitfalls: MEDIUM — ANSI256 color approximations are "best effort" by nature; 16-color fallbacks are Claude's discretion per CONTEXT.md

**Research date:** 2026-03-10
**Valid until:** 2026-06-10 (lipgloss API is stable; color values are static)
