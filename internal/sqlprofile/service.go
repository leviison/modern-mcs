package sqlprofile

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound     = errors.New("sql profile not found")
	ErrInvalidInput = errors.New("invalid sql profile input")
	allowedDBTypes  = map[string]struct{}{"mysql": {}, "mssql": {}, "pgsql": {}}
)

type Service struct {
	nowFunc   func() time.Time
	stateFile string

	mu       sync.RWMutex
	profiles map[string]Profile
}

func NewService() *Service {
	return &Service{
		nowFunc:   time.Now,
		stateFile: "",
		profiles:  make(map[string]Profile),
	}
}

func NewServiceWithFile(stateFile string) (*Service, error) {
	s := &Service{
		nowFunc:   time.Now,
		stateFile: strings.TrimSpace(stateFile),
		profiles:  make(map[string]Profile),
	}
	if s.stateFile == "" {
		return nil, fmt.Errorf("state file path is required")
	}
	if err := s.loadState(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) Create(p Profile) (Profile, error) {
	if err := validate(p); err != nil {
		return Profile{}, err
	}

	id, err := generateID(12)
	if err != nil {
		return Profile{}, fmt.Errorf("generate id: %w", err)
	}

	now := s.nowFunc().UTC()
	p.ID = id
	p.CreatedAt = now
	p.ModifiedAt = now
	p.Name = strings.TrimSpace(p.Name)
	p.DBType = strings.ToLower(strings.TrimSpace(p.DBType))

	s.mu.Lock()
	prev := cloneProfiles(s.profiles)
	s.profiles[p.ID] = p.Clone()
	if err := s.persistLocked(); err != nil {
		s.profiles = prev
		s.mu.Unlock()
		return Profile{}, err
	}
	s.mu.Unlock()

	return p, nil
}

func (s *Service) List() []Profile {
	s.mu.RLock()
	profiles := make([]Profile, 0, len(s.profiles))
	for _, p := range s.profiles {
		profiles = append(profiles, p.Clone())
	}
	s.mu.RUnlock()

	sort.Slice(profiles, func(i, j int) bool { return profiles[i].CreatedAt.Before(profiles[j].CreatedAt) })
	return profiles
}

func (s *Service) Get(id string) (Profile, error) {
	s.mu.RLock()
	p, ok := s.profiles[id]
	s.mu.RUnlock()
	if !ok {
		return Profile{}, ErrNotFound
	}
	return p.Clone(), nil
}

func (s *Service) Update(id string, p Profile) (Profile, error) {
	if err := validate(p); err != nil {
		return Profile{}, err
	}

	s.mu.Lock()
	prev := cloneProfiles(s.profiles)
	existing, ok := s.profiles[id]
	if !ok {
		s.mu.Unlock()
		return Profile{}, ErrNotFound
	}

	now := s.nowFunc().UTC()
	existing.Name = strings.TrimSpace(p.Name)
	existing.DBType = strings.ToLower(strings.TrimSpace(p.DBType))
	existing.Host = p.Host
	existing.Port = p.Port
	existing.Username = p.Username
	existing.Database = p.Database
	existing.Commands = p.Commands
	existing.UseSSL = p.UseSSL
	existing.ModifiedAt = now
	s.profiles[id] = existing.Clone()
	if err := s.persistLocked(); err != nil {
		s.profiles = prev
		s.mu.Unlock()
		return Profile{}, err
	}
	s.mu.Unlock()

	return existing, nil
}

func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev := cloneProfiles(s.profiles)
	if _, ok := s.profiles[id]; !ok {
		return ErrNotFound
	}
	delete(s.profiles, id)
	if err := s.persistLocked(); err != nil {
		s.profiles = prev
		return err
	}
	return nil
}

func (s *Service) loadState() error {
	b, err := os.ReadFile(s.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read sql profile state: %w", err)
	}
	if len(b) == 0 {
		return nil
	}
	var decoded []Profile
	if err := json.Unmarshal(b, &decoded); err != nil {
		return fmt.Errorf("decode sql profile state: %w", err)
	}
	for _, p := range decoded {
		if p.ID == "" {
			continue
		}
		s.profiles[p.ID] = p.Clone()
	}
	return nil
}

func (s *Service) persistLocked() error {
	if s.stateFile == "" {
		return nil
	}
	out := make([]Profile, 0, len(s.profiles))
	for _, p := range s.profiles {
		out = append(out, p.Clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("encode sql profile state: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.stateFile), 0o755); err != nil {
		return fmt.Errorf("mkdir sql profile state dir: %w", err)
	}
	if err := os.WriteFile(s.stateFile, b, 0o644); err != nil {
		return fmt.Errorf("write sql profile state: %w", err)
	}
	return nil
}

func cloneProfiles(src map[string]Profile) map[string]Profile {
	out := make(map[string]Profile, len(src))
	for k, v := range src {
		out[k] = v.Clone()
	}
	return out
}

func validate(p Profile) error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	dbType := strings.ToLower(strings.TrimSpace(p.DBType))
	if _, ok := allowedDBTypes[dbType]; !ok {
		return fmt.Errorf("%w: db_type must be mysql, mssql, or pgsql", ErrInvalidInput)
	}
	if strings.TrimSpace(p.Host) == "" {
		return fmt.Errorf("%w: host is required", ErrInvalidInput)
	}
	if p.Port <= 0 || p.Port > 65535 {
		return fmt.Errorf("%w: port must be between 1 and 65535", ErrInvalidInput)
	}
	if strings.TrimSpace(p.Database) == "" {
		return fmt.Errorf("%w: database is required", ErrInvalidInput)
	}
	if strings.TrimSpace(p.Commands) == "" {
		return fmt.Errorf("%w: commands is required", ErrInvalidInput)
	}
	return nil
}

func generateID(n int) (string, error) {
	if n < 8 {
		return "", fmt.Errorf("id length too short")
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
