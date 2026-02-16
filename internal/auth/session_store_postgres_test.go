package auth

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewPostgresSessionStore(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS auth_sessions").WillReturnResult(sqlmock.NewResult(0, 0))

	_, err = NewPostgresSessionStore(db)
	if err != nil {
		t.Fatalf("NewPostgresSessionStore() error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestPostgresSessionStoreLoadAndSave(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS auth_sessions").WillReturnResult(sqlmock.NewResult(0, 0))
	store, err := NewPostgresSessionStore(db)
	if err != nil {
		t.Fatalf("NewPostgresSessionStore() error: %v", err)
	}

	now := time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC)
	sessions := map[string]Session{
		"tok1": {
			ID:        "sid1",
			Token:     "tok1",
			UserID:    "u1",
			Username:  "admin",
			Roles:     []string{"admin"},
			CreatedAt: now,
			ExpiresAt: now.Add(time.Hour),
		},
	}

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM auth_sessions").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO auth_sessions").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := store.Save(sessions); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	rows := sqlmock.NewRows([]string{"token", "session_id", "user_id", "username", "roles", "created_at", "expires_at"}).
		AddRow("tok1", "sid1", "u1", "admin", []byte(`["admin"]`), now, now.Add(time.Hour))
	mock.ExpectQuery("SELECT token, session_id, user_id, username, roles, created_at, expires_at FROM auth_sessions").
		WillReturnRows(rows)

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(loaded) != 1 || loaded["tok1"].ID != "sid1" {
		t.Fatalf("unexpected loaded sessions: %+v", loaded)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
