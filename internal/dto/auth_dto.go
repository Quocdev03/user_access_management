package dto

import "time"
type UserInfoResponse struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

type RegisterRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	FullName    string `json:"full_name" binding:"required,max=100"`
	Phone       string `json:"phone" binding:"required,max=20"`
	DateOfBirth string `json:"date_of_birth" binding:"required,datetime=2006-01-02"`
}

type RegisterResponse struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Status   string `json:"status"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

type ResendVerificationEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
	User         UserInfoResponse `json:"user"`
}

// --------------- Refresh Token ---------------

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type ForceChangePasswordRequest struct {
	Email        string `json:"email" binding:"required,email"`
	TempPassword string `json:"temp_password" binding:"required"`
	NewPassword  string `json:"new_password" binding:"required,min=8"`
}

type SessionResponse struct {
	ID        uint64    `json:"id"`
	IPAddress *string   `json:"ip_address"`
	UserAgent *string   `json:"user_agent"`
	DeviceID  *uint64   `json:"device_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IsCurrent bool      `json:"is_current"`
}

type DeviceResponse struct {
	ID           uint64     `json:"id"`
	DeviceName   *string    `json:"device_name"`
	DeviceType   *string    `json:"device_type"`
	OS           *string    `json:"os"`
	Browser      *string    `json:"browser"`
	IPAddress    *string    `json:"ip_address"`
	LastActiveAt *time.Time `json:"last_active_at"`
}
