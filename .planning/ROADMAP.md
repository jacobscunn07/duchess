# Roadmap: duchess

## Overview

duchess is built in five phases that follow the natural dependency graph of the system. Foundation proves AWS authentication works across all three profile types before any UI is written. The UI shell establishes the Bubble Tea program structure and persistent status bar. S3 browsing validates the Controller -> Panel architecture on the simpler global service. ECS browsing proves the pattern generalizes to regional services with deeper hierarchy. Profile and region switching overlays complete the core UX — live account and region switching that validates global-vs-regional behavior across real panels.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation** - Working Go binary that authenticates across all three AWS profile types, loads config, and returns caller identity from STS (completed 2026-03-03)
- [x] **Phase 2: UI Shell** - Bubble Tea program with persistent status bar, loading states, and clean terminal handling — no resource panels yet (completed 2026-03-04)
- [x] **Phase 3: S3 Browsing** - Full S3 navigation from bucket list through prefix hierarchy to object metadata, with live refresh and filter (completed 2026-03-04)
- [x] **Phase 4: ECS Browsing** - Full ECS navigation from cluster list through service detail to task detail, with lazy hierarchical fetch and live refresh (completed 2026-03-05)
- [ ] **Phase 5: Profile and Region Switching** - In-session profile and region switching overlays that correctly isolate global vs. regional service behavior

## Phase Details

### Phase 1: Foundation
**Goal**: A working Go binary that authenticates with AWS using named profiles, role assumption, and SSO profiles, loads configuration from file and CLI flags, and can retrieve caller identity from STS
**Depends on**: Nothing (first phase)
**Requirements**: AUTH-02, AUTH-03, AUTH-04, AUTH-06, CONF-01, CONF-02, CONF-03, CONF-04, CONF-05, CONF-06
**Success Criteria** (what must be TRUE):
  1. Running `duchess --profile my-profile` authenticates with a named access key profile and prints caller identity (account ID + ARN) to stdout without error
  2. Running with a role assumption profile (`role_arn + source_profile`) successfully assumes the role and returns the assumed role's caller identity
  3. Running with an SSO profile returns the SSO session's caller identity; when the SSO token is expired, the binary exits with a message containing the exact `aws sso login --profile <name>` command to run
  4. `~/.duchess/config` is loaded on startup; CLI flags (`--profile`, `--region`, `--refresh-interval`) override file values
  5. Missing or invalid credentials produce a human-readable error message (not a stack trace) with a concrete remediation step
**Plans**: 2 plans

Plans:
- [x] 01-01-PLAN.md — cobra CLI skeleton + yaml config loading with flag-over-file precedence
- [x] 01-02-PLAN.md — AWS session factory (named/role/SSO) + STS GetCallerIdentity + credential error taxonomy

### Phase 2: UI Shell
**Goal**: A Bubble Tea TUI that launches cleanly, displays real AWS identity in a persistent status bar, handles terminal resize, and exits gracefully — no resource panels yet
**Depends on**: Phase 1
**Requirements**: STAT-01, STAT-02, STAT-03, STAT-04, STAT-05, STAT-06, NAV-04, NAV-06, NAV-07
**Success Criteria** (what must be TRUE):
  1. The TUI launches and shows a status bar containing the app version, current profile name, current region, IAM principal (account ID + role/user ARN), and a live clock
  2. The status bar updates immediately when profile or region changes (verified in later phase; structure must exist here)
  3. Pressing `q` from any screen quits the application cleanly with no error output
  4. A loading spinner is visible while the initial AWS identity fetch is in progress
  5. A simulated API error (injected via test) renders as inline text in the affected view area without crashing the program
**Plans**: 2 plans

Plans:
- [x] 02-01-PLAN.md — Upgrade Charmbracelet stack to v1.x; create internal/ui package with rootModel scaffold, message types, WindowSizeMsg, NO_COLOR guard, q-to-quit
- [x] 02-02-PLAN.md — Status bar component (lipgloss, left/right layout); async STS identity as tea.Cmd; loading spinner; inline error rendering; wire tea.NewProgram into cmd/root.go

### Phase 3: S3 Browsing
**Goal**: Full S3 navigation — bucket list to prefix hierarchy to object metadata — with live refresh, breadcrumb header, prefix filter, and paginated API calls that never truncate results
**Depends on**: Phase 2
**Requirements**: S3-01, S3-02, S3-03, S3-04, S3-05, NAV-01, NAV-02, NAV-03, NAV-05
**Success Criteria** (what must be TRUE):
  1. User sees a complete list of all S3 buckets accessible by the current profile, including buckets in any region
  2. User can press Enter on a bucket to navigate into it and see object prefixes (folder metaphor); pressing Esc returns to the bucket list; nested prefixes navigate recursively
  3. User can press Enter on an object to see its metadata (key, size, last modified, storage class) in a detail panel or inline row
  4. Pressing `/` enters filter mode and narrows the visible objects using a server-side ListObjectsV2 prefix — the filter does not fetch more items than the current prefix contains
  5. A breadcrumb header (e.g., `S3 > my-bucket > logs/`) updates correctly at every level; `j`/`k` scroll rows, `g`/`G` jump to top/bottom throughout all S3 views
**Plans**: 3 plans

Plans:
- [x] 03-01-PLAN.md — S3 client package (ListBuckets, HeadBucket, ListObjectsV2 paginated) + typed tea.Cmd message types; S3 SDK added to go.mod
- [x] 03-02-PLAN.md — S3 panel model (BucketList/PrefixList/ObjectDetail state machine), custom list delegate, breadcrumb header, filter mode, 30s refresh ticker; wire s3Panel into rootModel
- [x] 03-03-PLAN.md — Gap closure: widen / filter guard to panelBucketList; fix enter-to-apply to branch on state (client-side substring for bucket list, FetchPrefixCmd for prefix list)

### Phase 4: ECS Browsing
**Goal**: Full ECS navigation — cluster list to service detail to task detail — with lazy hierarchical fetch, per-level refresh intervals, and full pagination that prevents API throttling
**Depends on**: Phase 3
**Requirements**: ECS-01, ECS-02, ECS-03, ECS-04, ECS-05, ECS-06
**Success Criteria** (what must be TRUE):
  1. User sees a list of all ECS clusters in the current region; switching region updates the cluster list
  2. User can press Enter on a cluster to view its services, each showing desired count, running count, pending count, launch type, and task definition
  3. User can press Enter on a service to view its running tasks; each task row shows task ID, status, started-at time, and container name/image/status
  4. Pressing `L` on a task container opens the CloudWatch Logs console URL for that container in the system browser
  5. The app does not request ECS services or task data until the user navigates into the relevant level (lazy fetch); no ThrottlingException errors appear during normal use in a large account
**Plans**: 2 plans

Plans:
- [x] 04-01-PLAN.md — ECS client package: messages.go, delegate.go, client.go; promote ECS SDK to direct dependency; all paginated API calls (ListClusters/DescribeClusters, ListServices/DescribeServices batched-10, ListTasks/DescribeTasks batched-100, DescribeTaskDefinition); three typed refresh tick messages; ecsDelegate two-column row rendering
- [x] 04-02-PLAN.md — ECS panel model (ClusterList -> ServiceList -> TaskList -> TaskDetail state machine); value-receiver Model, ascend/descend helpers, breadcrumb rendering, TaskDetail pane with navigable container list and L-key CloudWatch Logs URL opener; wire ecsPanel into rootModel with Tab-key panel switching

### Phase 4.1: Wire Configurable Refresh Interval (INSERTED — gap closure)
**Goal**: Close CONF-01 and CONF-02 by threading cfg.RefreshInterval through NewRootModel into all panel constructors and replacing all five hardcoded tick durations with the configured value so that --refresh-interval and the config file key have an observable runtime effect
**Depends on**: Phase 4
**Requirements**: CONF-01, CONF-02
**Gap Closure**: Closes gaps from v1.0 milestone audit
**Success Criteria** (what must be TRUE):
  1. Running `duchess --profile X --refresh-interval 5s` causes the S3 and ECS panels to visibly refresh at approximately 5-second intervals (not the hardcoded 30s/10s defaults)
  2. Setting `refresh_interval: 10s` in `~/.duchess/config` produces the same effect without a CLI flag
  3. `cfg.RefreshInterval` is the sole source of truth for all tick durations; no hardcoded `30*time.Second`, `10*time.Second`, or `5*time.Second` literals remain in production code
**Plans**: 2 plans

Plans:
- [x] 04.1-01-PLAN.md — Thread cfg.RefreshInterval through NewRootModel into s3panel.NewModel and ecspanel.NewModel; parameterize all 4 tick cmd functions; fix all 12 call sites; no hardcoded durations remain
- [ ] 04.1-02-PLAN.md — Gap closure (UAT): add refresh interval indicator to status bar left group; add last-refreshed timestamp to S3 and ECS breadcrumbs so the active interval is observable to the user

### Phase 5: Profile and Region Switching
**Goal**: In-session AWS profile and region switching via modal overlays, with immediate status bar update, credential error surfacing, and correct isolation of global vs. regional service panel state
**Depends on**: Phase 4.1
**Requirements**: AUTH-01, AUTH-05
**Success Criteria** (what must be TRUE):
  1. User can open a profile picker overlay (from any screen), select a different profile, and see the status bar update immediately with the new profile name and IAM principal — without restarting the app
  2. User can open a region picker overlay, select a different region, and see the ECS cluster list refresh with clusters from the new region; the S3 bucket list remains unchanged (S3 is global)
  3. When a switched-to profile has an expired SSO token, the S3 and ECS panels display an inline error containing the `aws sso login --profile <name>` command rather than blank content or a crash
  4. Profile names are read directly from `~/.aws/config` via ini.v1 (no SDK enumeration API) and display correctly without the "profile " prefix
**Plans**: TBD

Plans:
- [ ] 05-01: Implement profile enumeration via ini.v1 parsing of ~/.aws/config (strip "profile " prefix); implement profile picker modal (tea.Model overlay); wire ProfileSelectedMsg to root model to rebuild AWS session and cancel in-flight goroutines via context cancellation; refresh STS identity and propagate to status bar
- [ ] 05-02: Implement region picker modal with AWS region list; wire RegionSelectedMsg to root model; reset ECS panel state on region change; confirm S3 panel does not reset; surface InvalidTokenError and role-chain expiry as specific inline error messages distinct from generic API failures

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 4.1 → 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation | 2/2 | Complete   | 2026-03-03 |
| 2. UI Shell | 2/2 | Complete   | 2026-03-03 |
| 3. S3 Browsing | 3/3 | Complete   | 2026-03-04 |
| 4. ECS Browsing | 2/2 | Complete   | 2026-03-05 |
| 4.1. Wire Configurable Refresh Interval | 1/2 | In progress | - |
| 5. Profile and Region Switching | 0/2 | Not started | - |
