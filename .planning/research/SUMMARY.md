# Project Research Summary

**Project:** duchess — v1.1 Visual Polish milestone
**Domain:** Go TUI — k9s-inspired layout redesign for an AWS CLI tool
**Researched:** 2026-03-10
**Confidence:** HIGH

## Executive Summary

duchess v1.1 is a visual polish milestone on top of a fully working v1.0 AWS TUI. The goal is a k9s-class layout: a persistent header with ASCII logo, a two-panel split (left nav + right content), a centralized AWS Dark color theme, a service switcher modal, and a simplified footer. Research confirms that no new dependencies are required — the existing stack (Bubble Tea v1.3.10, Lipgloss v1.1.0, Bubbles v1.0.0, bubbletea-overlay v0.6.5) contains every primitive needed. The four feature areas are tightly coupled by a single dependency root: the `DarkTheme` struct. It must be built first, because every other component in the milestone consumes it.

The recommended implementation order follows the dependency graph exactly: build the theme package first, then migrate the footer and implement the header (establishing the height constants that drive all body-height math), then wire the service switcher overlay (the third instance of a proven pattern already used for profile and region switching), and finally restructure `rootModel.baseView()` into a three-zone vertical layout with a two-panel horizontal body. The existing s3panel and ecspanel models require no logic changes — they render into whatever dimensions they receive. This is a structural refactor of the rendering pipeline, not a new feature build.

The primary risks are mechanical rather than conceptual: Lipgloss border-width off-by-one math, height budget drift across three call sites in model.go, style inheritance silent failures, and overlay compositing against a partial instead of full view string. All risks have documented prevention strategies sourced from confirmed Lipgloss GitHub issues and existing codebase patterns. Overall confidence in delivering this milestone is HIGH.

## Key Findings

### Recommended Stack

No new dependencies are needed. All four capability areas (centralized theme, ASCII header, two-panel layout, service switcher modal) are fully implemented within the existing go.mod. Lipgloss v1.1.0 provides `JoinHorizontal`, `JoinVertical`, `Style.Inherit()`, `RoundedBorder()`, and direct hex color support. bubbletea-overlay v0.6.5 provides `btoverlay.Composite()` for modal compositing — already live in model.go for the profile and region overlays.

**Core technologies:**
- `lipgloss v1.1.0`: All layout, borders, colors, style composition — `JoinHorizontal` for the two-panel body, `JoinVertical` for three-zone chrome, `CompleteColor` for 256-color fallbacks on AWS Orange
- `bubbletea v1.3.10`: Event loop and modal routing — service switcher follows the same `isOverlayOpen bool` routing pattern as existing profile/region overlays
- `bubbletea-overlay v0.6.5`: Floating modal compositing — `btoverlay.Composite(foreground, background, btoverlay.Center, btoverlay.Center, 0, 0)` is the exact call pattern already wired in model.go
- `bubbles v1.0.0`: `list.Model` inside the service switcher overlay, identical in usage to both existing overlay implementations

**Critical version constraint:** Do not upgrade to `lipgloss v2` (`charm.land/lipgloss/v2`). It uses a different import path and is incompatible with bubbletea v1.3.10. Stay on `github.com/charmbracelet/lipgloss` v1.1.0.

**AWS Dark color palette (primary tokens):**

| Token | Hex | Role |
|-------|-----|------|
| Accent | `#FF9900` | AWS Orange — logo, selected nav item, focus borders |
| Background | `#232F3E` | AWS Squid Ink — main surface |
| Surface | `#1C2B33` | Slightly darker panels and sidebars |
| Border | `#37475A` | Non-focused element borders |
| TextPrimary | `#FFFFFF` | Primary text |
| TextSecondary | `#8AA0B2` | Metadata, separators, muted labels |

Use `lipgloss.CompleteColor{TrueColor: "#FF9900", ANSI256: "214", ANSI: "3"}` for the Accent token to prevent muddy color degradation in 256-color terminals (tmux, SSH sessions without truecolor).

See `.planning/research/STACK.md` for full capability-area breakdowns, alternative analysis, and version compatibility table.

### Expected Features

All eight in-scope features are P1 — this milestone ships as a coherent whole or not at all. Shipping the theme without the layout, or the layout without the theme, produces an inconsistent result that is visually worse than the v1.0 state.

**Must have (table stakes — k9s-class TUI users expect these):**
- Persistent header with ASCII logo (4-6 rows; logo left, metadata right)
- Header metadata showing profile, region, version, refresh timestamp
- Two-panel layout: fixed-width left nav (~20 cols) and right content taking the remainder
- Left nav displaying the service list with active service highlighted
- Rounded borders on all panels
- Revised footer: principal ARN left, clock right (panel service indicator removed)
- Modal overlays following the existing profile/region overlay conventions exactly

**Should have (visual differentiators):**
- Centralized DarkTheme struct with AWS Orange accent — eliminates scattered inline `lipgloss.Color("XX")` strings across 5+ files and enables future theme switching
- Service switcher modal on `s` key — mnemonic, parallel to `p` (profile) and `r` (region)
- Right content panel service-name subheader for instant orientation

**Defer (v1.2+):**
- Additional AWS services (EC2, Lambda, CloudWatch) in left nav — only valuable after the two-panel structure is proven with 2 services
- Light theme variant — only if user feedback confirms a meaningful light-terminal audience

**Defer (v2+):**
- Config-driven theme selection — requires theme file parsing infrastructure
- Animated logo or environment-specific warning themes (k9s issue #2090 shows this is requested but complex)

**Anti-features explicitly out of scope:**
- Per-panel focus ring borders — left nav is display-only; a focus ring implies interactivity that does not exist
- Mouse hover effects — project is intentionally keyboard-only per PROJECT.md
- Full-width ASCII banner consuming 8+ rows — wastes 30%+ of vertical space in a 24-row terminal

See `.planning/research/FEATURES.md` for full feature table, dependency graph, and existing code integration points.

### Architecture Approach

The v1.1 architecture is a restructure of `rootModel`'s rendering pipeline, not a new model. `baseView()` becomes a three-zone vertical composition (`JoinVertical(header, body, footer)`). The body is a two-panel horizontal composition (`JoinHorizontal(navPanel, contentPanel)`). All components share a single `theme.Theme` value passed at construction time — the theme is immutable after initialization. The existing s3panel and ecspanel models are unchanged except that they receive narrower content widths via the existing WindowSizeMsg path.

**Major components:**
1. `internal/ui/theme/theme.go` — DarkTheme struct and constructor; all Lipgloss Style values constructed once; isolated package to prevent circular imports
2. `internal/ui/header.go` — stateless render function (not a model); ASCII logo + metadata row; `headerHeight` constant derived from logo line count
3. `rootModel.baseView()` / `contentView()` — three-zone vertical layout wrapping a two-panel horizontal body; all height and width budget math lives here via `bodyHeight()` helper
4. `overlay/service.go` — ServiceSwitcherOverlay; third instance of the profile/region pattern; dispatches `serviceSelectedMsg` (UI-only, no session reset)
5. `internal/ui/status.go` (revised footer) — panel indicator removed; DarkTheme tokens replace all inline color strings

**Dependency-aware build order:**
1. `theme/theme.go` — no dependencies; all other components depend on this
2. Footer migration — replaces inline colors; establishes `footerHeight` constant
3. `header.go` — depends on theme; establishes `headerHeight` constant
4. `overlay/service.go` + `serviceSelectedMsg` — follows profile.go pattern
5. `rootModel` wiring — `baseView()` restructure, `bodyHeight()` helper, two-panel `contentView()`
6. Panel color substitution — migrate s3panel/ecspanel inline colors to theme tokens
7. Validation — all three height-budget sites consistent; WindowSizeMsg forwarding unconditional; existing profile/region overlays composite correctly over new layout

See `.planning/research/ARCHITECTURE.md` for full component diagram, message flow, and anti-pattern catalog.

### Critical Pitfalls

The research identified 7 critical pitfalls. The top 5 most likely to cause silent failures or rework if not addressed in the right phase:

1. **Lipgloss `.Width()` does not include border size** — `Style.Width(n)` with `RoundedBorder()` renders at `n+2` columns. Two panels side by side overflow the terminal. Subtract `borderCost := 2` from each panel's content width before setting `.Width()`. Never use `GetHorizontalFrameSize()` (documented inconsistency in Lipgloss issue #298).

2. **Height budget drift across three sites in model.go** — Three locations compute content height. Define `headerHeight` and `footerHeight` as package-level constants referenced everywhere. Derive `headerHeight` from the actual ASCII art string (`strings.Count(logoStr, "\n") + 1`) rather than a hardcoded integer. If the logo changes, the budget adjusts automatically.

3. **Overlay compositing against partial view** — After the layout restructure, `btoverlay.Composite()` must receive `m.baseView()` (full terminal output, including header and footer) as the background argument. If it receives only `contentView()`, the header and footer disappear behind the modal. Verify all three existing overlay call sites after the layout restructure.

4. **Style `.Inherit()` silent divergence** — `childStyle.Inherit(parentStyle)` only fills unset properties; it does not override existing values. Build each style explicitly from `theme.Theme` token fields (`lipgloss.NewStyle().Background(t.Surface).Foreground(t.Text)...`). Do not use `.Inherit()` as the primary style propagation mechanism from the theme.

5. **AWS Orange degrades in 256-color terminals** — `lipgloss.Color("#FF9900")` falls back to a muddy brown-yellow in tmux or SSH sessions without truecolor. Use `lipgloss.CompleteColor{TrueColor: "#FF9900", ANSI256: "214", ANSI: "3"}` for the Accent token to provide an explicit fallback.

See `.planning/research/PITFALLS.md` for full pitfall catalog with recovery strategies, UX pitfalls, and a "looks done but isn't" verification checklist.

## Implications for Roadmap

Research produces a clear 4-phase build order driven by the dependency graph. The DarkTheme struct is the hard prerequisite for everything else. Subsequent phases each build on the previous with minimal rework risk.

### Phase 1: Centralized Theme
**Rationale:** DarkTheme is the dependency root. Every other component in the milestone consumes it. Building panels or the header before the theme exists means touching every file twice. This phase introduces zero interaction with existing rendering code — it is a new file in a new package.
**Delivers:** `internal/ui/theme/theme.go` with `DarkTheme()` constructor; all color tokens including `CompleteColor` for AWS Orange; composed Lipgloss styles for panel borders, modal borders, selected items, and text variants
**Addresses:** Centralized AWS Dark palette, rounded border styles, 256-color degradation prevention
**Avoids:** Inline color proliferation, style divergence across components (Pitfalls 4, 5)

### Phase 2: Chrome — Header and Footer
**Rationale:** The header establishes `headerHeight` as a constant, which drives all body-height math in Phase 3. The footer migration establishes `footerHeight` and proves the theme token replacement pattern before it is applied broadly. Both are stateless render functions — the lowest-risk implementation in the milestone.
**Delivers:** `internal/ui/header.go` with ASCII logo + metadata row; revised `status.go` footer (panel indicator removed; DarkTheme tokens replacing all inline `lipgloss.Color` strings); `headerHeight` and `footerHeight` package constants
**Addresses:** Persistent header with ASCII logo, header metadata display, revised footer format
**Avoids:** Hardcoded height integers that drift when ASCII art changes (Pitfall 3), ASCII art width miscalculation (Pitfall 7)

### Phase 3: Two-Panel Layout
**Rationale:** With header and footer heights established as constants, `rootModel` can safely restructure `baseView()` and `contentView()`. This is the highest mechanical complexity of the milestone — three height-budget sites in model.go must be updated consistently, `WindowSizeMsg` forwarding must be made unconditional, and the `navPanelRatio` constant must drive all split math.
**Delivers:** `rootModel.baseView()` restructured to `JoinVertical(header, body, footer)`; `rootModel.contentView()` restructured to `JoinHorizontal(navPanel, contentPanel)`; `navPanelRatio = 0.30` named constant; `bodyHeight()` helper; left nav with active-service highlight using accent color; right content service-name subheader
**Addresses:** Two-panel layout, left nav service list, active-service highlight, rounded panel borders
**Avoids:** Border width off-by-2 (Pitfall 1), WindowSizeMsg not forwarded during loading (Pitfall 2), height budget drift (Pitfall 3)

### Phase 4: Service Switcher Modal and Color Migration
**Rationale:** The service switcher is the third instance of the overlay pattern — structurally isolated and low-risk. Bundling it with the panel color migration sweep completes the theme rollout and validates that all `btoverlay.Composite()` calls target the fully restructured `baseView()`. Both efforts are fast; separating them into distinct phases adds coordination overhead without benefit.
**Delivers:** `overlay/service.go` (ServiceSwitcherOverlay, `s` key, `serviceSelectedMsg`); `serviceSelectedMsg` in messages.go; complete inline color migration for s3panel and ecspanel; verification that profile and region overlay composite calls still work correctly over the new three-zone layout
**Addresses:** Service switcher modal, complete visual consistency across all panels
**Avoids:** Overlay compositing against partial view (Pitfall 6), service switch triggering unnecessary session reset

### Phase Ordering Rationale

- **Theme first** because it is the dependency root — not a preference. Any component built before the theme is defined will need its colors replaced after the fact.
- **Chrome before layout** because header and footer heights must be fixed constants before the body-height calculation in `rootModel` is correct. Building the two-panel layout without known header height produces a bodyHeight that is immediately wrong.
- **Layout before modal** because the service switcher overlay composite call must target the fully restructured `baseView()` output. Implementing the overlay before the layout restructure means the `Composite()` background is the wrong view, requiring a second pass.
- **Color migration bundled with modal** because both are low-risk sweeps that finalize the visual coherence of the milestone. Neither introduces new patterns; both validate the work from previous phases.

### Research Flags

No phases require `/gsd:research-phase`. All four areas have HIGH-confidence sources and working reference implementations either in the existing duchess codebase or in official charmbracelet documentation.

Phases with standard patterns:
- **Phase 1 (Theme):** charmbracelet/crush provides a production-verified Styles struct pattern; Lipgloss v1.1.0 APIs are fully documented on pkg.go.dev
- **Phase 2 (Chrome):** Stateless render functions following the existing `status.go` pattern; `headerHeight` constant derivation from string line count is trivial
- **Phase 3 (Layout):** Lipgloss `JoinHorizontal`/`JoinVertical` are official first-class APIs; the specific border off-by-2 pitfall is fully documented with prevention in Lipgloss issues #449 and #298
- **Phase 4 (Modal + Migration):** Service switcher is the third instance of the profile/region overlay pattern; `overlay/profile.go` is a complete implementation reference

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | go.mod pins verified; Lipgloss and Bubble Tea APIs confirmed against pkg.go.dev; no new dependencies needed; lipgloss v2 incompatibility confirmed |
| Features | HIGH | Direct code reading of existing duchess codebase; k9s patterns sourced from official k9scli.io documentation; feature set is tightly scoped |
| Architecture | HIGH | Derived from reading existing overlay/profile.go, model.go, status.go; build order maps directly to confirmed dependency graph |
| Pitfalls | HIGH | Lipgloss issues #449 and #298 directly document the border-width and frame-size bugs; Bubble Tea WindowSizeMsg delivery behavior confirmed from official docs |

**Overall confidence:** HIGH

### Gaps to Address

- **AWS Dark color exact values:** `#232F3E` (Squid Ink) is community consensus, not official AWS design system documentation. ANSI256 fallback values for non-accent tokens are nearest-neighbor estimates. Verify colors visually on first render against a real 256-color terminal and tune before shipping.
- **Logo ASCII art finalization:** The logo constant shown in STACK.md is an example shape. The actual "duchess" ASCII art must be authored using a FIGlet tool (patorjk.com, Standard or Slant font) and verified visually at the intended header height (4-6 rows) before committing.
- **Left nav fixed width:** 20 columns is recommended based on inference from k9s and lazygit patterns (MEDIUM confidence). Verify visually at first render — it may need adjustment to 18 or 22 columns depending on how service names render with borders and padding.

## Sources

### Primary (HIGH confidence)
- `github.com/charmbracelet/lipgloss` v1.1.0 — `JoinHorizontal`, `JoinVertical`, `Style.Inherit()`, `RoundedBorder()`, `CompleteColor`, border width behavior
- lipgloss GitHub issue #449 — `.Width()` does not include border size
- lipgloss GitHub issue #298 — `GetHorizontalFrameSize()` inconsistency with padding vs. border
- `github.com/rmhubbert/bubbletea-overlay` v0.6.5 — `btoverlay.Composite()` signature; confirmed live in duchess model.go
- Bubble Tea v1.x docs — `WindowSizeMsg` delivered during all program states
- k9s skins documentation (k9scli.io) — theme struct pattern, header layout conventions, header row budget
- duchess codebase (direct reading) — `overlay/profile.go`, `overlay/region.go`, `model.go`, `status.go`
- AWS brand documentation (brandpalettes.com) — `#FF9900` AWS Orange, `#252F3E` AWS Navy

### Secondary (MEDIUM confidence)
- charmbracelet/crush project (DeepWiki analysis) — `Styles` struct pattern with `DefaultStyles()` constructor and `Common` carrier
- AWS Squid Ink `#232F3E` — community consensus from AWS dark theme projects; not official AWS design system docs
- ANSI256 fallback values for non-accent palette tokens — nearest-neighbor estimates from xterm 256-color chart; verify with `CompleteColor`
- Fixed left nav width of 20 columns — inference from k9s and lazygit visual patterns; not explicitly documented
- `moul.io/banner` as a dynamic ASCII art fallback option — confirmed on pkg.go.dev but not needed for this milestone

### Tertiary (LOW confidence)
- `common-nighthawk/go-figure` maintenance status — inferred from Debian package date (`0.0~git20210622`); not verified directly from GitHub

---
*Research completed: 2026-03-10*
*Ready for roadmap: yes*
