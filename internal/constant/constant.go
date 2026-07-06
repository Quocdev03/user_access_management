package constant

type UserStatus string

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"

	OTPTypeEmailVerification = "email_verification"
	OTPTypeForgotPassword    = "forgot_password"
	OTPTypeChangeEmailOld    = "change_email_old"
	OTPTypeChangeEmailNew    = "change_email_new"

	RoleAdmin     = "admin"
	RoleModerator = "moderator"
	RoleUser      = "user"

	StatusActive   UserStatus = "active"
	StatusInactive UserStatus = "inactive"
	StatusLocked   UserStatus = "locked"
)
