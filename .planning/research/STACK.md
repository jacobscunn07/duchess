# Stack Research

**Domain:** Go TUI — UI/UX and Visual Polish (v1.1 milestone)
**Researched:** 2026-03-10
**Confidence:** HIGH — existing go.mod pins verified; all new patterns confirmed against Lipgloss v1.1.0 source and charmbracelet project examples

---

## Context: What This Milestone Adds

This STACK.md covers only NEW capabilities required for the v1.1 Visual Polish milestone. The v1.0 base stack (Bubble Tea v1.3.10, Bubbles v1.0.0, Lipgloss v1.1.0, AWS SDK v2, Cobra, ini.v1, yaml.v3) is already in go.mod and remains unchanged.

The four new capability areas:

1. Centralized Lipgloss theme object (DarkTheme struct)
2. ASCII art rendering for the persistent header logo
3. Two-panel side-by-side layout (left nav + right content)
4. Service switcher modal following the existing profile/region overlay pattern

**Verdict up front:** No new dependencies are needed for any of these four areas. Everything required is already in go.mod or can be implemented with a hardcoded string constant.

---

## Recommended Stack

### Core Technologies (unchanged — already in go.mod)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `github.com/charmbracelet/lipgloss` | v1.1.0 | All layout, borders, colors, style composition | Already pinned. Provides `JoinHorizontal`, `JoinVertical`, `NewStyle().Inherit()`, `lipgloss.Color()`, `AdaptiveColor`, `RoundedBorder()`. Everything needed for themed layout and two-panel views. |
| `github.com/charmbracelet/bubbletea` | v1.3.10 | Event loop, modal routing | Already pinned. Service switcher follows same `isOverlayOpen bool` + `btoverlay.Composite()` pattern as existing profile/region overlays. |
| `github.com/charmbracelet/bubbles` | v1.0.0 | `list.Model` inside service switcher overlay | Already pinned. Existing ProfileOverlay and RegionOverlay both use `bubbles/list` — service switcher uses the same approach. |
| `github.com/rmhubbert/bubbletea-overlay` | v0.6.5 | Floating modal compositing | Already pinned. `btoverlay.Composite(foreground, background, btoverlay.Center, btoverlay.Center, 0, 0)` is the exact call pattern used for profile and region overlays today. |

### Supporting Libraries (no additions needed)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/charmbracelet/lipgloss` | v1.1.0 | Theme struct, style inheritance, panel layout | Used everywhere — all four capability areas live inside lipgloss |
| `github.com/muesli/termenv` | v0.16.0 | `termenv.Ascii` for NO_COLOR guard | Already imported in model.go. Continue using existing `lipgloss.SetColorProfile(termenv.Ascii)` pattern for NO_COLOR. |

### Development Tools (unchanged)

| Tool | Purpose | Notes |
|------|---------|-------|
| `go test ./...` | Run tests | Existing test files (`model_test.go`, `status_test.go`) should be extended for header and panel layout |
| `go vet` | Static analysis | Run after each layout change |

---

## Capability Area 1: Centralized Lipgloss Theme Object

**Pattern:** Define a `Theme` struct (or `Styles` struct) in a dedicated `internal/ui/theme/` package. All `lipgloss.Style` values and raw `lipgloss.Color` values live on this struct. Every component receives a pointer or value of this struct at construction time — zero inline colors anywhere else in the codebase.

**Why this pattern over package-level vars:** Package-level `var` styles (the current pattern in `status.go`) cannot be passed to child components without importing the parent package. A theme struct is portable — panels, overlays, and the header all receive the same theme without circular imports.

**Existing precedent in Charmbracelet ecosystem:** The `charmbracelet/crush` project uses this exact pattern: a `Styles` struct in `internal/ui/styles/styles.go` holds all `lipgloss.Style` fields; a `DefaultStyles()` constructor maps semantic roles to color values; a `Common` carrier struct propagates `*Styles` into every component at construction. (MEDIUM confidence — sourced from DeepWiki analysis of crush source; official docs do not prescribe this pattern, but it is the production-verified approach.)

**AWS Dark color palette for DarkTheme:**

| Token | Hex | ANSI256 fallback | Semantic role |
|-------|-----|-----------------|---------------|
| `ColorAccent` | `#FF9900` | `214` | AWS Orange — primary accent, selected items, borders |
| `ColorBackground` | `#232F3E` | `235` | AWS Navy / Squid Ink — main background |
| `ColorSurface` | `#1C2B33` | `234` | Slightly darker surface for panels, sidebars |
| `ColorBorder` | `#37475A` | `238` | Muted border color for non-focused elements |
| `ColorTextPrimary` | `#FFFFFF` | `15` | Primary text |
| `ColorTextSecondary` | `#8AA0B2` | `109` | Dimmed/secondary text, metadata, separators |
| `ColorTextMuted` | `#526778` | `103` | Timestamp, minor metadata |
| `ColorError` | `#FF6B6B` | `203` | Error state text |
| `ColorSuccess` | `#6BCB77` | `114` | Success / running state |

Color source: `#FF9900` and `#232F3E` are officially documented AWS brand colors (brandpalettes.com confirms `#252F3E` as official navy; `#232F3E` is the community-standard Squid Ink variant used in dark AWS community themes). All other tokens are derived from the AWS dark console inspection — MEDIUM confidence. The exact values should be verified visually on first render and tuned.

**Style inheritance pattern using `lipgloss v1.1.0`:**

`lipgloss.NewStyle().Inherit(baseStyle)` copies all unset properties from `baseStyle` onto the receiver. Use this to propagate `Background` from the panel base style to child text styles so background bleeding is eliminated. This is exactly how `status.go` currently works (e.g., `versionStyle.Inherit(statusBarStyle)`).

**Theme struct example shape** (implementation detail, not a recipe to follow verbatim):

```go
// internal/ui/theme/theme.go
type Theme struct {
    // Raw colors — used for runtime color computation
    Accent         lipgloss.Color
    Background     lipgloss.Color
    Surface        lipgloss.Color
    Border         lipgloss.Color
    TextPrimary    lipgloss.Color
    TextSecondary  lipgloss.Color

    // Composed styles — ready to use in View()
    Header         lipgloss.Style
    StatusBar      lipgloss.Style
    PanelLeft      lipgloss.Style
    PanelRight     lipgloss.Style
    ModalBorder    lipgloss.Style
    Selected       lipgloss.Style
    Separator      lipgloss.Style
}

func DarkTheme() Theme { ... }
```

Components accept `theme.Theme` as a constructor argument, not individual style values.

---

## Capability Area 2: ASCII Art Header Logo

**Recommendation: Hardcoded multi-line string constant. Do not add a new dependency.**

**Rationale:** "duchess" is a fixed word that never changes. A FIGlet-style library adds a dependency and startup cost for a string that can be authored once and embedded directly. The logo renders identically every time, requires no font files, and can be styled with Lipgloss after the fact (color, bold, foreground).

**How to implement:**

```go
// internal/ui/header/logo.go
const duchessLogo = `
 _           _
| |         | |
| |__   __ _| |_  ___  ___
|  _ \ / _' | __|/ _ \/ __|
| | | | (_| | |_|  __/\__ \
|_| |_|\__,_|\__|\___||___/
`
```

Author the ASCII art once using a FIGlet web tool (e.g., `patorjk.com/software/taag/` with the "Standard" or "Slant" font). Copy the output into the constant. Apply Lipgloss `Foreground(theme.Accent)` to color it at render time.

**Why not go-figure (`github.com/common-nighthawk/go-figure`):**
- Last committed ~2021, no releases since then, no semantic version tag — go.mod picks up a pseudo-version (MEDIUM confidence, based on Debian package date `0.0~git20210622`).
- Requires shipping a FIGlet font file or embedding the font bytes — unnecessary complexity for a fixed logo.
- `figure.String()` method returns the art as a string — the API works, but the dependency is not worth it for a static logo.

**Why not moul/banner (`moul.io/banner`):**
- Pure Go, returns a string, actively maintained. Technically viable.
- Still an unnecessary dependency for a word that never changes.
- The generated font is small and may not match the visual character of the intended logo.

**If dynamic ASCII art is ever needed** (e.g., for user-configured display names): use `moul.io/banner` as the lightest option — pure Go stdlib implementation, returns a string, no font files needed.

---

## Capability Area 3: Two-Panel Side-by-Side Layout

**API:** `lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)`

`JoinHorizontal` is a first-class function in Lipgloss v1.1.0 that concatenates two multi-line strings side-by-side, aligning them at the top, center, or bottom. This is the correct primitive — not manual string manipulation.

**Width budget calculation (critical — avoid common pitfall):**

In Lipgloss v1.1.0, `Style.Width(n)` sets the *content* width. When a border is present, the total rendered width is `n + borderWidth`. For `RoundedBorder()`, each side adds 1 character, so total rendered width = `n + 2` horizontally.

The correct pattern for splitting a terminal of width `W` into two panels:

```go
// Given: W = total terminal width
// Given: leftRatio = 0.3 (30% left, 70% right — typical for nav/content split)

borderCost := 2 // 1 char per side for RoundedBorder

leftContentW  := int(float64(W) * leftRatio) - borderCost
rightContentW := W - int(float64(W)*leftRatio) - borderCost

// Guard against negative widths on narrow terminals
if leftContentW < 1  { leftContentW = 1 }
if rightContentW < 1 { rightContentW = 1 }

leftRendered  := theme.PanelLeft.Width(leftContentW).Height(contentH).Render(leftContent)
rightRendered := theme.PanelRight.Width(rightContentW).Height(contentH).Render(rightContent)

view := lipgloss.JoinHorizontal(lipgloss.Top, leftRendered, rightRendered)
```

**Do not use `GetHorizontalFrameSize()` for the split calculation.** There are documented Lipgloss issues where `GetHorizontalFrameSize()` inconsistently includes padding but not borders depending on which border sides are explicitly set. Manual subtraction of `borderCost` is more predictable for this use case.

**Height budget:** The content height passed to each panel must account for the header row and status bar row(s). Use `lipgloss.Height(headerView)` and `lipgloss.Height(statusBarView)` to compute the actual consumed rows, then subtract from terminal height. This is the pattern already used in `rootModel.contentView()` for the status bar.

**Panel focus state:** The focused panel's border should use `theme.Accent` foreground; the unfocused panel uses `theme.Border`. Implement as two separate named styles on the theme, or as a method `theme.PanelStyle(focused bool) lipgloss.Style`.

---

## Capability Area 4: Service Switcher Modal

**No new dependency.** Follow the exact pattern of `ProfileOverlay` and `RegionOverlay` in `internal/ui/overlay/`.

**Pattern summary:**

```
rootModel fields:
  isServiceOverlayOpen bool
  serviceOverlay       overlay.ServiceOverlay

rootModel.Update():
  case "s": open overlay, same guard as "p" and "r"
  when open: intercept all keys, route to overlay.Update()
  on "enter": emit serviceSelectedMsg{service: selected}
  on "esc": close overlay

rootModel.View():
  if isServiceOverlayOpen {
      return btoverlay.Composite(m.serviceOverlay.View(), bg, btoverlay.Center, btoverlay.Center, 0, 0)
  }
```

`overlay.ServiceOverlay` is a struct with a `bubbles/list.Model` inside, a `lipgloss.Style` border (using `theme.ModalBorder`), and `SelectedService() string`. The service list is static: `["S3", "ECS"]` for v1.1, extensible for v2 services.

**Consistency with existing overlays:** Use `lipgloss.RoundedBorder()` (already used in `ProfileOverlay.borderStyle`) and `theme.Accent` as `BorderForeground`. Size the overlay to `termWidth/2` wide and list height + padding tall, matching the existing profile overlay sizing formula.

---

## Installation

```bash
# No new dependencies needed for v1.1 Visual Polish milestone.
# All four capability areas use only what is already in go.mod.

# Verify current state:
go mod verify

# After implementing the theme package, tidy if any indirect deps shift:
go mod tidy
```

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Hardcoded ASCII logo string | `go-figure` library | Only if the app logo needs to be configurable at runtime or generated from user input. Not applicable here. |
| Hardcoded ASCII logo string | `moul.io/banner` | If you want dynamically generated ASCII from arbitrary text (e.g., per-profile display names). Pure Go, returns a string — it is the least-bad library option if ever needed. |
| Manual border cost subtraction for panels | `GetHorizontalFrameSize()` | Acceptable when borders are explicitly set on all four sides and no implicit border styling is used. The manual approach is safer given known Lipgloss frame-size bugs. |
| `btoverlay.Composite()` for service switcher | Direct z-order rendering (manual string overlay) | Only if bubbletea-overlay were removed from the project. It is already a direct dependency — use it. |
| Theme struct in dedicated package | Package-level vars (current status.go pattern) | Acceptable for a single file. Breaks down when multiple packages (panels, overlays, header) need consistent styling — package-level vars create import cycles or duplication. |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `lipgloss/v2` (`charm.land/lipgloss/v2`) | Breaking import path change. Incompatible with Bubble Tea v1.3.10 and existing codebase. Causes go.mod split. | Continue using `github.com/charmbracelet/lipgloss` v1.1.0 |
| `common-nighthawk/go-figure` | Unmaintained since 2021, no semver tag, adds a dependency for a static logo that is more reliably expressed as a string constant. | Hardcoded multi-line string constant |
| Inline colors in component `View()` functions | Defeats the purpose of DarkTheme. Makes future theming (light mode, high-contrast) impossible. Creates inconsistency as the codebase grows. | All colors flow from the Theme struct |
| `lipgloss.Place()` for panel layout | `lipgloss.Place()` is for centering a widget inside a blank field — correct for loading spinners and error messages (already used correctly). Not for side-by-side panels. | `lipgloss.JoinHorizontal()` |
| Per-render `lipgloss.NewStyle()` construction in `View()` | Allocates on every render tick. Current `status.go` already avoids this with package-level vars. Theme struct init at startup is the right evolution. | Construct styles once in `DarkTheme()`, cache on struct |

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `lipgloss v1.1.0` | `bubbletea v1.3.10` | Both use `github.com/` import paths. lipgloss v2 is on `charm.land/` — do not use. |
| `bubbletea-overlay v0.6.5` | `bubbletea v1.3.10` | Already in go.mod and working for profile/region overlays. |
| `bubbles v1.0.0` | `bubbletea v1.3.10` + `lipgloss v1.1.0` | Stable release; all three charmbracelet core packages are version-compatible at these pins. |
| Hardcoded ASCII string | Any Go version | No compatibility concerns — it is a string constant. |

---

## Sources

- `github.com/charmbracelet/lipgloss` v1.1.0 — `JoinHorizontal`, `Style.Inherit()`, `RoundedBorder()`, `AdaptiveColor`, border width calculation behavior
- `github.com/charmbracelet/lipgloss` issue #449 / PR #451 — Border width not included in `Style.Width()` in v1; manual subtraction required
- `github.com/charmbracelet/lipgloss` issue #298 — `GetHorizontalFrameSize()` inconsistency with padding/border — motivates manual `borderCost` pattern
- `github.com/rmhubbert/bubbletea-overlay` v0.6.5 — `btoverlay.Composite()` signature; already live in `internal/ui/model.go`
- Charmbracelet crush project (via DeepWiki) — `Styles` struct pattern, `DefaultStyles()` constructor, `Common` carrier — MEDIUM confidence
- AWS brand colors: `brandpalettes.com/amazon-web-services-logo-colors/` — `#252F3E` navy, `#FF9900` orange (HIGH confidence)
- `#232F3E` Squid Ink variant: community consensus from AWS dark theme projects (MEDIUM confidence — not official AWS design system documentation)
- `common-nighthawk/go-figure` last commit ~2021 — Debian package tracker date `0.0~git20210622` (MEDIUM confidence on maintenance status)
- `moul.io/banner` — pure Go ASCII art, `pkg.go.dev/moul.io/banner` (MEDIUM confidence — exists and works but not needed here)

---

*Stack research for: duchess v1.1 Visual Polish — themed layout, ASCII header, two-panel layout, service switcher modal*
*Researched: 2026-03-10*
