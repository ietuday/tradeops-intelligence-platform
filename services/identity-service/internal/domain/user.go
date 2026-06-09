package domain

import "time"

type User struct {
	ID           string
	TenantID     string
	Email        string
	PasswordHash string
	FullName     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Roles        []string
}
