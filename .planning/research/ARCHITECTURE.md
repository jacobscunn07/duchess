# Architecture Patterns

**Domain:** Go TUI for AWS (k9s-style cloud resource navigator)
**Researched:** 2026-03-02
**Confidence:** HIGH (stack confirmed via go.mod; patterns from bubbletea docs, k9s source, and aws-sdk-go-v2 docs)

---

## Recommended Architecture

duchess follows a layered architecture with four discrete layers that have one-way dependencies:

```
┌─────────────────────────────────────────────────────────┐
│                     UI Layer                            │
│   Root Model → Panel Manager → Resource Panels          │
│   (Bubble Tea MVU — model/view/update)                  │
├─────────────────────────────────────────────────────────┤
│                  Controller Layer                        │
│   Resource Controllers (one per AWS service)            │
│   (orchestrate fetch, cache, refresh scheduling)        │
├─────────────────────────────────────────────────────────┤
│                   Store / Msg Layer                     │
│   Typed messages carry snapshots back to UI             │
│   (hold current data, delivered via program.Send())     │
├─────────────────────────────────────────────────────────┤
│                   Client Layer                          │
│   AWS API Clients (S3Client, ECSClient, STSClient)      │
│   (thin wrappers around aws-sdk-go-v2 service clients)  │
└─────────────────────────────────────────────────────────┘
```

Direction: UI → Controller → Client. Data returns as typed Msgs. Never reverse. UI never calls AWS directly.

---

## How k9s Structures This Problem (Reference Model)

k9s is the canonical reference for this architecture. Its internal packages map to:

| k9s Package | k9s Responsibility | duchess Equivalent |
|-------------|--------------------|--------------------|
| `internal/dao` | Data Access Objects — typed query+watch of k8s resources | `internal/aws/` — per-service clients |
| `internal/model` | Resource informers — hold the live cache, notify listeners | Typed Msgs carry snapshots from controllers to panels |
| `internal/ui` | View components (tables, headers, status bars) | `internal/ui/` — Bubble Tea panel components |
| `internal/view` | Compound views (bind model to UI, handle keybindings) | Handled inside each panel's tea.Model |
| `internal/config` | Config file + CLI flag merge | `internal/config/` |
| `internal/client` | Kubernetes client factory + context management | `internal/aws/session.go` — credential + session factory |

k9s uses a poll loop ("informers") rather than Kubernetes watch events for simpler resources. duchess should do the same — `time.Ticker` for periodic refresh, not event-driven subscriptions — because AWS APIs are polling-based by nature.

---

## Component Boundaries

### Component Map

| Component | Package | Responsibility | Communicates With |
|-----------|---------|---------------|-------------------|
| Root Model | `internal/ui/root.go` | Top-level Bubble Tea model; owns layout, global keybindings, active panel routing | Panel models (contains them), Config |
| Status Bar | `internal/ui/statusbar.go` | Renders profile, region, principal, time, version | Root Model passes state to it |
| S3 Panel | `internal/ui/s3/panel.go` | Bucket list → prefix list → object list navigation | Receives S3 Msgs; sends fetch Cmds |
| ECS Panel | `internal/ui/ecs/panel.go` | Cluster list → service list → task list navigation | Receives ECS Msgs; sends fetch Cmds |
| Profile Picker | `internal/ui/profile.go` | Overlay for switching AWS profiles | Root Model (sends ProfileSelectedMsg) |
| Region Picker | `internal/ui/region.go` | Overlay for switching regions | Root Model (sends RegionSelectedMsg) |
| S3 Controller | `internal/controller/s3.go` | Owns refresh ticker for S3; fetches on interval; sends Msgs | S3 Client (reads), program.Send() |
| ECS Controller | `internal/controller/ecs.go` | Owns refresh ticker for ECS; fetches clusters/services/tasks | ECS Client (reads), program.Send() |
| S3 Client | `internal/aws/s3.go` | Wraps aws-sdk-go-v2 S3; handles pagination transparently | Called by S3 Controller |
| ECS Client | `internal/aws/ecs.go` | Wraps aws-sdk-go-v2 ECS; handles pagination transparently | Called by ECS Controller |
| STS Client | `internal/aws/sts.go` | GetCallerIdentity for status bar principal display | Called at startup and profile switch |
| Session Factory | `internal/aws/session.go` | Loads aws-sdk-go-v2 config for a given profile+region; handles SSO, role assumption, named profiles | Called by Root Model on init and profile/region switch |
| Config | `internal/config/config.go` | Loads `~/.duchess/config` + CLI flags; CLI flags win on conflict | Called by main; passed to Root Model |

---

## Data Flow

### Startup Flow

```
main()
  → parse CLI flags
  → config.Load() merges ~/.duchess/config + flags
  → session.NewFactory(config.Profile, config.Region)
  → sts.GetCallerIdentity() → identity for status bar
  → start S3Controller (starts ticker goroutine)
  → start ECSController (starts ticker goroutine)
  → bubbletea.NewProgram(RootModel).Run()
```

### Refresh Flow (per controller, runs in background goroutine)

```
ticker fires every N seconds
  → S3Client.ListBuckets(ctx)        ← paginated, blocking, in goroutine
  → program.Send(S3BucketsRefreshedMsg{Buckets: buckets, Err: err})
      → Root Model.Update() receives Msg
      → delegates to S3Panel.Update()
      → S3Panel stores buckets in its own fields
      → next View() call renders updated table
```

### User Navigation Flow (drill-down into a bucket)

```
user presses Enter on a bucket row
  → S3Panel.Update(tea.KeyMsg{Type: tea.KeyEnter})
  → panel advances depth: BucketList → PrefixList
  → panel returns tea.Cmd: fetchPrefixes(client, selectedBucket, "")
  → Bubble Tea runtime executes Cmd in goroutine
  → S3ObjectsRefreshedMsg arrives at Root Model
  → delegates to S3Panel.Update()
  → panel stores objects, re-renders prefix list
```

### Profile/Region Switch Flow

```
user selects new profile from ProfilePicker overlay
  → ProfileSelectedMsg sent to Root Model
  → Root Model calls session.NewFactory(newProfile, currentRegion)
  → Root Model recreates S3Client, ECSClient with new session
  → S3Controller.Stop(), S3Controller.Start() with new client
  → ECSController.Stop(), ECSController.Start() with new client
  → sts.GetCallerIdentity() → sends IdentityFetchedMsg
  → status bar updates to new principal
```

---

## Bubble Tea MVU: How to Structure the Application

### The Golden Rule

Each resource panel is a self-contained `tea.Model`. The Root Model composes them and routes messages down. Panels never talk to each other directly.

```go
type RootModel struct {
    config      config.Config
    activePanelID PanelID
    s3Panel     s3.PanelModel
    ecsPanel    ecs.PanelModel
    statusBar   ui.StatusBarModel
    profile     string
    region      string
    principal   string
}
```

### Message Types Drive Everything

Define a typed message for each async event. This is the correct Bubble Tea idiom — all async results arrive as Msgs through `Update()`:

```go
// Background refresh (from controller goroutines)
type S3BucketsRefreshedMsg struct {
    Buckets []s3types.Bucket
    Err     error
}

// User-triggered navigation (from tea.Cmd)
type S3ObjectsRefreshedMsg struct {
    Prefix  string
    Objects []s3types.Object
    Err     error
}

type ECSClustersRefreshedMsg struct{ Clusters []ecstypes.Cluster; Err error }
type ECSServicesRefreshedMsg struct{ ClusterARN string; Services []ecstypes.Service; Err error }
type ECSTasksRefreshedMsg    struct{ ServiceARN string; Tasks []ecstypes.Task; Err error }

// Profile / region switching
type ProfileSelectedMsg struct{ Profile string }
type RegionSelectedMsg  struct{ Region string }
type IdentityFetchedMsg struct{ Principal string; Err error }
```

Root Model routes messages to the appropriate child panel in its `Update()` method.

### Background Polling Pattern

The refresh loop bridges goroutines and Bubble Tea cleanly using `program.Send()`, which is goroutine-safe:

```go
func (c *S3Controller) Start(program *tea.Program) {
    c.stop = make(chan struct{})
    go func() {
        ticker := time.NewTicker(c.interval)
        defer ticker.Stop()
        // Fetch immediately on start, then on each tick
        c.fetch(program)
        for {
            select {
            case <-ticker.C:
                c.fetch(program)
            case <-c.stop:
                return
            }
        }
    }()
}

func (c *S3Controller) fetch(program *tea.Program) {
    buckets, err := c.client.ListBuckets(context.Background())
    program.Send(S3BucketsRefreshedMsg{Buckets: buckets, Err: err})
}

func (c *S3Controller) Stop() {
    close(c.stop)
}
```

### Panel Navigation State Machine

Each panel owns its navigation depth as a state machine. This mirrors how k9s handles resource drill-down:

```go
type S3PanelModel struct {
    depth          S3NavDepth
    buckets        []s3types.Bucket
    objects        []s3types.Object
    selectedBucket string
    selectedPrefix string
    cursor         int
    filter         string
    client         aws.S3API  // needed for tea.Cmd fetches
}

type S3NavDepth int
const (
    S3NavBuckets S3NavDepth = iota
    S3NavObjects
)

func (m S3PanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "enter":
            if m.depth == S3NavBuckets {
                m.selectedBucket = m.currentBucketName()
                m.depth = S3NavObjects
                return m, fetchObjects(m.client, m.selectedBucket, "")
            }
        case "esc":
            if m.depth == S3NavObjects {
                m.depth = S3NavBuckets
            }
        }
    case S3BucketsRefreshedMsg:
        m.buckets = msg.Buckets
    case S3ObjectsRefreshedMsg:
        m.objects = msg.Objects
    }
    return m, nil
}
```

---

## Patterns to Follow

### Pattern 1: Typed AWS Wrappers (Thin Client Layer)

**What:** Define a Go interface for each AWS service. Implement against aws-sdk-go-v2. Mock the interface in tests.

**When:** Always — this is what enables testing without real AWS credentials.

```go
// internal/aws/s3.go
type S3API interface {
    ListBuckets(ctx context.Context) ([]s3types.Bucket, error)
    ListObjectsV2(ctx context.Context, bucket, prefix string) ([]s3types.Object, string, error)
}

type s3Client struct{ client *s3.Client }

func (c *s3Client) ListBuckets(ctx context.Context) ([]s3types.Bucket, error) {
    out, err := c.client.ListBuckets(ctx, &s3.ListBucketsInput{})
    if err != nil {
        return nil, err
    }
    return out.Buckets, nil
}
```

Pagination is handled inside the client wrapper — the controller and panel receive complete slices. Use aws-sdk-go-v2 paginators (`s3.NewListObjectsV2Paginator`) inside the wrapper.

### Pattern 2: tea.Cmd for User-Triggered Fetches

**What:** For navigation-driven fetches (drilling into a bucket), return a `tea.Cmd` from `Update()`. The result arrives as a typed Msg.

**When:** Any time a user action requires fresh data not yet available from the background refresh.

```go
func fetchObjects(client aws.S3API, bucket, prefix string) tea.Cmd {
    return func() tea.Msg {
        objects, nextToken, err := client.ListObjectsV2(
            context.Background(), bucket, prefix,
        )
        return S3ObjectsRefreshedMsg{Objects: objects, Err: err}
    }
}
```

`tea.Cmd` runs in a goroutine managed by the Bubble Tea runtime. This is the standard idiomatic pattern — never call blocking I/O inside `Update()` directly.

### Pattern 3: Configuration Merge (File + Flags)

**What:** Load file config first, then overlay CLI flags. Flags always win. Use sentinel values (empty string, zero) to detect "flag not provided by user."

**When:** Always — standard CLI tool convention.

```go
type Config struct {
    Profile         string
    Region          string
    RefreshInterval time.Duration
}

func Load(flags *Flags) (Config, error) {
    c := defaults()
    if fileConf, err := loadFile(configPath()); err == nil {
        merge(&c, fileConf)
    }
    overlayFlags(&c, flags) // CLI flags overwrite non-zero values
    return validate(c)
}
```

### Pattern 4: Session Factory for Profile/Region Switching

**What:** A factory that produces a fresh `aws.Config` for a given profile+region. Called on startup and on each switch. aws-sdk-go-v2's `LoadDefaultConfig` handles all three auth types (access key, role assumption, SSO) via standard credential chain resolution — no special-casing needed.

**When:** Startup, ProfileSelectedMsg, RegionSelectedMsg.

```go
type SessionFactory struct{}

func (f *SessionFactory) NewConfig(
    ctx context.Context,
    profile, region string,
) (aws.Config, error) {
    return config.LoadDefaultConfig(ctx,
        config.WithSharedConfigProfile(profile),
        config.WithRegion(region),
    )
}
```

### Pattern 5: Service-Specific Refresh Intervals

**What:** Each controller has its own ticker interval. S3 changes rarely; ECS task state changes frequently.

**When:** Always — a single global refresh interval is a poor fit across services.

Recommended defaults:
- S3 bucket list: 30 seconds
- ECS clusters/services: 10 seconds
- ECS tasks: 5 seconds (task status changes fast during deployments)

All configurable. Global `refresh_interval` in config acts as a multiplier baseline if desired.

### Pattern 6: Immediate First Fetch, Then Periodic

**What:** Controllers fetch immediately on `Start()`, then continue on the ticker. This avoids the initial blank-screen wait.

**When:** Always — users expect to see data as soon as the TUI loads.

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Calling AWS Directly from `View()`

**What:** Running AWS API calls inside the `View()` method of a Bubble Tea model.

**Why bad:** `View()` is called on every render tick (potentially many times per second during animation/resize). AWS calls are slow (10-500ms) and will block the render loop, freeze input handling, and generate unnecessary API traffic.

**Instead:** Return `tea.Cmd` from `Update()` for one-shot fetches. Use background goroutines with `program.Send()` for periodic refresh.

### Anti-Pattern 2: Mutating a Shared `aws.Config` on Profile Switch

**What:** Caching a single `aws.Config` instance and modifying it when the profile or region changes.

**Why bad:** aws-sdk-go-v2 configs embed credential providers, SSO token caches, and retry state. These are not designed for mutation after creation. Silent bugs and stale credentials result.

**Instead:** Call `SessionFactory.NewConfig()` to produce a fresh config on every profile/region switch. Recreate all service clients from it. Cost is negligible (no network call until first API use).

### Anti-Pattern 3: One Giant Root Model with All Resource State

**What:** Putting bucket list, object list, cluster list, task list, and all navigation state into `RootModel`.

**Why bad:** The `Update()` method becomes a thousand-line switch statement. Adding ECS requires touching S3 code. Components cannot be tested in isolation.

**Instead:** Each resource panel is its own `tea.Model`. Root Model delegates via `m.activePanel().Update(msg)`. Each panel is independently testable with mock clients.

### Anti-Pattern 4: Blocking Inside `Update()`

**What:** Making synchronous AWS API calls directly inside `Update()` and waiting for the result before returning.

**Why bad:** `Update()` runs on the main goroutine. Blocking it freezes keyboard input processing, causes TUI to appear hung, and prevents timer ticks from firing.

**Instead:** Always return a `tea.Cmd` for any I/O. The Bubble Tea runtime executes the Cmd in a goroutine; the result arrives later as a Msg.

### Anti-Pattern 5: Treating S3 as a Regional Service

**What:** Expecting `ListBuckets` to return only buckets in the selected region, or switching the S3 client session when the region changes.

**Why bad:** S3 is a global service. `ListBuckets` returns all buckets regardless of the region configured in the client. Confuses users who switch region and see the same bucket list; may break if you try to operate on a bucket using the wrong regional endpoint.

**Instead:** Use a single global S3 session (us-east-1 as default endpoint). Display bucket region in the detail view (via `GetBucketLocation`). Make it clear in the UI that S3 is global. Only ECS and other regional services change when the region picker is used.

### Anti-Pattern 6: Polling All Services at the Same Frequency

**What:** Using the same 2-second ticker for both S3 and ECS.

**Why bad:** S3 bucket lists change rarely (hours to days). Polling every 2 seconds generates unnecessary API calls, burns through rate limits, and wastes network. ECS task state needs frequent polling (seconds) during deployments.

**Instead:** Service-specific refresh intervals (see Pattern 5 above).

---

## Directory Structure

```
duchess/
├── cmd/
│   └── duchess/
│       └── main.go                  # Parse CLI flags, wire everything, start tea.Program
├── internal/
│   ├── aws/
│   │   ├── session.go               # SessionFactory — produces aws.Config per profile+region
│   │   ├── s3.go                    # S3API interface + s3Client implementation (paginated)
│   │   ├── ecs.go                   # ECSAPI interface + ecsClient implementation (paginated)
│   │   └── sts.go                   # STSAPI interface + GetCallerIdentity
│   ├── config/
│   │   └── config.go                # Config struct, Load(), file + flag merge, defaults
│   ├── controller/
│   │   ├── s3.go                    # S3Controller — refresh ticker, goroutine, program.Send()
│   │   └── ecs.go                   # ECSController — refresh ticker, goroutine, program.Send()
│   └── ui/
│       ├── root.go                  # RootModel — top-level tea.Model, panel routing
│       ├── statusbar.go             # StatusBarModel — bottom bar: profile/region/principal/time
│       ├── profile_picker.go        # ProfilePickerModel — overlay, reads ~/.aws/config profiles
│       ├── region_picker.go         # RegionPickerModel — overlay, list of AWS regions
│       ├── messages.go              # Shared Msg types (ProfileSelectedMsg, RegionSelectedMsg, etc.)
│       ├── s3/
│       │   ├── panel.go             # S3PanelModel — bucket→prefix→object navigation state machine
│       │   └── messages.go          # S3-specific Msg types (S3BucketsRefreshedMsg, etc.)
│       └── ecs/
│           ├── panel.go             # ECSPanelModel — cluster→service→task navigation state machine
│           └── messages.go          # ECS-specific Msg types (ECSClustersRefreshedMsg, etc.)
```

---

## Scalability Considerations

| Concern | At current scope (S3 + ECS) | Adding EC2 / Lambda / CloudWatch |
|---------|-----------------------------|---------------------------------|
| Adding a new service | Add `internal/aws/[svc].go` + `internal/controller/[svc].go` + `internal/ui/[svc]/panel.go` | Same pattern — no changes to existing services required |
| Rate limiting | S3/ECS read-only calls at 5-30s intervals; no throttling expected at 1 user | Add per-service rate limiters in client wrappers using `golang.org/x/time/rate` |
| Large resource counts | S3 ListObjectsV2 paginates at 1000/page; handle in client wrapper | Virtual scrolling in table panels; load next page on scroll-to-bottom |
| Multiple accounts | Not in v1; foundation exists: SessionFactory already takes profile as param | Add account switcher as peer of profile/region pickers |
| Credential expiry | aws-sdk-go-v2 refreshes SSO tokens automatically within a live session | Catch 401/403 errors from any client; surface "credentials expired" panel, prompt re-auth |

---

## Suggested Build Order

Dependencies determine sequence. This order minimizes re-work and provides clear validation gates.

### Stage 1: Foundation (nothing else starts without this)

1. `internal/config/config.go` — Config struct + Load(). Every other component reads config.
2. `internal/aws/session.go` — SessionFactory. All AWS clients depend on it.
3. `internal/aws/sts.go` — GetCallerIdentity. Validates credentials on startup.
4. `cmd/duchess/main.go` — Wire config load, session creation, minimal bubbletea loop.

**Gate:** Binary authenticates to AWS and retrieves caller identity. Proves auth works across all three profile types (named, role assumption, SSO).

### Stage 2: UI Shell (required before any resource panels)

5. `internal/ui/root.go` — Minimal RootModel: empty content area, status bar, `q` to quit.
6. `internal/ui/statusbar.go` — StatusBarModel: profile / region / principal / time / version.

**Gate:** TUI launches cleanly, shows real AWS identity in status bar, quits on `q`. Proves the bubbletea program structure and status bar work.

### Stage 3: S3 (first resource service — proves the full pattern end-to-end)

7. `internal/aws/s3.go` — S3API interface + ListBuckets / ListObjectsV2 implementations.
8. `internal/controller/s3.go` — S3Controller: refresh ticker, goroutine, program.Send().
9. `internal/ui/s3/panel.go` — S3PanelModel: bucket list → object list navigation.

**Gate:** Navigate S3 buckets and objects with live refresh. Proves the Controller → Msg → Panel pattern works end-to-end. All subsequent services follow this exact shape.

### Stage 4: ECS (second resource service — validates pattern generalizes to regional services)

10. `internal/aws/ecs.go` — ECSAPI interface + ListClusters / ListServices / ListTasks.
11. `internal/controller/ecs.go` — ECSController: refresh ticker at ECS-appropriate interval.
12. `internal/ui/ecs/panel.go` — ECSPanelModel: cluster → service → task navigation.

**Gate:** Navigate ECS resources with live refresh. Confirm the pattern works for a regional service (ECS cluster list changes when region switches; S3 bucket list does not).

### Stage 5: Navigation Overlays (profile and region switching)

13. `internal/ui/profile_picker.go` — Profile overlay: reads `~/.aws/config`, presents list.
14. `internal/ui/region_picker.go` — Region overlay: list of AWS regions.
15. Root Model wiring: handle ProfileSelectedMsg / RegionSelectedMsg, recreate sessions and controllers.

**Gate:** Switch profiles and regions mid-session without restart. ECS list changes; S3 list stays the same (confirms global vs regional behavior is correctly handled).

### Stage 6: Polish and Config File

16. `~/.duchess/config` file support — flesh out what Stage 1 left as stubs.
17. Configurable refresh intervals per service (flags + config file).
18. Vim-style keybinding audit across all panels (j/k/g/G/`/`/Esc).
19. Error state rendering — network errors, permission errors, expired SSO tokens shown in-panel (not crashing).
20. Object metadata detail view (size, last modified, storage class, ETag).

**Gate:** Feature-complete v1 as defined in PROJECT.md requirements checklist.

---

## Key Architectural Decisions

| Decision | Rationale |
|----------|-----------|
| Msg-as-snapshot (no separate shared store) | Simpler than a mutex-guarded store; Bubble Tea serializes all Msgs through Update(), so panel fields updated in Update() require no additional synchronization |
| `tea.Cmd` for user-triggered fetches, goroutine + `program.Send()` for background refresh | Cmd is natural for one-shot async; goroutine + Send is natural for ticker loops. Both produce Msgs that flow through Update() — consistent mental model |
| SessionFactory creates a fresh config on every switch | Avoids aws-sdk-go-v2 config mutation hazards; SSO token refresh, credential caching, and retry state all behave correctly on fresh configs |
| Interface per AWS service | Enables unit testing without real AWS credentials; mock implementations can be injected in tests |
| S3 is global; only ECS-and-below switches on region change | Matches AWS service reality; prevents user confusion when bucket list appears unchanged after region switch |
| Service-specific refresh intervals (not one global ticker) | Different services have fundamentally different change rates; one global interval is wrong for all of them |
| Initial fetch fires immediately on controller Start() | Eliminates blank-screen wait on launch; users see data before the first ticker interval elapses |

---

## Sources

- `go.mod` in duchess repo — confirms bubbletea v1.3.10, bubbles v0.21.0, lipgloss v1.1.0, aws-sdk-go-v2 v1.41.0 (HIGH confidence — direct file)
- `.planning/PROJECT.md` — confirms scope: S3 + ECS, read-only v1, profile/region switching, ~/.duchess/config (HIGH confidence — direct file)
- k9s source architecture (`github.com/derailed/k9s`) — dao, model, ui, view, config, client layer structure (MEDIUM confidence — training data; k9s is open source and widely studied; patterns are stable)
- Bubble Tea MVU pattern — Model interface, Cmd goroutine dispatch, program.Send() goroutine safety, composition via sub-models (`github.com/charmbracelet/bubbletea`) (HIGH confidence — stable API since v1.0, well-documented)
- aws-sdk-go-v2 `config.LoadDefaultConfig` with `WithSharedConfigProfile` + `WithRegion` — standard session creation pattern (HIGH confidence — official SDK, extensively documented, stable)
- S3 global service behavior — `ListBuckets` returns all buckets regardless of region; bucket location via `GetBucketLocation` (HIGH confidence — stable AWS behavior, AWS docs)
