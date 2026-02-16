# deployments

Place deployment manifests here:

- docker-compose for local integration
- Kubernetes manifests/Helm charts for production
- systemd units for VM/on-prem installs

## Local PostgreSQL for integration tests

```bash
cd modern-mcs
docker compose -f deployments/docker-compose.postgres.yml up -d

export TEST_POSTGRES_DSN='postgres://modern_mcs:modern_mcs@127.0.0.1:5432/modern_mcs?sslmode=disable'
make test-integration
```

## Production-like Docker Compose deployment

```bash
cd modern-mcs
cp deployments/.env.prod.example deployments/.env.prod
# edit deployments/.env.prod and replace all REPLACE_* secrets

docker compose --env-file deployments/.env.prod -f deployments/docker-compose.prod.yml up -d --build
```

Verify:

```bash
curl -f http://127.0.0.1:8080/healthz
curl -f http://127.0.0.1:8080/readyz
```

Update rollout:

```bash
git pull
docker compose --env-file deployments/.env.prod -f deployments/docker-compose.prod.yml up -d --build
```

Notes:
- The app image includes backend binary, `web/dist`, and `migrations/`.
- PostgreSQL initializes `0002_postgres_core_tables.sql` on first database creation.
- Put a reverse proxy (Caddy/Nginx/Traefik) in front for TLS and domain routing.

## Dockhand deployment

Use this compose file in Dockhand:
- `deployments/docker-compose.dockhand.yml`

Why:
- It avoids relative host-path bind mounts (which can fail on remote Docker nodes with permission/path errors).
- App bootstrap creates required tables at startup in PostgreSQL mode.

Required env vars in Dockhand:
- `APP_PORT`
- `POSTGRES_DB`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `AUTH_BOOTSTRAP_USERNAME`
- `AUTH_BOOTSTRAP_PASSWORD`
- `AUTH_PASSWORD_PEPPER`
- `AUTH_SESSION_TTL_SEC`
