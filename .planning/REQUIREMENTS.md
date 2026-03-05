# Requirements: duchess

**Defined:** 2026-03-02
**Core Value:** Navigate your AWS resources across accounts and regions in one terminal session — no console switching, no `--profile` flags, no context thrash.

## v1 Requirements

Requirements for initial release. All operations are read-only.

### Authentication

- [ ] **AUTH-01**: User can switch between AWS profiles defined in ~/.aws/config without restarting the app
- [x] **AUTH-02**: App authenticates with named profiles (access key + secret key)
- [x] **AUTH-03**: App authenticates with role assumption profiles (role_arn + source_profile)
- [x] **AUTH-04**: App authenticates with AWS SSO profiles (sso_start_url / sso_account_id)
- [ ] **AUTH-05**: User can switch between AWS regions without restarting the app
- [x] **AUTH-06**: App displays an actionable error when credentials are expired or invalid (e.g., "run aws sso login --profile X")

### Navigation

- [x] **NAV-01**: All list views support vim-style scrolling (j/k to move up/down rows)
- [x] **NAV-02**: User can jump to top of list with g and bottom with G
- [x] **NAV-03**: User can navigate back one level with Esc
- [x] **NAV-04**: User can quit the app with q
- [x] **NAV-05**: Each view displays a breadcrumb header showing current location (e.g., S3 > my-bucket > logs/)
- [x] **NAV-06**: List views display a loading spinner during initial data fetch
- [x] **NAV-07**: API errors display inline in the affected view without crashing the app

### Status Bar

- [x] **STAT-01**: Status bar displays the application version
- [x] **STAT-02**: Status bar displays the current AWS profile name
- [x] **STAT-03**: Status bar displays the current AWS region
- [x] **STAT-04**: Status bar displays the current IAM principal (account ID + role/user ARN from STS GetCallerIdentity)
- [x] **STAT-05**: Status bar displays the current date and time
- [x] **STAT-06**: Status bar updates immediately when profile or region changes

### S3

- [x] **S3-01**: User can view a list of all S3 buckets accessible by the current profile
- [x] **S3-02**: User can navigate into a bucket to browse object prefixes (folder metaphor, Enter to descend)
- [x] **S3-03**: User can navigate into prefixes recursively (Esc to ascend)
- [x] **S3-04**: User can view object metadata for the selected object (key, size, last modified, storage class)
- [x] **S3-05**: User can filter objects within the current prefix by typing (/ key enters filter mode, uses ListObjectsV2 server-side prefix)

### ECS

- [x] **ECS-01**: User can view a list of ECS clusters in the current region
- [x] **ECS-02**: User can navigate into a cluster to view its services
- [x] **ECS-03**: User can view service details (desired count, running count, pending count, launch type, task definition)
- [x] **ECS-04**: User can navigate into a service to view its running tasks
- [x] **ECS-05**: User can view task details (task ID, status, started at, containers with name, image, and status)
- [x] **ECS-06**: User can open the CloudWatch Logs console URL for a selected task's containers in their browser (L key)

### Configuration

- [x] **CONF-01**: Screen auto-refreshes at a configurable interval (default: 2 seconds)
- [x] **CONF-02**: User can set the refresh interval via CLI flag (--refresh-interval)
- [x] **CONF-03**: User can set the default profile via CLI flag (--profile)
- [x] **CONF-04**: User can set the default region via CLI flag (--region)
- [x] **CONF-05**: App loads configuration from ~/.duchess/config file on startup
- [x] **CONF-06**: CLI flags take precedence over config file values

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Navigation

- **NAV-v2-01**: Context-sensitive help overlay (? key shows keybindings for current view)

### S3

- **S3-v2-01**: User can copy S3 bucket name or object key to clipboard (y key)
- **S3-v2-02**: User can view total object count and size for a prefix (on-demand, S key to compute)
- **S3-v2-03**: User can download an object to local disk
- **S3-v2-04**: User can upload a local file to the current prefix
- **S3-v2-05**: User can delete an object

### ECS

- **ECS-v2-01**: User can copy cluster/service/task ARN to clipboard (y key)
- **ECS-v2-02**: User can exec into a running container (ECS Exec)
- **ECS-v2-03**: User can stop a running task
- **ECS-v2-04**: User can force a service redeployment

### Authentication

- **AUTH-v2-01**: App supports MFA token entry for role assumption
- **AUTH-v2-02**: App displays the full role assumption chain (source_profile → role_arn)

### Services

- **SVC-v2-01**: EC2 instance browsing (list, details, SSH)
- **SVC-v2-02**: Lambda function browsing (list, details, invoke)
- **SVC-v2-03**: CloudWatch Logs browsing (log groups, streams, tail)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Any write/mutation operations | v1 is strictly read-only — builds trust with open-source users before granting write IAM permissions |
| Mouse navigation | Target users are vim-native; keyboard-only is intentional |
| Multi-region simultaneous view | N parallel API calls + complex table merging; one region at a time is correct for v1 |
| Config UI inside the TUI | Over-engineered; edit ~/.duchess/config with any text editor |
| Browser-based SSO login flow | Too fragile; show error with the aws sso login command to run instead |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| AUTH-01 | Phase 5 | Pending |
| AUTH-02 | Phase 1 | Complete |
| AUTH-03 | Phase 1 | Complete |
| AUTH-04 | Phase 1 | Complete |
| AUTH-05 | Phase 5 | Pending |
| AUTH-06 | Phase 1 | Complete |
| NAV-01 | Phase 3 | Complete |
| NAV-02 | Phase 3 | Complete |
| NAV-03 | Phase 3 | Complete |
| NAV-04 | Phase 2 | Complete |
| NAV-05 | Phase 3 | Complete |
| NAV-06 | Phase 2 | Complete |
| NAV-07 | Phase 2 | Complete |
| STAT-01 | Phase 2 | Complete |
| STAT-02 | Phase 2 | Complete |
| STAT-03 | Phase 2 | Complete |
| STAT-04 | Phase 2 | Complete |
| STAT-05 | Phase 2 | Complete |
| STAT-06 | Phase 2 | Complete |
| S3-01 | Phase 3 | Complete |
| S3-02 | Phase 3 | Complete |
| S3-03 | Phase 3 | Complete |
| S3-04 | Phase 3 | Complete |
| S3-05 | Phase 3 | Complete |
| ECS-01 | Phase 4 | Complete |
| ECS-02 | Phase 4 | Complete |
| ECS-03 | Phase 4 | Complete |
| ECS-04 | Phase 4 | Complete |
| ECS-05 | Phase 4 | Complete |
| ECS-06 | Phase 4 | Complete |
| CONF-01 | Phase 1 | Complete |
| CONF-02 | Phase 1 | Complete |
| CONF-03 | Phase 1 | Complete |
| CONF-04 | Phase 1 | Complete |
| CONF-05 | Phase 1 | Complete |
| CONF-06 | Phase 1 | Complete |

**Coverage:**
- v1 requirements: 36 total
- Mapped to phases: 36
- Unmapped: 0

---
*Requirements defined: 2026-03-02*
*Last updated: 2026-03-02 after roadmap creation*
