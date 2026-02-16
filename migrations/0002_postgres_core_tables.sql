-- Core PostgreSQL tables for modern-mcs persistence.
-- These definitions mirror the runtime-created schemas in:
-- - internal/auth/store_postgres.go
-- - internal/auth/session_store_postgres.go
-- - internal/sqlprofile/service_postgres.go
-- - internal/migrations/service.go

CREATE TABLE IF NOT EXISTS auth_users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  roles JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS auth_sessions (
  token TEXT PRIMARY KEY,
  session_id TEXT NOT NULL UNIQUE,
  user_id TEXT NOT NULL,
  username TEXT NOT NULL,
  roles JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS sql_profiles (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  db_type TEXT NOT NULL,
  host TEXT NOT NULL,
  port INTEGER NOT NULL,
  username TEXT NOT NULL,
  database_name TEXT NOT NULL,
  commands TEXT NOT NULL,
  use_ssl BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL,
  modified_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS migration_applied (
  name TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL
);
