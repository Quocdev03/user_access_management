package dto

import "github.com/quocdev03/user-access-management/internal/model"

type AdminListUsersRequest struct {
	Username  string `form:"username"`
	Email     string `form:"email"`
	Status    string `form:"status"`
	Role      string `form:"role"`
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order"`
	Page      int    `form:"page,default=1"`
	PerPage   int    `form:"per_page,default=20"`
}

type AdminUserListItem struct {
	ID            uint64   `json:"id"`
	Username      string   `json:"username"`
	Email         string   `json:"email"`
	FullName      string   `json:"full_name"`
	Phone         string   `json:"phone"`
	Status        string   `json:"status"`
	EmailVerified bool     `json:"email_verified"`
	DateOfBirth   string   `json:"date_of_birth"`
	AvatarURL     *string  `json:"avatar_url,omitempty"`
	Roles         []string `json:"roles"`
	CreatedAt     string   `json:"created_at"`
}

type AdminListUsersResponse struct {
	Users []AdminUserListItem  `json:"users"`
	Meta  model.PaginationMeta `json:"meta"`
}

type AdminUpdateUserRequest struct {
	FullName      *string `json:"full_name" validate:"omitempty,min=2,max=100"`
	Phone         *string `json:"phone" validate:"omitempty,min=9,max=20"`
	Email         *string `json:"email" validate:"omitempty,email"`
	EmailVerified *bool   `json:"email_verified"`
	DateOfBirth   *string `json:"date_of_birth" validate:"omitempty,datetime=2006-01-02"`
	AvatarURL     *string `json:"avatar_url" validate:"omitempty,url"`
}

type AdminChangeStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=active inactive locked"`
}

type AdminNotifyRequest struct {
	Subject string `json:"subject" validate:"required,max=200"`
	Message string `json:"message" validate:"required,max=5000"`
}
