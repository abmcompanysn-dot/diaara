package model

import "time"

type User struct {
	ID                  string     `json:"id"`
	Email               string     `json:"email"`
	Phone               *string    `json:"phone,omitempty"`
	PasswordHash        string     `json:"-"`
	EmailVerifiedAt     *time.Time `json:"email_verified_at,omitempty"`
	PhoneVerifiedAt     *time.Time `json:"phone_verified_at,omitempty"`
	IsAdmin             bool       `json:"is_admin"`
	FailedLoginAttempts int        `json:"-"`
	LockedUntil         *time.Time `json:"-"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type RegisterInput struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Phone    *string `json:"phone,omitempty"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshInput struct {
	RefreshToken string `json:"refresh_token"`
}

type VerifyEmailInput struct {
	Token string `json:"token"`
}

type ForgotPasswordInput struct {
	Email string `json:"email"`
}

type ResetPasswordInput struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}
