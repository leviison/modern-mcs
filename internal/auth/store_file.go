package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type FileUserStore struct {
	path string

	mu    sync.RWMutex
	users map[string]User
}

func NewFileUserStore(path string) (*FileUserStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("user state file path is required")
	}

	s := &FileUserStore{
		path:  path,
		users: make(map[string]User),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *FileUserStore) GetByUsername(username string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.users[username]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

func (s *FileUserStore) Put(user User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.Username] = user
	return s.persistLocked()
}

func (s *FileUserStore) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read user store file: %w", err)
	}
	if len(b) == 0 {
		return nil
	}

	var decoded []User
	if err := json.Unmarshal(b, &decoded); err != nil {
		return fmt.Errorf("decode user store file: %w", err)
	}
	for _, u := range decoded {
		if strings.TrimSpace(u.Username) == "" {
			continue
		}
		s.users[u.Username] = u
	}
	return nil
}

func (s *FileUserStore) persistLocked() error {
	out := make([]User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("encode user store file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("mkdir user store dir: %w", err)
	}
	if err := os.WriteFile(s.path, b, 0o644); err != nil {
		return fmt.Errorf("write user store file: %w", err)
	}
	return nil
}
