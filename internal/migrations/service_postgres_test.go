package migrations

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewServiceWithPostgres(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS migration_applied").WillReturnResult(sqlmock.NewResult(0, 0))
	_, err = NewServiceWithPostgres(".", db)
	if err != nil {
		t.Fatalf("NewServiceWithPostgres() error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestServiceWithPostgresMarkApplied(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error: %v", err)
	}
	defer db.Close()

	dir := t.TempDir()
	migration := filepath.Join(dir, "0001_init.sql")
	if err := os.WriteFile(migration, []byte("select 1;"), 0o644); err != nil {
		t.Fatalf("write migration file: %v", err)
	}

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS migration_applied").WillReturnResult(sqlmock.NewResult(0, 0))
	svc, err := NewServiceWithPostgres(dir, db)
	if err != nil {
		t.Fatalf("NewServiceWithPostgres() error: %v", err)
	}

	mock.ExpectExec("INSERT INTO migration_applied").WithArgs("0001_init.sql", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	if err := svc.MarkApplied("0001_init.sql", time.Now()); err != nil {
		t.Fatalf("MarkApplied() error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
