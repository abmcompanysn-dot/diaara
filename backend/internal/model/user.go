package model

import "time"

// Rôles cumulables (client = implicite pour tous).
const (
	RoleVendeur = "vendeur"
	RoleCloser  = "closer"
)

// ValidRole retourne true si le rôle fait partie des rôles attribuables.
func ValidRole(role string) bool {
	return role == RoleVendeur || role == RoleCloser
}

type User struct {
	ID                  string     `json:"id"`
	Email               string     `json:"email"`
	Phone               *string    `json:"phone,omitempty"`
	PasswordHash        string     `json:"-"`
	EmailVerifiedAt     *time.Time `json:"email_verified_at,omitempty"`
	PhoneVerifiedAt     *time.Time `json:"phone_verified_at,omitempty"`
	IsAdmin             bool       `json:"is_admin"`
	Roles               []string   `json:"roles,omitempty"`
	FailedLoginAttempts int        `json:"-"`
	LockedUntil         *time.Time `json:"-"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type RegisterInput struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Phone    *string `json:"phone,omitempty"`
	// Rôles demandés à l'inscription (auto-inscription), cumulables :
	// "vendeur", "closer". Le rôle "client" est implicite pour tous.
	Roles []string `json:"roles,omitempty"`
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

type LogoutInput struct {
	RefreshToken string `json:"refresh_token"`
}

type RoleInput struct {
	Role   string `json:"role"`
	Action string `json:"action"` // "grant" ou "revoke"
}

type EmailVerification struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type PasswordReset struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type RefreshToken struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Canaux et usages des codes OTP.
const (
	OTPChannelEmail = "email"
	OTPChannelSMS   = "sms"

	OTPPurposeEmailVerify = "email_verify"
	OTPPurposePhoneVerify  = "phone_verify"
)

type OTPCode struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Channel   string    `json:"channel"`
	Purpose   string    `json:"purpose"`
	CodeHash  string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	Attempts  int       `json:"attempts"`
	CreatedAt time.Time `json:"created_at"`
}
