package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrWeakPassword       = errors.New("weak password")
)

const (
	minPasswordLength = 12
	maxPasswordLength = 128
)

type Service struct {
	users        UserStore
	pepper       string
	ttl          time.Duration
	nowFunc      func() time.Time
	stateFile    string
	sessionStore SessionStore

	sessMu   sync.RWMutex
	sessions map[string]Session
}

type ServiceConfig struct {
	PasswordPepper   string
	SessionTTL       time.Duration
	SessionStateFile string
	SessionStore     SessionStore
}

func NewService(userStore UserStore, cfg ServiceConfig) (*Service, error) {
	if userStore == nil {
		return nil, fmt.Errorf("user store is required")
	}
	if cfg.PasswordPepper == "" {
		return nil, fmt.Errorf("password pepper is required")
	}
	if cfg.SessionTTL <= 0 {
		return nil, fmt.Errorf("session TTL must be > 0")
	}

	return &Service{
		users:        userStore,
		pepper:       cfg.PasswordPepper,
		ttl:          cfg.SessionTTL,
		nowFunc:      time.Now,
		stateFile:    cfg.SessionStateFile,
		sessionStore: cfg.SessionStore,
		sessions:     make(map[string]Session),
		sessMu:       sync.RWMutex{},
	}, nil
}

func (s *Service) HashPassword(password string) string {
	sum := sha256.Sum256([]byte(s.pepper + ":" + password))
	return hex.EncodeToString(sum[:])
}

func (s *Service) VerifyPassword(password, storedHash string) bool {
	candidate := s.HashPassword(password)
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(storedHash)) == 1
}

func (s *Service) Login(username, password string) (Session, error) {
	u, err := s.users.GetByUsername(username)
	if err != nil {
		return Session{}, ErrInvalidCredentials
	}

	if !s.VerifyPassword(password, u.PasswordHash) {
		return Session{}, ErrInvalidCredentials
	}

	token, err := generateToken(32)
	if err != nil {
		return Session{}, fmt.Errorf("generate token: %w", err)
	}

	now := s.nowFunc()
	session := Session{
		ID:        mustID(16),
		Token:     token,
		UserID:    u.ID,
		Username:  u.Username,
		Roles:     append([]string(nil), u.Roles...),
		CreatedAt: now,
		ExpiresAt: now.Add(s.ttl),
	}

	s.sessMu.Lock()
	s.sessions[token] = session
	if err := s.persistSessionsLocked(); err != nil {
		delete(s.sessions, token)
		s.sessMu.Unlock()
		return Session{}, err
	}
	s.sessMu.Unlock()

	return session, nil
}

func (s *Service) ValidateToken(token string) (Session, error) {
	s.sessMu.RLock()
	session, ok := s.sessions[token]
	s.sessMu.RUnlock()
	if !ok {
		return Session{}, ErrInvalidToken
	}

	if s.nowFunc().After(session.ExpiresAt) {
		s.sessMu.Lock()
		delete(s.sessions, token)
		_ = s.persistSessionsLocked()
		s.sessMu.Unlock()
		return Session{}, ErrInvalidToken
	}

	return session, nil
}

func (s *Service) Logout(token string) error {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	if _, ok := s.sessions[token]; !ok {
		return ErrInvalidToken
	}
	delete(s.sessions, token)
	if err := s.persistSessionsLocked(); err != nil {
		return err
	}
	return nil
}

func (s *Service) ChangePassword(token, currentPassword, newPassword string) error {
	if err := validatePasswordPolicy(newPassword); err != nil {
		return ErrWeakPassword
	}

	session, err := s.ValidateToken(token)
	if err != nil {
		return err
	}

	user, err := s.users.GetByUsername(session.Username)
	if err != nil {
		return ErrInvalidCredentials
	}
	if !s.VerifyPassword(currentPassword, user.PasswordHash) {
		return ErrInvalidCredentials
	}
	user.PasswordHash = s.HashPassword(newPassword)
	if err := s.users.Put(user); err != nil {
		return fmt.Errorf("store updated password: %w", err)
	}
	return nil
}

func validatePasswordPolicy(password string) error {
	if strings.TrimSpace(password) != password {
		return ErrWeakPassword
	}
	if len(password) < minPasswordLength || len(password) > maxPasswordLength {
		return ErrWeakPassword
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return ErrWeakPassword
	}
	return nil
}

func (s *Service) ListSessions() []Session {
	now := s.nowFunc()

	s.sessMu.Lock()
	defer s.sessMu.Unlock()

	out := make([]Session, 0, len(s.sessions))
	dirty := false
	for token, sess := range s.sessions {
		if now.After(sess.ExpiresAt) {
			delete(s.sessions, token)
			dirty = true
			continue
		}
		out = append(out, sess)
	}
	if dirty {
		_ = s.persistSessionsLocked()
	}
	return out
}

func (s *Service) ListSessionViews() []SessionView {
	sessions := s.ListSessions()
	out := make([]SessionView, 0, len(sessions))
	for _, sess := range sessions {
		out = append(out, SessionView{
			ID:        sess.ID,
			UserID:    sess.UserID,
			Username:  sess.Username,
			Roles:     append([]string(nil), sess.Roles...),
			CreatedAt: sess.CreatedAt,
			ExpiresAt: sess.ExpiresAt,
		})
	}
	return out
}

func (s *Service) RevokeToken(token string) error {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	if _, ok := s.sessions[token]; !ok {
		return ErrInvalidToken
	}
	delete(s.sessions, token)
	if err := s.persistSessionsLocked(); err != nil {
		return err
	}
	return nil
}

func (s *Service) RevokeSessionByID(sessionID string) error {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()

	foundToken := ""
	for token, sess := range s.sessions {
		if sess.ID == sessionID {
			foundToken = token
			break
		}
	}
	if foundToken == "" {
		return ErrInvalidToken
	}
	delete(s.sessions, foundToken)
	if err := s.persistSessionsLocked(); err != nil {
		return err
	}
	return nil
}

func (s *Service) LoadSessionState() error {
	if s.sessionStore != nil {
		state, err := s.sessionStore.Load()
		if err != nil {
			return fmt.Errorf("load session state: %w", err)
		}
		s.sessMu.Lock()
		s.sessions = state
		s.sessMu.Unlock()
		return nil
	}

	if s.stateFile == "" {
		return nil
	}
	b, err := os.ReadFile(s.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read session state: %w", err)
	}
	if len(b) == 0 {
		return nil
	}
	state := make(map[string]Session)
	if err := json.Unmarshal(b, &state); err != nil {
		return fmt.Errorf("decode session state: %w", err)
	}

	s.sessMu.Lock()
	s.sessions = state
	s.sessMu.Unlock()
	return nil
}

func (s *Service) persistSessionsLocked() error {
	if s.sessionStore != nil {
		if err := s.sessionStore.Save(s.sessions); err != nil {
			return fmt.Errorf("save session state: %w", err)
		}
		return nil
	}

	if s.stateFile == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.stateFile), 0o755); err != nil {
		return fmt.Errorf("mkdir session state dir: %w", err)
	}
	b, err := json.MarshalIndent(s.sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session state: %w", err)
	}
	if err := os.WriteFile(s.stateFile, b, 0o644); err != nil {
		return fmt.Errorf("write session state: %w", err)
	}
	return nil
}

func generateToken(n int) (string, error) {
	if n < 16 {
		return "", fmt.Errorf("token length too short")
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func mustID(n int) string {
	id, err := generateToken(n)
	if err != nil {
		// Fallback preserves behavior even if randomness source fails unexpectedly.
		return fmt.Sprintf("sid-%d", time.Now().UnixNano())
	}
	return id
}
