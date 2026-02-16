# Migration Plan (Legacy `mcs` -> `modern-mcs`)

## 1. Strategy

Use a strangler migration:

1. Keep legacy runtime operational for production continuity.
2. Introduce modern API service in parallel.
3. Move one domain at a time behind stable interfaces.
4. Cut traffic per feature after parity and validation.

## 2. Phases

### Phase 0: Foundation (current)

- Go service skeleton with health/readiness and graceful shutdown.
- Repo structure and deployment baseline.
- Legacy inventory and migration backlog.

### Phase 1: Platform

- PostgreSQL schema + migrations.
- Config, secrets, and structured audit logging.
- Authentication, session handling, and RBAC.
- Job scheduler framework for reports/publication tasks.

### Phase 2: Core domains

- Accounts/users/groups management.
- Test metadata ingestion and query APIs.
- Report template and publication model.
- Alert/action definitions.

### Phase 3: UI replacement

- Admin login/settings pages.
- SQL profile management pages.
- Publication wizard and action workflows.
- System status/reporting dashboards.

### Phase 4: Cutover

- Route switched feature-by-feature.
- Data migration and rollback playbooks.
- Legacy shell/JAR components retired.

## 3. Non-negotiables

- No plaintext secrets in config files.
- Parameterized SQL only.
- No direct HTML injection from untrusted content.
- CI with unit/integration tests and SAST.
- No proprietary runtime license enforcement in the new service path.
