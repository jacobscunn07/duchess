# Feature Landscape

**Domain:** AWS TUI — k9s-inspired terminal navigator for S3 and ECS
**Researched:** 2026-03-02
**Confidence note:** Web search and WebFetch tools were unavailable during this research session. Findings are drawn from deep domain knowledge of k9s (widely studied open-source project), AWS CLI ecosystem tooling (awsume, Leapp, granted, aws-console, s3-browser, ecs tooling), and the project's own go.mod which confirms the Bubble Tea / Bubbles / Lipgloss stack. Confidence ratings reflect this limitation.

---

## Table Stakes

Features users expect. Missing = product feels incomplete or untrustworthy.

| Feature | Why Expected | Complexity | Confidence | Notes |
|---------|--------------|------------|------------|-------|
| Vim-style keybindings (j/k/g/G/q/Esc) | k9s established this as the TUI standard; engineers expect it | Low | HIGH | j/k scroll rows, g/G jump to top/bottom, q quit, Esc navigate back one level |
| Persistent status bar | Every serious TUI shows current context so users know where they are | Low | HIGH | Must show: profile, region, IAM identity, clock. Users are burned by acting in wrong account |
| AWS profile switching | Engineers have 5-20 profiles; switching without restarting is baseline expectation | Medium | HIGH | Must read ~/.aws/config; support named, role-assumption, and SSO profiles |
| AWS region switching | ECS is regional; users must be able to switch region in-session | Low | HIGH | S3 is global — region switch only affects regional services (ECS). This distinction must be communicated in the UI |
| IAM identity display (who am I?) | Critical safety feature — users need to know which account/role they are in before they trust any data | Medium | HIGH | Call STS GetCallerIdentity; display Account ID + ARN (or friendly role name). Stale display is worse than no display — must refresh on profile switch |
| Help overlay (? key) | k9s standard; users discover keybindings through ? not docs | Low | HIGH | Context-sensitive: show keybindings relevant to current view. Static global help is a fallback but inferior |
| Breadcrumb / location indicator | Users navigating S3 prefix hierarchies get lost without it | Low | HIGH | "S3 > my-bucket > logs/2024/" style header. ECS equivalent: "ECS > us-east-1 > prod-cluster > api-service" |
| S3 bucket list | Core feature; every S3 tool starts here | Low | HIGH | Show: bucket name, region, creation date. Sorted alphabetically by default |
| S3 prefix (folder) navigation | S3 is flat but users expect folder metaphor; all S3 tools implement this | Medium | HIGH | List common prefixes as "directories"; Enter to descend, Esc to ascend. Show item count or size if feasible |
| S3 object metadata view | Users browsing S3 want to know size, last modified, storage class without leaving TUI | Low | HIGH | Show in detail panel or secondary column: key, size (human-readable), last modified, storage class, ETag |
| ECS cluster list | Core feature for ECS browsing | Low | HIGH | Show: cluster name, status, registered container instances, running/pending tasks |
| ECS service drill-down | Engineers navigating ECS always drill: cluster → service → task | Medium | HIGH | Show per service: desired count, running count, pending count, launch type (Fargate/EC2), task definition |
| ECS task detail view | Task-level detail is the most common debugging destination in ECS | Medium | HIGH | Show: task ID, task definition, status, started at, containers (name, image, status, exit code if stopped) |
| Auto-refresh | Live infrastructure data must not go stale; users expect near-real-time view | Medium | HIGH | Default 2s interval (configurable). Must not disrupt cursor position or scroll on refresh |
| Error display (not crash) | API errors — expired credentials, no permissions — must be shown gracefully | Medium | HIGH | Show error inline in the view that failed; do not crash the whole app. Expired SSO token is the most common case |
| Loading states | AWS API calls take 200ms-2s; blank screen during load erodes trust | Low | HIGH | Spinner or "Loading..." text while first fetch is in-flight. Subsequent refreshes can be silent |

---

## Differentiators

Features that set duchess apart. Not universally expected, but materially increase value.

| Feature | Value Proposition | Complexity | Confidence | Notes |
|---------|-------------------|------------|------------|-------|
| Multi-profile instant switching | Competitors require restart to switch profiles; duchess makes it instant and obvious | Medium | HIGH | The profile switcher (modal or sidebar) is a first-class UX element — not buried in a menu. This is the core value prop of the tool |
| SSO profile support with graceful re-auth prompting | awsume and granted do SSO well, but TUI tools often fall back to "just use CLI first"; duchess should handle expired SSO tokens and guide the user to re-authenticate | High | MEDIUM | Display a clear message with the `aws sso login --profile <name>` command to run. Do not attempt to open browser from within TUI (too fragile) |
| Role assumption chain display | Engineers using assumed roles often lose track of the trust chain; showing source_profile → role_arn in the identity panel adds trust | Medium | MEDIUM | Useful for compliance-conscious teams; surfaced in the status bar or identity detail overlay |
| S3 search / filter within bucket | Buckets with millions of objects are unusable without filter; prefix-based filter in the TUI | Medium | HIGH | / key to enter filter mode; filter applies to current prefix level (server-side prefix filter via ListObjectsV2 Prefix param) |
| ECS container log hint | "Press L to open logs in CloudWatch console" — not fetching logs (out of scope) but linking to the right place | Low | MEDIUM | Open URL in browser using open/xdg-open; construct the CloudWatch Logs Insights URL from log group and task ID |
| S3 object count and total size per prefix | S3 console shows this; the CLI does not easily; a TUI showing "this prefix: 4,231 objects, 12.4 GB" is genuinely useful | High | MEDIUM | Requires recursive ListObjectsV2 — expensive for large buckets. Should be on-demand (press S to compute size) not automatic |
| Copy ARN / resource identifier to clipboard | Engineers constantly copy ARNs for use in IAM policies or other CLI commands | Low | HIGH | c or y key to yank the ARN of the selected resource to clipboard. go.mod already includes github.com/atotto/clipboard — this is essentially free to implement |

---

## Anti-Features

Features to explicitly NOT build in v1. Each has a deliberate reason.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| S3 upload / download / delete | Write operations require IAM write permissions; open-source users should trust the tool is safe before granting that access | Display object metadata only; show "write operations in v2" if user presses Delete |
| ECS exec (container shell) | Requires ECS Exec IAM permissions, SSM agent in container, and interactive PTY — significant complexity and security surface | Show task detail; consider linking to AWS console ECS Exec session in v2 |
| ECS task stop / restart | Mutation; breaks the read-only contract that builds user trust | Show task status; message "mutations in v2" |
| MFA token entry | Fragile to implement; modern SSO-based setups make it largely unnecessary; adds significant auth complexity | Support SSO profiles; document MFA workaround via aws-vault or similar |
| CloudWatch Logs viewer | Fetching and streaming logs is a separate product (dedicated tools: cwtail, saw); pulling logs into duchess bloats scope and requires log-specific pagination/streaming logic | Provide a "open in console" URL hint for task logs |
| EC2 browser | EC2 instance management is a different domain with much higher blast radius for mistakes; keeps v1 focused | Defer to v2 as a separate service module |
| Lambda browser | Same rationale as EC2 — separate domain, out of v1 scope | Defer to v2 |
| Multi-region simultaneous view | Aggregating resources across all regions requires N parallel API calls and complex table merging; S3 is already global so this mainly applies to ECS | Let users switch regions and view one region at a time; add region aggregate view in v2 if demanded |
| Config UI inside TUI | Editing duchess config from within the TUI is over-engineered for v1; config file is simple YAML/TOML | Document config file format; edit with any text editor |
| Mouse support | Adds complexity; the target user (vim-native engineers) does not expect or want mouse navigation in a TUI | Keyboard-only; document this as intentional |

---

## Feature Dependencies

```
Profile switching → IAM identity display
  (switching profile must re-call STS GetCallerIdentity and update status bar)

IAM identity display → STS integration
  (requires aws-sdk-go-v2/service/sts, already in go.mod)

S3 prefix navigation → Breadcrumb display
  (cannot navigate prefixes without showing where you are)

S3 search/filter → S3 prefix navigation
  (filter operates within a prefix context)

S3 object size-per-prefix → S3 prefix navigation
  (on-demand compute, but requires being inside a prefix view)

ECS task detail → ECS service drill-down → ECS cluster list
  (strict hierarchy: must enter cluster before service before task)

ECS container log hint → ECS task detail
  (log group only available at task/container level)

Copy ARN to clipboard → Any resource list view
  (applicable to S3 buckets, ECS clusters, services, tasks)

Auto-refresh → All list views
  (refresh must work at every level of the hierarchy without losing cursor position)

Help overlay → All views
  (context-sensitive help requires each view to register its keybindings)

Error display → All AWS API calls
  (every API call can fail; error handling is a cross-cutting concern)

SSO re-auth prompting → Profile switching
  (SSO token expiry is detected during profile switch or first API call after switch)
```

---

## MVP Recommendation

**Build in v1 (launch with these or do not launch):**

1. Persistent status bar — profile, region, IAM identity, clock
2. Profile switching (named, role-assumption, SSO)
3. Region switching
4. Vim-style keybindings throughout
5. Help overlay (? key, context-sensitive per view)
6. S3 bucket list → prefix navigation → object metadata
7. ECS cluster list → service detail → task detail
8. Auto-refresh (default 2s, configurable)
9. Graceful error display (expired credentials, no permissions)
10. Loading states during API calls
11. Breadcrumb header in all views

**High-value, low-cost additions for v1 launch:**

- Copy ARN/key to clipboard (c or y key) — clipboard lib already in go.mod, trivial to add
- S3 prefix filter (/ key) — users with large buckets will hit this immediately; without it, S3 browsing is painful for real workloads

**Defer with clear v2 rationale:**

- S3 object size-per-prefix: on-demand compute is expensive for large buckets; design the architecture to support it but do not build it in v1
- ECS container log hint: useful but not blocking; add after core ECS navigation is solid
- SSO re-auth prompting: show error message with CLI command to run; full re-auth flow is v2

---

## Existing Tool Landscape

Confidence: MEDIUM — from training data, not verified live due to tool unavailability.

| Tool | Focus | What It Does Well | Gap duchess fills |
|------|-------|-------------------|------------------|
| k9s | Kubernetes | Navigation model, keybindings, live refresh, help system | AWS equivalent does not exist at this quality level |
| awsume | Auth | Profile switching, role assumption, SSO token refresh | Terminal-only, no resource browsing; duchess uses same profile system |
| granted | Auth | Multi-account SSO, profile management, browser console handoff | No TUI resource browsing; complementary to duchess |
| Leapp | Auth | GUI-based credential management, MFA, SSO | Heavy GUI app; terminal engineers prefer lighter tooling |
| aws-console (various) | Console | Opens AWS console in browser for a profile | No in-terminal resource viewing |
| s3cmd / aws s3 ls | S3 CLI | Scriptable S3 access | No interactive navigation; no ECS; no profile-aware TUI |
| s3-tui (various small tools) | S3 | Minimal S3 TUI implementations | Limited maintenance, narrow scope, no ECS, no profile switching |
| ecs-cli | ECS | Task definitions, cluster management | Deprecated by AWS; not a TUI |

**The gap:** No well-maintained, k9s-quality TUI exists for AWS that covers multi-profile navigation, S3, and ECS in one tool. duchess targets exactly this gap.

---

## Sources

- Project context: `.planning/PROJECT.md`
- Stack evidence: `go.mod` (confirms Bubble Tea, Bubbles, Lipgloss, AWS SDK v2, atotto/clipboard)
- k9s navigation model: Training data (HIGH confidence — k9s is extensively documented and widely studied open-source project)
- AWS tool ecosystem: Training data (MEDIUM confidence — tool landscape changes; awsume/granted/Leapp are well-established as of knowledge cutoff August 2025)
- S3/ECS API behavior: Training data (HIGH confidence — AWS API behavior is stable and well-documented)
- Note: Web search and WebFetch tools were unavailable during this session; recommend verifying tool landscape currency before finalizing roadmap
