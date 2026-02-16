package auth

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewPostgresUserStore(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS auth_users").WillReturnResult(sqlmock.NewResult(0, 0))

	_, err = NewPostgresUserStore(db)
	if err != nil {
		t.Fatalf("NewPostgresUserStore() error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestPostgresUserStoreGetByUsernameNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS auth_users").WillReturnResult(sqlmock.NewResult(0, 0))
	store, err := NewPostgresUserStore(db)
	if err != nil {
		t.Fatalf("NewPostgresUserStore() error: %v", err)
	}

	mock.ExpectQuery("SELECT id, username, password_hash, roles FROM auth_users WHERE username = \\$1").
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	_, err = store.GetByUsername("missing")
	if err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestPostgresUserStorePut(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS auth_users").WillReturnResult(sqlmock.NewResult(0, 0))
	store, err := NewPostgresUserStore(db)
	if err != nil {
		t.Fatalf("NewPostgresUserStore() error: %v", err)
	}

	mock.ExpectExec("INSERT INTO auth_users").
		WithArgs("u1", "admin", "hash", []byte(`["admin"]`)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := store.Put(User{
		ID:           "u1",
		Username:     "admin",
		PasswordHash: "hash",
		Roles:        []string{"admin"},
	}); err != nil {
		t.Fatalf("Put() error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
