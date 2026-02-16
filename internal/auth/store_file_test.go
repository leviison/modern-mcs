package auth

import (
	"path/filepath"
	"testing"
)

func TestFileUserStorePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.json")
	store, err := NewFileUserStore(path)
	if err != nil {
		t.Fatalf("NewFileUserStore() error: %v", err)
	}

	u := User{ID: "u-1", Username: "admin", PasswordHash: "h", Roles: []string{"admin"}}
	if err := store.Put(u); err != nil {
		t.Fatalf("Put() error: %v", err)
	}

	store2, err := NewFileUserStore(path)
	if err != nil {
		t.Fatalf("NewFileUserStore() second error: %v", err)
	}
	got, err := store2.GetByUsername("admin")
	if err != nil {
		t.Fatalf("GetByUsername() error: %v", err)
	}
	if got.ID != "u-1" {
		t.Fatalf("expected id u-1, got %q", got.ID)
	}
}
