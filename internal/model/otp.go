package model

import "time"

// OTPCode đại diện cho thông tin các mã OTP được sinh ra trong hệ thống để phục vụ xác thực
type OTPCode struct {
	ID        uint64    `db:"id"`         // ID duy nhất của bản ghi OTP
	UserID    uint64    `db:"user_id"`    // ID của người dùng sở hữu mã OTP
	Code      string    `db:"code"`       // Mã OTP gồm 6 ký số được băm hoặc lưu dạng chuỗi
	Type      string    `db:"type"`       // Loại OTP (ví dụ: "email_verification", "reset_password")
	Attempts  int       `db:"attempts"`   // Số lần người dùng đã thử nhập mã OTP
	IsUsed    bool      `db:"is_used"`    // Trạng thái đã được sử dụng hay chưa
	ExpiresAt time.Time `db:"expires_at"` // Thời điểm mã OTP hết hạn sử dụng
	CreatedAt time.Time `db:"created_at"` // Thời điểm tạo mã OTP
}
