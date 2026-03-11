# Feature Research — v1.1 Visual Polish Milestone

**Domain:** TUI visual polish — k9s-inspired AWS TUI layout redesign
**Researched:** 2026-03-10
**Confidence:** MEDIUM-HIGH (web search available; WebFetch blocked; training data supplemented by code review of existing duchess codebase)

---

> **Scope note:** This file covers ONLY the four new feature areas for the v1.1 milestone.
> All v1.0 product features (S3/ECS browsing, profile/region switching, keybindings) are
> implemented and out of scope here. Research categories map to the four milestone targets:
> Header, Layout, Service Switcher, Theme.

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features that k9s-class TUI users assume exist. Missing these makes the redesigned UI feel
unfinished or amateurish compared to the tools that inspired duchess.

| Feature | Why Expected | Complexity | Category | Notes |
|---------|--------------|------------|----------|-------|
| Persistent header with app identity | k9s, lazygit, superfile all show a fixed top bar — users orient by the "you are in X tool" signal | LOW | Header | Rendered once per frame at a fixed height; height must be subtracted from content area |
| ASCII logo in header | k9s's dog logo and text logo are iconic; polished TUIs brand themselves visually | LOW | Header | Multi-line raw string literal in Go; lipgloss.Height() determines row cost; 4-6 rows is conventional |
| Header metadata (profile, region, version) | Status-bar metadata belongs in the header in a two-panel layout — users expect at-a-glance context without hunting | LOW | Header | The v1.0 status bar already has this data; the redesign moves it upward and reformats it alongside the logo |
| Two-panel layout (left nav + right content) | k9s, lazygit, and most professional TUIs use a left-sidebar/right-content split for navigation | MEDIUM | Layout | lipgloss.JoinHorizontal(lipgloss.Top, nav, content); left panel is fixed-width (20-25 chars), right takes remainder |
| Left nav showing current service + available services | Users need a visible, persistent list of what they can navigate to — not a hidden keybinding | LOW | Layout | Static list (S3, ECS) with the active service highlighted; no scrolling needed until there are 8+ services |
| Content panel with service-name subheader | Right panel should title itself with the current service so orientation is instant | LOW | Layout | Single-line header at top of right panel (e.g., "S3 Buckets") rendered as styled text, not ASCII art |
| Rounded borders on panels | k9s uses borders to visually separate regions; rounded corners are modern TUI convention | LOW | Theme | lipgloss.RoundedBorder() — already used in profile/region overlays in v1.0; extend to main panels |
| Modal overlays follow existing pattern | Service switcher should behave exactly like profile and region overlays — same key conventions, same visual style | LOW | Service Switcher | btoverlay.Composite is already wired in model.go; service switcher overlay is the third instance of this pattern |
| Footer revised to principal ARN + clock | v1.0 footer has service indicator which becomes redundant once the left nav shows active service | LOW | Layout | Remove panel indicator [S3]/[ECS] from footer; keep ARN left + clock right |

### Differentiators (Competitive Advantage)

Features that distinguish duchess's visual identity beyond what is merely functional.

| Feature | Value Proposition | Complexity | Category | Notes |
|---------|-------------------|------------|----------|-------|
| AWS Dark color theme (AWS Orange accent) | Immediately communicates "this is an AWS tool" without reading a word — color as brand signal | MEDIUM | Theme | #FF9900 (AWS Orange) as accent; #232F3E (Squid Ink) as surface backgrounds; lipgloss supports truecolor hex directly |
| Centralized DarkTheme object (no inline colors) | Inline hardcoded colors in v1.0 (e.g., lipgloss.Color("33")) are scattered across 5+ files; a theme struct eliminates future color drift and enables future theme switching | MEDIUM | Theme | Define a struct with named fields (Accent, Surface, Border, Text, Muted, etc.); pass to all rendering functions; this is architecturally the biggest change of the milestone |
| Left nav active-service highlight with accent color | The active service should be visually prominent (bold + accent foreground or accent background); passive entries should be clearly subordinate | LOW | Layout | Single style application using DarkTheme.Accent; uses the same selected-item pattern as existing overlay delegates |
| Logo + metadata layout cohesion | Logo on left, metadata tokens on right, vertically centered — mirrors the header layout of k9s's information bar | LOW | Header | lipgloss.JoinHorizontal with logo block and right-aligned metadata block; right block uses lipgloss.PlaceHorizontal or padding to push to the right |
| Service switcher key is `s` | The `s` key is semantically natural for "service"; matches k9s's colon-command pattern of using mnemonic keys for switchers | LOW | Service Switcher | Adds `case "s":` to rootModel.Update key handler; parallel to existing `p` (profile) and `r` (region) handlers |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Animated/dynamic ASCII logo | Looks impressive in demos | Wastes CPU on every render tick, adds complexity, distracts from data | Static ASCII art rendered once; use a spinner only during loading states |
| Per-panel border focus ring (active panel highlights its border) | Conveys which panel has keyboard focus visually | In the two-panel redesign, the left nav is display-only (no keyboard focus) and the right content always has focus — a focus ring implies interactivity that does not exist | Use active-service highlight in left nav instead; border focus rings are only appropriate when both panels accept keyboard input |
| Light theme variant | Appeals to users who run light terminals | Adds conditional rendering and a second full color set; doubles the theming surface area for v1.1 | Ship DarkTheme; design the theme struct to support a second theme later (LightTheme struct field matches) but do not implement it in v1.1 |
| Configurable theme via config file | Power-user request | Requires theme file parsing, validation, runtime color recalculation; significant complexity for a visual-polish milestone | Hardcode DarkTheme for v1.1; design the struct to be instantiatable (not a package-level singleton) to enable config-driven themes in v2 |
| Mouse hover effects on nav items | Visual richness | duchess is intentionally keyboard-only (PROJECT.md: "Mouse navigation — target users are vim-native; keyboard-only is intentional"); mouse interaction is explicitly out of scope | Keyboard highlight with accent color is sufficient |
| Full-width ASCII banner that consumes 8+ rows | Dramatic visual impact | Consumes 30%+ of a typical 24-row terminal; leaves inadequate vertical space for list content; forces truncation of usable content | Keep header to 5-6 rows maximum; a compact logo + inline metadata is the k9s convention |

---

## Feature Dependencies

```
DarkTheme struct
    └──required by──> Header rendering
    └──required by──> Left nav panel styling
    └──required by──> Right content panel styling
    └──required by──> Service switcher overlay styling
    └──required by──> Footer revised styling
    (DarkTheme is the foundation; all visual components depend on it)

Two-panel layout
    └──requires──> Height budget calculation
                       └──requires──> Header height (lipgloss.Height(header))
                       └──requires──> Footer height (lipgloss.Height(footer))
    └──requires──> Left nav width constant (fixed ~20 chars)
    └──right content width = total_width - left_nav_width

Service switcher modal
    └──follows pattern of──> Profile overlay (overlay/profile.go)
    └──follows pattern of──> Region overlay (overlay/region.go)
    └──uses──> btoverlay.Composite (already in model.go View())
    └──dispatches──> serviceSelectedMsg (new typed message, parallel to profileSelectedMsg / regionSelectedMsg)

Revised footer
    └──removes──> panelIndicator field from renderStatusBar()
    └──depends on──> DarkTheme (for consistent color tokens)
    (footer is a simplification, not a new component)

Header
    └──depends on──> DarkTheme (accent color for logo or metadata tokens)
    └──height consumed by──> reduces contentH passed to panels
    (existing: contentH = m.height - lipgloss.Height(statusBar)
     new:      contentH = m.height - lipgloss.Height(header) - lipgloss.Height(footer))
```

### Dependency Notes

- **DarkTheme must be built first:** Every other component in this milestone consumes it. Building
  panels before the theme struct means double-touching every file.

- **Height budget calculation changes are the highest-risk dependency:** Currently `model.go`
  computes `contentH` by subtracting one status bar from terminal height. The new layout subtracts
  both a header and a footer, and the content area is further split left/right. Every place that
  passes `width` and `height` to panels must be audited — there are currently at least 3 sites in
  `model.go` (identityLoadedMsg handler, regionSelectedMsg handler, and baseView()).

- **Service switcher is structurally isolated:** It follows the profile/region pattern exactly.
  The implementation risk is LOW because the pattern is already proven and tested in the codebase.
  The only novel part is defining `serviceSelectedMsg` and the panel-switching logic (which already
  exists via the `tab` key handler — the switcher just provides a UI to trigger it).

- **Two-panel layout does not require new panel models:** The existing `s3panel.Model` and
  `ecspanel.Model` render into whatever width/height they are given. The layout change is entirely
  in `rootModel.baseView()` — panels are not aware of the two-panel structure.

---

## MVP Definition

### This Milestone (v1.1 — all four areas are in scope together)

This is a visual polish milestone, not a phased feature build. All four areas should ship together
because they form a coherent redesign — shipping the theme without the layout, or the layout without
the theme, produces an inconsistent result.

- [ ] **DarkTheme struct** — centralized color token object; no inline colors anywhere after
  (this is the prerequisite for everything else)
- [ ] **Header with ASCII logo** — 4-6 rows, logo left, metadata right, themed with DarkTheme
- [ ] **Two-panel layout** — left nav (fixed ~20 cols) + right content (remainder); left nav
  shows service list with active item highlighted; right content renders existing panel views
- [ ] **Revised footer** — principal ARN left, clock right; panel indicator removed
- [ ] **Service switcher modal** — `s` key, lists available services, confirms with Enter,
  follows profile/region overlay pattern exactly

### Add After Validation (v1.2+)

- [ ] Additional AWS services (EC2, Lambda, CloudWatch) in left nav — only valuable once the
  service switcher and two-panel structure are proven with 2 services
- [ ] Light theme variant — only if user feedback indicates light-terminal users are a meaningful
  audience segment

### Future Consideration (v2+)

- [ ] Config-driven theme selection — requires theme file parsing infrastructure
- [ ] Animated logo or theme variants for different AWS environments (prod vs dev warning) — the
  k9s custom logo issue (#2090) shows this is requested but complex

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| DarkTheme struct (centralized colors) | HIGH | LOW | P1 |
| Header with ASCII logo | HIGH | LOW | P1 |
| Two-panel layout | HIGH | MEDIUM | P1 |
| Revised footer | MEDIUM | LOW | P1 |
| Service switcher modal | HIGH | LOW | P1 |
| Rounded borders (all panels) | MEDIUM | LOW | P1 |
| Left nav active-service highlight | HIGH | LOW | P1 |
| Right content service-name subheader | MEDIUM | LOW | P1 |

**Priority key:** All items are P1 — this is a focused milestone with a defined feature set.
No P2/P3 items exist because anything not in this list is explicitly deferred.

---

## Polished TUI Patterns Observed (Research Findings)

These findings from k9s, lazygit, and the Charmbracelet ecosystem inform the decisions above.

### Header patterns

**k9s:** Persistent header showing the ASCII dog/text logo plus cluster context info (cluster name,
API server URL). Header is approximately 5-6 rows. The `logoless` config option exists precisely
because some users need the rows back — indicating the logo is standard but its row cost is known.
The logo is a branding anchor that makes the tool instantly recognizable in screenshots.
Confidence: HIGH (widely documented, feature request #2090 directly references the logo system).

**Implication for duchess:** A 4-6 row header with the "duchess" ASCII logo plus profile/region/
version metadata is the correct pattern. Taller headers are extractable by config later.

### Two-panel layout

**lipgloss:** `JoinHorizontal(lipgloss.Top, leftBlock, rightBlock)` is the idiomatic approach.
The left block is rendered to a fixed column width; the right block gets the remainder. Both are
styled independently. `lipgloss.Width()` and `lipgloss.Height()` can measure rendered blocks to
verify dimensions. The layout function is well-tested and performant.
Confidence: HIGH (official lipgloss API, verified in pkg.go.dev documentation).

**Fixed-width left nav:** The service list (S3, ECS — eventually more) contains short names.
A fixed nav width of 20 columns is generous (the longest current name "ECS" is 3 chars; with
padding and borders, 18-22 columns is appropriate). Proportional widths create a jumpy layout
when terminal is resized; fixed-width nav is the right call for text-label navigation.
Confidence: MEDIUM (inference from k9s and lazygit patterns; lazygit uses fixed-width side panels).

### Service switcher modal

The existing codebase has two modal overlays (profile, region) using `btoverlay.Composite` and the
`rmhubbert/bubbletea-overlay` library. Both follow the same pattern:

1. A bool flag on rootModel controls visibility (`isProfileOverlayOpen`, `isRegionOverlayOpen`)
2. Key handler in rootModel.Update() opens the overlay and intercepts keys while open
3. On Enter, a typed message is dispatched (`profileSelectedMsg`, `regionSelectedMsg`)
4. rootModel.View() composites overlay over base view using btoverlay.Composite

The service switcher is structurally identical. It does not need the external overlay library for
something as simple as a list of 2-5 services — but using the same library maintains visual
consistency (same centering, same blur-background behavior).
Confidence: HIGH (direct code reading of overlay/profile.go and overlay/region.go).

### Color theme

**k9s theming:** k9s uses a YAML skin file with named sections: `frame` (border, title, crumb,
status), `views` (table header, selected row, error row), `status` (new, modified, add, error,
highlight, kill, completed colors). The theme is a flat config struct, not a polymorphic interface.
This is the right model for a v1.1 theme — a simple named-field struct, not an interface.
Confidence: HIGH (k9s skins documentation on k9scli.io, confirmed by multiple sources).

**Lipgloss color support:** Lipgloss v1 supports hex colors directly (`lipgloss.Color("#FF9900")`).
The library auto-detects terminal color capabilities (TrueColor, ANSI256, ANSI) via termenv and
degrades automatically. For maximum correctness, `lipgloss.CompleteColor` can specify all three
profiles explicitly. Since duchess targets engineers with modern terminals (iTerm2, Alacritty,
WezTerm, macOS Terminal 2019+), relying on auto-detection with hex colors is acceptable.
The `NO_COLOR` guard in v1.0 `NewRootModel()` continues to apply.
Confidence: HIGH (lipgloss pkg.go.dev documentation, confirmed by charmbracelet/mods DeepWiki).

**AWS Dark palette mapping:**

| Token | Hex | Usage | ANSI256 fallback |
|-------|-----|-------|-----------------|
| Accent | #FF9900 | Logo, active nav item, focus borders | 214 (Orange1, close match) |
| Surface | #232F3E | Panel backgrounds (Squid Ink) | 235 (Grey27, close match) |
| SurfaceAlt | #1A2332 | Header/footer backgrounds (darker) | 234 (Grey11) |
| Text | #FFFFFF | Primary text | 15 (White) |
| TextMuted | #8C9BAB | Secondary text, separators | 103 (LightSlateGrey) |
| Border | #3A4B5C | Panel borders (lighter than surface) | 237 (Grey23) |
| Selected | #FF9900 | Selected item foreground or background | 214 |
| Error | #FF4444 | Error states | 196 (Red1) |

Confidence: MEDIUM (hex values from AWS brand documentation and community dark theme implementations;
ANSI256 fallbacks are nearest-neighbor estimates from xterm 256-color chart — verify with CompleteColor).

---

## Existing Code Integration Points

These are the specific files that will need changes, identified from code reading:

| File | Change Type | Change Summary |
|------|-------------|----------------|
| `internal/ui/status.go` | REFACTOR | Replace inline `lipgloss.Color("236")` etc. with DarkTheme tokens; remove panel indicator |
| `internal/ui/model.go` | EXTEND + REFACTOR | Add `isServiceOverlayOpen`, `serviceOverlay` fields; add `s` key handler; add `serviceSelectedMsg`; rewrite `baseView()` for two-panel layout; change height budget to subtract header + footer |
| `internal/ui/overlay/profile.go` | MINOR | Update border color to use DarkTheme.Border instead of hardcoded `"62"` |
| `internal/ui/overlay/region.go` | MINOR | Same as profile.go |
| `internal/ui/overlay/service.go` | NEW | ServiceOverlay struct following profile.go pattern exactly |
| `internal/ui/theme.go` | NEW | DarkTheme struct definition and singleton; all color tokens |
| `internal/ui/header.go` | NEW | renderHeader(m rootModel) string — ASCII logo + metadata |

The `s3/model.go` and `ecs/model.go` panel files do not need changes — they render into
whatever width/height they receive. The layout change is entirely in rootModel.

---

## Sources

- Codebase: `internal/ui/model.go`, `internal/ui/status.go`, `internal/ui/overlay/profile.go`,
  `internal/ui/overlay/region.go` — direct code reading, HIGH confidence
- k9s skins documentation: https://k9scli.io/topics/skins/ — HIGH confidence
- k9s custom logo issue: https://github.com/derailed/k9s/issues/2090 — MEDIUM confidence
- lipgloss JoinHorizontal: https://pkg.go.dev/github.com/charmbracelet/lipgloss — HIGH confidence
- lipgloss color support (CompleteColor, auto-detection): pkg.go.dev + charmbracelet/mods DeepWiki — HIGH confidence
- AWS Squid Ink color (#232F3E): https://colorswall.com/color/232f3e — MEDIUM confidence
- AWS Orange (#FF9900): common knowledge from AWS branding, community dark theme implementations — HIGH confidence
- btoverlay.Composite: direct code reading of existing duchess `model.go` import — HIGH confidence

---

*Feature research for: duchess v1.1 Visual Polish — Header, Layout, Service Switcher, Theme*
*Researched: 2026-03-10*
