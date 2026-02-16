package integration

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"myconnectionsvr/modern-mcs/internal/auth"
	"myconnectionsvr/modern-mcs/internal/migrations"
	"myconnectionsvr/modern-mcs/internal/sqlprofile"
)

func openTestPostgres(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping Postgres integration tests")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping() error: %v", err)
	}
	return db
}

func TestPostgresAuthUserAndSessionRoundTrip(t *testing.T) {
	db := openTestPostgres(t)

	userStore, err := auth.NewPostgresUserStore(db)
	if err != nil {
		t.Fatalf("NewPostgresUserStore() error: %v", err)
	}
	sessionStore, err := auth.NewPostgresSessionStore(db)
	if err != nil {
		t.Fatalf("NewPostgresSessionStore() error: %v", err)
	}

	svc, err := auth.NewService(userStore, auth.ServiceConfig{
		PasswordPepper: "integration-pepper",
		SessionTTL:     time.Minute,
		SessionStore:   sessionStore,
	})
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	username := fmt.Sprintf("itest_auth_%d", time.Now().UnixNano())
	u := auth.User{
		ID:           fmt.Sprintf("u-%d", time.Now().UnixNano()),
		Username:     username,
		PasswordHash: svc.HashPassword("Password123!"),
		Roles:        []string{"admin"},
	}
	if err := userStore.Put(u); err != nil {
		t.Fatalf("userStore.Put() error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM auth_sessions WHERE username = $1", username)
		_, _ = db.Exec("DELETE FROM auth_users WHERE username = $1", username)
	})

	session, err := svc.Login(username, "Password123!")
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}
	if session.Token == "" || session.ID == "" {
		t.Fatalf("expected non-empty session token and id")
	}

	svc2, err := auth.NewService(userStore, auth.ServiceConfig{
		PasswordPepper: "integration-pepper",
		SessionTTL:     time.Minute,
		SessionStore:   sessionStore,
	})
	if err != nil {
		t.Fatalf("NewService() second instance error: %v", err)
	}
	if err := svc2.LoadSessionState(); err != nil {
		t.Fatalf("LoadSessionState() error: %v", err)
	}
	loaded, err := svc2.ValidateToken(session.Token)
	if err != nil {
		t.Fatalf("ValidateToken() error: %v", err)
	}
	if loaded.Username != username {
		t.Fatalf("expected username %q, got %q", username, loaded.Username)
	}
}

func TestPostgresSQLProfileCRUD(t *testing.T) {
	db := openTestPostgres(t)

	svc, err := sqlprofile.NewPGService(db)
	if err != nil {
		t.Fatalf("NewPGService() error: %v", err)
	}

	created, err := svc.Create(sqlprofile.Profile{
		Name:     fmt.Sprintf("itest_profile_%d", time.Now().UnixNano()),
		DBType:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "mcs",
		Database: "mcsdb",
		Commands: "SELECT 1",
		UseSSL:   false,
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	t.Cleanup(func() {
		_ = svc.Delete(created.ID)
	})

	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Name != created.Name {
		t.Fatalf("expected name %q, got %q", created.Name, got.Name)
	}

	updated, err := svc.Update(created.ID, sqlprofile.Profile{
		Name:     created.Name + "_updated",
		DBType:   "pgsql",
		Host:     "127.0.0.1",
		Port:     5432,
		Username: "postgres",
		Database: "modern_mcs",
		Commands: "SELECT NOW()",
		UseSSL:   true,
	})
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	if updated.DBType != "pgsql" {
		t.Fatalf("expected db_type pgsql, got %q", updated.DBType)
	}
}

func TestPostgresMigrationStateRoundTrip(t *testing.T) {
	db := openTestPostgres(t)

	dir := t.TempDir()
	migrationName := fmt.Sprintf("0001_itest_%d.sql", time.Now().UnixNano())
	if err := os.WriteFile(filepath.Join(dir, migrationName), []byte("SELECT 1;"), 0o644); err != nil {
		t.Fatalf("write migration file: %v", err)
	}

	svc, err := migrations.NewServiceWithPostgres(dir, db)
	if err != nil {
		t.Fatalf("NewServiceWithPostgres() error: %v", err)
	}

	if err := svc.MarkApplied(migrationName, time.Now().UTC()); err != nil {
		t.Fatalf("MarkApplied() error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM migration_applied WHERE name = $1", migrationName)
	})

	statuses, err := svc.Status()
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status row, got %d", len(statuses))
	}
	if !statuses[0].Applied {
		t.Fatalf("expected migration to be applied")
	}
}
