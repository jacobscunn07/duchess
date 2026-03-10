---
phase: 01-foundation
plan: "02"
subsystem: aws-session
tags: [aws, sts, ssocreds, smithy, error-taxonomy, tdd]
dependency_graph:
  requires: [01-01]
  provides: [NewAWSConfig, GetCallerIdentity, ClassifyCredentialError]
  affects: [02-01]
tech_stack:
  added:
    - github.com/aws/aws-sdk-go-v2 v1.41.2 (promoted to direct)
    - github.com/aws/aws-sdk-go-v2/config v1.32.10 (promoted to direct)
    - github.com/aws/aws-sdk-go-v2/credentials v1.19.10 (promoted to direct)
    - github.com/aws/aws-sdk-go-v2/service/sts v1.41.7 (promoted to direct)
    - github.com/aws/smithy-go v1.24.1 (promoted to direct)
  patterns:
    - ssocreds.InvalidTokenError checked before smithy.APIError (error chain order)
    - Package named "session" to avoid shadowing SDK's "aws" package
    - errors.As for error unwrapping through SDK wrapper types
    - fmt.Errorf with %w for wrapping generic errors (preserves errors.Is chain)
key_files:
  created:
    - internal/aws/session.go
    - internal/aws/session_test.go
  modified:
    - cmd/root.go
    - go.mod
    - go.sum
decisions:
  - "Package named 'session' (not 'aws') to avoid shadowing the SDK's top-level 'aws' package — import alias used in cmd/root.go: session \"github.com/jacobscunn07/duchess/internal/aws\""
  - "ssocreds.InvalidTokenError checked FIRST in ClassifyCredentialError because it is a struct type, not a smithy.APIError implementor — checking APIError first would miss it"
  - "errors.As used for all error checks to handle SDK wrapping via smithy.OperationError"
metrics:
  duration: "3 minutes"
  completed: "2026-03-03T20:09:20Z"
  tasks_completed: 2
  files_created: 2
  files_modified: 2
---

# Phase 1 Plan 2: AWS Session Factory and Credential Error Taxonomy Summary

**One-liner:** AWS session factory using LoadDefaultConfig with 5-case credential error taxonomy (SSO expiry, token expiry, bad access key, access denied, generic fallback) wired into cobra root command via STS GetCallerIdentity.

## What Was Built

### Task 1: AWS session factory and credential error taxonomy (TDD)

**internal/aws/session.go** (package `session`)

Three exported functions:

**`NewAWSConfig(ctx context.Context, profile, region string) (aws.Config, error)`**
- Builds `[]func(*config.LoadOptions) error` opts slice
- Appends `config.WithSharedConfigProfile(profile)` only if `profile != ""`
- Appends `config.WithRegion(region)` only if `region != ""`
- Calls `config.LoadDefaultConfig(ctx, opts...)` — supports named, role-assumption, and SSO profiles through SDK's unified config loading
- On load error: returns `ClassifyCredentialError(err, profile)`
- After successful load: if `cfg.Region == ""` returns actionable error instructing `--region` flag use

**`GetCallerIdentity(ctx context.Context, cfg aws.Config) (account, arn string, err error)`**
- Creates `sts.NewFromConfig(cfg)` client
- Calls `GetCallerIdentity` and returns `aws.ToString` of Account and Arn fields

**`ClassifyCredentialError(err error, profile string) error`**
- nil check first
- `ssocreds.InvalidTokenError` check BEFORE `smithy.APIError` — order is critical
- `smithy.APIError` switch: ExpiredTokenException, InvalidClientTokenId/AuthFailure, AccessDenied
- Fallback: `fmt.Errorf(...%w...)` to preserve error chain

**internal/aws/session_test.go**

9 tests with `mockAPIError` type satisfying `smithy.APIError` interface:
- 7 `TestClassifyCredentialError_*` tests (all error taxonomy cases including nil)
- `TestNewAWSConfig_EmptyProfile` — no panic verification
- `TestNewAWSConfig_NonexistentProfile` — confirms error for unknown profile

### Task 2: Wire AWS session and identity into cmd/root.go

Updated `runRoot` flow:
1. `config.LoadConfig(cmd)` — existing config loading
2. `session.NewAWSConfig(cmd.Context(), cfg.Profile, cfg.Region)` — AWS config
3. `session.GetCallerIdentity(cmd.Context(), awsCfg)` — STS call
4. On GetCallerIdentity error: `session.ClassifyCredentialError(err, cfg.Profile)` — no raw error
5. `fmt.Printf("Account: %s\nARN:     %s\n", account, arn)` — prints identity

Import alias `session "github.com/jacobscunn07/duchess/internal/aws"` avoids naming conflict.

## Error Taxonomy

| Error Type | Trigger | Message Pattern |
|-----------|---------|-----------------|
| `ssocreds.InvalidTokenError` | SSO token expired/missing | `"SSO session expired. Run: aws sso login --profile X"` |
| `ExpiredTokenException` (smithy APIError) | Temp credentials expired | `"credentials expired for profile X. Refresh your credentials and try again."` |
| `InvalidClientTokenId` (smithy APIError) | Bad access key ID | `"invalid credentials for profile X. Check your access key and secret key."` |
| `AuthFailure` (smithy APIError) | Bad access key/secret | `"invalid credentials for profile X. Check your access key and secret key."` |
| `AccessDenied` (smithy APIError) | Insufficient IAM permissions | `"access denied for profile X. Check IAM permissions."` |
| Any other error | Unknown auth failure | `"authentication failed for profile X: <wrapped original>"` |

## Verification Results

1. `go build ./...` exits 0 — no compilation errors
2. `go test ./internal/aws/... -v` — 9/9 tests pass
3. `go test ./...` — all tests pass (config + aws packages)
4. `go run . --help` shows all flags without error
5. Manual auth test: Not run (no real AWS profile in this environment — NewAWSConfig_NonexistentProfile confirms the error path is wired correctly)

## Key Decisions

### Decision 1: Package name "session" to avoid SDK shadowing

**Context:** The plan initially suggested package name "aws" noting it might shadow the SDK — then corrected to "session". The internal package path is `internal/aws/` (directory), but the Go package declaration is `package session`.

**Why it matters:** If the package were named `aws`, any file importing both the SDK's `github.com/aws/aws-sdk-go-v2/aws` and this package would need aliasing. Using `session` as the package name keeps imports clean: `session "github.com/jacobscunn07/duchess/internal/aws"` in root.go.

### Decision 2: errors.As order in ClassifyCredentialError

**Context:** `ssocreds.InvalidTokenError` is a concrete struct type (`*InvalidTokenError`), not a `smithy.APIError`. Checking `smithy.APIError` first would work for most cases, but if a future version of the SSO provider happens to also implement `APIError`, the wrong branch could fire.

**Decision:** Check `ssocreds.InvalidTokenError` FIRST, unconditionally, before the smithy branch. This matches the plan's explicit requirement and is more correct semantically.

## Commits

| Hash | Description |
|------|-------------|
| bcf18f8 | test(01-02): add failing tests for AWS session factory and credential error taxonomy |
| dfd435f | feat(01-02): implement AWS session factory, GetCallerIdentity, and credential error taxonomy |
| 859605e | feat(01-02): wire AWS session and GetCallerIdentity into cmd/root.go |

## Self-Check: PASSED

| Check | Result |
|-------|--------|
| internal/aws/session.go | FOUND |
| internal/aws/session_test.go | FOUND |
| cmd/root.go (modified) | FOUND |
| commit bcf18f8 | FOUND |
| commit dfd435f | FOUND |
| commit 859605e | FOUND |

## Deviations from Plan

None — plan executed exactly as written.
