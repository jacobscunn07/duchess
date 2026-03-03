# Domain Pitfalls

**Domain:** AWS TUI / Management Tool (Go + Bubble Tea)
**Researched:** 2026-03-02
**Confidence:** HIGH (AWS SDK docs, ECS throttling docs, official Go SDK pkg.go.dev)

---

## Critical Pitfalls

Mistakes that cause rewrites, data gaps, or outright broken behavior.

---

### Pitfall 1: S3 Pagination — Treating 1,000 Objects as "All Objects"

**What goes wrong:**
ListObjectsV2 and ListBuckets both paginate at 1,000 objects max per response. Tools that call the API once and render results silently show a truncated view. Users with large buckets see 1,000 keys and assume that's everything.

**Why it happens:**
The API returns `IsTruncated: true` and a `NextContinuationToken` but does not error. A developer who didn't read the docs carefully ships a working-looking feature that lies about completeness.

**Consequences:**
- Buckets with >1,000 objects appear to have exactly 1,000. No error. No warning.
- S3 ListBuckets also paginates — accounts with >1,000 buckets show a truncated bucket list.

**Prevention:**
Use the AWS SDK Go v2 built-in paginators. They handle this correctly:
```go
paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
    Bucket: aws.String(bucket),
    Prefix: aws.String(prefix),
})
for paginator.HasMorePages() {
    page, err := paginator.NextPage(ctx)
    // accumulate page.Contents
}
```
Never call `ListObjectsV2` directly and assume `Contents` is complete.

**Detection:**
- Test against a bucket you control that has >1,000 objects.
- Check: does your count match `aws s3 ls s3://bucket | wc -l`?

**Warning signs:**
- Object counts always end in round numbers (1,000, 2,000).
- No `NextContinuationToken` handling in the code path.

**Phase:** Implement pagination from day one in the S3 browsing phase. Do not add it "later."

---

### Pitfall 2: ECS Pagination — ListServices and ListTasks Are Also Paginated

**What goes wrong:**
ListClusters, ListServices, and ListTasks all paginate at 100 results max per response. A cluster with 150 services only shows 100. This is the same truncation problem as S3 but less obvious because most teams have fewer than 100 ECS services.

**Why it happens:**
ECS defaults look correct in development (small clusters). The bug only surfaces in prod-scale environments.

**Consequences:**
- Services silently missing from the UI.
- Tasks for large services truncated.
- `nextToken` is nil when all results fit in one page, so the code works until it doesn't.

**Prevention:**
Follow `nextToken` in a loop for every ECS list call:
```go
var nextToken *string
for {
    out, err := client.ListServices(ctx, &ecs.ListServicesInput{
        Cluster:    aws.String(clusterArn),
        MaxResults: aws.Int32(100),
        NextToken:  nextToken,
    })
    // accumulate out.ServiceArns
    if out.NextToken == nil {
        break
    }
    nextToken = out.NextToken
}
```

**Detection:**
- Test with a cluster that has >100 services.
- Compare your count vs `aws ecs list-services --cluster <name> | jq '.serviceArns | length'`.

**Warning signs:**
- No `nextToken` loop in ECS list calls.
- Service count always at or below 100.

**Phase:** ECS listing phase. Build pagination in from the start.

---

### Pitfall 3: AWS SSO Token Expiration — No Graceful User-Facing Error

**What goes wrong:**
SSO sessions have a configurable expiry (commonly 8-12 hours). When the cached token at `~/.aws/sso/cache/` expires, the AWS SDK returns `ssocreds.InvalidTokenError`. Tools that don't catch this specifically either crash with a cryptic error, silently show no data, or loop-retry on a non-retryable error.

**Why it happens:**
The SSO credential provider cannot initiate re-authentication. It can only report the token is invalid. The tool must translate this into a user-actionable message: "Run `aws sso login --profile <name>`."

**Consequences:**
- User sees generic "failed to load credentials" or a stack trace.
- Background refresh loops retry endlessly on a non-retryable auth failure, wasting CPU and producing no recovery.
- No path to resolution without understanding AWS CLI internals.

**Prevention:**
1. Detect `ssocreds.InvalidTokenError` specifically:
```go
var invalidToken *ssocreds.InvalidTokenError
if errors.As(err, &invalidToken) {
    // Display: "SSO session expired. Run: aws sso login --profile <profile>"
    // Stop background refresh for this profile
}
```
2. Stop the refresh loop on auth errors. Do not retry auth failures.
3. Surface a user-readable message with the exact CLI command to re-authenticate.
4. Distinguish SSO expiry from role assumption failure from network error — all three have different recovery paths.

**Detection:**
- Manually expire an SSO token by clearing `~/.aws/sso/cache/` and observe what the tool shows.
- Check for infinite retry loops after auth failure.

**Warning signs:**
- Credential errors in refresh loops treated identically to transient network errors.
- Generic `err.Error()` shown to user for any auth failure.

**Phase:** Authentication / profile switching phase. The SSO expiry path is the most common auth edge case for open source tools used across diverse AWS configurations.

---

### Pitfall 4: API Rate Throttling From 2-Second Auto-Refresh

**What goes wrong:**
ECS cluster-read operations share a burst bucket of 50 tokens and sustain at 20 requests/second. Service-read and task-read share a bucket of 100 burst / 20 sustained. A 2-second refresh that lists clusters, all services per cluster, and all tasks per service across a large account can easily exceed these limits — especially if the user is also running other tools against the same account.

**Why it happens:**
The refresh loop issues one API call per resource type per refresh cycle. With a deep hierarchy (clusters to services to tasks), the number of calls is O(clusters x services) per refresh tick.

**Consequences:**
- `ThrottlingException` errors cascade during refresh.
- Exponential backoff inside the SDK means refresh takes longer than 2 seconds, causing goroutine pileup.
- If retries are not bounded, in-flight goroutines stack up over time.

**Prevention:**
1. Fetch only what is visible on screen (lazy hierarchical fetching):
   - On clusters view: fetch cluster list only.
   - On services view for cluster X: fetch services for X only.
   - On tasks view for service Y: fetch tasks for Y only.
2. Implement refresh backoff when throttled: if a refresh returns ThrottlingException, double the next interval (cap at 30s), then return to normal.
3. Make the refresh interval user-configurable (already a requirement) — default 2s is aggressive for large accounts. Consider 5s default.
4. Use `retry.NewAdaptiveMode()` in the AWS SDK Go v2 config for automatic client-side rate limiting.

**Detection:**
- Run against an account with many clusters and services; watch for ThrottlingException in debug output.
- Check: does the refresh interval hold steady or drift under load?

**Warning signs:**
- Refresh fetches ALL clusters AND all services AND all tasks in a single tick regardless of what view is active.
- ThrottlingException treated as a generic error (not a signal to back off).

**Phase:** Core refresh loop design. This determines the architecture of data fetching and must be decided before building the refresh system.

---

### Pitfall 5: S3 Buckets in Different Regions — Wrong-Region Client Returns 301/400

**What goes wrong:**
S3 is "global" in listing (ListBuckets works from any region), but bucket operations (ListObjectsV2, GetObject, HeadObject) must use the bucket's home region. A client configured for `us-east-1` calling ListObjectsV2 on a bucket in `eu-west-1` gets HTTP 301 (older regions) or HTTP 400 (regions launched after March 2019). This is silent if the error is not properly surfaced.

**Additional quirk:** GetBucketLocation returns `null` (not "us-east-1") for buckets in us-east-1. This is a documented but widely missed behavior. AWS now recommends HeadBucket over GetBucketLocation for region detection because GetBucketLocation has cross-account policy behavior differences across regions.

**Why it happens:**
The S3 client is initialized with a single region. Listing buckets works globally. But once the user drills into a bucket, the same regional client is used for object listing — and it may be the wrong region for that bucket.

**Consequences:**
- Object listing fails silently or shows a cryptic redirect error.
- Buckets in us-east-1 break code that maps `null` location to an error instead of `"us-east-1"`.

**Prevention:**
1. When a user selects a bucket, call `HeadBucket` to discover the bucket's region before listing objects:
```go
head, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &bucketName})
// BucketRegion available in response header via SDK
```
2. Create a new S3 client configured for that region before listing objects.
3. Handle `null` LocationConstraint from GetBucketLocation: explicitly map it to `"us-east-1"`.

**Detection:**
- Test with buckets in at least 3 different regions including `us-east-1`.
- Confirm that GetBucketLocation's null response is handled without an error.

**Warning signs:**
- Single S3 client used for all bucket operations regardless of bucket location.
- No HeadBucket or GetBucketLocation call before listing bucket contents.

**Phase:** S3 bucket browsing. Must be solved before the "navigate into a bucket" feature is considered done.

---

### Pitfall 6: AWS Config INI Parsing — Profile Prefix Rules Break Profile Discovery

**What goes wrong:**
The `~/.aws/config` file requires a `[profile name]` prefix for all named profiles (except `[default]`). The `~/.aws/credentials` file does NOT use this prefix — it's just `[name]`. Tools that parse these files manually (or with generic INI parsers) get the profile list wrong.

**Additional edge cases:**
- `source_profile` and `credential_source` are mutually exclusive. Both in one profile is an error the tool must detect.
- SSO configuration lives in `[sso-session name]` sections — a separate section type that a generic INI parser ignores.
- `sso_account_id` and `sso_role_name` can be absent from SSO profiles that use bearer token auth only.
- All SSO config MUST be in `~/.aws/config`, never in `~/.aws/credentials`.
- Nested subsettings (like `s3 =` blocks) require indented continuation lines — custom parsers often miss this.

**Why it happens:**
Developers manually parse INI files or use generic INI libraries that don't know the AWS profile naming convention. The AWS CLI/SDK handles this correctly; custom parsers don't.

**Consequences:**
- Named profiles don't appear in the profile switcher.
- `[profile dev]` shows up as `profile dev` (with space) instead of `dev`.
- SSO sections ignored because the parser doesn't recognize `[sso-session ...]`.

**Prevention:**
Use the AWS SDK Go v2 config package to enumerate and load profiles. Do not write a custom INI parser:
```go
// The SDK reads ~/.aws/config correctly, handles all section types
cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profileName))
```
If you need to list all profile names for a switcher UI, use the SDK's shared config file abstractions, not raw string parsing.

**Detection:**
- Create a config with `[profile dev]` (correct), `[profile test-sso]` (SSO), and `[default]`.
- Verify the profile switcher shows: `dev`, `test-sso`, `default` — not `profile dev`.

**Warning signs:**
- Custom INI parser anywhere in the codebase.
- Hardcoded string splitting on the `[profile ` prefix.

**Phase:** Profile switcher implementation. Affects every user's first interaction with the tool.

---

### Pitfall 7: Goroutine Leaks in Background Refresh Loops

**What goes wrong:**
The 2-second refresh loop spawns goroutines to fetch AWS data. If the user switches profiles mid-fetch, navigates away from a view, or quits — any in-flight goroutine that doesn't respect context cancellation becomes a leak. Goroutines waiting on AWS SDK calls can block for 30+ seconds (default HTTP timeout). A tool that creates one leak per navigation action gradually accumulates goroutines.

**Why it happens:**
In Bubble Tea, commands (`tea.Cmd`) run in background goroutines managed by the framework. The trap is spawning raw goroutines (`go func()`) outside of the Bubble Tea Cmd system — those are unmanaged and leak on navigation or quit.

**Consequences:**
- Memory growth over time (goroutines hold stack and AWS response data).
- Stale data delivered to the UI after the user has moved to a different view.
- Potential race conditions if stale results update state they no longer own.

**Prevention:**
1. Use Bubble Tea's `tea.Cmd` pattern exclusively for async operations — never raw `go func()`.
2. Pass a context to every AWS SDK call. Cancel the context when the view is torn down or when the profile switches:
```go
func fetchClusters(ctx context.Context, client *ecs.Client) tea.Cmd {
    return func() tea.Msg {
        out, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
        return clustersMsg{clusters: out, err: err}
    }
}
```
3. Use `tea.WithContext(ctx)` when creating the program to propagate top-level cancellation.
4. When profile switches, cancel the current context and create a new one before re-fetching.

**Detection:**
- Monitor goroutine count with `runtime.NumGoroutine()` while switching profiles rapidly.
- Run with `GODEBUG=schedtrace=1000` and watch goroutine count over time.

**Warning signs:**
- Any `go func()` anywhere in refresh/fetch code paths.
- No context passed to AWS SDK calls.
- Context not cancelled on profile switch.

**Phase:** Refresh loop architecture. Must be designed correctly from the start — goroutine leaks are very hard to fix retroactively.

---

### Pitfall 8: Chained Role Assumption — 1-Hour Session Hard Cap

**What goes wrong:**
Profiles that use `source_profile` + `role_arn` to assume a role have a hard 1-hour maximum session duration when role chaining is involved. AWS docs are explicit: if role A assumes role B (a chain), the session for role B cannot exceed 1 hour regardless of the role's configured `MaxSessionDuration`. Tools that cache credentials and don't refresh before the 1-hour mark silently break.

**Why it happens:**
Developers store the `aws.Credentials` struct returned by `cfg.Credentials.Retrieve()` and reuse it for subsequent calls. The SDK's credential provider would auto-refresh, but bypassing it by caching raw credentials breaks the refresh cycle.

**Consequences:**
- Tool works for up to 1 hour, then all API calls fail with `ExpiredTokenException`.
- User doesn't understand why — no indication that it's a credential expiry issue.

**Prevention:**
1. Never cache raw `aws.Credentials`. Always use the SDK's credential provider through `aws.Config` — it handles refresh automatically.
2. When profile switches, create a new `aws.Config` with the new profile — do not reuse config objects across profile changes.
3. On `ExpiredTokenException`, surface a user-readable message and trigger a credential reload.

**Detection:**
- Test with a role-assumption profile and wait past the 1-hour mark.
- Verify `ExpiredTokenException` is caught and handled distinctly from ThrottlingException.

**Warning signs:**
- `Credentials.Retrieve()` called once at startup and result stored in a struct.
- No `ExpiredTokenException` handling in the error handling code.

**Phase:** Profile switching and authentication phase.

---

## Moderate Pitfalls

---

### Pitfall 9: ECS ListTasks Default Filters Out STOPPED Tasks

**What goes wrong:**
`ListTasks` with no `desiredStatus` parameter defaults to `RUNNING`. Services that have experienced recent failures or that are scaling down show zero tasks — the API returns an empty list, not an error. The tool shows "0 tasks" when there are actually STOPPED tasks worth seeing for debugging.

**Additional quirk:** Passing `desiredStatus: PENDING` always returns zero results. AWS never sets a task's desired status to PENDING (only `lastStatus` can be PENDING). This parameter value is a documented no-op that confuses developers.

**Prevention:**
For the task detail view, query both RUNNING and STOPPED tasks separately and display them with their status. Or provide a filter toggle. Never assume an empty task list means "no tasks ever existed for this service."

**Phase:** ECS task list view.

---

### Pitfall 10: Terminal Width/Height Not Updated on Resize

**What goes wrong:**
The terminal window is resized after the tool starts. The Bubble Tea framework sends a `tea.WindowSizeMsg` on resize. Tools that set layout dimensions once at startup and never update them render truncated or overflowing content after a resize.

**Prevention:**
Handle `tea.WindowSizeMsg` in `Update()` and recalculate all layout dimensions:
```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    // recalculate all component sizes
```

**Phase:** Layout/rendering phase.

---

### Pitfall 11: NO_COLOR and TERM=dumb Not Respected

**What goes wrong:**
Tools that hardcode ANSI color codes break in environments where `NO_COLOR` is set (CI, some SSH sessions, accessibility settings) or where `TERM=dumb`. The user sees escape sequences as literal characters.

**Prevention:**
Use `termenv` or `lipgloss` with `EnvColorProfile()` for color detection — it respects `NO_COLOR`, `CLICOLOR`, and `CLICOLOR_FORCE` automatically. Do not hardcode ANSI escape codes.

**Phase:** UI rendering, from the first commit.

---

### Pitfall 12: Box Drawing Characters Break in Non-UTF-8 Locales

**What goes wrong:**
Unicode box-drawing characters (U+2500 range) used for table borders and panels require a UTF-8 locale. On systems with `LANG=C` or `LC_ALL=C`, these render as `?` or garbage. Older terminal emulators and some SSH forwarding configurations have historically had issues.

**Prevention:**
- Detect terminal encoding capability via `LANG`/`LC_CTYPE` environment variables.
- Provide a `--no-unicode` flag or config option that uses ASCII fallback borders (`+`, `-`, `|`).
- Test in a Docker container with `LANG=C` to catch this early.

**Phase:** UI component design. Easier to build in a fallback early than retrofit later.

---

### Pitfall 13: S3 ListObjects (v1) vs ListObjectsV2

**What goes wrong:**
ListObjects (v1) is deprecated. It uses marker-based pagination (not token-based), returns different response fields, and has subtly different behavior with delete markers in versioned buckets. Using v1 in new code creates a maintenance burden.

**Prevention:**
Use `ListObjectsV2` exclusively. The AWS SDK Go v2 paginators support it natively. Never call `ListObjects` in new code.

**Phase:** S3 data fetching, from day one.

---

## Minor Pitfalls

---

### Pitfall 14: Profile Name Display — "profile dev" Instead of "dev"

**What goes wrong:**
If profiles are read from `~/.aws/config` without stripping the `profile ` prefix, the switcher displays `profile dev` instead of `dev`. This is a display/UX bug that looks unprofessional on first run.

**Prevention:**
Strip the `profile ` prefix when displaying profile names. The underlying SDK uses the full section name internally; your display layer should show clean names.

---

### Pitfall 15: AWS Region Not Set — Silent Endpoint Errors

**What goes wrong:**
If no region is configured (no `AWS_DEFAULT_REGION`, no `region` in profile, no `--region` flag), the AWS SDK Go v2 returns an empty region string. Regional API calls then fail with opaque endpoint errors rather than a clear "region not set" message.

**Prevention:**
Check that `cfg.Region != ""` after `LoadDefaultConfig`. If empty, prompt the user to select a region before making any regional API calls. Surface a clear message: "No region configured for this profile. Select a region to continue."

---

### Pitfall 16: GetCallerIdentity in Status Bar Blocking Render

**What goes wrong:**
The status bar shows the current IAM principal (sts:GetCallerIdentity). This call can fail if the profile is misconfigured, credentials are expired, or network is unavailable. Synchronous callers block rendering while the SDK waits for a response or timeout.

**Prevention:**
Fetch caller identity asynchronously with a Bubble Tea Cmd. Display "loading..." until resolved, then update. On error, display "unknown" with a log message. Never block the `View()` function on network calls.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|----------------|------------|
| Profile loading | INI prefix rules, SSO section types | Use SDK config package, not custom parser |
| S3 bucket list | Pagination >1,000 buckets | NewListBucketsPaginator from day one |
| S3 bucket drill-in | Wrong-region client (301/400), null LocationConstraint | HeadBucket to detect region before listing objects |
| S3 object list | Pagination >1,000 objects | NewListObjectsV2Paginator, never ListObjects v1 |
| ECS cluster list | Pagination >100 clusters | Manual nextToken loop |
| ECS service list | Pagination >100 services, large cluster throttling | Paginate; fetch only selected cluster's services |
| ECS task list | 0-task services confusing, STOPPED tasks invisible | Distinguish empty vs. stopped; consider STOPPED filter toggle |
| Refresh loop | Goroutine leaks, throttle cascade | Cmd-only goroutines; hierarchical lazy fetch; back off on ThrottlingException |
| Auth / SSO | Token expiry, 1-hour chain cap, cryptic errors | Detect InvalidTokenError; surface actionable CLI command |
| Status bar | GetCallerIdentity blocking render | Async Cmd fetch with loading state |
| UI layout | Resize not handled, NO_COLOR ignored | Handle WindowSizeMsg; use termenv color detection |
| Terminal output | Box drawing chars broken in C locale | ASCII fallback option via flag or config |

---

## Sources

- AWS S3 ListObjectsV2 pagination: https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html (HIGH confidence)
- AWS ECS ListTasks API: https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_ListTasks.html (HIGH confidence)
- ECS API throttling limits: https://docs.aws.amazon.com/AmazonECS/latest/APIReference/request-throttling.html (HIGH confidence)
- AWS SSO token provider Go SDK: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/credentials/ssocreds (HIGH confidence)
- AWS Config file format edge cases: https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html (HIGH confidence)
- S3 GetBucketLocation null for us-east-1: https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketLocation.html (HIGH confidence)
- S3 virtual hosting and region redirects (301/400): https://docs.aws.amazon.com/AmazonS3/latest/userguide/VirtualHosting.html (HIGH confidence)
- AWS SDK Go v2 LoadDefaultConfig / credential chain: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/config (HIGH confidence)
- Bubble Tea Cmd/goroutine pattern: https://pkg.go.dev/github.com/charmbracelet/bubbletea (HIGH confidence)
- termenv color detection / NO_COLOR: https://pkg.go.dev/github.com/muesli/termenv (HIGH confidence)
- Role chaining 1-hour session hard cap: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use_switch-role-api.html (HIGH confidence)
- AWS API retry modes: https://docs.aws.amazon.com/general/latest/gr/api-retries.html (HIGH confidence)
- S3 performance / request rate limits: https://docs.aws.amazon.com/AmazonS3/latest/userguide/optimizing-performance.html (HIGH confidence)
