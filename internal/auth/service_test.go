package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoginAndValidateToken(t *testing.T) {
	store := NewInMemoryUserStore()
	svc, err := NewService(store, ServiceConfig{PasswordPepper: "pepper", SessionTTL: 2 * time.Minute})
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	if err := store.Put(User{
		ID:           "u-1",
		Username:     "admin",
		PasswordHash: svc.HashPassword("secret123"),
		Roles:        []string{"admin"},
	}); err != nil {
		t.Fatalf("store.Put() error: %v", err)
	}

	session, err := svc.Login("admin", "secret123")
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}
	if session.Token == "" {
		t.Fatalf("expected non-empty token")
	}

	validated, err := svc.ValidateToken(session.Token)
	if err != nil {
		t.Fatalf("ValidateToken() error: %v", err)
	}
	if validated.Username != "admin" {
		t.Fatalf("expected username admin, got %q", validated.Username)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	store := NewInMemoryUserStore()
	svc, err := NewService(store, ServiceConfig{PasswordPepper: "pepper", SessionTTL: time.Minute})
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	_ = store.Put(User{ID: "u-1", Username: "admin", PasswordHash: svc.HashPassword("secret123"), Roles: []string{"admin"}})

	_, err = svc.Login("admin", "badpass")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestExpiredToken(t *testing.T) {
	store := NewInMemoryUserStore()
	svc, err := NewService(store, ServiceConfig{PasswordPepper: "pepper", SessionTTL: time.Second})
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	fakeNow := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	svc.nowFunc = func() time.Time { return fakeNow }

	_ = store.Put(User{ID: "u-1", Username: "admin", PasswordHash: svc.HashPassword("secret123"), Roles: []string{"admin"}})
	session, err := svc.Login("admin", "secret123")
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}

	svc.nowFunc = func() time.Time { return fakeNow.Add(2 * time.Second) }
	_, err = svc.ValidateToken(session.Token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestLogoutRevokesToken(t *testing.T) {
	store := NewInMemoryUserStore()
	svc, err := NewService(store, ServiceConfig{PasswordPepper: "pepper", SessionTTL: time.Minute})
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	_ = store.Put(User{ID: "u-1", Username: "admin", PasswordHash: svc.HashPassword("secret123"), Roles: []string{"admin"}})

	session, err := svc.Login("admin", "secret123")
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}

	if err := svc.Logout(session.Token); err != nil {
		t.Fatalf("Logout() error: %v", err)
	}

	_, err = svc.ValidateToken(session.Token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken after logout, got %v", err)
	}
}

func TestSessionStatePersistence(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "auth_sessions.json")

	store := NewInMemoryUserStore()
	svc, err := NewService(store, ServiceConfig{
		PasswordPepper:   "pepper",
		SessionTTL:       time.Minute,
		SessionStateFile: stateFile,
	})
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	_ = store.Put(User{ID: "u-1", Username: "admin", PasswordHash: svc.HashPassword("secret123"), Roles: []string{"admin"}})

	session, err := svc.Login("admin", "secret123")
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}

	raw, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read session state file: %v", err)
	}
	var decoded map[string]Session
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode session state file: %v", err)
	}
	if _, ok := decoded[session.Token]; !ok {
		t.Fatalf("expected token %s in session state file", session.Token)
	}

	store2 := NewInMemoryUserStore()
	svc2, err := NewService(store2, ServiceConfig{
		PasswordPepper:   "pepper",
		SessionTTL:       time.Minute,
		SessionStateFile: stateFile,
	})
	if err != nil {
		t.Fatalf("NewService() second instance error: %v", err)
	}
	if err := svc2.LoadSessionState(); err != nil {
		t.Fatalf("LoadSessionState() error: %v", err)
	}
	if _, err := svc2.ValidateToken(session.Token); err != nil {
		t.Fatalf("ValidateToken() for loaded token error: %v", err)
	}
}

func TestChangePassword(t *testing.T) {
	store := NewInMemoryUserStore()
	svc, err := NewService(store, ServiceConfig{
		PasswordPepper: "pepper",
		SessionTTL:     time.Minute,
	})
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	_ = store.Put(User{ID: "u-1", Username: "admin", PasswordHash: svc.HashPassword("oldpass123"), Roles: []string{"admin"}})

	session, err := svc.Login("admin", "oldpass123")
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}

	if err := svc.ChangePassword(session.Token, "oldpass123", "NewPassword123!"); err != nil {
		t.Fatalf("ChangePassword() error: %v", err)
	}

	_, err = svc.Login("admin", "oldpass123")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected old password to fail after change, got %v", err)
	}

	if _, err := svc.Login("admin", "NewPassword123!"); err != nil {
		t.Fatalf("expected login with new password to succeed, got %v", err)
	}
}

func TestChangePasswordWeakRejected(t *testing.T) {
	store := NewInMemoryUserStore()
	svc, err := NewService(store, ServiceConfig{
		PasswordPepper: "pepper",
		SessionTTL:     time.Minute,
	})
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	_ = store.Put(User{ID: "u-1", Username: "admin", PasswordHash: svc.HashPassword("oldpass123"), Roles: []string{"admin"}})
	session, _ := svc.Login("admin", "oldpass123")

	err = svc.ChangePassword(session.Token, "oldpass123", "short")
	if !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("expected ErrWeakPassword, got %v", err)
	}
}

func TestListAndRevokeSessions(t *testing.T) {
	store := NewInMemoryUserStore()
	svc, err := NewService(store, ServiceConfig{
		PasswordPepper: "pepper",
		SessionTTL:     time.Minute,
	})
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	_ = store.Put(User{ID: "u-1", Username: "admin", PasswordHash: svc.HashPassword("secret123"), Roles: []string{"admin"}})

	s1, _ := svc.Login("admin", "secret123")
	s2, _ := svc.Login("admin", "secret123")
	list := svc.ListSessions()
	if len(list) < 2 {
		t.Fatalf("expected at least 2 sessions, got %d", len(list))
	}

	if err := svc.RevokeToken(s1.Token); err != nil {
		t.Fatalf("RevokeToken() error: %v", err)
	}
	if _, err := svc.ValidateToken(s1.Token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected revoked token invalid, got %v", err)
	}
	if _, err := svc.ValidateToken(s2.Token); err != nil {
		t.Fatalf("expected second token still valid, got %v", err)
	}
}
