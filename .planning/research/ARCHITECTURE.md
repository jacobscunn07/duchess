# Architecture Research

**Domain:** Go TUI visual polish — duchess v1.1 layout restructure
**Researched:** 2026-03-10
**Confidence:** HIGH (derived from reading existing codebase + confirmed Bubble Tea v1.x + Lipgloss v1.1.0 APIs)

## Standard Architecture

### System Overview (v1.1 target)

```
┌─────────────────────────────────────────────────────────────┐
│  header.go — ASCII logo + version/profile/region/refresh     │
├──────────────────────────┬──────────────────────────────────┤
│  Left Nav Panel          │  Right Content Panel             │
│  (30% width)             │  (70% width)                     │
│  ServiceNav placeholder  │  "Amazon S3" / "Amazon ECS"      │
│                          │  header label                    │
│                          │  ┌────────────────────────────┐  │
│                          │  │  s3panel / ecspanel Model  │  │
│                          │  └────────────────────────────┘  │
├──────────────────────────┴──────────────────────────────────┤
│  footer — principal ARN (left)          clock (right)        │
├─────────────────────────────────────────────────────────────┤
│  Overlays (z-order above all): ProfileOverlay, RegionOverlay,│
│  ServiceSwitcherOverlay, HelpOverlay                         │
└─────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | New vs Modified |
|-----------|----------------|-----------------|
| `internal/ui/theme/theme.go` | Centralized DarkTheme — all colors, border styles, text styles | **New** |
| `internal/ui/header.go` | Stateless render function: ASCII logo + metadata row | **New** |
| `rootModel` (model.go) | baseView() restructure: JoinVertical(header, body, footer) | **Modified** |
| `rootModel` (model.go) | contentView(): JoinHorizontal(navPanel, contentPanel) | **Modified** |
| `overlay/service.go` | ServiceSwitcherOverlay — `s` key, serviceSelectedMsg | **New** |
| `overlay/help.go` | HelpOverlay — `?` key, contextual hotkey listing, Esc dismiss | **New** |
| `s3panel`, `ecspanel` | Receive theme styles at construction; no logic changes | **Modified (constructor only)** |

## Recommended Project Structure

```
internal/
├── ui/
│   ├── theme/
│   │   └── theme.go          # DarkTheme var — all Lipgloss styles
│   ├── header.go             # Render(theme, width, meta) string
│   ├── overlay/
│   │   ├── profile.go        # Existing — unchanged
│   │   ├── region.go         # Existing — unchanged
│   │   ├── service.go        # New — ServiceSwitcherOverlay
│   │   └── help.go           # New — HelpOverlay
│   └── status.go             # Existing footer — modify to use theme colors
```

### Structure Rationale

- **theme/:** Isolated package so all components import `theme.DarkTheme` without circular deps
- **overlay/:** All four overlay types live together — consistent pattern, easy to extend
- **header.go:** Stateless render function (not a model) — no state to manage, mirrors status.go pattern

## Architectural Patterns

### Pattern 1: Centralized Theme Object

**What:** A `Theme` struct in `internal/ui/theme/` with all `lipgloss.Style` fields constructed once in `DarkTheme()`. All component constructors accept a `theme.Theme` value and store it.

**When to use:** Whenever more than one component shares colors or border styles.

**Trade-offs:** One source of truth for all visual tokens. Slightly more constructor boilerplate.

**Example:**
```go
// internal/ui/theme/theme.go
type Theme struct {
    Accent        lipgloss.Color
    Background    lipgloss.Color
    Surface       lipgloss.Color
    Text          lipgloss.Color
    MutedText     lipgloss.Color

    BorderActive  lipgloss.Style
    BorderInactive lipgloss.Style
    // ... etc
}

func DarkTheme() Theme {
    return Theme{
        Accent:     lipgloss.Color("#FF9900"),
        Background: lipgloss.Color("#232F3E"),
        Surface:    lipgloss.Color("#2D2D2D"),
        Text:       lipgloss.Color("#FFFFFF"),
        MutedText:  lipgloss.Color("#888888"),
        // styles composed from these tokens
    }
}
```

### Pattern 2: Three-Zone baseView()

**What:** `rootModel.baseView()` uses `lipgloss.JoinVertical` to compose header + body + footer. The `bodyHeight()` helper computes `m.height - headerHeight - footerHeight` to give panels their correct vertical budget.

**When to use:** Any time persistent chrome (header/footer) surrounds scrollable content.

**Trade-offs:** All height math is in one place. Must be updated whenever chrome height changes.

**Example:**
```go
func (m Model) baseView() string {
    header := header.Render(m.theme, m.width, m.meta())
    body   := m.contentView()  // two-panel layout
    footer := m.footerView()
    return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m Model) bodyHeight() int {
    return m.height - header.Height() - footerHeight
}
```

### Pattern 3: ServiceSwitcherOverlay (mirrors ProfileOverlay)

**What:** `overlay/service.go` implements `ServiceSwitcherOverlay` with the exact same three-field pattern (`showing bool`, `selected int`, `items []string`) and the same `btoverlay.Composite()` compositing call already used by profile and region overlays.

**When to use:** Any floating selection modal in the app.

**Trade-offs:** Zero new patterns — third instance of proven code. serviceSelectedMsg is a new unexported typed message struct.

## Data Flow

### Key Message Flows

```
KeyMsg "s"
    → rootModel.Update()
    → m.serviceSwitcher.showing = true

ServiceSwitcherOverlay selection
    → serviceSelectedMsg{service: "s3"}
    → rootModel.Update()
    → m.activeService = "s3"
    → rootModel.View() renders correct right-panel content

KeyMsg "?"
    → rootModel.Update()
    → m.helpOverlay.showing = true
    → helpOverlay.SetContext(m.activeService)

tea.WindowSizeMsg
    → rootModel receives width/height
    → Computes: navWidth = width * 0.30, contentWidth = width * 0.70 - borderCost
    → Passes new dimensions to s3panel, ecspanel via sized message
```

### Height Budget (Critical — 3 sites in model.go)

```
m.height (terminal rows)
  - header.Height()       (header chrome)
  - footerHeight          (footer chrome)
  = bodyHeight            (available for panels)

navWidth  = int(float64(m.width) * 0.30) - 2   // border cost
contentWidth = m.width - navWidth - 2            // border cost
```

## Integration Points

### New vs Existing Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| theme → all components | Value passed at construction | Theme is immutable after init |
| rootModel ↔ ServiceSwitcherOverlay | Embed in rootModel, same as profile/region | Three new fields in rootModel |
| rootModel ↔ HelpOverlay | Embed in rootModel, context set on key press | Needs activeService to show contextual keys |
| rootModel → s3panel/ecspanel | Width passed via WindowSizeMsg-equivalent | Panels must handle new narrower contentWidth |
| header.go → rootModel | Stateless: rootModel passes meta values in | No state in header component |

## Build Order (dependency-aware)

1. **theme/theme.go** — all other components depend on this; no deps of its own
2. **Migrate footer (status.go)** — replace inline colors with theme tokens; establish footerHeight constant
3. **header.go** — stateless render function; depends on theme; establishes headerHeight
4. **overlay/service.go** — ServiceSwitcherOverlay; depends on theme; mirrors profile.go
5. **overlay/help.go** — HelpOverlay; depends on theme; contextual key listing per service
6. **serviceSelectedMsg / helpMsg** — typed message structs in messages.go
7. **rootModel wiring** — add overlay fields, key handlers ("s", "?"), baseView() restructure, bodyHeight() helper, two-panel contentView()
8. **Panel color substitution** — migrate s3panel and ecspanel inline colors to theme tokens
9. **Validation** — WindowSizeMsg propagation, all 3 height-budget sites consistent

## Anti-Patterns

### Anti-Pattern 1: Hardcoding Colors Outside Theme

**What people do:** Use `lipgloss.Color("#FF9900")` inline in panel or header render functions.
**Why it's wrong:** Makes future theme switching impossible; defeats the centralization goal.
**Do this instead:** All colors flow from `theme.Theme` fields passed at construction.

### Anti-Pattern 2: rootModel Owning the Nav Panel Split Logic

**What people do:** Compute 30/70 split inside `contentView()` with magic numbers.
**Why it's wrong:** Split ratio becomes invisible and hard to tune.
**Do this instead:** Define `navPanelRatio = 0.30` as a named constant; compute in `WindowSizeMsg` handler and store as `m.navWidth`, `m.contentWidth`.

### Anti-Pattern 3: Rendering to Measure Height

**What people do:** Call `header.Render(...)` twice — once to measure, once to display.
**Why it's wrong:** Performance regression on every frame; header renders at 60Hz.
**Do this instead:** `header.Height()` returns a constant (hardcoded line count of ASCII art + 1 metadata row). No render needed to know the height.

### Anti-Pattern 4: Height Budget Drift

**What people do:** Update header height in one place but forget to update the other 2 of the 3 sites in model.go.
**Why it's wrong:** Panels overflow or leave blank space; hard to debug.
**Do this instead:** Define `headerHeight` and `footerHeight` as package-level constants; all 3 sites reference the same constants.

### Anti-Pattern 5: Service Switch Triggering Session Reset

**What people do:** Wire service selection to the same `cancelSession` flow used by profile/region switches.
**Why it's wrong:** Service switching is purely a UI concern — S3 and ECS data are already loaded; no credential refresh needed.
**Do this instead:** `serviceSelectedMsg` only updates `m.activeService` — no session cancellation, no re-fetch.

## Sources

- Existing duchess codebase (internal/ui/overlay/profile.go, region.go, model.go, status.go)
- charmbracelet/lipgloss v1.1.0 — JoinHorizontal, JoinVertical, Style APIs
- charmbracelet/bubbles — btoverlay.Composite() (already in go.mod)
- k9s source — header/footer layout conventions

---
*Architecture research for: duchess v1.1 Visual Polish*
*Researched: 2026-03-10*
