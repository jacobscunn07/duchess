# Phase 2: UI Shell - Context

**Gathered:** 2026-03-03
**Status:** Ready for planning

<domain>
## Phase Boundary

A Bubble Tea TUI that launches cleanly, displays real AWS identity in a persistent status bar, handles terminal resize, and exits gracefully. No resource panels yet — this phase is pure shell scaffolding.

Delivers:
- Bubble Tea `tea.Program` wired into `cmd/root.go` replacing the current `fmt.Printf` approach
- Root model with `WindowSizeMsg` handling, `NO_COLOR` support, and `q` to quit
- Persistent status bar showing: version, profile name, region, IAM principal (account + ARN from STS), and live clock
- Async STS `GetCallerIdentity` as a `tea.Cmd` with loading spinner while in-flight
- Inline error rendering in the content area if the identity fetch fails

</domain>

<decisions>
## Implementation Decisions

### Status bar placement
- Single status bar at the bottom of the terminal (not top) — bottom placement is consistent with most terminal tools and leaves the top for future breadcrumb headers (Phase 3+)
- Status bar spans the full terminal width

### Status bar content layout
- Left side: `version | profile | region`
- Right side: `identity (account:role/user)` and `clock`
- Vertical separator `|` between fields; left/right groups pushed to edges with lipgloss flex layout
- Clock shows 24h format with seconds: `HH:MM:SS`; no explicit timezone label (local system time)

### Main viewport placeholder
- While loading: spinner centered in the content area (not just the status bar)
- After identity loads: empty content area — no welcome message, no menu hints
- The empty state is intentional: Phase 2 is scaffolding; future phases add panels into this space

### Inline error format
- When STS fetch fails: render the `ClassifyCredentialError` message as plain text in the content area, left-aligned, in a neutral color (no red styling in v1 — keep it simple and accessible)
- Error does not crash the program; `q` still works

### Quitting
- `q` key quits from any view — no confirmation prompt
- `ctrl+c` also quits cleanly (default Bubble Tea behavior via `tea.Quit`)

### Claude's Discretion
- Exact lipgloss color scheme, padding, and margin values
- Spinner style (choose from bubbles spinner component)
- Exact string format of the IAM principal in the status bar (e.g., `123456789012 / arn:aws:sts::123456789012:assumed-role/...` vs. truncated ARN)
- NO_COLOR handling (strip all lipgloss styling when env var is set)

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/aws/session.go`: `NewAWSConfig` + `GetCallerIdentity` — ready to be wrapped as a `tea.Cmd`; returns `(account, arn string, err error)`
- `internal/aws/session.go`: `ClassifyCredentialError` — already converts raw SDK errors to human-readable messages; use this output directly in the inline error display
- `internal/config/config.go`: `LoadConfig` — provides `cfg.Profile`, `cfg.Region`, `cfg.RefreshInterval` at startup; pass these into the root model's initial state

### Established Patterns
- cobra `RunE` in `cmd/root.go`: Phase 2 replaces the `fmt.Printf` block with a `tea.NewProgram(model).Run()` call — same entry point, just different output
- `internal/aws` package name is "session" (aliased as `session` in imports) — do not rename; note this alias when adding more aws packages
- Error handling: return `error` from `RunE`, don't `os.Exit` directly — cobra handles exit code

### Integration Points
- `cmd/root.go` `runRoot` function: swap `fmt.Printf(...)` for `p := tea.NewProgram(newRootModel(cfg, awsCfg)); return p.Run()`
- Initial root model must receive `cfg.Profile`, `cfg.Region`, and `aws.Config` so the status bar can render profile/region immediately (before STS responds)
- `GetCallerIdentity` becomes an async `tea.Cmd` fired from `Init()` — result delivered as a custom `identityLoadedMsg`

</code_context>

<specifics>
## Specific Ideas

- k9s as the visual reference: clean, monochrome-friendly status bar that works in 256-color and no-color terminals
- The status bar should be readable in a narrow terminal (80 cols) — truncate the ARN if needed rather than wrapping
- `bubbles/spinner` component for the loading state — matches the Charmbracelet ecosystem already locked in PROJECT.md

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 02-ui-shell*
*Context gathered: 2026-03-03*
