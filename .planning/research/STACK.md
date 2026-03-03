# Technology Stack

**Project:** duchess — k9s-inspired AWS TUI in Go
**Researched:** 2026-03-02
**Confidence:** HIGH — all versions verified against pkg.go.dev; go.mod already exists with real dependencies pinned

---

## Context

This project is not greenfield in the strictest sense: a `go.mod` already exists with a working dependency set. The decisions below align with what is already in go.mod and upgrade to current versions where the existing pins are behind latest. Every library here is verified against pkg.go.dev as of 2026-03-02.

---

## Recommended Stack

### TUI Framework

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `github.com/charmbracelet/bubbletea` | v1.3.10 | Core TUI event loop and rendering | The Elm Architecture (Model/Update/View) is the dominant paradigm for Go TUIs. v1.x is stable, battle-tested in production at scale (used by 10,000+ apps). Already in go.mod. The async Cmd model maps perfectly to concurrent AWS API calls. |
| `github.com/charmbracelet/bubbles` | v1.0.0 | Pre-built TUI components | Hit v1.0.0 stable on Feb 9, 2026. Provides List, Table, Viewport, Spinner, TextInput, Help, and Paginator — everything duchess needs for browse/navigate/detail views. The `list` component handles the k9s-style resource browser directly. Currently pinned at v0.21.0 in go.mod — upgrade to v1.0.0. |
| `github.com/charmbracelet/lipgloss` | v1.1.0 | Terminal styling and layout | CSS-for-the-terminal. Status bars, panes, borders, colors. v1.1.0 (Mar 12, 2025) added StyleRanges and ASCIIBorder. Already in go.mod at v1.1.0. |

**Why Bubble Tea over alternatives:** Bubble Tea is now effectively the Go TUI standard. The only real alternative is `tview` (backed by tcell), which uses a mutable widget tree rather than functional immutability. k9s itself uses tview, but for a new project, Bubble Tea's compositional model and first-class async support make it easier to reason about concurrent AWS polling. `termui` is maintenance-mode. `gocui` is minimal and lower-level. Bubble Tea is the correct choice.

### AWS SDK

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `github.com/aws/aws-sdk-go-v2` | v1.41.2 | AWS SDK core | v2 is the current, actively maintained SDK. v1 (classic) is in maintenance mode — no new features. Already in go.mod at v1.41.0; upgrade to v1.41.2. |
| `github.com/aws/aws-sdk-go-v2/config` | v1.32.10 | Config loading, profile selection, SSO | The single entry point for all auth concerns. `LoadDefaultConfig` reads `~/.aws/config`, resolves credential chains, handles role assumption and SSO automatically. Currently at v1.32.6 in go.mod; upgrade to v1.32.10. |
| `github.com/aws/aws-sdk-go-v2/service/s3` | v1.96.2 | S3 bucket and object listing | Provides `ListBuckets`, `ListObjectsV2`, `HeadObject`, `GetObjectAttributes` with built-in `*Paginator` types that abstract token management. Currently at v1.58.3 in go.mod — significant version gap; upgrade immediately. |
| `github.com/aws/aws-sdk-go-v2/service/ecs` | v1.73.0 | ECS cluster/service/task listing | Not currently in go.mod — add this. Provides `ListClusters`, `DescribeClusters`, `ListServices`, `DescribeServices`, `ListTasks`, `DescribeTasks` with `*Paginator` types for each. |
| `github.com/aws/aws-sdk-go-v2/service/sts` | v1.41.7 | GetCallerIdentity (status bar) | `GetCallerIdentity` returns Account, ARN, and UserId with zero input parameters — perfect for the status bar "who you are" display. Currently at v1.41.5 in go.mod; upgrade to v1.41.7. |
| `github.com/aws/aws-sdk-go-v2/credentials` | v1.x (indirect) | Credential providers (SSO, static, process) | Pulled in transitively by config; provides `ssocreds`, `stscreds`, `processcreds`. No direct import needed — config handles provider selection. |

**AWS SDK v1 vs v2:** Do not use `github.com/aws/aws-sdk-go` (v1). It is maintenance-mode, uses a different API surface (pointer everywhere vs value types in v2), and has no new service features. The project already uses v2 correctly.

### AWS Auth / Profile Handling

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `github.com/aws/aws-sdk-go-v2/config` | v1.32.10 | SSO + role assumption + profile switching | `LoadDefaultConfig(ctx, config.WithSharedConfigProfile("name"))` handles named profiles, role_arn/source_profile chains, and SSO profiles transparently. No additional auth library needed. |
| `gopkg.in/ini.v1` | v1.67.1 | Profile enumeration from `~/.aws/config` | The AWS SDK has no public API for listing all profiles — it can only load a named profile. To build a profile picker, you must parse `~/.aws/config` yourself. It is an INI file. ini.v1 is already in go.mod (as indirect dep of aws-sdk-go-v2), actively maintained (v1.67.1, Jan 2026), and 5,710+ importers confirm it as the standard. Use `config.DefaultSharedConfigFilename()` to get the path, then `ini.Load()` to enumerate sections. |

**SSO handling:** The SDK's SSO token provider (`ssocreds.SSOTokenProvider`) handles token refresh automatically — it calls the OIDC `CreateToken` API when cached tokens in `~/.aws/sso/cache/` expire. duchess does NOT need to implement login flows. If the token is expired and refresh fails, surface an error directing the user to run `aws sso login`. This is the same behavior as the AWS CLI and equivalent k9s-style tools.

**Role assumption:** Handled transparently by `LoadDefaultConfig` when the profile has `role_arn` + `source_profile`. No stscreds import needed at the call site.

### Configuration File

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `gopkg.in/yaml.v3` | v3.0.1 | `~/.duchess/config` parsing | YAML is the standard for Go TUI tool configs (k9s uses YAML, Helm uses YAML). yaml.v3 is already in go.mod as an indirect dep. Prefer YAML over TOML for this use case — TOML is better for project configs with complex types; YAML is more familiar to the AWS/Kubernetes audience duchess targets. |

**Why not TOML:** TOML (BurntSushi/toml v1.6.0) is technically fine, but duchess's target users come from a Kubernetes/AWS background where YAML is ambient. YAML reduces cognitive friction.

**Why not a custom format:** Do not invent another config format. YAML or TOML, pick one.

### CLI Argument Parsing

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `github.com/spf13/cobra` | v1.10.2 | CLI entrypoint, flags, subcommands | Cobra is the de facto standard for Go CLI tools (kubectl, k9s, hugo, etc.). v1.10.2 (Dec 2025). Provides flag definitions, `--profile`, `--region`, `--refresh-interval` flags, help text generation, and shell completion for free. |
| `github.com/spf13/pflag` | v1.0.10 | POSIX-style flags (pulled by cobra) | Transitive dep of cobra. No direct import needed. |

**Why not `flag` (stdlib):** Standard `flag` doesn't support POSIX-style `--flag` syntax or subcommands. The AWS CLI uses `--` style; duchess should match.

**Why not urfave/cli:** Cobra is more widely adopted in the Go infrastructure tooling ecosystem (the space duchess lives in). Familiarity matters for contributors.

### Testing

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `github.com/stretchr/testify` | v1.9.0 | Assertions and test helpers | Already in go.mod. Standard in the Go ecosystem. `require.*` for fatal assertions, `assert.*` for non-fatal. |

---

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| TUI Framework | Bubble Tea v1.3.10 | tview (tcell) | tview uses mutable widget tree — harder to reason about concurrent state. k9s uses it but duchess starts fresh. |
| TUI Framework | Bubble Tea v1.3.10 | termui | Maintenance-mode. Not actively developed. |
| TUI Framework | Bubble Tea v1.3.10 | gocui | Lower-level, no component library, more boilerplate for what duchess needs. |
| AWS SDK | aws-sdk-go-v2 | aws-sdk-go (v1) | Maintenance-mode. No new features. Different API conventions. |
| Config format | YAML (yaml.v3) | TOML (BurntSushi/toml) | TOML fine technically, but YAML matches the target audience's mental model (AWS/k8s users). |
| Config format | YAML (yaml.v3) | JSON | JSON has no comments; config files need comments. |
| CLI flags | cobra | urfave/cli | Cobra is dominant in Go infrastructure tooling. Better ecosystem fit. |
| CLI flags | cobra | stdlib flag | No POSIX-style flags, no subcommands, poor help text. |
| Profile enumeration | gopkg.in/ini.v1 | Manual string parsing | ini.v1 is already in go.mod, robust, handles edge cases (comments, continuation lines, whitespace). |

---

## Current go.mod Versions vs Recommended

| Package | go.mod (current) | Recommended | Action |
|---------|------------------|-------------|--------|
| bubbletea | v1.3.10 | v1.3.10 | None — already current |
| bubbles | v0.21.0 | v1.0.0 | Upgrade — v1.0.0 stable released Feb 2026 |
| lipgloss | v1.1.0 | v1.1.0 | None — already current |
| aws-sdk-go-v2 core | v1.41.0 | v1.41.2 | Upgrade — minor |
| aws-sdk-go-v2/config | v1.32.6 | v1.32.10 | Upgrade — minor |
| aws-sdk-go-v2/service/s3 | v1.58.3 | v1.96.2 | Upgrade — significant version gap |
| aws-sdk-go-v2/service/sts | v1.41.5 | v1.41.7 | Upgrade — minor |
| aws-sdk-go-v2/service/ecs | not in go.mod | v1.73.0 | Add — required for ECS features |
| stretchr/testify | v1.9.0 | v1.9.0 | None — already current |
| cobra | not in go.mod | v1.10.2 | Add — required for CLI flags |
| yaml.v3 | v3.0.1 (indirect) | v3.0.1 (direct) | Promote to direct dependency |

---

## Installation

```bash
# Upgrade existing packages
go get github.com/charmbracelet/bubbles@v1.0.0
go get github.com/aws/aws-sdk-go-v2@v1.41.2
go get github.com/aws/aws-sdk-go-v2/config@v1.32.10
go get github.com/aws/aws-sdk-go-v2/service/s3@v1.96.2
go get github.com/aws/aws-sdk-go-v2/service/sts@v1.41.7

# Add new packages
go get github.com/aws/aws-sdk-go-v2/service/ecs@v1.73.0
go get github.com/spf13/cobra@v1.10.2

# Promote indirect to direct
go get gopkg.in/yaml.v3@v3.0.1

# Tidy
go mod tidy
```

---

## Key Architectural Patterns Enabled by This Stack

**Profile switching:** `config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(name))` — call this with the new profile name whenever user switches. Rebuilds the AWS credential chain. Cheap enough to do on switch.

**Region switching:** Reconstruct service clients with `config.WithRegion(region)` option when user picks a new region. Service clients are cheap to reconstruct.

**Status bar identity:** `sts.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})` — runs once after profile/region switch. Returns ARN for display. Cache result; invalidate on switch.

**Profile list enumeration:**
```go
configPath := config.DefaultSharedConfigFilename()
cfg, _ := ini.Load(configPath)
for _, section := range cfg.Sections() {
    name := strings.TrimPrefix(section.Name(), "profile ")
    // filter out DEFAULT section
}
```

**Paginated ECS listing:**
```go
paginator := ecs.NewListClustersPaginator(client, &ecs.ListClustersInput{})
for paginator.HasMorePages() {
    page, err := paginator.NextPage(ctx)
    // collect page.ClusterArns
}
```

**Paginated S3 object listing:**
```go
paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
    Bucket: aws.String(bucket),
    Prefix: aws.String(prefix),
})
for paginator.HasMorePages() {
    page, err := paginator.NextPage(ctx)
    // collect page.Contents
}
```

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Bubble Tea / Bubbles / Lipgloss versions | HIGH | Verified directly on pkg.go.dev |
| AWS SDK v2 versions | HIGH | Verified directly on pkg.go.dev; all service packages confirmed |
| SSO handling via aws-sdk-go-v2/config | HIGH | Official SDK docs confirm SSOTokenProvider + automatic refresh behavior |
| Profile enumeration via ini.v1 | HIGH | SDK docs explicitly confirm no public API for listing profiles; ini.v1 already in go.mod |
| Cobra for CLI flags | HIGH | pkg.go.dev confirms v1.10.2; dominant pattern in Go tooling |
| YAML for config file | MEDIUM | Best-fit for audience; technically TOML is equivalent. No hard source confirms "YAML over TOML for AWS tools." |
| bubbles v1.0.0 API stability | HIGH | Stable v1.0.0 released Feb 9, 2026; verified on pkg.go.dev |

---

## Sources

- Bubble Tea: https://pkg.go.dev/github.com/charmbracelet/bubbletea (verified v1.3.10)
- Bubbles: https://pkg.go.dev/github.com/charmbracelet/bubbles (verified v1.0.0, Feb 2026)
- Lipgloss: https://pkg.go.dev/github.com/charmbracelet/lipgloss (verified v1.1.0, Mar 2025)
- AWS SDK v2 core: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2 (verified v1.41.2, Feb 2026)
- AWS SDK v2 config: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/config (SharedConfig, LoadDefaultConfig, profile enumeration limitation)
- AWS SDK v2 S3: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3 (paginator patterns confirmed)
- AWS SDK v2 ECS: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/ecs (verified v1.73.0, Feb 2026)
- AWS SDK v2 STS: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/sts (verified v1.41.7, Feb 2026)
- AWS SDK v2 ssocreds: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/credentials/ssocreds (SSOTokenProvider, legacy vs token provider distinction)
- Cobra: https://pkg.go.dev/github.com/spf13/cobra (verified v1.10.2, Dec 2025)
- ini.v1: https://pkg.go.dev/gopkg.in/ini.v1 (verified v1.67.1, Jan 2026)
- yaml.v3: https://pkg.go.dev/gopkg.in/yaml.v3 (verified v3.0.1)
