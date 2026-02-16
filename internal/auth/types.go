package auth

import "time"

type User struct {
	ID           string   `json:"id"`
	Username     string   `json:"username"`
	PasswordHash string   `json:"-"`
	Roles        []string `json:"roles"`
}

type Session struct {
	ID        string
	Token     string
	UserID    string
	Username  string
	Roles     []string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type SessionView struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
