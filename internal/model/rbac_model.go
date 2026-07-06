package model

import "time"

type Role struct {
	ID          uint64    `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type Permission struct {
	ID          uint64    `db:"id"`
	Name        string    `db:"name"`
	Description *string   `db:"description"`
	Resource    string    `db:"resource"`
	Action      string    `db:"action"`
	CreatedAt   time.Time `db:"created_at"`
}
