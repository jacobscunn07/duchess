# Project Research Summary

**Project:** duchess — k9s-inspired AWS TUI in Go
**Domain:** Terminal User Interface / AWS Resource Navigator
**Researched:** 2026-03-02
**Confidence:** HIGH

## Executive Summary

duchess is a k9s-style TUI for AWS written in Go, focused on read-only S3 and ECS navigation with multi-profile switching. The right way to build this is well-established: Bubble Tea (Model/Update/View) as the TUI framework, aws-sdk-go-v2 for all AWS API calls, and a strict four-layer architecture (UI -> Controller -> Store/Msg -> Client) that mirrors how k9s is internally structured. The project already has a working go.mod with most of these dependencies, which removes significant setup risk. The main engineering decision is committing to the Bubble Tea MVU pattern throughout — self-contained panel models, typed async messages, and background goroutines only via `tea.Cmd` and `program.Send()`. This architecture pays dividends as services are added.

The single biggest risk area is AWS credential handling. Three distinct failure modes — SSO token expiry, role assumption chain 1-hour hard cap, and misconfigured profiles — all produce errors that without deliberate handling appear as cryptic failures or silent empty states. Each must be caught and surfaced as specific, user-actionable messages. The second major risk is API correctness: both S3 and ECS silently truncate results without errors when pagination is not handled correctly, and S3 requires per-bucket regional client construction (not one global client) or object listing will return 301/400 errors for buckets outside the default region.

The tool fills a genuine gap in the AWS tooling ecosystem. k9s-quality TUIs do not exist for AWS S3 and ECS navigation with multi-profile support. The v1 scope (read-only S3 + ECS + profile/region switching) is well-defined, achievable, and delivers immediately on the gap. The phasing that naturally emerges from the architecture research — foundation first, then UI shell, then S3 (proving the pattern), then ECS (proving generalization), then navigation overlays, then polish — maps cleanly to the dependency graph and minimizes rework.

## Key Findings

### Recommended Stack

The stack builds on what is already in go.mod. Bubble Tea v1.3.10 is already pinned at the correct version. Bubbles needs upgrading from v0.21.0 to v1.0.0 (stable release Feb 2026 — critical upgrade since v1 stabilizes the API). The AWS SDK requires the ECS service package to be added entirely (`aws-sdk-go-v2/service/ecs@v1.73.0`) and the S3 service package has a significant version gap (v1.58.3 → v1.96.2). Cobra needs to be added for CLI flags. YAML config is already available as an indirect dep and should be promoted to direct. No new TUI or AWS auth libraries are needed.

**Core technologies:**
- `charmbracelet/bubbletea v1.3.10`: TUI event loop — Elm Architecture maps cleanly to concurrent AWS polling
- `charmbracelet/bubbles v1.0.0`: Pre-built components (List, Table, Viewport, Spinner) — upgrade from v0.21.0 immediately
- `charmbracelet/lipgloss v1.1.0`: Terminal styling — already at correct version
- `aws-sdk-go-v2`: AWS SDK — v2 only; v1 is maintenance-mode with different API conventions
- `aws-sdk-go-v2/config`: Profile loading, SSO, role assumption — `LoadDefaultConfig` handles all three auth types transparently
- `aws-sdk-go-v2/service/ecs v1.73.0`: ECS cluster/service/task listing — not yet in go.mod; must be added
- `aws-sdk-go-v2/service/sts`: GetCallerIdentity for status bar identity display
- `gopkg.in/ini.v1`: Profile enumeration from `~/.aws/config` — AWS SDK has no public API to list profiles; ini.v1 already in go.mod
- `spf13/cobra v1.10.2`: CLI flags and help — standard for Go infrastructure tooling; add to go.mod
- `gopkg.in/yaml.v3 v3.0.1`: Config file parsing — promote from indirect to direct dependency

See `/Users/jacobcunningham/_CODE/duchess/.planning/research/STACK.md` for full version table and go.mod upgrade instructions.

### Expected Features

The feature set has a clear v1 core and explicit anti-features that preserve trust and focus.

**Must have (table stakes) — do not launch without these:**
- Persistent status bar: profile, region, IAM identity, clock — users burned by acting in wrong account
- AWS profile switching (named, role-assumption, SSO) without restarting
- AWS region switching in-session (S3 is global; communicate this distinction in UI)
- IAM identity display via STS GetCallerIdentity — refresh on every profile switch
- Vim-style keybindings throughout (j/k/g/G/q/Esc)
- Context-sensitive help overlay (? key)
- Breadcrumb header in all views (S3 > bucket > prefix/, ECS > region > cluster > service)
- S3 bucket list → prefix navigation → object metadata
- ECS cluster list → service detail → task detail
- Auto-refresh (configurable interval, must not disrupt cursor position)
- Graceful error display for API failures — do not crash, show inline error
- Loading states during first fetch (spinner/loading text)

**Should have (high-value, low-cost additions for v1):**
- Copy ARN/resource key to clipboard (c or y key) — clipboard lib already in go.mod, essentially free
- S3 prefix filter (/ key, server-side prefix filter) — users with real workloads hit this immediately

**Defer to v2 with clear rationale:**
- S3 object size-per-prefix (on-demand compute, expensive for large buckets — design for it, don't build it)
- ECS container log hint (link to CloudWatch console URL — useful but not blocking v1)
- SSO re-auth flow within TUI (show error + `aws sso login --profile <name>` command; full flow is v2)
- S3 upload/download/delete, ECS exec, ECS task stop — all write operations (read-only contract builds trust first)
- EC2, Lambda, CloudWatch Logs browsing (separate domains, out of v1 scope)
- Mouse support (target user is vim-native; keyboard-only is intentional)

See `/Users/jacobcunningham/_CODE/duchess/.planning/research/FEATURES.md` for full feature table with confidence levels.

### Architecture Approach

duchess uses a strict four-layer architecture with one-way dependencies: UI -> Controller -> Client, with data returning as typed messages via Bubble Tea's message system. Each resource panel (S3, ECS) is a self-contained `tea.Model` with its own navigation state machine. The Root Model composes panels and routes messages down; panels never talk to each other directly. Background refresh runs via goroutines that push typed snapshots into the Bubble Tea event loop using `program.Send()` (goroutine-safe). User-triggered fetches (drill-down navigation) use `tea.Cmd`. AWS API clients are defined as Go interfaces and implemented against aws-sdk-go-v2, enabling unit testing without real credentials.

**Major components:**
1. `internal/config/config.go` — Config struct + Load(); file config + CLI flags merged (flags win); YAML format
2. `internal/aws/session.go` — SessionFactory; produces fresh `aws.Config` per profile+region; never mutates shared config
3. `internal/aws/{s3,ecs,sts}.go` — Typed interfaces + implementations; pagination handled internally; returns complete slices to callers
4. `internal/controller/{s3,ecs}.go` — Refresh tickers; goroutine-safe `program.Send()`; service-specific intervals (S3: 30s, ECS clusters/services: 10s, ECS tasks: 5s)
5. `internal/ui/root.go` — Root `tea.Model`; composes panels; routes all messages; handles ProfileSelectedMsg/RegionSelectedMsg
6. `internal/ui/statusbar.go` — Renders profile/region/principal/time; updated on every identity change
7. `internal/ui/s3/panel.go` — S3 navigation state machine (BucketList → PrefixList → ObjectList)
8. `internal/ui/ecs/panel.go` — ECS navigation state machine (ClusterList → ServiceList → TaskList)
9. `internal/ui/{profile_picker,region_picker}.go` — Modal overlays; profile list via ini.v1 parsing of `~/.aws/config`

See `/Users/jacobcunningham/_CODE/duchess/.planning/research/ARCHITECTURE.md` for full component map, data flow diagrams, and code patterns.

### Critical Pitfalls

The research identified 8 critical pitfalls and 5 moderate pitfalls. The top 5 that would cause the most damage if not addressed in the right phase:

1. **S3 and ECS pagination not implemented from day one** — Both S3 (`ListObjectsV2` at 1,000/page) and ECS (`ListServices`/`ListTasks` at 100/page) silently truncate results with no error. Use SDK paginators (`s3.NewListObjectsV2Paginator`, manual `nextToken` loop for ECS) from the first implementation. Never add pagination "later."

2. **S3 per-bucket regional client not constructed** — S3 is global for `ListBuckets` but bucket operations require the bucket's home region. A `us-east-1` client calling `ListObjectsV2` on an `eu-west-1` bucket returns HTTP 301 or 400. Use `HeadBucket` to detect bucket region before listing objects; create a region-specific client. Also: `GetBucketLocation` returns `null` for `us-east-1` buckets — map null to "us-east-1" explicitly.

3. **SSO token expiry not distinctly handled** — `ssocreds.InvalidTokenError` must be caught specifically, the refresh loop must stop (not retry), and a user-readable message with the exact `aws sso login --profile <name>` command must be displayed. Generic error handling produces cryptic output and retry loops that waste CPU.

4. **Goroutine leaks from raw `go func()` in refresh loops** — Only use `tea.Cmd` and `program.Send()` for async operations; never spawn raw goroutines in fetch paths. Pass context to every AWS SDK call; cancel context on profile switch. Raw goroutines become leaks when the user navigates away or switches profiles.

5. **ECS API throttling from fetching full hierarchy each tick** — Fetch only what is visible on screen (lazy hierarchical fetching: clusters view fetches clusters only, not services and tasks). Apply exponential backoff on `ThrottlingException`. Use `retry.NewAdaptiveMode()` in the SDK config. Default refresh interval should be 5s, not 2s, for large accounts.

See `/Users/jacobcunningham/_CODE/duchess/.planning/research/PITFALLS.md` for full pitfall catalog with detection and phase-specific warnings.

## Implications for Roadmap

The architecture research provides an explicit build order. The phase structure below is derived directly from the dependency graph in ARCHITECTURE.md and maps cleanly to the pitfall phase warnings.

### Phase 1: Foundation
**Rationale:** Every other component depends on config loading and AWS session creation. Nothing else starts without this. Auth correctness must be proven before any UI work begins — a broken credential chain makes every subsequent test unreliable.
**Delivers:** Working Go binary that authenticates to AWS across all three profile types (named, role assumption, SSO) and returns caller identity from STS. Proves auth works before any UI is built.
**Addresses:** Status bar identity (STS GetCallerIdentity), config file + CLI flag merge, YAML config
**Avoids:** Chained role assumption 1-hour expiry (design around SDK credential providers from start); missing region configuration (validate region is non-empty after LoadDefaultConfig)
**Stack elements:** aws-sdk-go-v2/config, aws-sdk-go-v2/service/sts, yaml.v3, cobra
**Needs research:** No — well-documented SDK patterns. Standard patterns apply.

### Phase 2: UI Shell
**Rationale:** Root Model and status bar must exist before any resource panels can be wired in. This phase proves the Bubble Tea program structure works and that real AWS identity appears correctly in the status bar before adding resource-specific complexity.
**Delivers:** TUI that launches cleanly, shows real AWS identity in status bar, handles terminal resize, respects NO_COLOR, and quits on q.
**Addresses:** Persistent status bar (profile/region/IAM identity/clock), loading states, vim-style quit
**Avoids:** WindowSizeMsg not handled (handle from the start); NO_COLOR not respected (use lipgloss EnvColorProfile from first commit); GetCallerIdentity blocking render (async Cmd fetch)
**Stack elements:** bubbletea, bubbles, lipgloss
**Needs research:** No — established Bubble Tea patterns.

### Phase 3: S3 Browsing
**Rationale:** S3 is the simpler of the two services (global, no regional service-client matrix), making it the right first proof of the Controller -> Msg -> Panel pattern. Getting S3 right validates the architecture for all subsequent services.
**Delivers:** Full S3 navigation (bucket list -> prefix navigation -> object metadata) with live refresh, breadcrumb header, filter, and clipboard copy.
**Addresses:** S3 bucket list, S3 prefix navigation, S3 object metadata, breadcrumb display, S3 prefix filter (/ key), copy ARN to clipboard
**Avoids:** S3 pagination bug (use NewListObjectsV2Paginator from day one); wrong-region client (HeadBucket before listing objects; handle null LocationConstraint); ListObjects v1 (use ListObjectsV2 exclusively)
**Stack elements:** aws-sdk-go-v2/service/s3, bubbles List/Table components
**Needs research:** No — pagination patterns and regional client construction are fully documented in PITFALLS.md.

### Phase 4: ECS Browsing
**Rationale:** ECS proves the pattern generalizes to regional services. It also introduces a deeper navigation hierarchy (cluster -> service -> task vs. bucket -> prefix) and the lazy hierarchical fetch pattern that prevents throttling.
**Delivers:** Full ECS navigation (cluster list -> service detail -> task detail) with live refresh, service-specific refresh intervals, and STOPPED task visibility.
**Addresses:** ECS cluster list, ECS service drill-down, ECS task detail, auto-refresh with per-service intervals
**Avoids:** ECS pagination (manual nextToken loop for ListServices/ListTasks from the start); throttle cascade (lazy hierarchical fetch — fetch only visible level); ECS ListTasks filtering STOPPED tasks (show both RUNNING and STOPPED)
**Stack elements:** aws-sdk-go-v2/service/ecs (add to go.mod)
**Needs research:** No — ECS pagination and throttle patterns fully documented in PITFALLS.md.

### Phase 5: Navigation Overlays (Profile and Region Switching)
**Rationale:** Profile and region switching must be built after the resource panels exist, because the critical validation is confirming that switching profile correctly resets resource panels and that S3 bucket list stays unchanged while ECS list changes (global vs. regional behavior). Building overlays without panels to test against would leave the most important behavior unvalidated.
**Delivers:** Profile picker overlay (reads ~/.aws/config via ini.v1), region picker overlay, mid-session profile/region switching without restart. ECS list changes on region switch; S3 list remains unchanged.
**Addresses:** AWS profile switching (table stakes), AWS region switching, IAM identity refresh on switch, SSO graceful re-auth prompting
**Avoids:** INI profile prefix rules (strip "profile " prefix; use SDK config package for loading, ini.v1 only for listing); SSO token expiry not handled (catch InvalidTokenError specifically, stop refresh loop, surface actionable message); S3 treated as regional service (confirm S3 panel does not reset on region switch)
**Stack elements:** gopkg.in/ini.v1, aws-sdk-go-v2/config WithSharedConfigProfile + WithRegion
**Needs research:** Possibly — SSO re-auth flow edge cases (expired token vs. network failure vs. role expiry) may benefit from research during planning.

### Phase 6: Polish and Config File
**Rationale:** Polish comes last because it depends on all panels being complete. Keybinding audit, error state rendering, and configurable refresh intervals require the full feature set to be in place before auditing for completeness.
**Delivers:** Feature-complete v1: full vim keybindings, graceful error display across all views, configurable refresh intervals, ASCII fallback for non-UTF-8 terminals, help overlay, object metadata detail view.
**Addresses:** Help overlay (context-sensitive, all views), graceful error display (inline, not crashing), configurable refresh intervals (file + flags), vim keybinding audit (j/k/g/G throughout)
**Avoids:** Box drawing chars in C locale (provide --no-unicode or ASCII fallback); profile name display bug (strip "profile " prefix consistently)
**Needs research:** No — UI polish patterns are standard.

### Phase Ordering Rationale

- Foundation before UI: AWS auth must work across all three profile types before any UI work begins. A broken credential chain makes every render test unreliable.
- S3 before ECS: S3 is global (simpler client management) and proves the Controller -> Panel pattern. ECS adds regional complexity and deeper hierarchy — better to validate the pattern on the simpler case first.
- Resource panels before navigation overlays: Profile switching must be tested against real resource panels to validate that S3 stays unchanged and ECS changes on region switch. This is the most important validation of global vs. regional service handling.
- Polish last: Keybinding audit and error state review require all panels to exist. Building polish incrementally during earlier phases is fine; the audit gate belongs at the end.

### Research Flags

Phases with standard patterns (skip research-phase during planning):
- **Phase 1 (Foundation):** Config loading, session creation, and STS GetCallerIdentity are thoroughly documented in official AWS SDK docs. Patterns are fully specified in STACK.md and ARCHITECTURE.md.
- **Phase 2 (UI Shell):** Bubble Tea Root Model structure and status bar patterns are standard. ARCHITECTURE.md provides the exact component structure.
- **Phase 3 (S3 Browsing):** S3 pagination and regional client construction are fully documented in PITFALLS.md with code examples.
- **Phase 4 (ECS Browsing):** ECS pagination, throttle mitigation, and task status filtering are fully documented in PITFALLS.md.
- **Phase 6 (Polish):** Standard UI refinement; no novel research needed.

Phases that may benefit from deeper research during planning:
- **Phase 5 (Navigation Overlays):** SSO token expiry detection and the distinction between SSO expiry vs. role chain expiry vs. network failure each have different recovery paths. The credential error taxonomy may warrant a focused research spike before implementation, especially if the codebase will attempt any re-auth prompting beyond a simple static message.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All versions verified against pkg.go.dev; go.mod confirms existing deps; cobra and ecs service package are the only additions needed |
| Features | HIGH | k9s navigation model is extensively documented; AWS API behavior is stable; tool landscape confidence is MEDIUM (training data only, web search unavailable during research) |
| Architecture | HIGH | Directly derived from Bubble Tea docs, k9s source structure, and aws-sdk-go-v2 official docs; all patterns verified |
| Pitfalls | HIGH | All 8 critical pitfalls sourced from official AWS API docs and SDK docs; pagination limits and throttle rates are documented AWS behavior |

**Overall confidence:** HIGH

### Gaps to Address

- **Tool landscape currency:** FEATURES.md research was conducted without web search access; the competitive landscape (awsume, granted, Leapp, s3 TUI tools) was assessed from training data (confidence MEDIUM). Before finalizing v1 scope, verify that no new well-maintained AWS TUI has shipped since August 2025 that would change the differentiation story. Low risk to the build plan — the architecture is sound regardless.
- **Config format (YAML vs TOML):** STACK.md rates YAML confidence as MEDIUM — the choice is best-fit for the AWS/Kubernetes audience but not definitively proven by external sources. Either works; YAML is the recommendation and should be committed to early.
- **SSO re-auth UX:** The precise UX for expired SSO tokens is intentionally punted to v2. During Phase 5 planning, the team should decide the exact error message format and whether to surface the `aws sso login` command inline in the TUI or via a dedicated error overlay. This does not affect architecture but affects user experience.
- **Bubbles v1.0.0 API migration:** The upgrade from bubbles v0.21.0 to v1.0.0 may include breaking API changes. This should be done in Phase 2 (UI Shell) before any panel code is written against the bubbles API, to avoid migrating panel code after the fact.

## Sources

### Primary (HIGH confidence)
- `go.mod` in duchess repo — confirms bubbletea v1.3.10, bubbles v0.21.0, lipgloss v1.1.0, aws-sdk-go-v2 v1.41.0, atotto/clipboard, ini.v1, yaml.v3, testify
- `pkg.go.dev/github.com/charmbracelet/bubbletea` — v1.3.10, Cmd/goroutine patterns, program.Send() goroutine safety
- `pkg.go.dev/github.com/charmbracelet/bubbles` — v1.0.0 stable (Feb 9, 2026)
- `pkg.go.dev/github.com/charmbracelet/lipgloss` — v1.1.0
- `pkg.go.dev/github.com/aws/aws-sdk-go-v2` — v1.41.2; LoadDefaultConfig, WithSharedConfigProfile, WithRegion
- `pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/ecs` — v1.73.0 (Feb 2026); ListClusters/Services/Tasks paginators
- `pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3` — v1.96.2; ListObjectsV2Paginator, ListBucketsPaginator
- `pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/sts` — v1.41.7; GetCallerIdentity
- `pkg.go.dev/github.com/aws/aws-sdk-go-v2/credentials/ssocreds` — InvalidTokenError, SSOTokenProvider
- `pkg.go.dev/github.com/spf13/cobra` — v1.10.2 (Dec 2025)
- `pkg.go.dev/gopkg.in/ini.v1` — v1.67.1 (Jan 2026); profile enumeration
- AWS S3 ListObjectsV2 API docs — 1,000/page pagination, IsTruncated/NextContinuationToken behavior
- AWS ECS ListServices/ListTasks API docs — 100/page pagination, nextToken loop requirement
- AWS ECS throttling limits docs — burst/sustained rate limits per API family
- AWS SSO credential docs — InvalidTokenError, token cache location, no-login-from-SDK constraint
- AWS S3 VirtualHosting docs — regional redirect behavior (301/400 for wrong-region client)
- AWS GetBucketLocation API docs — null LocationConstraint for us-east-1
- AWS IAM role chaining docs — 1-hour session hard cap for chained assumptions
- AWS API retry docs — adaptive retry mode, exponential backoff

### Secondary (MEDIUM confidence)
- k9s source architecture (github.com/derailed/k9s) — dao/model/ui/view/config/client layer mapping; training data, widely studied open source
- AWS tool ecosystem landscape (awsume, granted, Leapp, s3 TUI tools) — training data, knowledge cutoff August 2025; verify currency before finalizing v1 positioning

### Tertiary (LOW confidence)
- YAML vs TOML preference for AWS tooling audience — reasoned inference from ecosystem observation; not verified by survey or external source

---
*Research completed: 2026-03-02*
*Ready for roadmap: yes*
