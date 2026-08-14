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
const AdminPermissionsKey contextKey = "admin_permissions"

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
			ctx = context.WithValue(ctx, AdminPermissionsKey, claims.AdminPermissions)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth remplit le contexte utilisateur si un token Bearer valide est
// présent, mais ne bloque jamais la requête (token absent ou invalide = accès
// public normal). Utile pour les routes publiques dont le contenu dépend de
// l'identité du visiteur (ex: un vendeur qui consulte son propre produit en
// attente de modération).
func OptionalAuth(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				next.ServeHTTP(w, r)
				return
			}
			token := strings.TrimPrefix(header, "Bearer ")
			claims, err := jwtManager.ValidateAccessToken(token)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, IsAdminKey, claims.IsAdmin)
			ctx = context.WithValue(ctx, RolesKey, claims.Roles)
			ctx = context.WithValue(ctx, AdminPermissionsKey, claims.AdminPermissions)
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

// RequireAutomation autorise soit un admin authentifié (JWT classique), soit
// une clé d'automatisation valide envoyée dans le header X-Automation-Key —
// pour que les endpoints de création automatisée de produit (IA/script)
// n'exigent pas forcément une session de connexion complète. getKey lit la
// clé actuellement stockée (vide = aucune clé générée, header refusé).
func RequireAutomation(jwtManager *auth.JWTManager, getKey func(ctx context.Context) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key := r.Header.Get("X-Automation-Key"); key != "" {
				stored := getKey(r.Context())
				if stored == "" || key != stored {
					http.Error(w, `{"error":"invalid_automation_key"}`, http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			// Repli sur l'authentification admin classique.
			RequireAuth(jwtManager)(RequireAdmin(next)).ServeHTTP(w, r)
		})
	}
}

func GetAdminPermissions(ctx context.Context) []string {
	if v, ok := ctx.Value(AdminPermissionsKey).([]string); ok {
		return v
	}
	return nil
}

// IsUnrestrictedAdmin : un admin sans scope assigné a l'accès complet (legacy).
func IsUnrestrictedAdmin(ctx context.Context) bool {
	return GetIsAdmin(ctx) && len(GetAdminPermissions(ctx)) == 0
}

// RequireAdminScope exige le scope donné. Un admin sans aucun scope assigné
// garde l'accès complet (comportement historique, pas de migration de données
// nécessaire pour le compte admin existant). Doit suivre RequireAuth+RequireAdmin.
func RequireAdminScope(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			perms := GetAdminPermissions(r.Context())
			if len(perms) > 0 {
				allowed := false
				for _, p := range perms {
					if p == perm {
						allowed = true
						break
					}
				}
				if !allowed {
					http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireUnrestrictedAdmin réserve l'action aux admins à accès complet — utilisé
// pour les opérations de gestion des accès elles-mêmes (promouvoir un admin,
// changer ses scopes), afin qu'un admin restreint ne puisse pas s'auto-accorder
// plus de droits. Doit suivre RequireAuth+RequireAdmin.
func RequireUnrestrictedAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsUnrestrictedAdmin(r.Context()) {
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
