package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("HTTP_READ_TIMEOUT_SEC", "")
	t.Setenv("HTTP_WRITE_TIMEOUT_SEC", "")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT_SEC", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("AUTH_BOOTSTRAP_USERNAME", "")
	t.Setenv("AUTH_BOOTSTRAP_PASSWORD", "")
	t.Setenv("AUTH_PASSWORD_PEPPER", "")
	t.Setenv("AUTH_SESSION_TTL_SEC", "")
	t.Setenv("AUTH_SESSION_STATE_FILE", "")
	t.Setenv("AUTH_USER_STATE_FILE", "")
	t.Setenv("FRONTEND_DIST_DIR", "")
	t.Setenv("SQL_PROFILE_STATE_FILE", "")
	t.Setenv("MIGRATIONS_DIR", "")
	t.Setenv("MIGRATION_STATE_FILE", "")
	t.Setenv("AUDIT_LOG_FILE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("expected default HTTP addr :8080, got %q", cfg.HTTP.Addr)
	}
	if cfg.HTTP.ReadTimeout != 10*time.Second {
		t.Fatalf("expected default read timeout 10s, got %v", cfg.HTTP.ReadTimeout)
	}
	if cfg.HTTP.WriteTimeout != 15*time.Second {
		t.Fatalf("expected default write timeout 15s, got %v", cfg.HTTP.WriteTimeout)
	}
	if cfg.HTTP.ShutdownTimeout != 20*time.Second {
		t.Fatalf("expected default shutdown timeout 20s, got %v", cfg.HTTP.ShutdownTimeout)
	}
	if cfg.DatabaseURL != "" {
		t.Fatalf("expected default database url to be empty, got %q", cfg.DatabaseURL)
	}
	if cfg.Auth.BootstrapUsername != "admin" {
		t.Fatalf("expected default bootstrap username admin, got %q", cfg.Auth.BootstrapUsername)
	}
	if cfg.Auth.BootstrapPassword != "admin123" {
		t.Fatalf("expected default bootstrap password admin123, got %q", cfg.Auth.BootstrapPassword)
	}
	if cfg.Auth.PasswordPepper != "change-me-in-production" {
		t.Fatalf("expected default password pepper, got %q", cfg.Auth.PasswordPepper)
	}
	if cfg.Auth.SessionTTL != 3600*time.Second {
		t.Fatalf("expected default session ttl 3600s, got %v", cfg.Auth.SessionTTL)
	}
	if cfg.Auth.SessionStateFile != "./data/auth_sessions.json" {
		t.Fatalf("expected default auth session state file ./data/auth_sessions.json, got %q", cfg.Auth.SessionStateFile)
	}
	if cfg.Auth.UserStateFile != "./data/auth_users.json" {
		t.Fatalf("expected default auth user state file ./data/auth_users.json, got %q", cfg.Auth.UserStateFile)
	}
	if cfg.FrontendDistDir != "./web/dist" {
		t.Fatalf("expected default frontend dist dir ./web/dist, got %q", cfg.FrontendDistDir)
	}
	if cfg.SQLProfileStateFile != "./data/sql_profiles.json" {
		t.Fatalf("expected default sql profile state file ./data/sql_profiles.json, got %q", cfg.SQLProfileStateFile)
	}
	if cfg.MigrationsDir != "./migrations" {
		t.Fatalf("expected default migrations dir ./migrations, got %q", cfg.MigrationsDir)
	}
	if cfg.MigrationStateFile != "./data/migration_state.json" {
		t.Fatalf("expected default migration state file ./data/migration_state.json, got %q", cfg.MigrationStateFile)
	}
	if cfg.AuditLogFile != "./data/audit.log" {
		t.Fatalf("expected default audit log file ./data/audit.log, got %q", cfg.AuditLogFile)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("HTTP_READ_TIMEOUT_SEC", "3")
	t.Setenv("HTTP_WRITE_TIMEOUT_SEC", "5")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT_SEC", "9")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/modern_mcs?sslmode=disable")
	t.Setenv("AUTH_BOOTSTRAP_USERNAME", "ops")
	t.Setenv("AUTH_BOOTSTRAP_PASSWORD", "secret")
	t.Setenv("AUTH_PASSWORD_PEPPER", "pepper")
	t.Setenv("AUTH_SESSION_TTL_SEC", "600")
	t.Setenv("AUTH_SESSION_STATE_FILE", "/data/auth_sessions.json")
	t.Setenv("AUTH_USER_STATE_FILE", "/data/auth_users.json")
	t.Setenv("FRONTEND_DIST_DIR", "/app/web/dist")
	t.Setenv("SQL_PROFILE_STATE_FILE", "/data/sql_profiles.json")
	t.Setenv("MIGRATIONS_DIR", "/data/migrations")
	t.Setenv("MIGRATION_STATE_FILE", "/data/migration_state.json")
	t.Setenv("AUDIT_LOG_FILE", "/data/audit.log")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.HTTP.Addr != ":9090" {
		t.Fatalf("expected overridden HTTP addr :9090, got %q", cfg.HTTP.Addr)
	}
	if cfg.HTTP.ReadTimeout != 3*time.Second {
		t.Fatalf("expected overridden read timeout 3s, got %v", cfg.HTTP.ReadTimeout)
	}
	if cfg.HTTP.WriteTimeout != 5*time.Second {
		t.Fatalf("expected overridden write timeout 5s, got %v", cfg.HTTP.WriteTimeout)
	}
	if cfg.HTTP.ShutdownTimeout != 9*time.Second {
		t.Fatalf("expected overridden shutdown timeout 9s, got %v", cfg.HTTP.ShutdownTimeout)
	}
	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/modern_mcs?sslmode=disable" {
		t.Fatalf("expected overridden database url, got %q", cfg.DatabaseURL)
	}
	if cfg.Auth.BootstrapUsername != "ops" {
		t.Fatalf("expected overridden bootstrap username ops, got %q", cfg.Auth.BootstrapUsername)
	}
	if cfg.Auth.BootstrapPassword != "secret" {
		t.Fatalf("expected overridden bootstrap password secret, got %q", cfg.Auth.BootstrapPassword)
	}
	if cfg.Auth.PasswordPepper != "pepper" {
		t.Fatalf("expected overridden password pepper pepper, got %q", cfg.Auth.PasswordPepper)
	}
	if cfg.Auth.SessionTTL != 600*time.Second {
		t.Fatalf("expected overridden session ttl 600s, got %v", cfg.Auth.SessionTTL)
	}
	if cfg.Auth.SessionStateFile != "/data/auth_sessions.json" {
		t.Fatalf("expected overridden auth session state file, got %q", cfg.Auth.SessionStateFile)
	}
	if cfg.Auth.UserStateFile != "/data/auth_users.json" {
		t.Fatalf("expected overridden auth user state file, got %q", cfg.Auth.UserStateFile)
	}
	if cfg.FrontendDistDir != "/app/web/dist" {
		t.Fatalf("expected overridden frontend dist dir, got %q", cfg.FrontendDistDir)
	}
	if cfg.SQLProfileStateFile != "/data/sql_profiles.json" {
		t.Fatalf("expected overridden sql profile state file, got %q", cfg.SQLProfileStateFile)
	}
	if cfg.MigrationsDir != "/data/migrations" {
		t.Fatalf("expected overridden migrations dir, got %q", cfg.MigrationsDir)
	}
	if cfg.MigrationStateFile != "/data/migration_state.json" {
		t.Fatalf("expected overridden migration state file, got %q", cfg.MigrationStateFile)
	}
	if cfg.AuditLogFile != "/data/audit.log" {
		t.Fatalf("expected overridden audit log file, got %q", cfg.AuditLogFile)
	}
}

func TestLoadInvalidIntFallsBack(t *testing.T) {
	t.Setenv("HTTP_READ_TIMEOUT_SEC", "not-a-number")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.HTTP.ReadTimeout != 10*time.Second {
		t.Fatalf("expected fallback read timeout 10s, got %v", cfg.HTTP.ReadTimeout)
	}
}
