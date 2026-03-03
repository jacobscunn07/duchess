---
phase: 01-foundation
plan: "01"
subsystem: cli-config
tags: [cobra, yaml, config, cli, tdd]
dependency_graph:
  requires: []
  provides: [LoadConfig, Config, cobra-root-cmd, main-entrypoint]
  affects: [01-02]
tech_stack:
  added:
    - github.com/spf13/cobra@v1.10.2
    - gopkg.in/yaml.v3@v3.0.1 (promoted from indirect to direct)
    - github.com/aws/aws-sdk-go-v2@v1.41.2 (upgraded)
    - github.com/aws/aws-sdk-go-v2/config@v1.32.10 (upgraded)
    - github.com/aws/aws-sdk-go-v2/service/sts@v1.41.7 (upgraded)
  patterns:
    - Flag-over-file precedence with cmd.Flags().Changed() guards
    - TDD Red-Green with t.TempDir() + t.Setenv() isolation
    - Cobra PersistentFlags on root command for inherited flags
key_files:
  created:
    - internal/config/config.go
    - internal/config/config_test.go
    - cmd/root.go
    - main.go
  modified:
    - go.mod
    - go.sum
decisions:
  - "Use cmd.Flags() (not PersistentFlags()) in test helper to allow Changed() to work without a full Execute() call — during real Execute(), cobra merges persistent flags into Flags() so the LoadConfig implementation using cmd.Flags().Changed() works correctly in production"
  - "AWS SDK deps re-added after go mod tidy removed them (no imports yet in Plan 01) — kept as direct deps per plan success criteria; Plan 02 will import them directly"
metrics:
  duration: "4 minutes"
  completed: "2026-03-03T20:04:34Z"
  tasks_completed: 2
  files_created: 4
  files_modified: 2
---

# Phase 1 Plan 1: cobra CLI skeleton and config loading Summary

**One-liner:** cobra root command with --profile/--region/--refresh-interval flags and YAML config loading using cmd.Flags().Changed() flag-over-file precedence.

## What Was Built

### Task 1: Config struct and LoadConfig (TDD)

**internal/config/config.go**
- `Config` struct with yaml struct tags: `Profile string`, `Region string`, `RefreshInterval int`
- `LoadConfig(cmd *cobra.Command) (*Config, error)` function:
  1. Starts with defaults: `RefreshInterval: 2`, `Profile: ""`, `Region: ""`
  2. Resolves `~/.duchess/config` path via `os.UserHomeDir()` (never string `~` expansion)
  3. Reads and yaml-unmarshals the config file; silently ignores missing file; wraps YAML parse errors with `"parse config: ..."` prefix
  4. Applies flag values ONLY when `cmd.Flags().Changed("flag-name")` is true — preventing unset flags with defaults from overriding file values

**internal/config/config_test.go**
- 8 tests using TDD Red-Green cycle
- Uses `t.TempDir()` for isolated home directories; `t.Setenv("HOME", ...)` to redirect `os.UserHomeDir()`
- Tests: defaults, file loading, profile/region/refresh-interval flag overrides, unset flags don't override, missing file silently ignored, malformed YAML error

### Task 2: cobra root command and main.go

**cmd/root.go**
- `rootCmd` with `Use: "duchess"`, `Short: "Navigate your AWS resources..."`, `RunE: runRoot`
- Three `PersistentFlags()`: `--profile`, `--region`, `--refresh-interval` (default 2)
- `runRoot` calls `config.LoadConfig(cmd)` and prints config values (Phase 1 placeholder; Phase 2 replaces with Bubble Tea launch)
- `Execute() error` exported function for main.go

**main.go**
- Single function: calls `cmd.Execute()`, exits 1 on error

**go.mod updates**
- cobra v1.10.2 — direct dependency (new)
- yaml.v3 v3.0.1 — promoted from indirect to direct
- aws-sdk-go-v2 upgraded: core v1.41.2, config v1.32.10, sts v1.41.7

## Decisions Made

### Decision 1: Cobra flag registration in tests uses Flags() not PersistentFlags()

**Context:** In tests, calling `cmd.Flags().Set("profile", "value")` fails with "no such flag" when the flag is registered via `PersistentFlags()`. `cmd.PersistentFlags().Set()` works but `cmd.Flags().Changed()` returns false.

**Root cause:** Cobra's `Flags()` and `PersistentFlags()` are separate `pflag.FlagSet` instances. During a real `cobra.Execute()`, cobra merges persistent flags into the active command's `Flags()` set — making `cmd.Flags().Changed()` see persistent flags. This merge doesn't happen when constructing a command manually in tests.

**Decision:** Test helper `newTestCmd()` registers flags via `cmd.Flags()` (not `PersistentFlags()`). The production `rootCmd` uses `PersistentFlags()` as intended. The implementation uses `cmd.Flags().Changed()` which works correctly in both contexts. Documented in test file comment.

### Decision 2: AWS SDK deps kept in go.mod after Plan 01

**Context:** `go mod tidy` removed AWS SDK deps since no Plan 01 code imports them. The plan's success criteria requires them upgraded in go.mod.

**Decision:** Re-added as direct deps after tidy. Plan 02 will import them with actual code. This avoids Plan 02 needing to re-add deps and matches the plan's stated success criteria.

## Test Results

```
ok  github.com/jacobscunn07/duchess/internal/config  0.314s
--- PASS: TestLoadConfig_Defaults
--- PASS: TestLoadConfig_FileValues
--- PASS: TestLoadConfig_ProfileFlagOverridesFile
--- PASS: TestLoadConfig_RegionFlagOverridesFile
--- PASS: TestLoadConfig_RefreshIntervalFlagOverridesFile
--- PASS: TestLoadConfig_UnsetFlagsDoNotOverrideFile
--- PASS: TestLoadConfig_MissingFileIgnored
--- PASS: TestLoadConfig_MalformedYAMLReturnsError
```

## Verification Results

1. `go build ./...` exits 0
2. `go test ./internal/config/... -v` — 8/8 tests pass
3. `go run . --help` shows --profile, --region, --refresh-interval
4. `go run . --profile testprofile --region us-west-2` prints `Profile: testprofile`, `Region: us-west-2`, `RefreshInterval: 2`
5. go.mod has cobra v1.10.2 and yaml.v3 v3.0.1 as direct deps

## Downstream Contract (for Plan 02)

Plan 02 imports `LoadConfig` and `Config` from `github.com/jacobscunn07/duchess/internal/config`:

```go
import "github.com/jacobscunn07/duchess/internal/config"

// Signature
func LoadConfig(cmd *cobra.Command) (*Config, error)

type Config struct {
    Profile         string // "" if not set
    Region          string // "" if not set
    RefreshInterval int    // default 2
}
```

Plan 02's `runRoot` will call `LoadConfig(cmd)` then pass `cfg.Profile` and `cfg.Region` to the AWS session factory (`internal/aws/session.go`).

## Commits

| Hash | Description |
|------|-------------|
| 9430cfb | test(01-01): add failing tests for LoadConfig config struct |
| d609fce | feat(01-01): implement Config struct and LoadConfig with flag-over-file precedence |
| aff80eb | feat(01-01): add cobra root command, main.go entrypoint, and go.mod updates |

## Self-Check: PASSED

All created files exist on disk. All commits verified in git log.

| Check | Result |
|-------|--------|
| main.go | FOUND |
| cmd/root.go | FOUND |
| internal/config/config.go | FOUND |
| internal/config/config_test.go | FOUND |
| 01-01-SUMMARY.md | FOUND |
| commit 9430cfb | FOUND |
| commit d609fce | FOUND |
| commit aff80eb | FOUND |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed cobra flag Changed() behavior in tests**
- **Found during:** Task 1 GREEN phase
- **Issue:** Test helper registered flags with `PersistentFlags()` but called `cmd.Flags().Set()` which returns "no such flag". Using `PersistentFlags().Set()` succeeds but `cmd.Flags().Changed()` still returns false without a full `Execute()` run.
- **Fix:** Changed test helper to use `cmd.Flags()` for flag registration. The implementation's `cmd.Flags().Changed()` works correctly in production (cobra merges persistent flags into Flags() during Execute) and in tests (flags registered directly in Flags()).
- **Files modified:** internal/config/config_test.go
- **Commit:** d609fce (included in GREEN commit)
