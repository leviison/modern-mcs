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
