package sqlprofile

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type PGService struct {
	db      *sql.DB
	nowFunc func() time.Time
}

func NewPGService(db *sql.DB) (*PGService, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	s := &PGService{
		db:      db,
		nowFunc: time.Now,
	}
	if err := s.ensureSchema(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *PGService) ensureSchema() error {
	const q = `
CREATE TABLE IF NOT EXISTS sql_profiles (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	db_type TEXT NOT NULL,
	host TEXT NOT NULL,
	port INTEGER NOT NULL,
	username TEXT NOT NULL,
	database_name TEXT NOT NULL,
	commands TEXT NOT NULL,
	use_ssl BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMPTZ NOT NULL,
	modified_at TIMESTAMPTZ NOT NULL
)`
	if _, err := s.db.Exec(q); err != nil {
		return fmt.Errorf("ensure sql_profiles schema: %w", err)
	}
	return nil
}

func (s *PGService) Create(p Profile) (Profile, error) {
	if err := validate(p); err != nil {
		return Profile{}, err
	}

	id, err := generateID(12)
	if err != nil {
		return Profile{}, fmt.Errorf("generate id: %w", err)
	}
	now := s.nowFunc().UTC()
	p.ID = id
	p.Name = strings.TrimSpace(p.Name)
	p.DBType = strings.ToLower(strings.TrimSpace(p.DBType))
	p.CreatedAt = now
	p.ModifiedAt = now

	const q = `
INSERT INTO sql_profiles
  (id, name, db_type, host, port, username, database_name, commands, use_ssl, created_at, modified_at)
VALUES
  ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	if _, err := s.db.Exec(q, p.ID, p.Name, p.DBType, p.Host, p.Port, p.Username, p.Database, p.Commands, p.UseSSL, p.CreatedAt, p.ModifiedAt); err != nil {
		return Profile{}, fmt.Errorf("insert sql profile: %w", err)
	}
	return p, nil
}

func (s *PGService) List() []Profile {
	const q = `
SELECT id, name, db_type, host, port, username, database_name, commands, use_ssl, created_at, modified_at
FROM sql_profiles
ORDER BY created_at ASC`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil
	}
	defer rows.Close()

	out := make([]Profile, 0)
	for rows.Next() {
		var p Profile
		if err := rows.Scan(&p.ID, &p.Name, &p.DBType, &p.Host, &p.Port, &p.Username, &p.Database, &p.Commands, &p.UseSSL, &p.CreatedAt, &p.ModifiedAt); err != nil {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (s *PGService) Get(id string) (Profile, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Profile{}, ErrNotFound
	}
	const q = `
SELECT id, name, db_type, host, port, username, database_name, commands, use_ssl, created_at, modified_at
FROM sql_profiles
WHERE id = $1`
	var p Profile
	if err := s.db.QueryRow(q, id).Scan(&p.ID, &p.Name, &p.DBType, &p.Host, &p.Port, &p.Username, &p.Database, &p.Commands, &p.UseSSL, &p.CreatedAt, &p.ModifiedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Profile{}, ErrNotFound
		}
		return Profile{}, fmt.Errorf("get sql profile: %w", err)
	}
	return p, nil
}

func (s *PGService) Update(id string, p Profile) (Profile, error) {
	if err := validate(p); err != nil {
		return Profile{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Profile{}, ErrNotFound
	}

	now := s.nowFunc().UTC()
	p.Name = strings.TrimSpace(p.Name)
	p.DBType = strings.ToLower(strings.TrimSpace(p.DBType))

	const q = `
UPDATE sql_profiles
SET name = $2,
	db_type = $3,
	host = $4,
	port = $5,
	username = $6,
	database_name = $7,
	commands = $8,
	use_ssl = $9,
	modified_at = $10
WHERE id = $1`
	res, err := s.db.Exec(q, id, p.Name, p.DBType, p.Host, p.Port, p.Username, p.Database, p.Commands, p.UseSSL, now)
	if err != nil {
		return Profile{}, fmt.Errorf("update sql profile: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return Profile{}, fmt.Errorf("read update affected rows: %w", err)
	}
	if affected == 0 {
		return Profile{}, ErrNotFound
	}
	return s.Get(id)
}

func (s *PGService) Delete(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrNotFound
	}
	const q = `DELETE FROM sql_profiles WHERE id = $1`
	res, err := s.db.Exec(q, id)
	if err != nil {
		return fmt.Errorf("delete sql profile: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read delete affected rows: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
