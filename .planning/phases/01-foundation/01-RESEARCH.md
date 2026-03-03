# Phase 1: Foundation - Research

**Researched:** 2026-03-03
**Domain:** Go CLI binary — AWS SDK v2 authentication (named profiles, role assumption, SSO), cobra CLI flags, YAML config loading, STS identity
**Confidence:** HIGH

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| AUTH-02 | App authenticates with named profiles (access key + secret key) | AWS SDK v2 `LoadDefaultConfig` + `WithSharedConfigProfile` handles named profiles transparently |
| AUTH-03 | App authenticates with role assumption profiles (role_arn + source_profile) | SDK config package resolves role chains automatically via STS AssumeRole — no additional code needed |
| AUTH-04 | App authenticates with AWS SSO profiles (sso_start_url / sso_account_id) | SDK SSO token provider handles cached tokens in `~/.aws/sso/cache/` automatically |
| AUTH-06 | App displays actionable error when credentials are expired/invalid | `ssocreds.InvalidTokenError` detected via `errors.As`; surface "run aws sso login --profile X" message |
| CONF-01 | Screen auto-refreshes at configurable interval (default: 2 seconds) | Config struct holds refresh interval; loaded from YAML then overridden by cobra flag |
| CONF-02 | User can set refresh interval via CLI flag (--refresh-interval) | cobra `PersistentFlags().IntVar` with default 2 |
| CONF-03 | User can set default profile via CLI flag (--profile) | cobra `PersistentFlags().StringVar` |
| CONF-04 | User can set default region via CLI flag (--region) | cobra `PersistentFlags().StringVar` |
| CONF-05 | App loads configuration from ~/.duchess/config on startup | yaml.v3 `yaml.Unmarshal` from `os.UserHomeDir()/.duchess/config` |
| CONF-06 | CLI flags take precedence over config file values | Load YAML config first, then apply cobra flag values over it only if flags were explicitly set |
</phase_requirements>

---

## Summary

Phase 1 builds the CLI foundation with no TUI — a working Go binary that proves AWS authentication across all three profile types and reads configuration from both a YAML file and CLI flags. The Go module already exists (`github.com/jacobscunn07/duchess`) with AWS SDK v2 and Bubble Tea already present; Phase 1 adds cobra and promotes yaml.v3 from indirect to direct dependency.

The authentication work is entirely handled by the AWS SDK v2 config package (`LoadDefaultConfig` + `WithSharedConfigProfile`). Named profiles, role assumption chains, and SSO sessions all resolve through the same call — no custom credential wiring is needed. The one exception is error handling: `ssocreds.InvalidTokenError` must be caught specifically with `errors.As` to surface the "run aws sso login --profile X" message instead of a cryptic SDK error.

The two plans outlined in ROADMAP.md match the natural dependency split: Plan 1 is pure Go/CLI setup (cobra + config loading), Plan 2 is AWS-specific (session factory + STS identity + error taxonomy). This phase produces no TUI code — Bubble Tea comes in Phase 2. The binary output at phase completion is a `duchess --profile X` command that prints caller identity to stdout.

**Primary recommendation:** Use `LoadDefaultConfig` with `WithSharedConfigProfile` as the single AWS config entry point; never cache raw `aws.Credentials` structs; detect `ssocreds.InvalidTokenError` explicitly before all other error handling.

---

## Standard Stack

### Core (Phase 1 specific)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/spf13/cobra` | v1.10.2 | CLI entrypoint, persistent flags | De facto standard for Go infrastructure tools (kubectl, k9s, Hugo). Provides `--flag` POSIX-style syntax, automatic help text, shell completion. |
| `gopkg.in/yaml.v3` | v3.0.1 | Parse `~/.duchess/config` | Already in go.mod as indirect dep; YAML matches the AWS/k8s audience's muscle memory. Promote to direct dep. |
| `github.com/aws/aws-sdk-go-v2/config` | v1.32.10 | AWS config loading, all profile types | Single entry point for all auth: resolves named profiles, role chains, SSO tokens transparently via `LoadDefaultConfig`. |
| `github.com/aws/aws-sdk-go-v2/service/sts` | v1.41.7 | `GetCallerIdentity` | Proves auth works; returns Account + ARN used in status bar later. Zero-input call — no parameters needed. |
| `github.com/aws/aws-sdk-go-v2/credentials/ssocreds` | v1.19.10 | SSO error type detection | Provides `InvalidTokenError` for typed error handling of expired SSO sessions. |

### Already Present (go.mod — no action needed for Phase 1)

| Library | Version | Notes |
|---------|---------|-------|
| `github.com/aws/aws-sdk-go-v2` | v1.41.0 (upgrade to v1.41.2) | SDK core — already direct dep |
| `github.com/stretchr/testify` | v1.9.0 | Test assertions — already present |
| `gopkg.in/ini.v1` | v1.67.0 (upgrade to v1.67.1) | Profile enumeration — NOT needed in Phase 1 (Phase 5 only) |

### Phase 1 Does NOT Need (deferred)

| Library | Phase |
|---------|-------|
| `github.com/charmbracelet/bubbletea` | Phase 2 |
| `github.com/charmbracelet/bubbles` | Phase 2 |
| `github.com/charmbracelet/lipgloss` | Phase 2 |
| `github.com/aws/aws-sdk-go-v2/service/ecs` | Phase 4 |
| `github.com/aws/aws-sdk-go-v2/service/s3` | Phase 3 |
| `gopkg.in/ini.v1` (direct use) | Phase 5 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| cobra v1.10.2 | stdlib `flag` | stdlib has no `--flag` POSIX-style, no subcommands, poor help text |
| cobra v1.10.2 | urfave/cli | cobra dominates Go infra tooling; better ecosystem fit for duchess audience |
| yaml.v3 | BurntSushi/toml | YAML matches AWS/k8s audience mental model; TOML fine technically |
| yaml.v3 | JSON | JSON has no comments; config files need comments |

**Installation (Phase 1 additions only):**
```bash
# Add cobra
go get github.com/spf13/cobra@v1.10.2

# Promote yaml.v3 to direct dependency
go get gopkg.in/yaml.v3@v3.0.1

# Minor upgrade of existing deps
go get github.com/aws/aws-sdk-go-v2@v1.41.2
go get github.com/aws/aws-sdk-go-v2/config@v1.32.10
go get github.com/aws/aws-sdk-go-v2/service/sts@v1.41.7

# Tidy
go mod tidy
```

---

## Architecture Patterns

### Recommended Project Structure (Phase 1 scope only)

```
duchess/
├── main.go                  # cobra rootCmd + Execute(); nothing else
├── cmd/
│   └── root.go              # cobra command definition, flag vars, cobra.Command.RunE
├── internal/
│   ├── config/
│   │   └── config.go        # Config struct, LoadConfig(), flag-over-file merge logic
│   └── aws/
│       └── session.go       # NewSession(profile, region) -> aws.Config; error taxonomy
└── go.mod
```

**Why this structure:**
- `main.go` is minimal — cobra Execute() only, never grows
- `cmd/` is the standard cobra convention; all packages that add subcommands live here
- `internal/config/` owns the Config struct and loading — no AWS package imports here
- `internal/aws/` owns session creation — returns `aws.Config`, not service clients (clients created at call site)

### Pattern 1: Config Struct with Flag-Over-File Precedence

**What:** Load YAML config first, then only override fields where cobra flags were explicitly set (not just default values).

**When to use:** Any time you have both a config file and CLI flags with defaults.

**The trap with defaults:** cobra flag defaults make every flag "set" — you can't distinguish `--profile default` from "user didn't pass --profile." Use `cobra.Command.Flags().Changed("profile")` to check whether a flag was explicitly provided.

```go
// Source: cobra official docs + AWS CLI pattern
type Config struct {
    Profile         string `yaml:"profile"`
    Region          string `yaml:"region"`
    RefreshInterval int    `yaml:"refresh_interval"`
}

func LoadConfig(cmd *cobra.Command) (*Config, error) {
    // 1. Start with defaults
    cfg := &Config{
        RefreshInterval: 2,
    }

    // 2. Load from ~/.duchess/config if it exists
    home, _ := os.UserHomeDir()
    configPath := filepath.Join(home, ".duchess", "config")
    if data, err := os.ReadFile(configPath); err == nil {
        if err := yaml.Unmarshal(data, cfg); err != nil {
            return nil, fmt.Errorf("parse ~/.duchess/config: %w", err)
        }
    }

    // 3. Override only flags that were explicitly set on the command line
    if cmd.Flags().Changed("profile") {
        cfg.Profile, _ = cmd.Flags().GetString("profile")
    }
    if cmd.Flags().Changed("region") {
        cfg.Region, _ = cmd.Flags().GetString("region")
    }
    if cmd.Flags().Changed("refresh-interval") {
        cfg.RefreshInterval, _ = cmd.Flags().GetInt("refresh-interval")
    }

    return cfg, nil
}
```

### Pattern 2: AWS Session Factory

**What:** A single function that takes profile + region and returns a fully configured `aws.Config`. All three auth types (named, role assumption, SSO) resolve through the same call.

**When to use:** Whenever profile or region changes. `LoadDefaultConfig` is cheap enough to call on every switch.

```go
// Source: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/config
func NewAWSConfig(ctx context.Context, profile, region string) (aws.Config, error) {
    opts := []func(*config.LoadOptions) error{
        config.WithSharedConfigProfile(profile),
    }
    if region != "" {
        opts = append(opts, config.WithRegion(region))
    }
    return config.LoadDefaultConfig(ctx, opts...)
}
```

### Pattern 3: Typed Error Taxonomy for Credentials

**What:** Catch specific AWS credential errors before falling through to a generic handler. Each error type has a distinct user-facing message and recovery instruction.

**When to use:** After every `LoadDefaultConfig` call and every AWS API call where credential expiry could surface.

```go
// Source: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/credentials/ssocreds
import (
    "errors"
    "github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
    "github.com/aws/smithy-go"
)

func classifyAWSError(err error, profile string) error {
    if err == nil {
        return nil
    }

    // SSO token expired — user must re-authenticate
    var invalidToken *ssocreds.InvalidTokenError
    if errors.As(err, &invalidToken) {
        return fmt.Errorf("SSO session expired. Run: aws sso login --profile %s", profile)
    }

    // Credentials expired (role chaining 1-hour cap, static key rotation)
    var apiErr smithy.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.ErrorCode() {
        case "ExpiredTokenException":
            return fmt.Errorf("credentials expired for profile %q. Refresh your credentials and try again.", profile)
        case "InvalidClientTokenId", "AuthFailure":
            return fmt.Errorf("invalid credentials for profile %q. Check your access key and secret.", profile)
        case "AccessDenied":
            return fmt.Errorf("access denied for profile %q. Check IAM permissions.", profile)
        }
    }

    // No region configured
    if strings.Contains(err.Error(), "no EC2 IMDS role found") || cfg.Region == "" {
        return fmt.Errorf("no region configured. Set AWS_DEFAULT_REGION or use --region flag.")
    }

    return fmt.Errorf("AWS error: %w", err)
}
```

### Pattern 4: GetCallerIdentity for Identity Verification

**What:** Call STS to prove authentication succeeded and retrieve the IAM principal.

**When to use:** At phase completion to verify all three auth types work; later reused async in Phase 2 status bar.

```go
// Source: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/sts
func GetIdentity(ctx context.Context, cfg aws.Config) (account, arn string, err error) {
    client := sts.NewFromConfig(cfg)
    result, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
    if err != nil {
        return "", "", err
    }
    return aws.ToString(result.Account), aws.ToString(result.Arn), nil
}
```

### Pattern 5: cobra Root Command Setup

**What:** Minimal cobra root command with three persistent flags. `RunE` is used instead of `Run` so errors propagate correctly.

```go
// Source: https://pkg.go.dev/github.com/spf13/cobra v1.10.2
var rootCmd = &cobra.Command{
    Use:   "duchess",
    Short: "Navigate your AWS resources across accounts and regions",
    RunE: func(cmd *cobra.Command, args []string) error {
        cfg, err := config.LoadConfig(cmd)
        if err != nil {
            return err
        }
        // Phase 1: print identity to stdout to prove auth works
        awsCfg, err := aws.NewAWSConfig(cmd.Context(), cfg.Profile, cfg.Region)
        if err != nil {
            return aws.ClassifyAWSError(err, cfg.Profile)
        }
        account, arn, err := aws.GetIdentity(cmd.Context(), awsCfg)
        if err != nil {
            return aws.ClassifyAWSError(err, cfg.Profile)
        }
        fmt.Printf("Account: %s\nARN: %s\n", account, arn)
        return nil
    },
}

func init() {
    rootCmd.PersistentFlags().String("profile", "", "AWS profile name")
    rootCmd.PersistentFlags().String("region", "", "AWS region")
    rootCmd.PersistentFlags().Int("refresh-interval", 2, "Refresh interval in seconds")
}
```

### Anti-Patterns to Avoid

- **Caching `aws.Credentials` structs:** Never call `cfg.Credentials.Retrieve()` and store the result. Always use `aws.Config` directly — the SDK handles token refresh. Cached raw credentials break on the 1-hour role chain cap.
- **Single SDK error handler for all auth failures:** SSO expiry, key expiry, role access denied, and no-region all require different user messages. Generic `err.Error()` shown to users is a product failure.
- **Checking cobra flag values to detect "was flag set":** Use `cmd.Flags().Changed("name")` — cobra flag defaults make the value non-nil even when not explicitly passed.
- **Parsing `~/.aws/config` in Phase 1:** Profile enumeration (ini.v1 parsing) is NOT needed until Phase 5. In Phase 1, the profile name comes from the CLI flag or config file — the SDK loads it by name.
- **Putting business logic in `main.go`:** `main.go` calls `rootCmd.Execute()` and returns. All logic lives in `cmd/` or `internal/`.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| AWS named profile auth | Custom credential loading from `~/.aws/credentials` | `config.LoadDefaultConfig` + `WithSharedConfigProfile` | SDK handles credential chain, env var precedence, process credentials, EC2 IMDS — custom parsers miss all of these |
| Role assumption | Manual STS AssumeRole calls with token refresh loops | `LoadDefaultConfig` with `role_arn` profile | SDK's `stscreds.AssumeRoleProvider` handles session refresh, MFA token prompts (future), and the 1-hour chaining cap tracking automatically |
| SSO token loading | Reading `~/.aws/sso/cache/*.json` manually | `LoadDefaultConfig` with SSO profile | SDK's `ssocreds.SSOTokenProvider` handles token refresh, OIDC CreateToken, and expiry detection — JSON cache format is undocumented and subject to change |
| INI parsing for profile names | Custom string splitting on `[profile ...]` | `gopkg.in/ini.v1` | NOT needed in Phase 1 at all. When needed in Phase 5, ini.v1 handles continuation lines, comments, and `[sso-session]` sections that custom parsers miss |
| CLI flag parsing | Custom `os.Args` parsing | cobra v1.10.2 | POSIX flag semantics, `--flag=value` and `--flag value` both work, automatic help text, shell completion for free |
| YAML config parsing | Custom string splitting or regex | yaml.v3 | Handles nested structs, tags, type coercion, comment stripping |

**Key insight:** The AWS SDK v2 config package's credential chain is the most complex piece of software in this phase. It handles ~12 credential source types, environment variable precedence, INI file format edge cases, and token refresh loops. Using it as a black box is the correct architecture. Fighting it by building custom credential code guarantees missing edge cases.

---

## Common Pitfalls

### Pitfall 1: SSO Token Expiry Not Caught Specifically

**What goes wrong:** When SSO token at `~/.aws/sso/cache/` expires, the SDK returns `ssocreds.InvalidTokenError`. If this is not caught before generic error handling, the user sees a cryptic SDK error instead of the actionable "run aws sso login" message.

**Why it happens:** The error wraps deeply — `errors.As` is required; direct type assertion fails.

**How to avoid:** Always check `errors.As(err, &invalidToken)` before any other error handling in auth paths.

**Warning signs:** Generic `err.Error()` displayed to users for auth failures.

### Pitfall 2: cobra Flag Default Confused for User-Set Value

**What goes wrong:** `--profile` has default `""` and `--refresh-interval` has default `2`. If you read these values unconditionally and override the config file, users who don't pass `--profile` get `""` overriding their config file's profile.

**Why it happens:** cobra always provides a value for every flag (the default). You cannot tell "was this flag set?" by reading the value alone.

**How to avoid:** Use `cmd.Flags().Changed("profile")` to check whether a flag was explicitly provided before overriding the config file value.

**Warning signs:** Config file values are always ignored when the flag has a non-empty default.

### Pitfall 3: Raw Credentials Cached Across the 1-Hour Role Chain Cap

**What goes wrong:** Role assumption profiles (`role_arn + source_profile`) have a hard 1-hour session cap in AWS when role chains are involved. Caching `aws.Credentials` at startup means the tool silently breaks after 60 minutes.

**Why it happens:** `cfg.Credentials.Retrieve()` returns a snapshot credential. The snapshot's `Expires` field is set, but nothing enforces refresh if you store the result.

**How to avoid:** Never store the result of `Retrieve()`. Always pass `aws.Config` to service client constructors — the SDK refreshes automatically.

**Warning signs:** `Retrieve()` called once at startup; result stored in a struct field.

### Pitfall 4: Missing Region Produces Opaque Endpoint Error

**What goes wrong:** If no region is set (no `AWS_DEFAULT_REGION`, no `region` in profile, no `--region` flag), the SDK's `cfg.Region` is `""`. Regional API calls (STS GetCallerIdentity, ECS, etc.) fail with "no region configured" wrapped in an endpoint resolution error — not a clear user message.

**Why it happens:** STS and ECS require a region. S3 ListBuckets does not but object operations do.

**How to avoid:** After `LoadDefaultConfig`, check `cfg.Region == ""`. If so, surface "No region configured. Use --region or set region in ~/.duchess/config." before any API call.

**Warning signs:** STS or ECS calls fail with endpoint resolution errors on first run without region config.

### Pitfall 5: Config File Path Not Using `os.UserHomeDir()`

**What goes wrong:** Hardcoding `~/.duchess/config` as a string fails on systems where `~` is not expanded by Go (it's shell expansion, not a Go feature). The file is never found.

**Why it happens:** Developers assume `~` works in Go path strings — it does not.

**How to avoid:** Always use `os.UserHomeDir()` to get the home directory:
```go
home, err := os.UserHomeDir()
configPath := filepath.Join(home, ".duchess", "config")
```

**Warning signs:** `os.Open("~/.duchess/config")` anywhere in the codebase — this will always fail.

---

## Code Examples

Verified patterns from official sources:

### Config Loading with Flag Precedence

```go
// Source: cobra v1.10.2 docs — cmd.Flags().Changed()
func LoadConfig(cmd *cobra.Command) (*Config, error) {
    cfg := &Config{RefreshInterval: 2}

    home, err := os.UserHomeDir()
    if err != nil {
        return nil, fmt.Errorf("resolve home dir: %w", err)
    }

    configPath := filepath.Join(home, ".duchess", "config")
    if data, err := os.ReadFile(configPath); err == nil {
        if err := yaml.Unmarshal(data, cfg); err != nil {
            return nil, fmt.Errorf("parse config: %w", err)
        }
    }
    // err from ReadFile: if file doesn't exist, that's fine — use defaults

    if cmd.Flags().Changed("profile") {
        cfg.Profile, _ = cmd.Flags().GetString("profile")
    }
    if cmd.Flags().Changed("region") {
        cfg.Region, _ = cmd.Flags().GetString("region")
    }
    if cmd.Flags().Changed("refresh-interval") {
        cfg.RefreshInterval, _ = cmd.Flags().GetInt("refresh-interval")
    }

    return cfg, nil
}
```

### AWS Session Factory (All Three Auth Types)

```go
// Source: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/config v1.32.10
// All three auth types (named, role assumption, SSO) resolve through LoadDefaultConfig
func NewAWSConfig(ctx context.Context, profile, region string) (aws.Config, error) {
    opts := []func(*config.LoadOptions) error{
        config.WithSharedConfigProfile(profile),
    }
    if region != "" {
        opts = append(opts, config.WithRegion(region))
    }
    cfg, err := config.LoadDefaultConfig(ctx, opts...)
    if err != nil {
        return aws.Config{}, err
    }
    if cfg.Region == "" {
        return aws.Config{}, fmt.Errorf("no region configured for profile %q — use --region flag", profile)
    }
    return cfg, nil
}
```

### SSO InvalidTokenError Detection

```go
// Source: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/credentials/ssocreds v1.19.10
import (
    "errors"
    "github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
)

func classifyCredentialError(err error, profile string) error {
    var invalidToken *ssocreds.InvalidTokenError
    if errors.As(err, &invalidToken) {
        return fmt.Errorf("SSO session expired. Run: aws sso login --profile %s", profile)
    }

    var apiErr smithy.APIError
    if errors.As(err, &apiErr) && apiErr.ErrorCode() == "ExpiredTokenException" {
        return fmt.Errorf("credentials expired for profile %q", profile)
    }

    return fmt.Errorf("authentication failed for profile %q: %w", profile, err)
}
```

### GetCallerIdentity

```go
// Source: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/sts v1.41.7
func GetCallerIdentity(ctx context.Context, cfg aws.Config) (account, arn string, err error) {
    client := sts.NewFromConfig(cfg)
    out, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
    if err != nil {
        return "", "", err
    }
    return aws.ToString(out.Account), aws.ToString(out.Arn), nil
}
```

### Config Struct (YAML)

```go
// Source: https://pkg.go.dev/gopkg.in/yaml.v3 v3.0.1
type Config struct {
    Profile         string `yaml:"profile"`
    Region          string `yaml:"region"`
    RefreshInterval int    `yaml:"refresh_interval"`
}

// Example ~/.duchess/config:
// profile: prod
// region: us-east-1
// refresh_interval: 5
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| aws-sdk-go v1 (pointers everywhere) | aws-sdk-go-v2 (value types, modular) | 2021 | v1 is maintenance-mode; v2 has paginators, modular service packages, context-aware |
| Manual SSO token cache parsing | `ssocreds.SSOTokenProvider` in config package | SDK v2 early releases | SDK handles OIDC refresh, cache file format, expiry detection |
| `flag` stdlib for CLI | cobra v1.x | N/A — cobra has dominated for years | POSIX flags, subcommands, auto-help |

**Deprecated/outdated:**
- `aws-sdk-go` (v1 at `github.com/aws/aws-sdk-go`): Maintenance-mode. No new features. All new Go AWS code uses v2.
- `ListObjects` (S3 v1 API): Deprecated in favor of `ListObjectsV2`. Not relevant in Phase 1 but do not import.

---

## Open Questions

1. **Default profile when none configured**
   - What we know: cobra flag `--profile` can have default `""`, YAML config may have no profile set
   - What's unclear: Should the binary fall through to the SDK's default credential chain (no profile specified = use `[default]` section) or require explicit profile?
   - Recommendation: If `cfg.Profile == ""`, call `LoadDefaultConfig` without `WithSharedConfigProfile` — the SDK uses the `[default]` profile naturally. Document this in help text.

2. **STS GetCallerIdentity region requirement**
   - What we know: STS is a global service but the Go SDK still requires a region to construct an endpoint
   - What's unclear: Does calling STS without a region always fail, or is there a global endpoint?
   - Recommendation: Always require region before calling STS. If region is empty after loading, show a clear error rather than letting STS fail with an endpoint resolution error. The PITFALLS.md confirms this is a real failure mode.

3. **~/.duchess/ directory creation**
   - What we know: Config file lives at `~/.duchess/config`; directory may not exist on first run
   - What's unclear: Should Phase 1 create the directory if absent, or just skip config loading if missing?
   - Recommendation: Skip silently if `~/.duchess/config` does not exist (use all defaults + flags). Print a note if `--verbose` flag is added later. Do NOT create the directory in Phase 1 — that's a Phase 2+ concern with proper UX.

---

## Sources

### Primary (HIGH confidence)
- `https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/config` v1.32.10 — `LoadDefaultConfig`, `WithSharedConfigProfile`, `WithRegion`, credential chain order
- `https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/credentials/ssocreds` v1.19.10 — `InvalidTokenError` type, `errors.As` pattern, import path
- `https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/sts` v1.41.7 — `GetCallerIdentity`, response fields (`Account`, `Arn`, `UserId`)
- `https://pkg.go.dev/github.com/spf13/cobra` v1.10.2 — root command pattern, `PersistentFlags`, `Changed()` method
- `https://pkg.go.dev/gopkg.in/yaml.v3` v3.0.1 — `yaml.Unmarshal`, struct tags, `omitempty`
- `.planning/research/STACK.md` — full version table, go.mod upgrade actions, architectural patterns
- `.planning/research/PITFALLS.md` — SSO expiry (Pitfall 3), role chain cap (Pitfall 8), INI prefix rules (Pitfall 6), GetCallerIdentity async (Pitfall 16), region not set (Pitfall 15)

### Secondary (MEDIUM confidence)
- `.planning/ROADMAP.md` — Phase 1 plan breakdown (01-01 cobra + config, 01-02 AWS session + STS); confirmed two-plan structure
- `go.mod` (project file) — confirmed existing direct/indirect deps, exact current versions

### Tertiary (LOW confidence)
- None for this phase — all claims verified against official sources or project files

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all versions verified on pkg.go.dev as of 2026-03-03
- Architecture: HIGH — patterns sourced from SDK official docs and STACK.md research (also 2026-03-03)
- Pitfalls: HIGH — sourced from AWS official docs and PITFALLS.md research (2026-03-03)

**Research date:** 2026-03-03
**Valid until:** 2026-04-03 (stable libraries; AWS SDK v2 releases frequently but API patterns are stable)
