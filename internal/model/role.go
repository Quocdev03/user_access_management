package model

import "time"

// Role đại diện cho bảng roles định nghĩa các vai trò của người dùng trong hệ thống (RBAC)
type Role struct {
	ID          uint64    `db:"id"`          // ID duy nhất của vai trò
	Name        string    `db:"name"`        // Tên vai trò (ví dụ: "admin", "user")
	Description string    `db:"description"` // Mô tả chi tiết vai trò
	CreatedAt   time.Time `db:"created_at"`  // Thời điểm tạo vai trò
	UpdatedAt   time.Time `db:"updated_at"`  // Thời điểm cập nhật vai trò
}
