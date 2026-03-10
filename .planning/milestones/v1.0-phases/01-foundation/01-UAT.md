---
status: complete
phase: 01-foundation
source: 01-01-SUMMARY.md, 01-02-SUMMARY.md
started: 2026-03-03T20:15:00Z
updated: 2026-03-03T20:20:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Help flag shows correct flags
expected: Run `go run . --help` — output shows --profile, --region, and --refresh-interval flags with defaults
result: pass

### 2. Flag values are respected
expected: Run `go run . --profile myprofile --region us-west-2` — prints `Profile: myprofile` and `Region: us-west-2` (and RefreshInterval: 2)
result: pass

### 3. Config file loading with flag precedence
expected: Create `~/.duchess/config` with `profile: fileprofile`, then run `go run .` (no flags) → uses file value. Run again with `--profile cliprofile` → CLI flag wins over file.
result: pass

### 4. Valid AWS profile prints caller identity
expected: Run `duchess --profile <your-real-profile>` — prints two lines: `Account: 123456789012` and `ARN: arn:aws:iam::123456789012:user/yourname`
result: pass

### 5. Invalid/nonexistent profile shows human-readable error
expected: Run `go run . --profile doesnotexist` — prints a readable error message (not a raw SDK stack trace). Should mention the profile name and what to do.
result: pass

### 6. SSO expiry message (if applicable)
expected: If you have an expired SSO profile, running `duchess --profile <expired-sso-profile>` shows: `SSO session expired. Run: aws sso login --profile <name>` — not a raw SDK error.
result: skipped
reason: No expired SSO session available to test

## Summary

total: 6
passed: 5
issues: 0
pending: 0
skipped: 1

## Gaps

[none yet]
