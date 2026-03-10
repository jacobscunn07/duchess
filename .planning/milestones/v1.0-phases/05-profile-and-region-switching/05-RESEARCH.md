# Phase 5: Profile and Region Switching - Research

**Researched:** 2026-03-06
**Domain:** Bubble Tea overlay models, INI file parsing, AWS context cancellation, credential error taxonomy
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Keybindings:**
- `p` opens the profile picker overlay (from any stateReady screen)
- `r` opens the region picker overlay (from any stateReady screen)
- Both overlays dismissed with `Esc` (no switch — return to previous state)
- Both overlays navigate with j/k (scroll), g/G (top/bottom), Enter (confirm selection)

**Overlay visual style:**
- Centered floating window rendered over the existing content
- Overlay has a border (lipgloss border style, consistent with existing panel styling)
- Content behind the overlay is still rendered but visually "behind" the overlay
- Overlay shows: title ("Switch Profile" / "Switch Region"), list of options with current selection highlighted
- Current active profile/region is pre-selected (cursor starts on the currently active item)
- Width: ~50% of terminal width, height: dynamic to content (capped at 80% terminal height)

**Panel state on profile switch:**
- Both S3 and ECS panels fully reset to top level on profile switch
- All cached data cleared (no attempt to preserve position)
- Loading state shown in both panels immediately after switch
- STS GetCallerIdentity re-fetched for new profile; status bar shows "loading..." until identity confirmed
- In-flight goroutines from previous session cancelled via context cancellation before panel rebuild

**Panel state on region switch:**
- ECS panel resets to cluster list for the new region
- S3 panel does NOT reset — bucket list stays (S3 is global)
- S3 per-bucket regional clients still work correctly (use HeadBucket to detect per-bucket region)
- Active panel after region switch stays on whatever panel was active (no forced panel jump)

**Credential error UX on switch:**
- Each panel independently shows its own inline error if credential check fails after switch
- Error message for expired SSO token: exact `aws sso login --profile <name>` command (consistent with Phase 1 error taxonomy via ClassifyCredentialError)
- Error message for role chain expiry: identifies the source_profile by name so user knows which chain broke
- Profile picker remains accessible (`p` key) even when panels are in error state — user can switch back
- Region picker also remains accessible (`r` key) when in error state

**Profile enumeration:**
- Profile names read directly from `~/.aws/config` via `gopkg.in/ini.v1` (no SDK enumeration API)
- Display without the "profile " prefix

### Claude's Discretion
- Exact overlay border style (rounded, normal, thick — pick consistent with existing panel borders)
- Exact overlay sizing formula
- Whether to dim/fade non-overlay content (lipgloss opacity not well-supported in all terminals; optional)
- Animation/transition on overlay open (can be instant)
- How to handle extremely long profile names (truncate with "…")
- Region list source: hardcoded list of standard AWS commercial regions (not fetched dynamically — avoids bootstrapping problem of needing credentials to list regions)
- Exact set of regions to include in region picker

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| AUTH-01 | User can switch between AWS profiles defined in ~/.aws/config without restarting the app | ini.v1 for profile enumeration; overlay list model pattern; profileSelectedMsg wired to rootModel rebuild; context cancellation pattern for in-flight goroutines |
| AUTH-05 | User can switch between AWS regions without restarting the app | Hardcoded region list; regionSelectedMsg wired to rootModel; ECS panel reset vs S3 panel no-reset; overlay list model reused |
</phase_requirements>

---

## Summary

Phase 5 builds two modal overlay pickers — profile and region — implemented as Bubble Tea child models that are composited over the existing background view. The core overlay rendering is handled by `github.com/rmhubbert/bubbletea-overlay` v0.6.5, which was explicitly confirmed to depend on bubbletea v1.3.10 and lipgloss v1.1.0 — exact matches for this project's current dependencies. No version conflicts exist; this library can be added directly.

Profile enumeration uses `gopkg.in/ini.v1` v1.67.1, a well-established INI parser. The AWS config file uses `[profile name]` section headers for non-default profiles and `[default]` for the default — the "profile " prefix must be stripped when displaying names. The approach is explicit: load `~/.aws/config` with `ini.Load()`, iterate `cfg.Sections()`, strip the "profile " prefix from non-default sections. This is simpler and faster than invoking the AWS SDK's credential chain to enumerate profiles.

The context cancellation pattern requires adding a `cancelSession context.CancelFunc` to `rootModel`. On every profile or region switch, the existing cancel function is called first (which cancels all in-flight tea.Cmd goroutines holding that context), then a new `context.WithCancel(m.ctx)` is created and passed to reconstructed panel models. This is standard Go context propagation — no Bubble Tea-specific mechanism is needed.

**Primary recommendation:** Use `rmhubbert/bubbletea-overlay` for overlay compositing (verified v1 compatibility), `gopkg.in/ini.v1` for profile enumeration (no SDK dependency), `bubbles/list` for the item selector inside each overlay (reuses established project pattern), and `context.WithCancel` for per-session goroutine cancellation.

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/rmhubbert/bubbletea-overlay` | v0.6.5 | Overlay compositing of foreground model over background | Explicitly targets bubbletea v1.3.10 + lipgloss v1.1.0; MIT license; maintained for v1 users; avoids custom line-by-line ANSI scanner |
| `gopkg.in/ini.v1` | v1.67.1 | Parse `~/.aws/config` to enumerate profile names | No AWS credential bootstrapping needed; direct file read; strips "profile " prefix trivially |
| `github.com/charmbracelet/bubbles/list` | v1.0.0 (already in project) | Item list inside overlay (profile/region selector) | Already in project; SetFilteringEnabled(false) + DisableQuitKeybindings() pattern already established |
| `github.com/charmbracelet/lipgloss` | v1.1.0 (already in project) | Overlay border, sizing, style | Already in project; Border() for overlay container |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `context.WithCancel` (stdlib) | — | Per-session context cancellation | On every profile/region switch to cancel in-flight goroutines |
| `strings.TrimPrefix` (stdlib) | — | Strip "profile " prefix from ini section names | In profile enumeration function |
| `os.UserHomeDir` (stdlib) | — | Resolve `~/.aws/config` path correctly | Never use `~` string expansion; already established pattern in `config.go` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `rmhubbert/bubbletea-overlay` | Hand-rolled ANSI line scanner | Hand-rolling requires preserving ANSI escape sequences across line splits — error-prone; library solves this correctly |
| `gopkg.in/ini.v1` | `bufio.Scanner` + `strings.HasPrefix` | Manual parse is ~15 lines but brittle for quoted values, comments, multi-value keys; ini.v1 handles all edge cases |
| `gopkg.in/ini.v1` | `aws-sdk-go-v2/internal/ini` | Internal package is not part of the public API; subject to breaking change without notice |
| Hardcoded region list | `ec2.DescribeRegions` API | API call requires credentials already loaded — chicken-and-egg on region picker; hardcoded list is the standard pattern for region pickers |

**Installation:**
```bash
go get gopkg.in/ini.v1
go get github.com/rmhubbert/bubbletea-overlay
```

---

## Architecture Patterns

### Recommended Project Structure

```
internal/ui/
├── model.go              # rootModel — add cancelSession, isProfileOverlayOpen, isRegionOverlayOpen
├── messages.go           # add profileSelectedMsg, regionSelectedMsg
├── status.go             # unchanged (reads m.cfg.Profile and m.cfg.Region automatically)
├── overlay/
│   ├── profile.go        # profileOverlayModel — bubbles/list + ini.v1 enumeration
│   └── region.go         # regionOverlayModel — bubbles/list + hardcoded region list
├── s3/                   # unchanged
└── ecs/                  # unchanged
```

Placing overlays in `internal/ui/overlay/` keeps them isolated and mirrors the established `s3/` and `ecs/` subpackage pattern.

### Pattern 1: Overlay Model Lifecycle

**What:** Each overlay (profile, region) is a self-contained `tea.Model` with its own `bubbles/list`. The rootModel stores the overlay instance and an open/closed flag. The overlay is not part of the update loop until opened.

**When to use:** Whenever the user presses `p` or `r` in stateReady.

**Example:**
```go
// Source: project pattern + rmhubbert/bubbletea-overlay API
// In rootModel:
type rootModel struct {
    // ... existing fields ...
    cancelSession      context.CancelFunc      // cancels the current session's context
    sessionCtx         context.Context         // per-session cancellable context
    isProfileOverlayOpen bool
    isRegionOverlayOpen  bool
    profileOverlay     overlaymodel.ProfileOverlay
    regionOverlay      overlaymodel.RegionOverlay
}

// Open profile overlay on 'p' key:
case "p":
    if m.state == stateReady || m.state == stateError {
        m.isProfileOverlayOpen = true
        m.profileOverlay = overlaymodel.NewProfileOverlay(m.cfg.Profile, m.width, m.height)
        return m, m.profileOverlay.Init()
    }
```

### Pattern 2: Overlay Compositing in View()

**What:** When an overlay is open, `rootModel.View()` uses `overlay.Composite()` to render the foreground (overlay) on top of the background (existing panel view).

**When to use:** In `rootModel.View()` whenever `m.isProfileOverlayOpen || m.isRegionOverlayOpen`.

**Example:**
```go
// Source: rmhubbert/bubbletea-overlay API + existing View() pattern
import btoverlay "github.com/rmhubbert/bubbletea-overlay"

func (m rootModel) View() string {
    if m.width == 0 {
        return ""
    }
    background := m.baseView()  // existing content (panels + status bar)

    if m.isProfileOverlayOpen {
        fg := m.profileOverlay.View()
        return btoverlay.Composite(fg, background, btoverlay.Center, btoverlay.Center, 0, 0)
    }
    if m.isRegionOverlayOpen {
        fg := m.regionOverlay.View()
        return btoverlay.Composite(fg, background, btoverlay.Center, btoverlay.Center, 0, 0)
    }
    return background
}
```

### Pattern 3: Profile Enumeration via ini.v1

**What:** Read `~/.aws/config` directly, return profile names without "profile " prefix.

**When to use:** When constructing the profile overlay model.

**Example:**
```go
// Source: gopkg.in/ini.v1 API (pkg.go.dev/gopkg.in/ini.v1), verified against AWS config format
import ini "gopkg.in/ini.v1"

// ListAWSProfiles returns profile names from ~/.aws/config, stripping the "profile " prefix.
// The [default] section is returned as "default" without modification.
func ListAWSProfiles() ([]string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }
    configPath := filepath.Join(home, ".aws", "config")

    cfg, err := ini.LooseLoad(configPath)  // LooseLoad ignores missing file gracefully
    if err != nil {
        return nil, err
    }

    var profiles []string
    for _, section := range cfg.Sections() {
        name := section.Name()
        if name == ini.DefaultSection {
            continue  // skip the ini library's own DEFAULT section
        }
        if name == "default" {
            profiles = append(profiles, "default")
            continue
        }
        // AWS config uses "[profile name]" for non-default profiles
        name = strings.TrimPrefix(name, "profile ")
        profiles = append(profiles, name)
    }
    return profiles, nil
}
```

### Pattern 4: Per-Session Context Cancellation

**What:** Store a `cancelSession` function in rootModel. On profile/region switch, call cancel, create new cancellable context, rebuild panels with new context.

**When to use:** In the `profileSelectedMsg` and `regionSelectedMsg` handlers in `rootModel.Update()`.

**Example:**
```go
// Source: standard Go context pattern, established in CONTEXT.md
case profileSelectedMsg:
    // 1. Cancel in-flight goroutines from current session
    if m.cancelSession != nil {
        m.cancelSession()
    }
    // 2. Create new cancellable context for the new session
    sessionCtx, cancel := context.WithCancel(m.ctx)
    m.sessionCtx = sessionCtx
    m.cancelSession = cancel
    // 3. Update config and reset to loading state
    m.cfg.Profile = msg.profile
    m.state = stateLoading
    m.isProfileOverlayOpen = false
    m.account = ""
    m.arn = ""
    // 4. Re-fetch identity with new profile (panels rebuilt on identityLoadedMsg)
    return m, tea.Batch(
        fetchIdentityCmd(m.sessionCtx, m.cfg.Profile, m.cfg.Region),
        m.spinner.Tick,
    )

case regionSelectedMsg:
    if m.cancelSession != nil {
        m.cancelSession()
    }
    sessionCtx, cancel := context.WithCancel(m.ctx)
    m.sessionCtx = sessionCtx
    m.cancelSession = cancel
    m.cfg.Region = msg.region
    m.isRegionOverlayOpen = false
    // Reset ECS, keep S3 (S3 is global)
    refreshInterval := time.Duration(m.cfg.RefreshInterval) * time.Second
    statusBar := renderStatusBar(m)
    contentH := m.height - lipgloss.Height(statusBar)
    if contentH < 1 {
        contentH = 1
    }
    m.ecsPanel = ecspanel.NewModel(m.sessionCtx, m.awsCfg, m.width, contentH, refreshInterval)
    // Update awsCfg region for future API calls
    m.awsCfg.Region = msg.region
    return m, m.ecsPanel.Init()
```

### Pattern 5: Overlay Key Interception

**What:** When an overlay is open, rootModel intercepts all key messages before forwarding to panels. Esc closes overlay without switching. Enter triggers selection and fires profileSelectedMsg/regionSelectedMsg.

**When to use:** In `rootModel.Update()` when `m.isProfileOverlayOpen || m.isRegionOverlayOpen`.

**Example:**
```go
// Source: established project pattern from s3panel key interception (Phase 3)
case tea.KeyMsg:
    // If overlay is open, overlay handles all keys
    if m.isProfileOverlayOpen {
        switch msg.String() {
        case "esc":
            m.isProfileOverlayOpen = false
            return m, nil
        case "enter":
            selected := m.profileOverlay.SelectedProfile()
            if selected != "" {
                return m, func() tea.Msg { return profileSelectedMsg{profile: selected} }
            }
            m.isProfileOverlayOpen = false
            return m, nil
        default:
            var cmd tea.Cmd
            m.profileOverlay, cmd = m.profileOverlay.Update(msg)
            return m, cmd
        }
    }
```

### Anti-Patterns to Avoid

- **Opening overlay from stateLoading:** Both pickers should only be accessible when `stateReady` OR `stateError`. Do not allow overlay while initial identity load is in progress.
- **Using m.ctx directly after adding cancelSession:** Once per-session cancellation is in place, always use `m.sessionCtx` for panel and identity commands — never `m.ctx`.
- **Calling rootModel.View() recursively for background:** Extract background rendering into a `baseView()` helper and call it directly — do not call `m.View()` from within `m.View()`.
- **Forwarding key messages to panels when overlay is open:** The `p` and `r` key intercepts in the `stateReady` block must return before the "forward all messages to both panels" fallthrough.
- **Using m.cfg.Region directly after region switch for ECS panel:** The aws.Config embedded region needs explicit override — use `m.awsCfg.Region = msg.region` and pass the updated config to the new ECS panel.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Overlay compositing over ANSI-styled background | Custom line scanner that splits on `\n`, strips ANSI, measures widths | `rmhubbert/bubbletea-overlay` `Composite()` | ANSI escape sequences span byte boundaries; naive string splitting corrupts color state; the library handles this correctly |
| INI file parsing | Custom `bufio.Scanner` with regex for `[profile name]` headers | `gopkg.in/ini.v1` | AWS config can contain multi-line values, comments, escaped characters; ini.v1 handles all edge cases correctly |
| Profile enumeration via API | AWS SDK `ListProfiles` / `ec2:DescribeInstances` | Direct file read via ini.v1 | No such SDK API exists for listing locally-configured profiles; file read is the only correct approach |
| Region enumeration via API | `ec2.DescribeRegions` on switch | Hardcoded list | Requires credentials to be already valid — bootstrapping problem; new region may not have EC2; opt-in regions appear that aren't relevant |

**Key insight:** Overlay compositing over ANSI-colored terminal content is the hardest problem here. Using the library saves significant debugging time on ANSI state corruption.

---

## Common Pitfalls

### Pitfall 1: ini.DefaultSection vs "default" Profile

**What goes wrong:** `ini.v1` always creates a `ini.DefaultSection` (named `DEFAULT` or empty string) even when the file has no such section. Iterating `cfg.Sections()` without filtering this out adds a spurious entry to the profile list.

**Why it happens:** ini.v1 follows INI convention of having an implicit DEFAULT section; AWS config does not follow this convention.

**How to avoid:** Filter out sections where `section.Name() == ini.DefaultSection` (which equals `""` or `"DEFAULT"`) before adding to the profile list.

**Warning signs:** Profile list shows an empty entry or "DEFAULT" entry that doesn't correspond to any real profile.

### Pitfall 2: Using m.ctx vs m.sessionCtx After Migration

**What goes wrong:** After adding per-session context cancellation, some panel or command still uses the original `m.ctx`. When `m.cancelSession()` is called on switch, only the `m.sessionCtx`-based goroutines are cancelled — the `m.ctx`-based ones continue running and deliver stale messages.

**Why it happens:** Refactoring `m.ctx` to `m.sessionCtx` in all call sites is easy to miss, especially in the `Init()` command which runs before any session context exists.

**How to avoid:** In `Init()`, create the initial session context: `m.sessionCtx, m.cancelSession = context.WithCancel(m.ctx)`. Use `m.sessionCtx` everywhere panels and `fetchIdentityCmd` are called. The root `m.ctx` is only used as the parent context.

**Warning signs:** After profile switch, old S3 or ECS fetch results arrive and overwrite the new panel state.

### Pitfall 3: Region Switch Not Updating awsCfg

**What goes wrong:** ECS panel is rebuilt with `m.awsCfg` but `m.awsCfg.Region` was not updated. The new ECS panel calls ECS APIs but they go to the old region.

**Why it happens:** `m.awsCfg` is a value type (`aws.Config`). After `fetchIdentityCmd` sets it from the identity loaded message, it contains the initial region. A region switch must explicitly update `m.awsCfg.Region` before passing it to the new ECS panel, OR call `fetchIdentityCmd` with the new region and let it return a fresh aws.Config.

**How to avoid:** For region switch (which does not require new credentials), set `m.awsCfg.Region = msg.region` directly and pass the updated config. For profile switch (which requires new credentials), go through `fetchIdentityCmd` and let `identityLoadedMsg` rebuild the whole state — the new aws.Config will have the correct region.

**Warning signs:** ECS cluster list shows clusters from wrong region after switching.

### Pitfall 4: Status Bar Shows Stale Profile/Region During Switch

**What goes wrong:** After pressing Enter in the profile picker, the status bar continues to show the old profile until identity loads.

**Why it happens:** `renderStatusBar` reads `m.cfg.Profile`. If `m.cfg.Profile` is not updated immediately on `profileSelectedMsg`, the status bar lags.

**How to avoid:** Update `m.cfg.Profile = msg.profile` (and/or `m.cfg.Region = msg.region`) in the message handler before returning — the status bar reads these fields on every render, so the update is immediate.

**Warning signs:** STAT-06 fails UAT — user sees old profile name in status bar after confirming switch.

### Pitfall 5: Overlay Key Forwarding Race

**What goes wrong:** When the overlay is open, pressing Tab or other keys intended for the overlay get forwarded to the panels via the "forward all messages to both panels" fallthrough at the bottom of `rootModel.Update()`.

**Why it happens:** The fallthrough runs unconditionally if no earlier case matched. Overlay key handling must `return` before reaching the fallthrough.

**How to avoid:** The overlay key handling block must always `return (m, cmd)` — never fall through to the panel forwarding code. Test with Tab pressed while overlay is open.

**Warning signs:** Tab key closes overlay by toggling panel or causes unexpected behavior.

### Pitfall 6: Context Cancellation Sends context.Canceled to Panels

**What goes wrong:** After `m.cancelSession()`, in-flight goroutines complete with `context.Canceled` errors. If this error reaches the panel via a `xxxErrMsg`, the panel displays "context canceled" to the user.

**Why it happens:** The goroutine holding the old context returns an error when cancelled, which is dispatched as an error message.

**How to avoid:** In panel message handlers for error messages (e.g., `bucketsErrMsg`, `clustersErrMsg`), check for `errors.Is(err, context.Canceled)` and silently ignore these. A cancelled context means the panel is being replaced — no error should be shown.

**Warning signs:** After profile/region switch, a brief "context canceled" error flashes in a panel.

---

## Code Examples

Verified patterns from official sources:

### ini.v1 — Load AWS Config and List Profiles
```go
// Source: pkg.go.dev/gopkg.in/ini.v1 (verified API)
import (
    "os"
    "path/filepath"
    "strings"
    ini "gopkg.in/ini.v1"
)

func ListAWSProfiles() ([]string, error) {
    home, _ := os.UserHomeDir()
    path := filepath.Join(home, ".aws", "config")

    cfg, err := ini.LooseLoad(path) // LooseLoad: returns empty file if not found
    if err != nil {
        return nil, err
    }

    var profiles []string
    for _, s := range cfg.Sections() {
        name := s.Name()
        if name == "" || name == "DEFAULT" {
            continue // skip ini.v1 implicit default section
        }
        if name == "default" {
            profiles = append(profiles, "default")
        } else {
            // AWS config: non-default profiles are "[profile myname]"
            profiles = append(profiles, strings.TrimPrefix(name, "profile "))
        }
    }
    return profiles, nil
}
```

### rmhubbert/bubbletea-overlay — Composite in View()
```go
// Source: pkg.go.dev/github.com/rmhubbert/bubbletea-overlay (verified API, v0.6.5)
import btoverlay "github.com/rmhubbert/bubbletea-overlay"

func (m rootModel) View() string {
    if m.width == 0 {
        return ""
    }
    bg := m.baseView() // renders panels + status bar as a string

    if m.isProfileOverlayOpen {
        return btoverlay.Composite(m.profileOverlay.View(), bg,
            btoverlay.Center, btoverlay.Center, 0, 0)
    }
    if m.isRegionOverlayOpen {
        return btoverlay.Composite(m.regionOverlay.View(), bg,
            btoverlay.Center, btoverlay.Center, 0, 0)
    }
    return bg
}
```

### bubbles/list Overlay Model Construction
```go
// Source: established project pattern (s3/model.go NewModel) + bubbles v1.0.0
import (
    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/lipgloss"
)

func NewProfileOverlay(currentProfile string, termWidth, termHeight int, profiles []string) ProfileOverlay {
    overlayW := termWidth / 2
    if overlayW < 30 {
        overlayW = 30
    }
    overlayH := len(profiles) + 4 // items + title + padding
    maxH := termHeight * 4 / 5
    if overlayH > maxH {
        overlayH = maxH
    }

    items := make([]list.Item, len(profiles))
    selectedIdx := 0
    for i, p := range profiles {
        items[i] = profileItem{name: p}
        if p == currentProfile {
            selectedIdx = i
        }
    }

    l := list.New(items, profileDelegate{}, overlayW-4, overlayH-4)
    l.SetShowTitle(false)
    l.SetShowStatusBar(false)
    l.SetFilteringEnabled(false)
    l.DisableQuitKeybindings()
    l.Select(selectedIdx) // pre-select current profile

    borderStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("62")).
        Width(overlayW - 2).
        Height(overlayH - 2)

    return ProfileOverlay{list: l, borderStyle: borderStyle, width: overlayW, height: overlayH}
}
```

### context.WithCancel — Per-Session Pattern
```go
// Source: Go standard library context docs (go.dev/blog/context)

// In NewRootModel — initialize with first session context
func NewRootModel(ctx context.Context, cfg *config.Config) rootModel {
    sessionCtx, cancelSession := context.WithCancel(ctx)
    // ...
    return rootModel{
        ctx:           ctx,         // root context — parent of session contexts
        sessionCtx:    sessionCtx,  // cancellable session context — passed to panels
        cancelSession: cancelSession,
        // ...
    }
}

// In rootModel.Init() — use sessionCtx for identity fetch
func (m rootModel) Init() tea.Cmd {
    return tea.Batch(
        fetchIdentityCmd(m.sessionCtx, m.cfg.Profile, m.cfg.Region),
        m.spinner.Tick,
        tickCmd(),
    )
}
```

### AWS Regions Hardcoded List
```go
// Source: aws.amazon.com/about-aws/global-infrastructure/regions_az/ (verified 2026-03-06)
// 36 standard commercial regions (excludes GovCloud and China)
var awsRegions = []string{
    // North America
    "us-east-1",      // US East (N. Virginia)
    "us-east-2",      // US East (Ohio)
    "us-west-1",      // US West (N. California)
    "us-west-2",      // US West (Oregon)
    "ca-central-1",   // Canada (Central)
    "ca-west-1",      // Canada West (Calgary)
    "mx-central-1",   // Mexico (Central)
    // South America
    "sa-east-1",      // South America (São Paulo)
    // Europe
    "eu-west-1",      // Europe (Ireland)
    "eu-west-2",      // Europe (London)
    "eu-west-3",      // Europe (Paris)
    "eu-central-1",   // Europe (Frankfurt)
    "eu-central-2",   // Europe (Zurich)
    "eu-north-1",     // Europe (Stockholm)
    "eu-south-1",     // Europe (Milan)
    "eu-south-2",     // Europe (Spain)
    // Middle East
    "me-south-1",     // Middle East (Bahrain)
    "me-central-1",   // Middle East (UAE)
    "il-central-1",   // Israel (Tel Aviv)
    // Africa
    "af-south-1",     // Africa (Cape Town)
    // Asia Pacific
    "ap-northeast-1", // Asia Pacific (Tokyo)
    "ap-northeast-2", // Asia Pacific (Seoul)
    "ap-northeast-3", // Asia Pacific (Osaka)
    "ap-southeast-1", // Asia Pacific (Singapore)
    "ap-southeast-2", // Asia Pacific (Sydney)
    "ap-southeast-3", // Asia Pacific (Jakarta)
    "ap-southeast-4", // Asia Pacific (Melbourne)
    "ap-southeast-5", // Asia Pacific (Malaysia)
    "ap-south-1",     // Asia Pacific (Mumbai)
    "ap-south-2",     // Asia Pacific (Hyderabad)
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual ANSI line scanner for overlay | `rmhubbert/bubbletea-overlay` | 2024-2025 | No manual ANSI parsing required; compositing handled correctly |
| `aws-sdk-go-v1` ini parsing | `gopkg.in/ini.v1` direct | Long-standing | No SDK version coupling; reads file directly |
| lipgloss v2 compositing (Layer/Canvas) | lipgloss v1 + overlay library | v2 still beta | Project on v1; v2 compositing not available yet in this project's stack |
| Global context in rootModel | Per-session context.WithCancel | Phase 5 introduces | Enables clean cancellation of in-flight goroutines on switch |

**Deprecated/outdated:**
- lipgloss v1 has no `PlaceOverlay` — this was added only in v2 beta (canvas/layer API). Using v1 requires the overlay library.
- `aws-sdk-go-v2/internal/ini` — internal package, not a public API; never import internal packages directly.

---

## Open Questions

1. **Whether to include `ap-northeast-4` (Taipei) and `ap-southeast-7` (New Zealand) in region list**
   - What we know: AWS listed these on their global infrastructure page as of March 2026, but they may be newly announced/opt-in only
   - What's unclear: Whether these are generally available to all accounts or require opt-in enrollment
   - Recommendation: Include only the well-established default-enabled regions. The Claude's Discretion note says "Exact set of regions to include in region picker" — start with the 17 default-enabled regions and add newer ones if the user requests. Or include all 36 and trust the user to know which apply to their account.

2. **How `rmhubbert/bubbletea-overlay` handles the case where the overlay is taller than the terminal**
   - What we know: The library positions by center; if overlay height exceeds terminal height, compositing may clip or scroll unexpectedly
   - What's unclear: Whether the library hard-clips or panics on overflow
   - Recommendation: Cap overlay height at `termHeight * 4 / 5` before creating the list model. This is specified in the locked decisions.

3. **context.Canceled messages to panel after switch — does ClassifyCredentialError handle it?**
   - What we know: `ClassifyCredentialError` handles `ssocreds.InvalidTokenError` and `smithy.APIError` codes; it does NOT explicitly handle `context.Canceled`
   - What's unclear: Whether a cancelled context returns a `context.Canceled` error or a wrapped smithy error
   - Recommendation: Add `errors.Is(err, context.Canceled)` check at the TOP of each panel's error message handler to silently drop it. Do NOT add this to `ClassifyCredentialError` since it lives in the aws/session package and shouldn't need to know about context.

---

## Sources

### Primary (HIGH confidence)
- `pkg.go.dev/gopkg.in/ini.v1` — API functions (Load, LooseLoad, Sections, Section.Name), version v1.67.1
- `pkg.go.dev/github.com/rmhubbert/bubbletea-overlay` — API (New, Composite, Position constants), version v0.6.5
- `github.com/rmhubbert/bubbletea-overlay` go.mod — confirmed bubbletea v1.3.10 + lipgloss v1.1.0 dependency (exact match)
- `pkg.go.dev/github.com/aws/aws-sdk-go-v2/credentials/ssocreds` — InvalidTokenError type and behavior, version v1.19.11
- `pkg.go.dev/github.com/charmbracelet/lipgloss` — confirmed NO PlaceOverlay in v1.1.0
- `aws.amazon.com/about-aws/global-infrastructure/regions_az/` — complete region list verified 2026-03-06
- Existing codebase (`internal/ui/model.go`, `internal/ui/s3/model.go`, `internal/aws/session.go`) — established patterns for list initialization, key interception, message types, ClassifyCredentialError

### Secondary (MEDIUM confidence)
- `rmhubbert/bubbletea-overlay` README — "I'll keep maintaining this package for users of Bubbletea and Lipgloss v1" — explicitly v1 maintained
- `lmika.org/2022/09/24/overlay-composition-using.html` — explains why ANSI line scanner is complex (verifies that hand-rolling is inadvisable)
- `go.dev/blog/context` — canonical Go context.WithCancel cancellation pattern

### Tertiary (LOW confidence)
- None — all critical claims verified with official sources.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — rmhubbert/bubbletea-overlay go.mod exact version match confirmed; ini.v1 API verified on pkg.go.dev; bubbles/list already in project
- Architecture: HIGH — patterns derived directly from existing codebase (s3/model.go, model.go) with verified library APIs layered on top
- Pitfalls: HIGH — ini.DefaultSection pitfall verified against ini.v1 docs; context.Canceled pitfall is standard Go behavior; others derived from direct code inspection of existing model

**Research date:** 2026-03-06
**Valid until:** 2026-04-06 (stable libraries; region list may need periodic update as AWS adds regions)
