package models

import "time"

// User represents an administrator or member user in the Control Plane
type User struct {
	ID           string    `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         string    `db:"role" json:"role"` // SUPER_ADMIN, ADMIN, MEMBER
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
