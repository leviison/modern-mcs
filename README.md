# modern-mcs

New codebase for replacing the legacy MyConnection Server package in `../mcs`.

## Goals

- Replace legacy shell + monolithic JAR runtime with a maintainable service architecture.
- Deliver an API-first backend in Go, then modernize admin/public UIs.
- Migrate incrementally with compatibility endpoints and phased cutover.
- Keep runtime fully open-source with no proprietary license-gating logic.

## Quick start

```bash
cd modern-mcs
make run
```

Deploy with containers:
- Production-like compose stack: `deployments/docker-compose.prod.yml`
- Setup steps: `deployments/README.md`
- GHCR image publish workflow: `.github/workflows/publish-image.yml`

Health endpoints:

- `GET /healthz`
- `GET /readyz`
- `GET /v1/info`

Testing:
- Unit/default test suite: `make test`
- Postgres integration tests: set `TEST_POSTGRES_DSN` then run `make test-integration`
- CI workflow: `.github/workflows/ci.yml` runs format check, `make test`, `make build`, and Postgres integration tests on push/PR
- Postgres readiness helper used by CI: `go run ./cmd/waitforpostgres`
- Branch protection recommendations: `docs/BRANCH-PROTECTION.md`
- Node version pinning: `.nvmrc` (repo root) and `web/.nvmrc` target `20.19.0`

Frontend hosting:
- If `FRONTEND_DIST_DIR` contains `index.html`, backend serves it at `/` with SPA fallback.
- API routes remain under `/v1/*` and are not shadowed by static hosting.
- Frontend build toolchain currently expects Node.js `20.19+` (or `22.12+`) in `web/`.

Auth endpoints (bootstrap only for now):

- `POST /v1/auth/login`
- `GET /v1/auth/me` (Bearer token)
- `POST /v1/auth/logout` (Bearer token)
- `POST /v1/auth/change-password` (Bearer token)

State persistence (JSON files):

- Auth sessions: `AUTH_SESSION_STATE_FILE`
- Auth users: `AUTH_USER_STATE_FILE`
- SQL Profiles: `SQL_PROFILE_STATE_FILE`
- Migration apply status: `MIGRATION_STATE_FILE`
- Audit trail: `AUDIT_LOG_FILE`

Optional PostgreSQL mode:
- Set `DATABASE_URL` (PostgreSQL DSN) to persist auth users, auth sessions, SQL profiles, and migration apply state in Postgres.
- If `DATABASE_URL` is empty, file-backed JSON persistence is used (default).

Admin endpoints (require `admin` role):

- `GET /v1/sql-profiles`
- `POST /v1/sql-profiles`
- `GET /v1/sql-profiles/{id}`
- `PUT /v1/sql-profiles/{id}`
- `DELETE /v1/sql-profiles/{id}`
- `GET /v1/system/migrations`
- `GET /v1/system/migrations/status`
- `POST /v1/system/migrations/{name}/apply`
- `GET /v1/system/sessions`
- `DELETE /v1/system/sessions/{id}`

## Project layout

- `cmd/server`: main entrypoint
- `internal/config`: environment configuration
- `internal/httpserver`: HTTP transport and routing
- `internal/app`: app lifecycle wiring
- `migrations`: SQL migration files for schema tracking
- `docs`: migration and architecture docs
- `api`: API contracts
- `web`: placeholder for modern frontend

## Next implementation milestones

1. Add PostgreSQL integration and schema migrations.
2. Implement auth/session APIs and RBAC.
3. Port account, report, and scheduler flows from legacy templates.
4. Replace legacy admin pages with TypeScript frontend.
