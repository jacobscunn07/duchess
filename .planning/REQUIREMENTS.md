# Requirements: duchess

**Defined:** 2026-03-10
**Core Value:** Navigate your AWS resources across accounts and regions in one terminal session — no console switching, no `--profile` flags, no context thrash.

## v1.1 Requirements

Requirements for v1.1 Visual Polish milestone. All changes are purely presentational and structural — no business logic or data-fetching changes.

### Layout

- [ ] **LAYOUT-01**: App displays a persistent header at the top of every screen
- [ ] **LAYOUT-02**: Header shows ASCII art "duchess" logo on the left
- [ ] **LAYOUT-03**: Header shows app version, AWS profile, region, and refresh interval to the right of the logo
- [ ] **LAYOUT-04**: Main content area is split into a left navigation panel and a right content panel
- [ ] **LAYOUT-05**: Right content panel displays the active service name as a section header (e.g., "Amazon S3", "Amazon ECS")
- [ ] **LAYOUT-06**: Footer shows the current IAM principal ARN on the left
- [ ] **LAYOUT-07**: Footer shows the clock on the right
- [ ] **LAYOUT-08**: Footer no longer shows a service indicator

### Theme

- [ ] **THEME-01**: All panel borders and modal dialogs use rounded corners
- [x] **THEME-02**: App uses AWS Dark color theme (AWS Orange `#FF9900` accent, Squid Ink `#232F3E` background, `#2D2D2D` surfaces)
- [x] **THEME-03**: Selected items, active borders, and focus states use AWS Orange as accent color
- [x] **THEME-04**: All colors and styles flow from a centralized `DarkTheme` object — no inline hardcoded colors in components

### Navigation

- [ ] **NAV-01**: App displays a left navigation panel alongside the content panel (placeholder structure for v1.1; ready to populate with service-specific items in a future milestone)
- [ ] **NAV-02**: User can open a service switcher modal with the `s` key
- [ ] **NAV-03**: Service switcher modal follows the same pattern as existing region and profile switcher dialogs
- [ ] **NAV-04**: User can dismiss the service switcher with Esc or by selecting a service

### Help

- [ ] **HELP-01**: User can open a help overlay with `?` key showing available hotkeys for the current screen
- [ ] **HELP-02**: Help overlay shows contextual hotkeys relevant to the active panel (S3 keys vs ECS keys)
- [ ] **HELP-03**: User can dismiss the help overlay with Esc

## v1.2 Requirements

Deferred to future release.

### Navigation

- **NAV-05**: Left navigation panel shows populated service-specific navigation items (currently placeholder in v1.1)

### Theme

- **THEME-05**: App supports a configurable theme (e.g., LightTheme) switchable at runtime or via config

## Out of Scope

| Feature | Reason |
|---------|--------|
| Any AWS API / data-fetching changes | v1.1 is purely presentational — no logic changes |
| Mouse hover effects | Project is intentionally keyboard-only per PROJECT.md |
| Per-panel focus ring borders | Left nav is display-only; a focus ring implies interactivity that does not exist |
| Full-width ASCII banner (8+ rows) | Wastes 30%+ of vertical space in a 24-row terminal |
| Config-driven theme selection | Requires theme file parsing infrastructure — defer to v2 |
| Additional AWS services (EC2, Lambda, CloudWatch) | Only valuable after two-panel structure is proven; v1.2+ |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| THEME-02 | Phase 6 | Complete |
| THEME-03 | Phase 6 | Complete |
| THEME-04 | Phase 6 | Complete |
| LAYOUT-01 | Phase 7 | Pending |
| LAYOUT-02 | Phase 7 | Pending |
| LAYOUT-03 | Phase 7 | Pending |
| LAYOUT-06 | Phase 8 | Pending |
| LAYOUT-07 | Phase 8 | Pending |
| LAYOUT-08 | Phase 8 | Pending |
| LAYOUT-04 | Phase 9 | Pending |
| LAYOUT-05 | Phase 9 | Pending |
| NAV-01 | Phase 9 | Pending |
| THEME-01 | Phase 9 | Pending |
| NAV-02 | Phase 10 | Pending |
| NAV-03 | Phase 10 | Pending |
| NAV-04 | Phase 10 | Pending |
| HELP-01 | Phase 11 | Pending |
| HELP-02 | Phase 11 | Pending |
| HELP-03 | Phase 11 | Pending |

**Coverage:**
- v1.1 requirements: 19 total
- Mapped to phases: 19
- Unmapped: 0 ✓

---
*Requirements defined: 2026-03-10*
*Last updated: 2026-03-10 — traceability updated after roadmap revision (6 phases: 6–11)*
