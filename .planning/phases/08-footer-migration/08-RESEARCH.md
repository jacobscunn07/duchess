# Phase 8: Footer Migration - Research

**Researched:** 2026-03-11
**Domain:** lipgloss TUI layout — footer bar refactor; IAM ARN left / clock right; DarkTheme token compliance
**Confidence:** HIGH

## Summary

Phase 8 refactors `internal/ui/status.go` — the existing status bar — into a dedicated footer with a new, simpler contract: IAM principal ARN on the left, live clock on the right, and no service/panel indicator anywhere. This is a targeted transformation of one existing file plus a parallel update to `model.go` to introduce `footerHeight` and wire it identically to how Phase 7 wired `headerHeight`.

The existing `renderStatusBar` function already handles: width-zero guard, left-group / right-group construction via gap-fill, `m.arn` display with `truncateARN`, and clock rendering. The Phase 8 work is surgical: rename the function to `renderFooter`, drop the left group's `version | profile | region | interval` block, remove the `panelIndicator` branching, consolidate the right group to ARN-left + clock-right, and confirm every style var still references `theme.DefaultTheme.*` (all currently do — the inline color audit shows status.go is already compliant post-Phase 6).

The critical downstream side-effect: introducing `footerHeight` as a package-level var in `status.go` (analogous to `headerHeight` in `header.go`) enables Phase 9 to compute `bodyHeight` correctly with `m.height - headerHeight - footerHeight`. All four `contentH` computation sites in `model.go` currently subtract `lipgloss.Height(statusBar)` dynamically; Phase 8 must replace this with `- footerHeight` (a constant) everywhere to match the `headerHeight` pattern, or keep the dynamic call — the research below recommends the constant approach for consistency.

**Primary recommendation:** Rewrite `status.go` as `footer.go` (or keep the filename and rename the function), declare `footerHeight = 1`, replace `renderStatusBar` with `renderFooter`, strip the left group and service indicator, and update `model.go` at all four `contentH` sites plus `baseView()` to call `renderFooter`.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LAYOUT-06 | Footer shows the current IAM principal ARN on the left | `m.arn` is on `rootModel`; `truncateARN` already exists in status.go; left-side placement uses same gap-fill pattern |
| LAYOUT-07 | Footer shows the clock on the right | `m.now.Format("15:04:05")` already in `renderStatusBar`; `clockStyle` already declared with `theme.DefaultTheme.TextPrimary` |
| LAYOUT-08 | Footer no longer shows a service indicator | `panelIndicator` branching and the `[ECS]` / `[S3]` strings are removed; left group (version/profile/region/interval) also removed |
</phase_requirements>

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/charmbracelet/lipgloss` | v1.1.0 | Style, layout, width/height measurement | Already in go.mod; all UI uses it |
| `github.com/jacobscunn07/duchess/internal/ui/theme` | — | Color tokens via `DefaultTheme` | Project-wide rule: no inline `lipgloss.Color()` |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `strings` (stdlib) | — | `strings.Repeat(" ", gap)` for gap-fill | Already used in status.go |
| `fmt` (stdlib) | — | Clock formatting (`m.now.Format`) | Already used; may be removable after refactor |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Keeping `status.go` filename | Renaming to `footer.go` | Either works; keeping the file avoids a rename + git history disruption. Recommend: keep `status.go`, rename function from `renderStatusBar` to `renderFooter`. |
| Dynamic `lipgloss.Height(footer)` for contentH | `footerHeight = 1` constant | footerHeight is always 1 row (no multi-line footer); a constant is self-documenting and consistent with `headerHeight` pattern |

**Installation:** No new dependencies needed.

---

## Architecture Patterns

### Current State of `status.go`

The file currently renders a two-group status bar:

**Left group** (to be removed entirely):
```
version | profile | region | interval
```

**Right group** (to be restructured):
```
[spinner/arn/error state] | [ECS]/[S3] panel indicator | HH:MM:SS
```

**After Phase 8**, the footer renders:
```
[ARN / loading... / error]          [HH:MM:SS]
```
Left: IAM identity string. Right: clock. No version, no profile, no region, no interval, no panel indicator.

### Recommended Project Structure

```
internal/ui/
├── status.go          # MODIFIED — rename renderStatusBar→renderFooter, add footerHeight=1,
│                      #   drop left group, remove panelIndicator, simplify right group
├── status_test.go     # MODIFIED — update tests to match new contract
├── model.go           # MODIFIED — replace renderStatusBar→renderFooter, add footerHeight to
│                      #   all four contentH sites, update baseView()
└── header.go          # UNCHANGED — reference pattern
```

### Pattern 1: footerHeight as Package-Level Var (mirrors headerHeight)

**What:** A single named var in `status.go` that declares how many terminal rows the footer occupies. Phase 9 needs this to compute `bodyHeight`.

```go
// footerHeight is the number of terminal rows the footer occupies.
// The footer is always a single row.
const footerHeight = 1
```

**Why `const` (not `var`):** Unlike `headerHeight` (derived from ASCII art at init), footer height is always 1 row and will never change based on content. `const footerHeight = 1` is clearer and avoids init-order concerns. The STATE.md decision record confirms: "Phase 9 depends on both height constants (headerHeight and footerHeight) before bodyHeight() math is correct" — this implies it should be a stable constant.

### Pattern 2: Simplified renderFooter Function

**Current `renderStatusBar` structure:**
```
gap-fill: leftStr + spaces + rightStr
leftStr  = version + sep + profile + sep + region + sep + interval
rightStr = identityStr [+ sep + panelIndicator] + sep + clock
```

**New `renderFooter` structure:**
```
gap-fill: leftStr + spaces + rightStr
leftStr  = identityStr   (ARN / "loading..." / "error")
rightStr = clock
```

The gap-fill idiom, the width-zero guard, and `statusBarStyle.Width(m.width).Render(...)` all stay identical. Only the content of `leftStr` and `rightStr` changes.

**Code pattern:**
```go
// footerHeight is the number of terminal rows the footer occupies (always 1).
const footerHeight = 1

func renderFooter(m rootModel) string {
    if m.width == 0 {
        return ""
    }

    // Left: IAM identity — state-dependent
    var leftStr string
    switch m.state {
    case stateLoading:
        leftStr = identityStyle.Render(m.spinner.View() + " loading...")
    case stateError:
        leftStr = identityStyle.Render("error")
    case stateReady:
        leftStr = identityStyle.Render(m.account + " " + truncateARN(m.arn, 40))
    }

    // Right: live clock
    rightStr := clockStyle.Render(m.now.Format("15:04:05"))

    // Gap fill to full width
    gap := m.width - lipgloss.Width(leftStr) - lipgloss.Width(rightStr)
    if gap < 0 {
        gap = 0
    }

    return statusBarStyle.Width(m.width).Render(leftStr + strings.Repeat(" ", gap) + rightStr)
}
```

**Style vars already declared in status.go and still valid:**
- `statusBarStyle` — Surface background, used as the outer wrapper
- `identityStyle` — TextPrimary foreground (was used for identity, still correct for ARN)
- `clockStyle` — TextPrimary foreground
- `separatorStyle` — no longer needed; can be removed or left (harmless)
- `versionStyle`, `profileStyle`, `regionStyle`, `intervalStyle` — no longer used; remove to keep code clean

### Pattern 3: model.go Integration — Replace `renderStatusBar` with `renderFooter`

**Current state** (model.go, after Phase 7 — all four `contentH` sites already subtract `headerHeight`):

```go
// Site 1 — baseView()
statusBar := renderStatusBar(m)
contentH := m.height - headerHeight - lipgloss.Height(statusBar)
...
return lipgloss.JoinVertical(lipgloss.Left, header, content, statusBar)

// Sites 2, 3, 4 — contentView(), identityLoadedMsg handler, regionSelectedMsg handler
statusBar := renderStatusBar(m)
contentH := m.height - headerHeight - lipgloss.Height(statusBar)
```

**After Phase 8** — replace dynamic `lipgloss.Height(statusBar)` with constant `footerHeight`:

```go
// Site 1 — baseView()
footer := renderFooter(m)
contentH := m.height - headerHeight - footerHeight
...
return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)

// Sites 2, 3, 4 — contentView(), identityLoadedMsg handler, regionSelectedMsg handler
contentH := m.height - headerHeight - footerHeight
// Remove "statusBar := renderStatusBar(m)" lines that become unused
```

**Note:** Sites 2, 3, and 4 only use `statusBar` to compute `lipgloss.Height(statusBar)`. Once replaced by the `footerHeight` constant, the `statusBar` local variable is unused and the `renderStatusBar(m)` call should be removed entirely from those sites. Only `baseView()` needs the rendered footer string.

### Pattern 4: DarkTheme Token Audit

**All style vars in current `status.go` already use `theme.DefaultTheme.*` — no inline colors.**

Confirmation (from source inspection):
```go
statusBarStyle  → Background(theme.DefaultTheme.Surface)       ✓
versionStyle    → Foreground(theme.DefaultTheme.TextPrimary)   ✓ (to be removed)
profileStyle    → Foreground(theme.DefaultTheme.Accent)        ✓ (to be removed)
regionStyle     → Foreground(theme.DefaultTheme.Accent)        ✓ (to be removed)
intervalStyle   → Foreground(theme.DefaultTheme.Accent)        ✓ (to be removed)
identityStyle   → Foreground(theme.DefaultTheme.TextPrimary)   ✓ (retained)
clockStyle      → Foreground(theme.DefaultTheme.TextPrimary)   ✓ (retained)
separatorStyle  → Foreground(theme.DefaultTheme.Muted)         ✓ (to be removed)
```

Phase 8 success criterion "all inline `lipgloss.Color` strings in status.go are replaced with DarkTheme token references" is ALREADY MET by Phase 6. Phase 8 just needs to not introduce any new inline colors while removing the unwanted style vars. No remediation work needed.

### Anti-Patterns to Avoid

- **Leaving unused style vars:** `versionStyle`, `profileStyle`, `regionStyle`, `intervalStyle`, `separatorStyle` will be unused after the refactor. Unused package-level vars in Go compile but `go vet` warns about them in some linters. Remove them to keep the file clean and avoid confusing future maintainers.
- **Forgetting to remove `renderStatusBar` call sites:** The function rename from `renderStatusBar` to `renderFooter` will cause a compile error at every old call site in model.go. There are exactly four call sites — all must be updated in the same pass.
- **Leaving `statusBar` local variable in Sites 2/3/4:** After replacing `lipgloss.Height(statusBar)` with `footerHeight`, the `statusBar` variable in those sites is unused. Go will refuse to compile an unused variable. Remove the entire `statusBar := renderStatusBar(m)` line from those three sites.
- **Using `var footerHeight` instead of `const`:** `var` works but `const` is cleaner for a value that never changes, and it avoids any risk of accidental mutation.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ARN truncation | Custom string slicer | `truncateARN(arn, 40)` (already in status.go) | Handles exact-length edge case; already tested |
| Terminal-width filling | Character counting loop | `statusBarStyle.Width(m.width).Render(leftStr + spaces + rightStr)` | lipgloss handles ANSI-aware width fill |
| Height measurement | `len(strings.Split(footer, "\n"))` | `const footerHeight = 1` | Footer is always 1 row; constant is self-documenting |

**Key insight:** The gap-fill idiom (compute `gap = width - leftW - rightW`, fill with `strings.Repeat(" ", gap)`) is already battle-tested in `renderStatusBar`. Reuse it verbatim — it correctly handles ANSI-colored substrings because `lipgloss.Width()` strips ANSI before measuring.

---

## Common Pitfalls

### Pitfall 1: Unused local variable compile error at Sites 2/3/4
**What goes wrong:** After replacing `contentH := m.height - headerHeight - lipgloss.Height(statusBar)` with `contentH := m.height - headerHeight - footerHeight`, the line `statusBar := renderStatusBar(m)` immediately above becomes unused. Go refuses to compile unused variables.
**Why it happens:** Sites 2, 3, and 4 only used `statusBar` as an argument to `lipgloss.Height`. Once that argument is replaced by `footerHeight`, the variable has no readers.
**How to avoid:** Remove the `statusBar := renderStatusBar(m)` (or its rename `footer := renderFooter(m)`) from contentView, identityLoadedMsg handler, and regionSelectedMsg handler. Only `baseView()` needs the rendered string.
**Warning signs:** `go build` error: "statusBar declared and not used".

### Pitfall 2: Missing `renderStatusBar` call site in model.go
**What goes wrong:** If one of the four `renderStatusBar` call sites in model.go is not updated, the code fails to compile (undefined: `renderStatusBar`) or silently uses stale logic.
**Why it happens:** Four distinct sites call this function. A grep-and-replace may miss one if the context differs.
**How to avoid:** After renaming, run `grep -rn 'renderStatusBar' internal/ui/` — expected: zero matches. Any remaining match is a missed update.
**Warning signs:** Compile error "undefined: renderStatusBar" OR lingering grep hit.

### Pitfall 3: `separatorValue` const left orphaned
**What goes wrong:** `separatorValue` is a `const` used only by `separatorStyle.Render(separatorValue)`. After removing `separatorStyle`, the const is unused.
**Why it happens:** Go does NOT error on unused constants (only unused vars and imports). So the code compiles but leaves dead code.
**How to avoid:** Remove both `separatorValue` const and `separatorStyle` var together. Run `go vet ./internal/ui/...` and `staticcheck ./internal/ui/...` if available.

### Pitfall 4: contentH negative on terminal height < headerHeight + footerHeight
**What goes wrong:** `m.height - headerHeight - footerHeight` = 24 - 7 - 1 = 16 normally. On a very short terminal (e.g., 5 rows), this goes negative. All four `contentH` sites already have guards — ensure the guards survive the edit.
**Why it happens:** The existing guards (`< 0`, `< 1`, `<= 0`) are in place but only correct if the surrounding subtraction is correct.
**How to avoid:** All four guard conditions must remain in place after the refactor. Do not delete them.

### Pitfall 5: `footerHeight` not visible to Phase 9
**What goes wrong:** Phase 9 needs `footerHeight` to compute `bodyHeight`. If `footerHeight` is declared inside a function instead of at package level, Phase 9 cannot reference it.
**Why it happens:** Temptation to hardcode `1` inline rather than naming it.
**How to avoid:** Declare `const footerHeight = 1` at package level in `status.go`, just as `var headerHeight` is declared at package level in `header.go`.

---

## Code Examples

### Minimal renderFooter (verified pattern from status.go)

```go
// Source: refactored from internal/ui/status.go gap-fill pattern
const footerHeight = 1

func renderFooter(m rootModel) string {
    if m.width == 0 {
        return ""
    }

    var leftStr string
    switch m.state {
    case stateLoading:
        leftStr = identityStyle.Render(m.spinner.View() + " loading...")
    case stateError:
        leftStr = identityStyle.Render("error")
    case stateReady:
        leftStr = identityStyle.Render(m.account + " " + truncateARN(m.arn, 40))
    }

    rightStr := clockStyle.Render(m.now.Format("15:04:05"))

    gap := m.width - lipgloss.Width(leftStr) - lipgloss.Width(rightStr)
    if gap < 0 {
        gap = 0
    }

    return statusBarStyle.Width(m.width).Render(leftStr + strings.Repeat(" ", gap) + rightStr)
}
```

### baseView() after Phase 8

```go
// Source: internal/ui/model.go — updated in this phase
func (m rootModel) baseView() string {
    header := renderHeader(m, m.width)
    footer := renderFooter(m)
    contentH := m.height - headerHeight - footerHeight
    if contentH < 0 {
        contentH = 0
    }
    content := lipgloss.NewStyle().Width(m.width).Height(contentH).Render(m.contentView())
    return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}
```

### contentView() after Phase 8 (Site 2)

```go
// Before:
statusBar := renderStatusBar(m)
contentH := m.height - headerHeight - lipgloss.Height(statusBar)
if contentH <= 0 {
    return m.spinner.View() + " Connecting to AWS..."
}

// After:
contentH := m.height - headerHeight - footerHeight
if contentH <= 0 {
    return m.spinner.View() + " Connecting to AWS..."
}
```

### identityLoadedMsg handler after Phase 8 (Site 3)

```go
// Before:
statusBar := renderStatusBar(m)
contentH := m.height - headerHeight - lipgloss.Height(statusBar)
if contentH < 1 {
    contentH = 1
}

// After:
contentH := m.height - headerHeight - footerHeight
if contentH < 1 {
    contentH = 1
}
```

### Remaining style vars in status.go (keep these, remove the rest)

```go
var (
    statusBarStyle = lipgloss.NewStyle().Background(theme.DefaultTheme.Surface)

    identityStyle = lipgloss.NewStyle().
            Foreground(theme.DefaultTheme.TextPrimary).
            Inherit(statusBarStyle)

    clockStyle = lipgloss.NewStyle().
            Foreground(theme.DefaultTheme.TextPrimary).
            Inherit(statusBarStyle)
)
```

Style vars to **remove** from status.go: `versionStyle`, `profileStyle`, `regionStyle`, `intervalStyle`, `separatorStyle`, and `const separatorValue`.

---

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|-----------------|--------|
| Status bar showed version + profile + region + interval on left | Footer shows only IAM ARN on left | Simpler; version/config info moved to header in Phase 7 |
| Panel indicator `[ECS]` / `[S3]` in status bar | Removed; service shown in right content panel header (Phase 9) | Cleaner footer; panel context comes from content area |
| Dynamic `lipgloss.Height(statusBar)` in contentH | `footerHeight` constant | Consistent with `headerHeight` pattern; faster (no ANSI measurement) |

**Deprecated/outdated:**
- `renderStatusBar` function name: replaced by `renderFooter` in this phase.
- `separatorValue` const and `separatorStyle` var: no longer used.
- Left group styles (`versionStyle`, `profileStyle`, `regionStyle`, `intervalStyle`): version/config info now lives only in the header.

---

## Open Questions

1. **Rename file `status.go` → `footer.go`?**
   - What we know: The filename `status.go` no longer matches its content after the refactor. `footer.go` would be more self-describing and consistent with `header.go`.
   - What's unclear: Whether the team prefers minimal git diff (keep `status.go`) or semantic naming (`footer.go`).
   - Recommendation: Rename to `footer.go` for consistency with `header.go`. Update `status_test.go` → `footer_test.go` as well. This makes the codebase fully self-describing. If git history continuity matters more, keep `status.go` — the choice does not affect behavior.

2. **Rename `statusBarStyle` → `footerStyle`?**
   - What we know: `statusBarStyle` is the outer wrapper used in `renderFooter`. Renaming it would match the new semantic.
   - What's unclear: Whether the rename adds cognitive overhead vs. clarity.
   - Recommendation: Rename to `footerStyle` for consistency. It's a one-line change and makes the code self-describing.

3. **Should `truncateARN` stay in `status.go`/`footer.go`?**
   - What we know: `truncateARN` is tested in `status_test.go`. It is only used in the footer for display. The function name is independent of file name.
   - What's unclear: Nothing — it stays wherever `renderFooter` is defined.
   - Recommendation: Keep it in the same file as `renderFooter`. No action needed.

---

## Sources

### Primary (HIGH confidence)
- `internal/ui/status.go` — authoritative source; inspected all style vars and confirmed DarkTheme compliance
- `internal/ui/model.go` — all four `contentH` computation sites identified via direct grep; confirmed they all currently call `renderStatusBar`
- `internal/ui/header.go` — confirmed `headerHeight = logoHeight + 1 = 7` (logoHeight=6, version row adds 1) and `footerHeight` must be a parallel package-level constant
- `internal/ui/theme/theme.go` — confirmed token names in use: `Surface`, `TextPrimary`, `Muted`
- `.planning/STATE.md` — confirmed Phase 8 depends on Phase 6 only; footerHeight needed by Phase 9

### Secondary (MEDIUM confidence)
- lipgloss v1.1.0 `Width()`, `JoinVertical`, `Style.Width()` behavior — based on direct codebase usage (these APIs are already working in production in this repo)

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; same libraries as all existing UI; all patterns already in use
- Architecture: HIGH — renderFooter is a simplification of renderStatusBar; integration pattern mirrors header exactly
- Pitfalls: HIGH — derived from direct code inspection and Go language rules (unused variable compile error is deterministic)

**Research date:** 2026-03-11
**Valid until:** 2026-04-10 (lipgloss v1.1.0 is stable; no fast-moving dependencies)
