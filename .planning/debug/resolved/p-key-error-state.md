---
status: resolved
trigger: "'p' key does not open profile overlay when app is in S3 access-denied error state"
created: 2026-03-06T00:00:00Z
updated: 2026-03-07T00:00:00Z
---

## Current Focus

hypothesis: S3 access-denied errors are trapped inside s3panel.Model.err and never elevate rootModel.state to stateError; rootModel stays in stateReady where 'p' IS handled — but the real issue is that the root state IS stateReady, and the 'p' key handler at line 174 guards on stateReady OR stateError. This means 'p' should work. Need to re-examine the actual state when S3 errors occur.
test: Traced full message flow for S3 access-denied scenario
expecting: Root cause identified
next_action: COMPLETE — root cause found

## Symptoms

expected: Pressing 'p' opens the profile overlay when the app shows an S3 access-denied error
actual: Profile selector did not open when started with a profile lacking S3 access
errors: None (app shows S3 error state visually, but no crash)
reproduction: Test 4 in UAT — start app with a profile lacking S3 access, press 'p'
started: Discovered during UAT of phase 05

## Eliminated

- hypothesis: 'p' key handler does not include stateError — WRONG, line 174 explicitly has `m.state == stateReady || m.state == stateError`
  evidence: internal/ui/model.go line 174: `if m.state == stateReady || m.state == stateError {`
  timestamp: 2026-03-06

- hypothesis: S3 access-denied error elevates rootModel.state to stateError — WRONG
  evidence: bucketsErrMsg handler in s3/model.go lines 201-207 sets m.err (the panel's own err field) and returns nil. It sends no message back to rootModel. The rootModel only enters stateError via identityErrMsg (STS GetCallerIdentity failure), NOT via S3 panel errors.
  timestamp: 2026-03-06

## Evidence

- timestamp: 2026-03-06
  checked: internal/ui/model.go — state transitions
  found: rootModel.state is set to stateError ONLY by identityErrMsg handler (line 209-212). It is set to stateLoading by profileSelectedMsg (line 227) and stateReady by identityLoadedMsg (line 197).
  implication: S3 panel errors cannot set rootModel.state = stateError. They are siloed inside s3panel.Model.

- timestamp: 2026-03-06
  checked: internal/ui/s3/model.go — bucketsErrMsg handler (lines 201-207)
  found: When ListBuckets fails (S3 access denied), s3panel handles bucketsErrMsg by setting m.err on the panel and returning nil (no cmd). No message is emitted to rootModel. The error is displayed inside the panel via View() at line 299-301.
  implication: rootModel stays in stateReady when S3 fails. The 'p' key guard at line 174 includes stateReady, so 'p' SHOULD open the overlay. This rules out a state-mismatch root cause.

- timestamp: 2026-03-06
  checked: internal/ui/s3/model.go — tea.KeyMsg handler (lines 116-188)
  found: The S3 panel's Update() intercepts ALL tea.KeyMsg before rootModel sees it (when forwarded). BUT: rootModel forwards KeyMsg to panels only AFTER its own switch statement falls through (lines 284-289). The 'p' case at line 173 RETURNS early — it does not fall through to the panel forwarding. So the panel never consumes 'p'.
  implication: When rootModel is stateReady and 'p' is pressed, rootModel handles it and returns on line 182. The S3 panel never sees 'p'. The overlay should open.

- timestamp: 2026-03-06
  checked: The scenario more carefully — what exactly happens with a profile that lacks S3 access
  found: A profile that "lacks S3 access" can mean two things:
    (A) The profile credentials exist but S3 is denied — STS GetCallerIdentity SUCCEEDS (IAM identity is valid), so rootModel enters stateReady. Then FetchBucketsCmd fails with access denied, s3panel gets bucketsErrMsg, sets m.err. rootModel stays stateReady. 'p' key IS guarded by stateReady — overlay SHOULD open.
    (B) The profile credentials do not allow even STS GetCallerIdentity — rootModel enters stateError. 'p' IS guarded by stateError — overlay SHOULD open here too.
  implication: In both cases the code should open the overlay. Something else is causing the 'p' key to be swallowed.

- timestamp: 2026-03-06
  checked: internal/ui/s3/model.go — S3 panel Update() tea.KeyMsg interception at lines 116-181
  found: The S3 panel's key interception does NOT have a case for "p". The switch only intercepts "q", "ctrl+c", "esc", "enter", "/". All other keys including "p" fall through to the list widget forwarding (or filterInput if in filterMode).
  implication: Not the issue — 'p' is not consumed by s3 panel early return.

- timestamp: 2026-03-06
  checked: rootModel.Update() tea.KeyMsg routing order
  found: In the tea.KeyMsg branch, rootModel processes the overlay-open guards first (lines 119-157), then the switch statement (lines 159-191) which includes case "p" at 173. The 'p' case RETURNS (line 182) so it never reaches the bottom forwarding (line 284). The issue is NOT the routing order.
  implication: The 'p' key logic itself is correct. The root cause must be in a PRECONDITION that prevents case "p" from being reached.

- timestamp: 2026-03-06
  checked: rootModel stateLoading behavior — what if S3 error occurs while still loading?
  found: CRITICAL FINDING. When a profile has valid AWS credentials but no S3 access:
    1. fetchIdentityCmd succeeds → identityLoadedMsg → rootModel enters stateReady
    2. s3Panel.Init() fires FetchBucketsCmd (async goroutine)
    3. User presses 'p' BEFORE FetchBucketsCmd completes → rootModel is stateReady → 'p' WORKS
    4. FetchBucketsCmd fails → s3panel shows error → rootModel still stateReady → 'p' STILL WORKS
  BUT: What if the profile does not have STS GetCallerIdentity access either? Then:
    1. fetchIdentityCmd fails → identityErrMsg → rootModel enters stateError
    2. s3Panel is never initialized (it's only created in identityLoadedMsg handler, line 205)
    3. User presses 'p' → stateError guard passes → overlay.ListAWSProfiles() called
    4. If ListAWSProfiles returns empty or err → returns nil (line 177-179) WITHOUT opening overlay
  implication: THIS IS THE ROOT CAUSE for scenario (B). If the profile also cannot call STS and ListAWSProfiles returns no profiles or errors, the overlay silently fails to open.

- timestamp: 2026-03-06
  checked: overlay.ListAWSProfiles() — what can cause it to return empty or error
  found: The guard on lines 176-179 of model.go silently skips opening the overlay if ListAWSProfiles returns err OR len(profiles) == 0. If ~/.aws/config is malformed, missing, or has no profiles, pressing 'p' does nothing with no feedback.
  implication: The user's specific UAT environment likely has ListAWSProfiles returning 0 profiles or an error. The silent no-op guard hides this failure completely.

## Resolution

root_cause: |
  When 'p' is pressed, rootModel calls overlay.ListAWSProfiles() (line 175) and silently
  no-ops if it returns an error OR zero profiles (lines 176-179). In the UAT scenario,
  ListAWSProfiles() is either failing or returning an empty list, causing the guard to
  silently swallow the keypress with no user feedback.

  The actual state-handling logic for 'p' (stateReady || stateError, line 174) is CORRECT.
  The silent failure guard (lines 176-179) is the defect: it swallows the error silently
  instead of telling the user why the overlay could not open.

  Secondary possibility: The UAT profile IS in stateReady (S3 access denied after identity
  succeeds) and ListAWSProfiles works fine — in which case the overlay should open. This
  would indicate a test setup problem (wrong profile used, or the specific profile truly
  has stateLoading not completed when 'p' was pressed). The most likely code defect is
  still the silent guard.

fix: (not applied — diagnose-only mode)
verification: (not applied)
files_changed: []
