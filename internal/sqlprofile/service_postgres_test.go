package sqlprofile

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewPGService(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS sql_profiles").WillReturnResult(sqlmock.NewResult(0, 0))
	_, err = NewPGService(db)
	if err != nil {
		t.Fatalf("NewPGService() error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestPGServiceCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS sql_profiles").WillReturnResult(sqlmock.NewResult(0, 0))
	svc, err := NewPGService(db)
	if err != nil {
		t.Fatalf("NewPGService() error: %v", err)
	}
	svc.nowFunc = func() time.Time { return time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC) }

	mock.ExpectExec("INSERT INTO sql_profiles").WillReturnResult(sqlmock.NewResult(1, 1))

	_, err = svc.Create(Profile{
		Name:     "Main",
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
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
