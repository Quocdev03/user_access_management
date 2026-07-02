package dto

import "time"

type UserProfileResponse struct {
	ID            uint64    `json:"id"`
	AvatarURL     *string   `json:"avatar_url"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	FullName      string    `json:"full_name"`
	Phone         string    `json:"phone"`
	Status        string    `json:"status"`
	DateOfBirth   string    `json:"date_of_birth"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	Roles         []string  `json:"roles"`
}

type UpdateProfileRequest struct {
	FullName string `json:"full_name" binding:"required,max=100"`
	Phone    string `json:"phone" binding:"required,max=20"`
}

type RequestEmailChangeRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewEmail        string `json:"new_email" binding:"required,email"`
}

type VerifyOldEmailRequest struct {
	OTP string `json:"otp" binding:"required,len=6"`
}

type VerifyOldEmailResponse struct {
	EmailChangeToken string `json:"email_change_token"`
}

type VerifyNewEmailRequest struct {
	EmailChangeToken string `json:"email_change_token" binding:"required"`
	OTP              string `json:"otp" binding:"required,len=6"`
}

type UploadAvatarResponse struct {
	AvatarURL string `json:"avatar_url"`
}
