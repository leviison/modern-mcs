package migrations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestList(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "0001_init.sql"), []byte("CREATE TABLE x;"), 0o644); err != nil {
		t.Fatalf("write migration: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatalf("write non-migration file: %v", err)
	}

	svc := NewService(dir, filepath.Join(dir, "state.json"))
	list, err := svc.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 migration file, got %d", len(list))
	}
	if list[0].Name != "0001_init.sql" {
		t.Fatalf("expected migration name 0001_init.sql, got %q", list[0].Name)
	}
	if list[0].Checksum == "" {
		t.Fatalf("expected non-empty checksum")
	}
}

func TestStatusAndMarkApplied(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state", "migration_state.json")
	if err := os.WriteFile(filepath.Join(dir, "0001_init.sql"), []byte("CREATE TABLE x;"), 0o644); err != nil {
		t.Fatalf("write migration: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "0002_more.sql"), []byte("ALTER TABLE x ADD COLUMN y int;"), 0o644); err != nil {
		t.Fatalf("write migration: %v", err)
	}

	svc := NewService(dir, stateFile)
	now := time.Date(2026, 2, 16, 12, 30, 0, 0, time.UTC)
	if err := svc.MarkApplied("0001_init.sql", now); err != nil {
		t.Fatalf("MarkApplied() error: %v", err)
	}

	status, err := svc.Status()
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if len(status) != 2 {
		t.Fatalf("expected 2 migration statuses, got %d", len(status))
	}
	if !status[0].Applied {
		t.Fatalf("expected first migration to be marked applied")
	}
	if status[0].AppliedAt == "" {
		t.Fatalf("expected AppliedAt to be set for applied migration")
	}
	if status[1].Applied {
		t.Fatalf("expected second migration to be unapplied")
	}

	b, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	decoded := map[string]string{}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("decode state file: %v", err)
	}
	if decoded["0001_init.sql"] == "" {
		t.Fatalf("expected persisted applied timestamp for 0001_init.sql")
	}
}
