package handler

import (
	"encoding/json"
	"net/http"

	"github.com/diarra/backend/internal/auth"
	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": map[string]string{
			"id":    result.User.ID,
			"email": result.User.Email,
		},
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
	})
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
	json.NewEncoder(w).Encode(map[string]string{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
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

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var input model.RefreshInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	result, err := h.authService.RefreshToken(r.Context(), input.RefreshToken)
	if err != nil {
		http.Error(w, `{"error":"invalid_token"}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var input model.LogoutInput
	_ = json.NewDecoder(r.Body).Decode(&input)

	if err := h.authService.Logout(r.Context(), input.RefreshToken); err != nil {
		http.Error(w, `{"error":"logout_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "logged_out"})
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
