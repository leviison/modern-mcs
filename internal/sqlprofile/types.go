package sqlprofile

import "time"

type Profile struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	DBType     string    `json:"db_type"`
	Host       string    `json:"host"`
	Port       int       `json:"port"`
	Username   string    `json:"username"`
	Database   string    `json:"database"`
	Commands   string    `json:"commands"`
	UseSSL     bool      `json:"use_ssl"`
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
}

func (p Profile) Clone() Profile {
	return p
}
