package sqlprofile

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestServiceCRUD(t *testing.T) {
	svc := NewService()

	created, err := svc.Create(Profile{
		Name:     "Primary Export",
		DBType:   "mysql",
		Host:     "db.local",
		Port:     3306,
		Username: "mcs",
		Database: "mcsdb",
		Commands: "INSERT INTO t VALUES (%RECORDID%)",
		UseSSL:   true,
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Name != "Primary Export" {
		t.Fatalf("expected profile name Primary Export, got %q", got.Name)
	}

	updated, err := svc.Update(created.ID, Profile{
		Name:     "Primary Export V2",
		DBType:   "mysql",
		Host:     "db.local",
		Port:     3306,
		Username: "mcs",
		Database: "mcsdb",
		Commands: "UPDATE t SET c=1",
		UseSSL:   false,
	})
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	if updated.Name != "Primary Export V2" {
		t.Fatalf("expected updated name, got %q", updated.Name)
	}

	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err = svc.Get(created.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestValidation(t *testing.T) {
	svc := NewService()
	_, err := svc.Create(Profile{DBType: "invalid"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestServicePersistsToFile(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "sql_profiles.json")

	svc, err := NewServiceWithFile(stateFile)
	if err != nil {
		t.Fatalf("NewServiceWithFile() error: %v", err)
	}

	created, err := svc.Create(Profile{
		Name:     "Persisted Profile",
		DBType:   "pgsql",
		Host:     "db.local",
		Port:     5432,
		Username: "mcs",
		Database: "mcsdb",
		Commands: "INSERT INTO t VALUES (1)",
		UseSSL:   true,
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	raw, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	var profiles []Profile
	if err := json.Unmarshal(raw, &profiles); err != nil {
		t.Fatalf("decode state file: %v", err)
	}
	if len(profiles) != 1 || profiles[0].ID != created.ID {
		t.Fatalf("expected one persisted profile with id %s", created.ID)
	}

	svc2, err := NewServiceWithFile(stateFile)
	if err != nil {
		t.Fatalf("NewServiceWithFile() reload error: %v", err)
	}
	got, err := svc2.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() from reloaded service error: %v", err)
	}
	if got.Name != "Persisted Profile" {
		t.Fatalf("expected persisted name, got %q", got.Name)
	}
}
