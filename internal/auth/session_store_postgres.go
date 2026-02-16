package auth

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type SessionStore interface {
	Load() (map[string]Session, error)
	Save(sessions map[string]Session) error
}

type PostgresSessionStore struct {
	db *sql.DB
}

func NewPostgresSessionStore(db *sql.DB) (*PostgresSessionStore, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	s := &PostgresSessionStore{db: db}
	if err := s.ensureSchema(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *PostgresSessionStore) ensureSchema() error {
	const q = `
CREATE TABLE IF NOT EXISTS auth_sessions (
	token TEXT PRIMARY KEY,
	session_id TEXT NOT NULL UNIQUE,
	user_id TEXT NOT NULL,
	username TEXT NOT NULL,
	roles JSONB NOT NULL DEFAULT '[]'::jsonb,
	created_at TIMESTAMPTZ NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL
)`
	if _, err := s.db.Exec(q); err != nil {
		return fmt.Errorf("ensure auth_sessions schema: %w", err)
	}
	return nil
}

func (s *PostgresSessionStore) Load() (map[string]Session, error) {
	const q = `
SELECT token, session_id, user_id, username, roles, created_at, expires_at
FROM auth_sessions`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	out := make(map[string]Session)
	for rows.Next() {
		var sess Session
		var rolesJSON []byte
		if err := rows.Scan(&sess.Token, &sess.ID, &sess.UserID, &sess.Username, &rolesJSON, &sess.CreatedAt, &sess.ExpiresAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		if len(rolesJSON) > 0 {
			if err := json.Unmarshal(rolesJSON, &sess.Roles); err != nil {
				return nil, fmt.Errorf("decode session roles: %w", err)
			}
		}
		out[sess.Token] = sess
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}
	return out, nil
}

func (s *PostgresSessionStore) Save(sessions map[string]Session) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM auth_sessions`); err != nil {
		return fmt.Errorf("clear sessions: %w", err)
	}

	const q = `
INSERT INTO auth_sessions (token, session_id, user_id, username, roles, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)`
	for token, sess := range sessions {
		rolesJSON, err := json.Marshal(sess.Roles)
		if err != nil {
			return fmt.Errorf("encode session roles: %w", err)
		}
		if _, err := tx.Exec(q, token, sess.ID, sess.UserID, sess.Username, rolesJSON, sess.CreatedAt, sess.ExpiresAt); err != nil {
			return fmt.Errorf("insert session: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit session tx: %w", err)
	}
	return nil
}
