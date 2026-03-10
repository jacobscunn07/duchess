# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.0 — MVP

**Shipped:** 2026-03-10
**Phases:** 6 | **Plans:** 14 | **Timeline:** 5 days (2026-03-03 → 2026-03-07)

### What Was Built

- Go binary with 3-mode AWS authentication (named profiles, role assumption, SSO) via SDK credential chain + STS GetCallerIdentity with actionable error taxonomy
- Bubble Tea TUI with persistent status bar (version, profile, region, IAM principal, live clock), loading spinner, and inline error rendering
- Full S3 navigation: bucket list → prefix hierarchy → object metadata, with breadcrumb, server-side prefix filter (`/`), and live refresh
- Full ECS navigation: cluster → service → task → task detail, with lazy hierarchical fetch, per-level refresh tickers, and CloudWatch log URL deep-links (`L` key)
- Configurable refresh interval threaded through all panels, observable in status bar left group and breadcrumb timestamps
- In-session profile and region switching via modal overlays with per-session context cancellation and SSO error surfacing

### What Worked

- **Dependency-graph phasing** — The 5-phase structure (Foundation → UI Shell → S3 → ECS → Overlays) matched the natural dependency graph perfectly. No phase had to wait for work it depended on. No backtracking.
- **Controller→Panel architecture** — rootModel routing all messages to both panels with unexported typed message structs prevented cross-panel interference structurally. No special routing needed.
- **Audit-first milestone close** — Running the milestone audit before completing surfaced SUMMARY.md frontmatter gaps and the stale ROADMAP.md checkboxes. Confirms the audit step is valuable even when confident.
- **Decimal phases for gap closure** — Phase 4.1 inserted cleanly between Phase 4 and Phase 5. The decimal naming convention made ordering unambiguous.
- **Client package isolation** — `s3/client.go` and `ecs/client.go` owning all SDK calls (panel models fire `FetchXxxCmd` closures, never import SDK directly) kept panel logic SDK-free and testable.

### What Was Inefficient

- **SUMMARY.md frontmatter not populated for Phase 01** — 8 requirements satisfied by VERIFICATION.md evidence but missing from SUMMARY frontmatter, creating audit tooling mismatch. Should populate frontmatter at plan-complete time, not retroactively.
- **`milestone complete` CLI recorded 0 accomplishments** — Because SUMMARY.md `one_liner` fields weren't populated in the structured frontmatter, the CLI extracted nothing. Accomplishments had to be written manually. Fix: populate `one_liner` in plan SUMMARY.md at completion.
- **14 deferred human-verification items** — Live AWS / PTY checks accumulated across all phases. These require access to real credentials and a real terminal, so they can't be automated. A dedicated "pre-ship smoke test" checklist would make this backlog explicit and actionable for v1.1.
- **UAT identified 2 issues after Phase 5 plans completed** — Required a gap-closure plan (05-03). Phase-level UAT runs should happen before the phase is marked Complete, not after all plans finish.

### Patterns Established

- **Value receivers + `(Model, tea.Cmd)` return from helpers** — All ascend/descend helpers return the updated model explicitly. This is the correct Go Bubble Tea convention and prevents copy-mutation loss.
- **Per-session context cancellation** — `cancelSession` called before creating new `sessionCtx` on every profile switch. `context.Canceled` guard placed FIRST in all 7 error handlers. This pattern is the correct way to handle rapid switching in Bubble Tea.
- **ssocreds.InvalidTokenError before smithy.APIError** — Critical ordering in `ClassifyCredentialError`. Document this explicitly in future auth-adjacent code.
- **HeadBucket over GetBucketLocation** — `GetBucketLocation` returns null for us-east-1 (AWS API bug). HeadBucket is the correct approach for bucket region detection.
- **Delimiter="/" mandatory in ListObjectsV2** — Without it, S3 returns all objects recursively (potentially millions). Always set delimiter.

### Key Lessons

1. **Populate SUMMARY.md frontmatter at plan completion, not retroactively.** The `one_liner`, `requirements_completed`, and other fields are used by downstream tooling (milestone complete, audit). Leaving them empty creates manual cleanup work.
2. **Run phase-level UAT before marking a phase complete.** Don't let UAT issues accumulate until milestone audit time. Each phase should be UAT-verified before moving to the next.
3. **The `milestone complete` CLI needs populated frontmatter to be useful.** If SUMMARYs don't have `one_liner` fields, it records "(none recorded)". Make structured frontmatter a plan-complete requirement.
4. **Decimal phases work well for inserted gap closure.** Phase 4.1 slotted between Phase 4 and Phase 5 without disrupting numbering. Use this pattern freely for urgent insertions.
5. **Charmbracelet v1.x is the correct stack (not v2).** v2 uses `charm.land/` import paths with breaking API changes. Document this clearly for any future contributors.

### Cost Observations

- Model mix: 100% Sonnet (claude-sonnet-4-6, quality profile)
- Sessions: 5 days of active development (all phases executed in a single continuous milestone)
- Notable: Very fast average plan execution (~3-14 min per plan) with parallel wave execution in most phases. ECS panel model (04-02) was the longest at ~25 min including a human checkpoint pause.

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Phases | Plans | Key Change |
|-----------|--------|-------|------------|
| v1.0 | 6 | 14 | Initial baseline — 5-phase MVP structure |

### Cumulative Quality

| Milestone | Requirements | Satisfied | Gaps |
|-----------|-------------|-----------|------|
| v1.0 | 36 | 36/36 | 0 |

### Top Lessons (Verified Across Milestones)

1. *(Accumulates after v1.1)*
