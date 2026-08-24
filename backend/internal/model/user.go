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
	DisplayName         *string    `json:"display_name,omitempty"`
	ShopName            *string    `json:"shop_name,omitempty"`
	FacebookPixelID     *string    `json:"facebook_pixel_id,omitempty"`
	GoogleTagID         *string    `json:"google_tag_id,omitempty"`
	PasswordHash        string     `json:"-"`
	EmailVerifiedAt     *time.Time `json:"email_verified_at,omitempty"`
	PhoneVerifiedAt     *time.Time `json:"phone_verified_at,omitempty"`
	IsAdmin             bool       `json:"is_admin"`
	Roles               []string   `json:"roles,omitempty"`
	FailedLoginAttempts int        `json:"-"`
	LockedUntil         *time.Time `json:"locked_until,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type RegisterInput struct {
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	Phone       *string `json:"phone,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	ShopName    *string `json:"shop_name,omitempty"`
	// Rôles demandés à l'inscription (auto-inscription), cumulables :
	// "vendeur", "closer". Le rôle "client" est implicite pour tous.
	Roles []string `json:"roles,omitempty"`
}

// UpdateProfileInput — PUT /api/account/profile (nom + nom de boutique,
// typiquement rempli au moment de devenir vendeur).
type UpdateProfileInput struct {
	DisplayName *string `json:"display_name,omitempty"`
	ShopName    *string `json:"shop_name,omitempty"`
}

// UpdateAdTrackingInput — PUT /api/account/ad-tracking. Chaque vendeur gère
// sa propre publicité (Facebook Ads, Google Ads) : ces identifiants sont
// injectés côté frontend sur sa boutique et ses pages produit, pour qu'il
// voie ses propres visites/conversions dans ses outils pub. Champs vides
// (chaîne vide) effacent l'identifiant enregistré.
type UpdateAdTrackingInput struct {
	FacebookPixelID *string `json:"facebook_pixel_id,omitempty"`
	GoogleTagID     *string `json:"google_tag_id,omitempty"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// GoogleLoginInput — POST /api/auth/google. IDToken est le jeton Firebase
// obtenu côté frontend après la connexion Google (Firebase Auth JS SDK).
type GoogleLoginInput struct {
	IDToken string `json:"id_token"`
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

// RoleGrantInput — libre-service (POST /api/account/roles) : un utilisateur
// connecté ne peut que s'ajouter un rôle, jamais le révoquer.
type RoleGrantInput struct {
	Role string `json:"role"`
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
