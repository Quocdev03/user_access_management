package model

import "time"

type AuditLog struct {
	ID         uint64    `db:"id" json:"id"`
	UserID     *uint64   `db:"user_id" json:"user_id"`
	Action     string    `db:"action" json:"action"`
	Resource   *string   `db:"resource" json:"resource,omitempty"`
	ResourceID *string   `db:"resource_id" json:"resource_id,omitempty"`
	IPAddress  *string   `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent  *string   `db:"user_agent" json:"user_agent,omitempty"`
	OldValues  *string   `db:"old_values" json:"old_values,omitempty"`
	NewValues  *string   `db:"new_values" json:"new_values,omitempty"`
	Status     string    `db:"status" json:"status"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}
