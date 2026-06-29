package dto

// --------------- Shared ---------------

// UserInfoResponse dùng chung cho các endpoint trả thông tin user
type UserInfoResponse struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

// --------------- Register ---------------

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

// --------------- Verify Email ---------------

type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

// VerifyEmail không cần dữ liệu trả về đặc biệt trong Response, chỉ cần thông điệp thành công (Success message)

// --------------- Login ---------------

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

