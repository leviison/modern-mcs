package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTP                HTTPConfig
	DatabaseURL         string
	Auth                AuthConfig
	FrontendDistDir     string
	SQLProfileStateFile string
	MigrationsDir       string
	MigrationStateFile  string
	AuditLogFile        string
}

type HTTPConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type AuthConfig struct {
	BootstrapUsername string
	BootstrapPassword string
	PasswordPepper    string
	SessionTTL        time.Duration
	SessionStateFile  string
	UserStateFile     string
}

func Load() (Config, error) {
	cfg := Config{
		HTTP: HTTPConfig{
			Addr:            getEnv("HTTP_ADDR", ":8080"),
			ReadTimeout:     time.Duration(getEnvInt("HTTP_READ_TIMEOUT_SEC", 10)) * time.Second,
			WriteTimeout:    time.Duration(getEnvInt("HTTP_WRITE_TIMEOUT_SEC", 15)) * time.Second,
			ShutdownTimeout: time.Duration(getEnvInt("HTTP_SHUTDOWN_TIMEOUT_SEC", 20)) * time.Second,
		},
		DatabaseURL: getEnv("DATABASE_URL", ""),
		Auth: AuthConfig{
			BootstrapUsername: getEnv("AUTH_BOOTSTRAP_USERNAME", "admin"),
			BootstrapPassword: getEnv("AUTH_BOOTSTRAP_PASSWORD", "admin123"),
			PasswordPepper:    getEnv("AUTH_PASSWORD_PEPPER", "change-me-in-production"),
			SessionTTL:        time.Duration(getEnvInt("AUTH_SESSION_TTL_SEC", 3600)) * time.Second,
			SessionStateFile:  getEnv("AUTH_SESSION_STATE_FILE", "./data/auth_sessions.json"),
			UserStateFile:     getEnv("AUTH_USER_STATE_FILE", "./data/auth_users.json"),
		},
		FrontendDistDir:     getEnv("FRONTEND_DIST_DIR", "./web/dist"),
		SQLProfileStateFile: getEnv("SQL_PROFILE_STATE_FILE", "./data/sql_profiles.json"),
		MigrationsDir:       getEnv("MIGRATIONS_DIR", "./migrations"),
		MigrationStateFile:  getEnv("MIGRATION_STATE_FILE", "./data/migration_state.json"),
		AuditLogFile:        getEnv("AUDIT_LOG_FILE", "./data/audit.log"),
	}

	if cfg.HTTP.Addr == "" {
		return Config{}, fmt.Errorf("HTTP_ADDR must not be empty")
	}
	if cfg.Auth.BootstrapUsername == "" {
		return Config{}, fmt.Errorf("AUTH_BOOTSTRAP_USERNAME must not be empty")
	}
	if cfg.Auth.BootstrapPassword == "" {
		return Config{}, fmt.Errorf("AUTH_BOOTSTRAP_PASSWORD must not be empty")
	}
	if cfg.Auth.PasswordPepper == "" {
		return Config{}, fmt.Errorf("AUTH_PASSWORD_PEPPER must not be empty")
	}
	if cfg.Auth.SessionTTL <= 0 {
		return Config{}, fmt.Errorf("AUTH_SESSION_TTL_SEC must be > 0")
	}
	if cfg.Auth.SessionStateFile == "" {
		return Config{}, fmt.Errorf("AUTH_SESSION_STATE_FILE must not be empty")
	}
	if cfg.Auth.UserStateFile == "" {
		return Config{}, fmt.Errorf("AUTH_USER_STATE_FILE must not be empty")
	}
	if cfg.FrontendDistDir == "" {
		return Config{}, fmt.Errorf("FRONTEND_DIST_DIR must not be empty")
	}
	if cfg.SQLProfileStateFile == "" {
		return Config{}, fmt.Errorf("SQL_PROFILE_STATE_FILE must not be empty")
	}
	if cfg.MigrationsDir == "" {
		return Config{}, fmt.Errorf("MIGRATIONS_DIR must not be empty")
	}
	if cfg.MigrationStateFile == "" {
		return Config{}, fmt.Errorf("MIGRATION_STATE_FILE must not be empty")
	}
	if cfg.AuditLogFile == "" {
		return Config{}, fmt.Errorf("AUDIT_LOG_FILE must not be empty")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return fallback
	}
	return val
}

func getEnvInt(key string, fallback int) int {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}
