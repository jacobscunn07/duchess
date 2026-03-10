# duchess

## What This Is

duchess is a k9s-inspired Terminal User Interface (TUI) for AWS that lets engineers navigate AWS resources across profiles, accounts, and regions without browser tab juggling or CLI complexity. Switch profiles, switch regions, and explore S3 and ECS — all from the terminal with vim-style keybindings. v1.0 shipped as a working Go binary with ~3,000 LOC using the Charmbracelet (Bubble Tea + Lipgloss + Bubbles) TUI stack.

## Core Value

Navigate your AWS resources across accounts and regions in one terminal session — no console switching, no `--profile` flags, no context thrash.

## Requirements

### Validated

- ✓ User can switch between AWS profiles defined in ~/.aws/config — v1.0 (modal overlay, p key, no restart)
- ✓ User can switch between AWS regions — v1.0 (modal overlay, r key, ECS resets / S3 unchanged)
- ✓ App supports named profiles (access key/secret) — v1.0
- ✓ App supports role assumption profiles (role_arn + source_profile) — v1.0
- ✓ App supports AWS SSO profiles (sso_start_url / sso_account_id) — v1.0
- ✓ All navigation uses vim-style keybindings (j/k scroll, g/G top/bottom, / search, q quit, Esc back) — v1.0
- ✓ Status bar displays application version — v1.0
- ✓ Status bar displays current IAM principal (who you are) — v1.0
- ✓ Status bar displays current AWS region — v1.0
- ✓ Status bar displays current date and time — v1.0
- ✓ User can browse S3 buckets (all regions, read-only) — v1.0
- ✓ User can navigate bucket → prefix → object hierarchy — v1.0
- ✓ User can view object metadata (size, last modified, storage class) — v1.0
- ✓ User can browse ECS clusters in the current region — v1.0
- ✓ User can navigate cluster → service → task hierarchy — v1.0
- ✓ User can view task details (status, task definition, container info) — v1.0
- ✓ User can view service details (desired count, running count, launch type) — v1.0
- ✓ Screen auto-refreshes every 2 seconds (default) — v1.0 (configurable, default 2s)
- ✓ Refresh interval is configurable — v1.0 (`--refresh-interval` flag + config file key)
- ✓ Configuration can be set via CLI flags at startup — v1.0
- ✓ Configuration can be set via ~/.duchess/config file — v1.0
- ✓ CLI flags override config file values — v1.0

### Active

(None — planning next milestone. See `/gsd:new-milestone` to define v1.1 requirements.)

### Out of Scope

- S3 upload/download/delete — v1 is read-only; mutations in v2
- ECS exec into containers — v1 is read-only; exec in v2
- ECS restart/stop tasks — v1 is read-only; mutations in v2
- EC2, Lambda, CloudWatch Logs browsing — not in v1 scope; candidate for v2
- MFA prompting — deferred; SSO covers most modern setups
- Any IAM write operations — safety-first for open source
- Mouse navigation — target users are vim-native; keyboard-only is intentional
- Multi-region simultaneous view — one region at a time is correct for v1

## Context

- **v1.0 state:** ~3,036 Go LOC (production), 57 files, Charmbracelet stack (Bubble Tea v1.x, Lipgloss, Bubbles). All 36 v1 requirements satisfied. 14 human-in-the-loop verification items deferred (require live AWS credentials / PTY).
- **Architecture:** Controller→Panel pattern. rootModel routes messages to s3panel and ecspanel. Panels own state machines; client.go packages own SDK calls; typed tea.Cmd messages cross the boundary.
- **Existing repo:** Go module at `github.com/jacobscunn07/duchess`
- **Inspiration:** k9s — the Kubernetes TUI. Users already know the navigation model; duchess extends it to AWS
- **Audience:** Open source. Must work across diverse AWS configurations and setups
- **Safety constraint:** Read-only in v1 builds trust with open source users and eliminates the need for any IAM write permissions in the tool

## Constraints

- **Tech stack**: Go — existing project, already committed
- **TUI framework**: Charmbracelet ecosystem — Bubble Tea v1.x (Elm-style MVU), Lipgloss (styling), Bubbles (pre-built components). NOTE: use v1.x, NOT v2 — v2 uses charm.land/ import paths with breaking API changes.
- **Safety**: Read-only operations only in v1 — no writes, no deletes, no mutations
- **Config location**: `~/.duchess/config` — follows XDG-adjacent conventions
- **Auth scope**: Named profiles, role assumption, and SSO for v1; MFA deferred

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Read-only v1 | Safety first — open source users should trust the tool before it has write access | ✓ Good — zero incident risk during v1 development |
| Go | Existing project; Go is the dominant language for TUI tools (k9s, lazygit) | ✓ Good — fast compilation, static binary, easy distribution |
| Charmbracelet ecosystem (Bubble Tea v1.x + Lipgloss + Bubbles) | Modern, actively maintained Go TUI stack; Elm-style architecture is composable | ✓ Good — clean MVU architecture; v1.x was correct choice (v2 not yet stable) |
| S3 is global, region switch only affects regional services | S3 buckets span regions; ECS clusters are regional | ✓ Good — validated by implementation; S3 panel unchanged on region switch works correctly |
| Controller→Panel architecture | rootModel owns program lifecycle; panels own their state machines | ✓ Good — clean separation; both panels receive all messages with no cross-interference via unexported typed message structs |
| Lazy ECS fetch (no prefetch) | ECS has strict API rate limits (DescribeServices batched at 10, DescribeTasks at 100) | ✓ Good — no ThrottlingException in practice; cluster-list tier deferred until user descends |
| Per-session context cancellation (cancelSession) | Rapid profile switching could leave orphaned goroutines updating stale state | ✓ Good — context.Canceled guard in all 7 error handlers prevents stale state |
| Value receivers throughout panels | Standard Bubble Tea convention; state mutations return new model copy | ✓ Good — all ascend/descend helpers explicitly return (Model, tea.Cmd) |
| ssocreds.InvalidTokenError checked before smithy.APIError in ClassifyCredentialError | InvalidTokenError is not a smithy.APIError implementor — order is critical | ✓ Good — prevents silent fallthrough on SSO expiry |
| HeadBucket (not GetBucketLocation) for region detection | GetBucketLocation returns null for us-east-1 (documented AWS API bug) | ✓ Good — avoids null-region handling edge case |

---
*Last updated: 2026-03-10 after v1.0 milestone*
