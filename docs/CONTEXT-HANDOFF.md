# CONTEXT HANDOFF

Last updated: 2026-02-16

## Goal
Migrate legacy `mcs` package to a modern, license-free Go codebase (`modern-mcs`) using incremental replacement.

## Current status
- `modern-mcs` backend builds and tests pass.
- No proprietary runtime license gate.
- Core APIs implemented for auth, sessions, SQL profiles, migration status/apply.
- Storage mode now supports:
  - default JSON-file persistence for users/sessions/sql-profiles/migration-state/audit
  - optional PostgreSQL persistence for auth users/sessions, SQL profiles, and migration apply state via `DATABASE_URL`
- React+TypeScript frontend scaffold exists under `web/` and is wired to current APIs.
- Backend serves frontend static assets when `FRONTEND_DIST_DIR/index.html` exists.
- Frontend production build now succeeds and is served by backend in a live localhost smoke test.

## Implemented modules
- Auth service: `internal/auth`
  - Login/logout/me/change-password
  - Session persistence:
    - file-backed via `AUTH_SESSION_STATE_FILE` (default)
    - PostgreSQL via `internal/auth/session_store_postgres.go` when `DATABASE_URL` is set
  - User persistence:
    - file-backed via `AUTH_USER_STATE_FILE` (default)
    - PostgreSQL via `internal/auth/store_postgres.go` when `DATABASE_URL` is set
  - Session listing/revoke uses session IDs and safe views (token fallback removed)
- SQL profiles: `internal/sqlprofile`
  - CRUD + validation
  - Persistence:
    - file-backed via `SQL_PROFILE_STATE_FILE` (default)
    - PostgreSQL via `internal/sqlprofile/service_postgres.go` when `DATABASE_URL` is set
- Migration service: `internal/migrations`
  - List/status/apply
  - Apply-state persistence:
    - file-backed via `MIGRATION_STATE_FILE` (default)
    - PostgreSQL via `migration_applied` table when `DATABASE_URL` is set
- Audit logger: `internal/audit`
  - JSONL events to `AUDIT_LOG_FILE`
- HTTP middleware/routes: `internal/httpserver`
  - Adds/propagates `X-Request-Id`
  - Request ID in context for audit details
  - Static frontend hosting + SPA fallback from `FRONTEND_DIST_DIR`

## Backend API surface (current)
- Health/info:
  - `GET /healthz`
  - `GET /readyz`
  - `GET /v1/info`
- Auth:
  - `POST /v1/auth/login` (returns `token` and `session_id`)
  - `GET /v1/auth/me`
  - `POST /v1/auth/logout`
  - `POST /v1/auth/change-password`
- Admin:
  - `GET /v1/system/sessions`
  - `DELETE /v1/system/sessions/{id}`
  - `GET /v1/sql-profiles`
  - `POST /v1/sql-profiles`
  - `GET /v1/sql-profiles/{id}`
  - `PUT /v1/sql-profiles/{id}`
  - `DELETE /v1/sql-profiles/{id}`
  - `GET /v1/system/migrations`
  - `GET /v1/system/migrations/status`
  - `POST /v1/system/migrations/{name}/apply`

## Frontend scaffold (`web/`)
- Vite + React + TypeScript
- Pages wired to backend APIs:
  - `/login`
  - `/sql-profiles` (list/create/update/delete)
  - `/sessions` (list/revoke)
  - `/migrations` (status/apply)
  - `/change-password`
- Auth context with localStorage persistence
- Route protection via `RequireAuth`
- API clients in `web/src/api/*`
- Shared frontend API error helper in `web/src/api/errors.ts`
- Consistent loading/busy UI states added across SQL profiles, sessions, and migrations pages

## Password policy (change-password)
Implemented in `internal/auth/service.go`:
- length: 12 to 128
- requires uppercase, lowercase, digit, and special character
- no leading/trailing whitespace

## Environment variables in use
Backend:
- `HTTP_ADDR`
- `HTTP_READ_TIMEOUT_SEC`
- `HTTP_WRITE_TIMEOUT_SEC`
- `HTTP_SHUTDOWN_TIMEOUT_SEC`
- `DATABASE_URL` (optional; enables PostgreSQL-backed repositories)
- `AUTH_BOOTSTRAP_USERNAME`
- `AUTH_BOOTSTRAP_PASSWORD`
- `AUTH_PASSWORD_PEPPER`
- `AUTH_SESSION_TTL_SEC`
- `AUTH_SESSION_STATE_FILE`
- `AUTH_USER_STATE_FILE`
- `FRONTEND_DIST_DIR`
- `SQL_PROFILE_STATE_FILE`
- `MIGRATIONS_DIR`
- `MIGRATION_STATE_FILE`
- `AUDIT_LOG_FILE`

Frontend:
- `VITE_API_BASE`

## Persistence files (defaults)
- `./data/auth_users.json`
- `./data/auth_sessions.json`
- `./data/sql_profiles.json`
- `./data/migration_state.json`
- `./data/audit.log`

## Audit details
Current audit detail payload includes:
- request ID (`rid`)
- client IP (`ip`)
- user-agent (`ua`)
- actor session ID when known (`sid`)
- optional detail text for failures/special cases

## Build/test commands
Backend (verified in this environment on 2026-02-16):
```bash
cd modern-mcs
GOMODCACHE=/tmp/gomodcache GOPATH=/tmp/gopath GOCACHE=/tmp/gocache make test
GOMODCACHE=/tmp/gomodcache GOPATH=/tmp/gopath GOCACHE=/tmp/gocache make build
GOMODCACHE=/tmp/gomodcache GOPATH=/tmp/gopath GOCACHE=/tmp/gocache make test-integration
```

Frontend (verified on 2026-02-16):
```bash
cd modern-mcs/web
cp .env.example .env
npm install
npm run build
```

Observed result:
- `npm install` succeeded after elevated execution in this environment.
- `npm run build` succeeded and produced `web/dist` with `index.html` and bundled assets.
- `npm audit` is now clean (`0` vulnerabilities) after upgrading frontend toolchain deps:
  - `vite` -> `^7.3.1`
  - `@vitejs/plugin-react` -> `^5.1.4`
  - Note: this toolchain expects Node.js `20.19+` (or `22.12+`), and `web/package.json` now declares this engine requirement.

Additional backend verification on 2026-02-16:
- Removed backward-compat revoke fallback in `internal/httpserver/server.go`; admin revoke now calls `RevokeSessionByID` only.
- Updated HTTP server tests accordingly and re-ran `make fmt`, `make test`, `make build` successfully.
- Live smoke test passed (elevated due sandbox socket limits):
  - `GET /` returned `200 text/html` from `web/dist/index.html`
  - `GET /v1/info` returned JSON `{"service":"modern-mcs-api","version":"0.1.0"}`
- Added optional `DATABASE_URL` in config and app wiring:
  - `internal/app/app.go` opens Postgres (`lib/pq`) when configured
  - app selects Postgres-backed auth user store, auth session store, SQL profile service, and migration apply-state store
  - app keeps JSON-file fallback when `DATABASE_URL` is empty
  - backend tests/build pass with this wiring
- Added Postgres-path unit tests using `sqlmock`:
  - `internal/auth/session_store_postgres_test.go`
  - `internal/auth/store_postgres_test.go`
  - `internal/sqlprofile/service_postgres_test.go`
  - `internal/migrations/service_postgres_test.go`
  - full `make test` and `make build` pass after these additions
- Added real Postgres integration tests (env-gated by `TEST_POSTGRES_DSN`):
  - `internal/integration/postgres_integration_test.go`
  - Covers auth user+session round-trip, SQL profile CRUD, migration apply-state round-trip
  - `make test-integration` target added in `Makefile`
  - In this environment, `make test-integration` passed with tests skipped because `TEST_POSTGRES_DSN` is not set
- Added local Postgres compose manifest:
  - `deployments/docker-compose.postgres.yml`
  - `deployments/README.md` includes run instructions and `TEST_POSTGRES_DSN` example
- Added CI workflow:
  - `.github/workflows/ci.yml`
  - Runs on push/PR
  - Includes format check (`gofmt -l`), `make test`, `make build`
  - Starts Postgres service and runs `make test-integration` with `TEST_POSTGRES_DSN`
  - Uses `go run ./cmd/waitforpostgres` for readiness probe (no `pg_isready` dependency)
  - Uses Node `20.19.0` via `actions/setup-node`
  - Runs frontend `npm ci`, `npm run build`, and `npm audit --audit-level=moderate`
  - Local backend tests/build remain green after adding workflow
- Added explicit Postgres DDL migration file:
  - `migrations/0002_postgres_core_tables.sql`
  - Defines `auth_users`, `auth_sessions`, `sql_profiles`, and `migration_applied`
  - Mirrors runtime `CREATE TABLE IF NOT EXISTS` schemas for controlled rollout/change tracking
- Added branch-protection guidance doc:
  - `docs/BRANCH-PROTECTION.md`
  - Recommends requiring CI check `test-build` before merge
- Added Node version pinning files:
  - `.nvmrc` and `web/.nvmrc` set to `20.19.0`
  - Aligns local developer environment with CI/frontend toolchain requirements
- Added deployable container baseline:
  - `Dockerfile` now builds frontend (`web/dist`) and copies `migrations/` into runtime image
  - `.dockerignore` added for leaner build context
  - `deployments/docker-compose.prod.yml` for app + postgres
  - `deployments/docker-compose.dockhand.yml` for Dockhand/remote-node friendly deploy (no relative bind mounts; no build context required)
  - `deployments/.env.prod.example` for server env/secrets bootstrap
  - `deployments/README.md` now has production-like rollout commands and verification steps
- Added GHCR image publish workflow:
  - `.github/workflows/publish-image.yml`
  - Publishes `ghcr.io/leviison/modern-mcs:latest` and sha tags on push to `main`
  - Dockhand compose now uses `APP_IMAGE` (registry pull) instead of local `build` directives

To serve frontend via backend:
- Build frontend to `web/dist` (or set `FRONTEND_DIST_DIR` to your output dir)
- Run backend normally; it serves static assets at `/` and API at `/v1/*`

## Priority next steps
1. Ensure GHCR package visibility/credentials are configured so Dockhand can pull `APP_IMAGE`.
2. Decide whether to remove runtime auto-create DDL once migration execution is managed externally.
3. Add reverse-proxy/TLS compose profile (Caddy or Nginx) for direct internet exposure.

## Notes
- Repo root (`/home/user/Downloads/myconnectionsvr`) is not a git repository.
- Legacy code under `mcs/` remains migration input only.
