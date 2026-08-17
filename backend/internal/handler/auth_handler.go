package handler

import (
	"encoding/json"
	"net/http"
	"net/mail"
	"os"

	"github.com/diarra/backend/internal/auth"
	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/otp"
	"github.com/diarra/backend/internal/service"
)

// minPasswordLength — seuil minimal appliqué à l'inscription et à la
// réinitialisation de mot de passe. Ni l'un ni l'autre chemin ne vérifiait
// de longueur minimale auparavant (bcrypt accepte n'importe quelle longueur
// non nulle, y compris un seul caractère).
const minPasswordLength = 8

type AuthHandler struct {
	authService *service.AuthService
	firebase    *auth.FirebaseVerifier
}

func NewAuthHandler(authService *service.AuthService, firebase *auth.FirebaseVerifier) *AuthHandler {
	return &AuthHandler{authService: authService, firebase: firebase}
}

// setRefreshCookie pose le cookie httpOnly pour le refresh token.
func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, token string) {
	secure := os.Getenv("APP_ENV") == "production"
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/api/auth/refresh",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600, // 7 jours
	})
}

// clearRefreshCookie supprime le cookie refresh token.
func (h *AuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/auth/refresh",
		HttpOnly: true,
		Secure:   os.Getenv("APP_ENV") == "production",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input model.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if input.Email == "" || input.Password == "" {
		http.Error(w, `{"error":"email_and_password_required"}`, http.StatusBadRequest)
		return
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		http.Error(w, `{"error":"invalid_email"}`, http.StatusBadRequest)
		return
	}
	if len(input.Password) < minPasswordLength {
		http.Error(w, `{"error":"password_too_short"}`, http.StatusBadRequest)
		return
	}

	result, err := h.authService.Register(r.Context(), input)
	if err != nil {
		switch err {
		case service.ErrUserAlreadyExists:
			http.Error(w, `{"error":"user_already_exists"}`, http.StatusConflict)
		case service.ErrInvalidRole:
			http.Error(w, `{"error":"invalid_role"}`, http.StatusBadRequest)
		default:
			http.Error(w, `{"error":"registration_failed"}`, http.StatusInternalServerError)
		}
		return
	}

	resp := map[string]interface{}{
		"user": map[string]interface{}{
			"id":                result.User.ID,
			"email":             result.User.Email,
			"email_verified_at": result.User.EmailVerifiedAt,
			"phone_verified_at": result.User.PhoneVerifiedAt,
			"phone":             result.User.Phone,
			"roles":             result.User.Roles,
			"is_admin":          result.User.IsAdmin,
		},
		// L'access token est requis immédiatement pour pouvoir vérifier l'OTP
		// (POST /api/auth/verify-otp est authentifié).
		"access_token": result.AccessToken,
		// Vérifications en attente : l'email est toujours requis à l'inscription,
		// le téléphone uniquement s'il a été fourni.
		"pending_verifications": []string{"email"},
	}

	// En mode dev, on renvoie les codes OTP pour tester sans email/SMS
	if os.Getenv("APP_ENV") != "production" {
		resp["dev_email_otp"] = result.EmailOTP
		if result.PhoneOTP != nil && *result.PhoneOTP != "" {
			resp["dev_phone_otp"] = *result.PhoneOTP
			pv := append(resp["pending_verifications"].([]string), "phone")
			resp["pending_verifications"] = pv
		}
	}

	w.Header().Set("Content-Type", "application/json")
	h.setRefreshCookie(w, result.RefreshToken)
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input model.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if input.Email == "" || input.Password == "" {
		http.Error(w, `{"error":"email_and_password_required"}`, http.StatusBadRequest)
		return
	}

	result, err := h.authService.Login(r.Context(), input)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			http.Error(w, `{"error":"invalid_credentials"}`, http.StatusUnauthorized)
		case service.ErrAccountLocked:
			http.Error(w, `{"error":"account_locked"}`, http.StatusLocked)
		default:
			http.Error(w, `{"error":"login_failed"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	h.setRefreshCookie(w, result.RefreshToken)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": result.AccessToken,
	})
}

// Me — GET /api/auth/me (authentifié). Retourne l'utilisateur + ses rôles.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	user, err := h.authService.Me(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"me_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"user": user})
}

// AddRole — POST /api/account/roles (libre-service, sans validation admin).
// Permet à un client déjà connecté de devenir vendeur/closer sans passer par
// l'admin — ex: bouton "Devenir vendeur" dans l'espace client.
func (h *AuthHandler) AddRole(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var input model.RoleGrantInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if err := h.authService.AddRole(r.Context(), userID, input.Role); err != nil {
		switch err {
		case service.ErrInvalidRole:
			http.Error(w, `{"error":"invalid_role"}`, http.StatusBadRequest)
		default:
			http.Error(w, `{"error":"role_grant_failed"}`, http.StatusInternalServerError)
		}
		return
	}

	user, err := h.authService.Me(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"me_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"user": user})
}

// UpdateProfile — PUT /api/account/profile (nom + nom de boutique).
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var input model.UpdateProfileInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if err := h.authService.UpdateProfile(r.Context(), userID, input); err != nil {
		http.Error(w, `{"error":"profile_update_failed"}`, http.StatusInternalServerError)
		return
	}

	user, err := h.authService.Me(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"me_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"user": user})
}

// GoogleLogin — POST /api/auth/google. Le frontend authentifie l'utilisateur
// avec Firebase Auth (bouton "Continuer avec Google") et envoie ici l'ID
// token obtenu ; le backend le vérifie puis émet une session DIARRA classique
// (Firebase ne sert que de vérificateur d'identité, voir AuthService.LoginWithFirebase).
func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	if h.firebase == nil {
		http.Error(w, `{"error":"google_login_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	var input model.GoogleLoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.IDToken == "" {
		http.Error(w, `{"error":"id_token_required"}`, http.StatusBadRequest)
		return
	}

	claims, err := h.firebase.VerifyIDToken(input.IDToken)
	if err != nil {
		http.Error(w, `{"error":"invalid_google_token"}`, http.StatusUnauthorized)
		return
	}

	result, err := h.authService.LoginWithFirebase(r.Context(), claims.Email, claims.EmailVerified, claims.Name)
	if err != nil {
		http.Error(w, `{"error":"google_login_failed"}`, http.StatusInternalServerError)
		return
	}

	h.setRefreshCookie(w, result.RefreshToken)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"access_token": result.AccessToken})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, `{"error":"missing_token"}`, http.StatusUnauthorized)
		return
	}

	result, err := h.authService.RefreshToken(r.Context(), cookie.Value)
	if err != nil {
		h.clearRefreshCookie(w)
		http.Error(w, `{"error":"invalid_token"}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	h.setRefreshCookie(w, result.RefreshToken)
	json.NewEncoder(w).Encode(map[string]string{
		"access_token": result.AccessToken,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		_ = h.authService.Logout(r.Context(), cookie.Value)
	}
	h.clearRefreshCookie(w)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "logged_out"})
}

// VerifyOTP — POST /api/auth/verify-otp
// Valide un code OTP et marque l'email ou le téléphone comme vérifié.
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var input struct {
		Channel string `json:"channel"`
		Code    string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Channel == "" || input.Code == "" {
		http.Error(w, `{"error":"channel_and_code_required"}`, http.StatusBadRequest)
		return
	}

	purpose := ""
	switch input.Channel {
	case model.OTPChannelEmail:
		purpose = model.OTPPurposeEmailVerify
	case model.OTPChannelSMS:
		purpose = model.OTPPurposePhoneVerify
	default:
		http.Error(w, `{"error":"invalid_channel"}`, http.StatusBadRequest)
		return
	}

	if err := h.authService.VerifyOTP(r.Context(), userID, input.Channel, purpose, input.Code); err != nil {
		switch err {
		case service.ErrInvalidOrExpiredToken:
			http.Error(w, `{"error":"invalid_or_expired_code"}`, http.StatusBadRequest)
		case otp.ErrTooManyAttempts:
			http.Error(w, `{"error":"too_many_attempts"}`, http.StatusTooManyRequests)
		default:
			http.Error(w, `{"error":"verification_failed"}`, http.StatusBadRequest)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "verified",
		"channel": input.Channel,
	})
}

// SendOTP — POST /api/auth/send-otp
// Réémet un code OTP (resend). Auth requis.
func (h *AuthHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var input struct {
		Channel string `json:"channel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Channel == "" {
		http.Error(w, `{"error":"channel_required"}`, http.StatusBadRequest)
		return
	}

	purpose := ""
	switch input.Channel {
	case model.OTPChannelEmail:
		purpose = model.OTPPurposeEmailVerify
	case model.OTPChannelSMS:
		purpose = model.OTPPurposePhoneVerify
	default:
		http.Error(w, `{"error":"invalid_channel"}`, http.StatusBadRequest)
		return
	}

	code, err := h.authService.SendOTP(r.Context(), userID, input.Channel, purpose)
	if err != nil {
		if err == otp.ErrResendTooSoon {
			http.Error(w, `{"error":"resend_too_soon"}`, http.StatusTooManyRequests)
			return
		}
		http.Error(w, `{"error":"send_failed"}`, http.StatusInternalServerError)
		return
	}

	resp := map[string]string{"status": "sent", "channel": input.Channel}

	// En dev, on renvoie le code pour tester
	if os.Getenv("APP_ENV") != "production" {
		resp["dev_code"] = code
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var input model.VerifyEmailInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Token == "" {
		http.Error(w, `{"error":"token_required"}`, http.StatusBadRequest)
		return
	}

	if err := h.authService.VerifyEmail(r.Context(), input.Token); err != nil {
		http.Error(w, `{"error":"invalid_or_expired_token"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "verified"})
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input model.ForgotPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Email == "" {
		http.Error(w, `{"error":"email_required"}`, http.StatusBadRequest)
		return
	}

	// Toujours renvoyer "email_sent" même si le compte n'existe pas (anti-énumération)
	if err := h.authService.ForgotPassword(r.Context(), input.Email); err != nil {
		http.Error(w, `{"error":"forgot_password_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "email_sent"})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var input model.ResetPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Token == "" || input.NewPassword == "" {
		http.Error(w, `{"error":"token_and_password_required"}`, http.StatusBadRequest)
		return
	}
	if len(input.NewPassword) < minPasswordLength {
		http.Error(w, `{"error":"password_too_short"}`, http.StatusBadRequest)
		return
	}

	if err := h.authService.ResetPassword(r.Context(), input.Token, input.NewPassword); err != nil {
		if err == auth.ErrInvalidToken || err == service.ErrInvalidOrExpiredToken {
			http.Error(w, `{"error":"invalid_or_expired_token"}`, http.StatusBadRequest)
			return
		}
		http.Error(w, `{"error":"reset_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "password_reset"})
}
