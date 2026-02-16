package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type PostgresUserStore struct {
	db *sql.DB
}

func NewPostgresUserStore(db *sql.DB) (*PostgresUserStore, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	s := &PostgresUserStore{db: db}
	if err := s.ensureSchema(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *PostgresUserStore) ensureSchema() error {
	const q = `
CREATE TABLE IF NOT EXISTS auth_users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	roles JSONB NOT NULL DEFAULT '[]'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`
	if _, err := s.db.Exec(q); err != nil {
		return fmt.Errorf("ensure auth_users schema: %w", err)
	}
	return nil
}

func (s *PostgresUserStore) GetByUsername(username string) (User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return User{}, ErrUserNotFound
	}

	var u User
	var rolesJSON []byte
	const q = `SELECT id, username, password_hash, roles FROM auth_users WHERE username = $1`
	if err := s.db.QueryRow(q, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &rolesJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("query auth user: %w", err)
	}
	if len(rolesJSON) > 0 {
		if err := json.Unmarshal(rolesJSON, &u.Roles); err != nil {
			return User{}, fmt.Errorf("decode roles: %w", err)
		}
	}
	return u, nil
}

func (s *PostgresUserStore) Put(user User) error {
	user.Username = strings.TrimSpace(user.Username)
	if user.ID == "" || user.Username == "" || user.PasswordHash == "" {
		return fmt.Errorf("id, username, and password hash are required")
	}

	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return fmt.Errorf("encode roles: %w", err)
	}

	const q = `
INSERT INTO auth_users (id, username, password_hash, roles, updated_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (username) DO UPDATE
SET id = EXCLUDED.id,
	password_hash = EXCLUDED.password_hash,
	roles = EXCLUDED.roles,
	updated_at = NOW()`
	if _, err := s.db.Exec(q, user.ID, user.Username, user.PasswordHash, rolesJSON); err != nil {
		return fmt.Errorf("upsert auth user: %w", err)
	}
	return nil
}
