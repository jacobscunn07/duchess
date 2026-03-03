# duchess

## What This Is

duchess is a k9s-inspired Terminal User Interface (TUI) for AWS that lets engineers navigate AWS resources across profiles, accounts, and regions without browser tab juggling or CLI complexity. Switch profiles, switch regions, and explore services like S3 and ECS — all from the terminal with vim-style keybindings.

## Core Value

Navigate your AWS resources across accounts and regions in one terminal session — no console switching, no `--profile` flags, no context thrash.

## Requirements

### Validated

(None yet — ship to validate)

### Active

**Authentication & Navigation**
- [ ] User can switch between AWS profiles defined in ~/.aws/config
- [ ] User can switch between AWS regions
- [ ] App supports named profiles (access key/secret)
- [ ] App supports role assumption profiles (role_arn + source_profile)
- [ ] App supports AWS SSO profiles (sso_start_url / sso_account_id)
- [ ] All navigation uses vim-style keybindings (j/k scroll, g/G top/bottom, / search, q quit, Esc back)

**Status Bar**
- [ ] Status bar displays application version
- [ ] Status bar displays current IAM principal (who you are)
- [ ] Status bar displays current AWS region
- [ ] Status bar displays current date and time

**S3 (Global)**
- [ ] User can browse S3 buckets (all regions, read-only)
- [ ] User can navigate bucket → prefix → object hierarchy
- [ ] User can view object metadata (size, last modified, storage class)

**ECS (Regional)**
- [ ] User can browse ECS clusters in the current region
- [ ] User can navigate cluster → service → task hierarchy
- [ ] User can view task details (status, task definition, container info)
- [ ] User can view service details (desired count, running count, launch type)

**Refresh & Config**
- [ ] Screen auto-refreshes every 2 seconds (default)
- [ ] Refresh interval is configurable
- [ ] Configuration can be set via CLI flags at startup
- [ ] Configuration can be set via ~/.duchess/config file
- [ ] CLI flags override config file values

### Out of Scope

- S3 upload/download/delete — v1 is read-only; mutations in v2
- ECS exec into containers — v1 is read-only; exec in v2
- ECS restart/stop tasks — v1 is read-only; mutations in v2
- EC2, Lambda, CloudWatch Logs — not in v1 scope
- MFA prompting — deferred; SSO covers most modern setups
- Any IAM write operations — safety-first for open source

## Context

- **Existing repo**: Go module already initialized (go.mod / go.sum present)
- **Inspiration**: k9s — the Kubernetes TUI. Users already know the navigation model; duchess extends it to AWS
- **Audience**: Open source. Must work across diverse AWS configurations and setups
- **Safety constraint**: Read-only in v1 builds trust with open source users and eliminates the need for any IAM write permissions in the tool

## Constraints

- **Tech stack**: Go — existing project, already committed
- **TUI framework**: Charmbracelet ecosystem — Bubble Tea (Elm-style MVU), Lipgloss (styling), Bubbles (pre-built components)
- **Safety**: Read-only operations only in v1 — no writes, no deletes, no mutations
- **Config location**: `~/.duchess/config` — follows XDG-adjacent conventions
- **Auth scope**: Named profiles, role assumption, and SSO for v1; MFA deferred

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Read-only v1 | Safety first — open source users should trust the tool before it has write access | — Pending |
| Go | Existing project; Go is the dominant language for TUI tools (k9s, lazygit) | — Pending |
| Charmbracelet ecosystem (Bubble Tea + Lipgloss + Bubbles) | Modern, actively maintained Go TUI stack; Elm-style architecture is composable; Lipgloss handles styling cleanly | — Pending |
| S3 is global, region switch only affects regional services | S3 buckets span regions; ECS clusters are regional — region picker filters regional services | — Pending |

---
*Last updated: 2026-03-02 after initialization*
