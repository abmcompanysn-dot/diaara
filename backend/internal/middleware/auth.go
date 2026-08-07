package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/diarra/backend/internal/auth"
	"github.com/diarra/backend/internal/model"
)

type contextKey string

const UserIDKey contextKey = "user_id"
const IsAdminKey contextKey = "is_admin"
const RolesKey contextKey = "roles"
const EmailVerifiedKey contextKey = "email_verified"
const PhoneVerifiedKey contextKey = "phone_verified"

func RequireAuth(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Header Authorization (API classique)
			header := r.Header.Get("Authorization")
			token := ""
			if strings.HasPrefix(header, "Bearer ") {
				token = strings.TrimPrefix(header, "Bearer ")
			} else if isWebSocketUpgrade(r) {
				// Un navigateur ne peut pas envoyer de header sur un WebSocket :
				// on accepte le token via la query string, réservé aux upgrades.
				token = r.URL.Query().Get("token")
			}
			if token == "" {
				http.Error(w, `{"error":"missing_token"}`, http.StatusUnauthorized)
				return
			}
			claims, err := jwtManager.ValidateAccessToken(token)
			if err != nil {
				http.Error(w, `{"error":"invalid_token"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, IsAdminKey, claims.IsAdmin)
			ctx = context.WithValue(ctx, RolesKey, claims.Roles)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// isWebSocketUpgrade détecte une tentative d'upgrade WebSocket (RFC 6455).
func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

func GetIsAdmin(ctx context.Context) bool {
	if v, ok := ctx.Value(IsAdminKey).(bool); ok {
		return v
	}
	return false
}

func GetRoles(ctx context.Context) []string {
	if v, ok := ctx.Value(RolesKey).([]string); ok {
		return v
	}
	return nil
}

// HasRole vérifie la présence d'un rôle dans le contexte (rôles cumulables).
func HasRole(ctx context.Context, role string) bool {
	for _, r := range GetRoles(ctx) {
		if r == role {
			return true
		}
	}
	return false
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAdmin := GetIsAdmin(r.Context())
		if !isAdmin {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole exige au moins un des rôles donnés. À utiliser après RequireAuth.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, role := range roles {
				if HasRole(r.Context(), role) {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		})
	}
}

// GetVerifiedFlags récupère les drapeaux de vérification email/téléphone
// depuis le contexte (positionnés par le middleware RequireVerified).
func GetEmailVerified(ctx context.Context) bool {
	if v, ok := ctx.Value(EmailVerifiedKey).(bool); ok {
		return v
	}
	return false
}

func GetPhoneVerified(ctx context.Context) bool {
	if v, ok := ctx.Value(PhoneVerifiedKey).(bool); ok {
		return v
	}
	return false
}

// RequireVerifiedEmail bloque si l'email n'est pas vérifié.
// Doit être placé après RequireAuth.
func RequireVerifiedEmail(userRepo interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserID(r.Context())
			if userID == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			user, err := userRepo.FindByID(r.Context(), userID)
			if err != nil || user.EmailVerifiedAt == nil {
				http.Error(w, `{"error":"email_not_verified"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireVerifiedPhone bloque si le téléphone n'est pas vérifié.
// Doit être placé après RequireAuth.
func RequireVerifiedPhone(userRepo interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserID(r.Context())
			if userID == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			user, err := userRepo.FindByID(r.Context(), userID)
			if err != nil || user.Phone == nil || user.PhoneVerifiedAt == nil {
				http.Error(w, `{"error":"phone_not_verified"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
