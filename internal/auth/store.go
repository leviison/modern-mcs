package auth

import (
	"errors"
	"sync"
)

var ErrUserNotFound = errors.New("user not found")

type UserStore interface {
	GetByUsername(username string) (User, error)
	Put(user User) error
}

type InMemoryUserStore struct {
	mu    sync.RWMutex
	users map[string]User
}

func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{users: make(map[string]User)}
}

func (s *InMemoryUserStore) GetByUsername(username string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.users[username]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

func (s *InMemoryUserStore) Put(user User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.Username] = user
	return nil
}
