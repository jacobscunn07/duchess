# Pitfalls Research

**Domain:** Go TUI visual polish — duchess v1.1 layout restructure (Bubble Tea v1.x + Lipgloss)
**Researched:** 2026-03-10
**Confidence:** HIGH (confirmed via lipgloss GitHub issues, Bubble Tea docs, codebase reading)

## Critical Pitfalls

### Pitfall 1: Lipgloss `.Width()` Does Not Include Border Size

**What goes wrong:**
A panel styled with `.Width(n).Border(lipgloss.RoundedBorder())` renders at `n + 2` columns wide (border adds 1 column each side). Two side-by-side panels overflow the terminal width, causing wrapping or content truncation.

**Why it happens:**
Lipgloss `.Width()` sets the content area width, not the outer width. Border cost is additive. `GetHorizontalFrameSize()` is also unreliable (documented inconsistency between border and padding accounting in v1.x).

**How to avoid:**
Manually subtract `borderCost := 2` from each panel's content width before setting `.Width()`. Define `borderCost` as a named constant. Never rely on `GetHorizontalFrameSize()`.

**Warning signs:**
Right panel wraps or is clipped; total visual width exceeds terminal width; layout looks correct in narrow terminals but breaks wide.

**Phase to address:** Theme + layout phase (whichever builds the two-panel contentView).

---

### Pitfall 2: `WindowSizeMsg` Not Forwarded During Loading State

**What goes wrong:**
If `rootModel` guards `WindowSizeMsg` forwarding with a state check (e.g., only forwarding when `state == stateReady`), panels receive stale dimensions after a terminal resize during loading. The layout is permanently wrong until the next resize.

**Why it happens:**
Early implementations add `if m.state != stateLoading { forwardMsg() }` thinking panels aren't ready. But Bubble Tea still delivers `WindowSizeMsg` during all states.

**How to avoid:**
Always forward `WindowSizeMsg` to panels unconditionally. Store `m.width` and `m.height` on every `WindowSizeMsg` regardless of state.

**Warning signs:**
Layout breaks after resizing during a profile switch; panels render at wrong dimensions after reconnect.

**Phase to address:** Layout wiring phase (rootModel contentView restructure).

---

### Pitfall 3: Hardcoding Header Height Instead of Deriving It

**What goes wrong:**
`bodyHeight = m.height - 5 - 1` (5 for header, 1 for footer). When the ASCII art is later adjusted or the metadata row wraps, the hardcoded 5 is wrong. Content overflows or leaves blank rows.

**Why it happens:**
Developer counts lines of ASCII art manually and embeds the number. Works until the art changes.

**How to avoid:**
Define `headerHeight` as a constant derived from the actual ASCII art string (`strings.Count(logoStr, "\n") + 1` for the metadata row). Reference this constant in all 3 height-budget sites in model.go.

**Warning signs:**
Blank gap between footer and content; content clips below footer; off-by-one that only appears at certain terminal heights.

**Phase to address:** Header implementation phase.

---

### Pitfall 4: Lipgloss Style `.Inherit()` Silent Divergence

**What goes wrong:**
`childStyle.Inherit(parentStyle)` only copies *unset* rules. If `childStyle` already has a background set (e.g., from a previous inline call), the parent background is silently ignored. The child renders with a different background than expected.

**Why it happens:**
Developers assume `Inherit()` is like CSS inheritance — it overrides. It actually only fills gaps.

**How to avoid:**
Construct all styles fresh from `theme.Theme` token fields (colors, border). Do not chain `.Inherit()` as a way to apply theme. Instead, build each style explicitly: `lipgloss.NewStyle().Background(t.Surface).Foreground(t.Text)...`.

**Warning signs:**
Component looks right in isolation but has wrong background when embedded; theme token change doesn't propagate to some components.

**Phase to address:** Theme implementation phase — establish the pattern before any component uses it.

---

### Pitfall 5: AWS Orange Degrades in 256-Color Terminals

**What goes wrong:**
`lipgloss.Color("#FF9900")` renders as a muddy brown-yellow in 256-color terminals (the nearest ANSI256 neighbor is color 214, which is close but noticeably different). On a user's SSH terminal or tmux session without truecolor, the accent looks wrong.

**Why it happens:**
Lipgloss passes hex colors through unchanged to terminals. Terminals without truecolor support (COLORTERM=truecolor) do their own nearest-neighbor mapping, which varies by terminal emulator.

**How to avoid:**
Use `lipgloss.CompleteColor{TrueColor: "#FF9900", ANSI256: "214", ANSI: "3"}` for the accent color. This provides explicit fallback values rather than relying on terminal mapping. Apply to all theme accent uses.

**Warning signs:**
Colors look washed out or wrong on SSH sessions; tmux users report different appearance.

**Phase to address:** Theme implementation phase.

---

### Pitfall 6: Modal Overlay Background Composite Fails After Layout Changes

**What goes wrong:**
`btoverlay.Composite()` composites the modal over the full terminal view string. After the header + two-panel layout change, the background string passed to `Composite()` must be the full `baseView()` output — not just the content area. If the overlay code still composites over just `contentView()`, the header and footer disappear behind the overlay.

**Why it happens:**
Existing profile/region overlay code was written against a simpler layout. The composite call may reference a partial view string.

**How to avoid:**
Ensure all overlay `Composite()` calls receive `m.baseView()` (the full terminal view including header + footer) as the background. New overlays (service switcher, help) must follow this same pattern from the start.

**Warning signs:**
Header disappears when profile switcher is open; overlay is positioned relative to content area instead of full terminal.

**Phase to address:** Service switcher and help overlay phases — and verify existing profile/region overlays still work after layout restructure.

---

### Pitfall 7: ASCII Art Width Miscalculation

**What goes wrong:**
`lipgloss.Width(logoStr)` returns incorrect width if the ASCII art string contains escape sequences or if line lengths vary. The header metadata row misaligns with the logo.

**Why it happens:**
`strings.SplitAfter` and `len()` count bytes, not visible rune widths. Lipgloss `Width()` is correct for plain ASCII but can diverge if the logo string ever gains non-ASCII characters.

**How to avoid:**
Use only plain ASCII characters in the logo constant. Measure logo width as the length of its widest line: `maxLineWidth(logoStr)` using `lipgloss.Width(line)` per line. Store as `logoWidth` constant; use for header layout math.

**Warning signs:**
Metadata row starts at wrong column; header looks misaligned at certain terminal widths.

**Phase to address:** Header implementation phase.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Inline colors in one component | Faster first render | Blocks theme switching; diverges from DarkTheme | Never — theme is the foundation |
| Hardcoded panel split ratio (0.30) inline | One less constant | Hard to tune; invisible to future devs | OK if documented as named const in same file |
| Skip CompleteColor fallback for non-accent colors | Simpler code | Background/surface colors look wrong in 256-color | Acceptable for v1.1 if documented as known limitation |
| ASCII logo as hardcoded string vs. generated | Zero deps | Logo changes require manual update | Acceptable — logo is stable |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Theme → panels | Pass theme through message bus | Pass as constructor value — theme is static after init |
| Header height → bodyHeight | Recalculate in each WindowSizeMsg handler | Define as package constant; reference everywhere |
| Service switcher → session state | Call cancelSession on service switch | Service switch is UI-only — no session reset needed |
| Help overlay → active service | Always show same key list | Pass activeService to HelpOverlay so it shows contextual keys |
| Rounded borders → width math | Forget border cost in width calc | Subtract `borderCost = 2` per bordered panel, always |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Rendering header to measure its height | Slight CPU spike every frame (60Hz render loop) | Use constant for header height | Immediately — avoids on first implementation |
| Re-creating `lipgloss.Style` objects every render | GC pressure at 60Hz | Construct styles once in `DarkTheme()`, reuse | High-frequency render loops |
| Recomputing panel widths every View() call | Negligible but noisy | Compute on WindowSizeMsg, store in model | Not a real problem for TUI scale |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Help modal shows all global keys, not context-specific keys | Overwhelming; irrelevant entries | Show only keys valid for the current panel/state |
| Service switcher doesn't show currently active service | User loses orientation | Highlight/mark current service in the modal list |
| `s` key conflicts with search `/` shortcut | Accidental service switch during search | Only handle `s` when not in search mode |
| Help overlay blocks content for tall key lists | User can't see what they're navigating | Constrain help modal height; add scroll if needed |

## "Looks Done But Isn't" Checklist

- [ ] **Two-panel layout:** Verify panels fill full bodyHeight — check at narrow (80 col) and wide (220 col) terminals
- [ ] **Theme:** Verify all inline color strings from v1.0 replaced — grep for `lipgloss.Color` outside `theme/theme.go`
- [ ] **Header:** Verify metadata (version, profile, region, refresh) matches footer metadata removed — no info lost
- [ ] **Footer:** Verify principal ARN doesn't truncate on long ARNs (>80 chars) — test with real role assumption ARN
- [ ] **Service switcher:** Verify dismissing with Esc doesn't accidentally trigger any nav key
- [ ] **Help overlay:** Verify keys shown are accurate per-panel — S3 panel keys ≠ ECS panel keys
- [ ] **Existing overlays:** Verify profile and region switcher still composite correctly over new layout
- [ ] **Resize:** Resize terminal while overlay is open — layout should recover cleanly
- [ ] **256-color:** Test in Terminal.app (non-truecolor) — AWS Orange accent should still be recognizable

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Border width off-by-2 discovered late | LOW | Add `borderCost` constant; subtract in both panel width calculations |
| Theme colors hardcoded in many files | MEDIUM | grep for `lipgloss.Color` outside theme package; replace with theme field references |
| Header height hardcoded in 3+ places | LOW | Extract to constant; find all uses with grep |
| Overlay compositing broken after layout | MEDIUM | Trace `Composite()` call site; ensure it receives full `baseView()` output |
| WindowSizeMsg not forwarded | LOW | Remove the state guard; always forward |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Border width off-by-2 | Two-panel layout phase | Visual check: no wrapping at 80-col and 220-col terminals |
| WindowSizeMsg forwarding | Layout wiring phase | Resize terminal during S3 browse — no layout break |
| Header height hardcoding | Header implementation phase | Change ASCII art line count — bodyHeight adjusts automatically |
| Style inheritance divergence | Theme phase | All non-theme `lipgloss.Color` uses eliminated (grep check) |
| AWS Orange 256-color degradation | Theme phase | Test in Terminal.app; verify CompleteColor used for accent |
| Overlay composite background | Service switcher + help phases | Open each overlay — header and footer remain visible |
| ASCII art width miscalculation | Header implementation phase | Metadata row visually aligns with logo at varying terminal widths |

## Sources

- lipgloss GitHub issue #449 — Style's Width doesn't include border size
- lipgloss GitHub issue #298 — GetHorizontalFrameSize includes padding but not border
- btoverlay v0.6.5 — Composite() API (already in duchess go.mod)
- Bubble Tea docs — WindowSizeMsg delivery during all states
- lipgloss CompleteColor API — explicit fallback for 256-color terminals

---
*Pitfalls research for: duchess v1.1 Visual Polish (Go TUI layout restructure)*
*Researched: 2026-03-10*
