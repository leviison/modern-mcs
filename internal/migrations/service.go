package migrations

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FileInfo struct {
	Name     string `json:"name"`
	Checksum string `json:"checksum"`
}

type Status struct {
	Name      string `json:"name"`
	Checksum  string `json:"checksum"`
	Applied   bool   `json:"applied"`
	AppliedAt string `json:"applied_at,omitempty"`
}

type Service struct {
	dir   string
	store appliedStore
}

type appliedState map[string]string

type appliedStore interface {
	Load() (appliedState, error)
	SetApplied(name string, appliedAt time.Time) error
}

func NewService(dir, stateFile string) *Service {
	return &Service{
		dir:   dir,
		store: &fileAppliedStore{stateFile: stateFile},
	}
}

func NewServiceWithPostgres(dir string, db *sql.DB) (*Service, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	store := &pgAppliedStore{db: db}
	if err := store.ensureSchema(); err != nil {
		return nil, err
	}
	return &Service{
		dir:   dir,
		store: store,
	}, nil
}

func (s *Service) List() ([]FileInfo, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	out := make([]FileInfo, 0)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		full := filepath.Join(s.dir, e.Name())
		checksum, err := fileSHA256(full)
		if err != nil {
			return nil, fmt.Errorf("hash migration %s: %w", e.Name(), err)
		}
		out = append(out, FileInfo{Name: e.Name(), Checksum: checksum})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Service) Status() ([]Status, error) {
	files, err := s.List()
	if err != nil {
		return nil, err
	}

	applied, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	out := make([]Status, 0, len(files))
	for _, f := range files {
		appliedAt, ok := applied[f.Name]
		out = append(out, Status{
			Name:      f.Name,
			Checksum:  f.Checksum,
			Applied:   ok,
			AppliedAt: appliedAt,
		})
	}
	return out, nil
}

func (s *Service) MarkApplied(name string, appliedAt time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" || !strings.HasSuffix(name, ".sql") || strings.Contains(name, "/") {
		return fmt.Errorf("invalid migration name")
	}

	// Ensure migration exists.
	full := filepath.Join(s.dir, name)
	if _, err := os.Stat(full); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("migration does not exist")
		}
		return fmt.Errorf("stat migration: %w", err)
	}

	return s.store.SetApplied(name, appliedAt.UTC())
}

type fileAppliedStore struct {
	stateFile string
}

func (s *fileAppliedStore) Load() (appliedState, error) {
	state := make(appliedState)
	if s.stateFile == "" {
		return state, nil
	}

	b, err := os.ReadFile(s.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, fmt.Errorf("read migration state: %w", err)
	}
	if len(b) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, fmt.Errorf("decode migration state: %w", err)
	}
	return state, nil
}

func (s *fileAppliedStore) SetApplied(name string, appliedAt time.Time) error {
	state, err := s.Load()
	if err != nil {
		return err
	}
	state[name] = appliedAt.UTC().Format(time.RFC3339)
	return s.saveState(state)
}

func (s *fileAppliedStore) saveState(state appliedState) error {
	if s.stateFile == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.stateFile), 0o755); err != nil {
		return fmt.Errorf("mkdir state dir: %w", err)
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode migration state: %w", err)
	}
	if err := os.WriteFile(s.stateFile, b, 0o644); err != nil {
		return fmt.Errorf("write migration state: %w", err)
	}
	return nil
}

type pgAppliedStore struct {
	db *sql.DB
}

func (s *pgAppliedStore) ensureSchema() error {
	const q = `
CREATE TABLE IF NOT EXISTS migration_applied (
	name TEXT PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL
)`
	if _, err := s.db.Exec(q); err != nil {
		return fmt.Errorf("ensure migration_applied schema: %w", err)
	}
	return nil
}

func (s *pgAppliedStore) Load() (appliedState, error) {
	const q = `SELECT name, applied_at FROM migration_applied`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("query migration state: %w", err)
	}
	defer rows.Close()

	out := make(appliedState)
	for rows.Next() {
		var name string
		var appliedAt time.Time
		if err := rows.Scan(&name, &appliedAt); err != nil {
			return nil, fmt.Errorf("scan migration state: %w", err)
		}
		out[name] = appliedAt.UTC().Format(time.RFC3339)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migration state: %w", err)
	}
	return out, nil
}

func (s *pgAppliedStore) SetApplied(name string, appliedAt time.Time) error {
	const q = `
INSERT INTO migration_applied (name, applied_at)
VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE SET applied_at = EXCLUDED.applied_at`
	_, err := s.db.Exec(q, name, appliedAt.UTC())
	if err != nil {
		return fmt.Errorf("upsert migration state: %w", err)
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
