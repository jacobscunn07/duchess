# Roadmap: duchess

## Milestones

- ✅ **v1.0 MVP** — Phases 1–5 (shipped 2026-03-10)
- 🚧 **v1.1 Visual Polish** — Phases 6–11 (in progress)

## Phases

<details>
<summary>✅ v1.0 MVP (Phases 1–5) — SHIPPED 2026-03-10</summary>

- [x] Phase 1: Foundation (2/2 plans) — completed 2026-03-03
- [x] Phase 2: UI Shell (2/2 plans) — completed 2026-03-03
- [x] Phase 3: S3 Browsing (3/3 plans) — completed 2026-03-04
- [x] Phase 4: ECS Browsing (2/2 plans) — completed 2026-03-05
- [x] Phase 4.1: Wire Configurable Refresh Interval — INSERTED (2/2 plans) — completed 2026-03-06
- [x] Phase 5: Profile and Region Switching (3/3 plans) — completed 2026-03-07

Full details: `.planning/milestones/v1.0-ROADMAP.md`

</details>

### 🚧 v1.1 Visual Polish (In Progress)

**Milestone Goal:** Transform the UI into a polished, themed AWS TUI — centralized AWS Dark color theme, persistent header with ASCII logo, two-panel layout, and a service switcher modal — without touching business logic.

- [x] **Phase 6: DarkTheme Package** — Build the centralized theme struct with all AWS Dark color tokens and border styles in an isolated package (completed 2026-03-11)
- [ ] **Phase 7: Header** — Implement the persistent ASCII art "duchess" logo header with app metadata (version, profile, region, refresh interval)
- [ ] **Phase 8: Footer Migration** — Revise the footer — principal ARN on left, clock on right, service indicator removed; migrate to DarkTheme tokens
- [ ] **Phase 9: Two-Panel Layout** — Restructure rootModel into three vertical zones (header/body/footer) with a two-panel horizontal body; add rounded borders throughout
- [ ] **Phase 10: Service Switcher Modal** — Wire the `s` key service switcher overlay following the existing profile/region modal pattern
- [ ] **Phase 11: Help Overlay** — Wire the `?` key contextual help overlay showing hotkeys for the active panel

## Phase Details

### Phase 6: DarkTheme Package
**Goal**: A single isolated package exports `DarkTheme()` — every color token and border style in the app flows from this one object
**Depends on**: Phase 5 (v1.0 complete)
**Requirements**: THEME-02, THEME-03, THEME-04
**Success Criteria** (what must be TRUE):
  1. A `DarkTheme()` constructor exists in `internal/ui/theme/theme.go` and can be imported by any component without circular dependency
  2. Every color in the AWS Dark palette (Squid Ink background, Surface, Border, AWS Orange accent, TextPrimary, TextSecondary) is defined as a typed field on the theme struct — no hex values anywhere else in the codebase
  3. AWS Orange accent uses `CompleteColor` with explicit TrueColor (`#FF9900`), ANSI256 (`214`), and ANSI (`3`) fallbacks so it renders correctly in tmux and SSH sessions without truecolor
  4. Selected items and active focus states reference the accent color field from the theme struct — not hardcoded inline
**Plans**: 2 plans

Plans:
- [x] 06-01-PLAN.md — Create Theme struct + DarkTheme() constructor + DefaultTheme var; verify clean build
- [ ] 06-02-PLAN.md — Gap closure: migrate all inline lipgloss.Color() strings in existing components to theme.DefaultTheme fields (THEME-04)

### Phase 7: Header
**Goal**: Users see a persistent ASCII art "duchess" logo header at the top of every screen, alongside app version, AWS profile, region, and refresh interval
**Depends on**: Phase 6
**Requirements**: LAYOUT-01, LAYOUT-02, LAYOUT-03
**Success Criteria** (what must be TRUE):
  1. Every screen shows a header row at the top with the ASCII "duchess" logo rendered on the left
  2. App version, current AWS profile, current region, and refresh interval are displayed to the right of the logo on every screen
  3. The header height is a package-level constant derived from the actual ASCII art line count — not a hardcoded integer — so the layout math updates automatically if the logo changes
  4. Header styling uses DarkTheme tokens — no inline `lipgloss.Color` strings remain in header.go
**Plans**: TBD

### Phase 8: Footer Migration
**Goal**: The footer shows the IAM principal ARN on the left and the clock on the right, with the service indicator removed and all inline colors replaced by DarkTheme tokens
**Depends on**: Phase 6
**Requirements**: LAYOUT-06, LAYOUT-07, LAYOUT-08
**Success Criteria** (what must be TRUE):
  1. The footer displays the current IAM principal ARN on the left side
  2. The footer displays the live clock on the right side
  3. The footer does not display a service indicator anywhere
  4. All inline `lipgloss.Color` strings in status.go are replaced with DarkTheme token references
**Plans**: TBD

### Phase 9: Two-Panel Layout
**Goal**: The app renders as three vertical zones (header / body / footer) with a two-panel horizontal body split — left nav panel and right content panel — using rounded borders throughout
**Depends on**: Phase 7, Phase 8
**Requirements**: LAYOUT-04, LAYOUT-05, NAV-01, THEME-01
**Success Criteria** (what must be TRUE):
  1. The app renders as three vertical zones: header, body, footer — with body height derived from `windowHeight - headerHeight - footerHeight` so no content overflows
  2. The body is split horizontally into a fixed-width left nav panel and a right content panel that takes the remaining width
  3. The right content panel displays the active service name (e.g., "Amazon S3", "Amazon ECS") as a visible section header above the panel content
  4. The left nav panel renders without errors and serves as a structural placeholder ready to receive service-specific items in a future milestone
  5. All panel borders and modal dialogs use rounded corners; border width cost is accounted for in all panel width calculations so no content overflows the terminal
**Plans**: TBD

### Phase 10: Service Switcher Modal
**Goal**: Pressing `s` opens a service switcher overlay that follows the identical pattern as the existing profile and region switcher dialogs
**Depends on**: Phase 9
**Requirements**: NAV-02, NAV-03, NAV-04
**Success Criteria** (what must be TRUE):
  1. Pressing `s` opens a service switcher modal overlaid on the full three-zone view (header and footer remain visible behind the modal)
  2. The service switcher modal is visually identical in structure and style to the existing profile and region switcher dialogs
  3. Pressing Esc dismisses the modal without changing the active service
  4. Selecting a service from the modal dismisses it and switches the active service
**Plans**: TBD

### Phase 11: Help Overlay
**Goal**: Pressing `?` opens a contextual help overlay listing the hotkeys available for the currently active panel
**Depends on**: Phase 9
**Requirements**: HELP-01, HELP-02, HELP-03
**Success Criteria** (what must be TRUE):
  1. Pressing `?` opens a help overlay showing available hotkeys for the current screen
  2. The hotkeys shown are contextual to the active panel — S3 navigation keys appear when browsing S3; ECS navigation keys appear when browsing ECS
  3. Pressing Esc dismisses the help overlay and returns to the previous state
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 6 → 7 → 8 → 9 → 10 → 11
Note: Phase 8 depends on Phase 6 only (parallel with Phase 7); Phase 9 depends on both Phase 7 and Phase 8.

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Foundation | v1.0 | 2/2 | Complete | 2026-03-03 |
| 2. UI Shell | v1.0 | 2/2 | Complete | 2026-03-03 |
| 3. S3 Browsing | v1.0 | 3/3 | Complete | 2026-03-04 |
| 4. ECS Browsing | v1.0 | 2/2 | Complete | 2026-03-05 |
| 4.1. Wire Configurable Refresh Interval | v1.0 | 2/2 | Complete | 2026-03-06 |
| 5. Profile and Region Switching | v1.0 | 3/3 | Complete | 2026-03-07 |
| 6. DarkTheme Package | 2/2 | Complete   | 2026-03-11 | - |
| 7. Header | v1.1 | 0/? | Not started | - |
| 8. Footer Migration | v1.1 | 0/? | Not started | - |
| 9. Two-Panel Layout | v1.1 | 0/? | Not started | - |
| 10. Service Switcher Modal | v1.1 | 0/? | Not started | - |
| 11. Help Overlay | v1.1 | 0/? | Not started | - |
