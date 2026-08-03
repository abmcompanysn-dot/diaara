package service

import (
	"context"
	"errors"
	"time"

	"github.com/diarra/backend/internal/auth"
	"github.com/diarra/backend/internal/email"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
)

var (
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrAccountLocked         = errors.New("account is locked")
	ErrUserAlreadyExists     = errors.New("user already exists")
	ErrInvalidOrExpiredToken = errors.New("invalid or expired token")
	ErrInvalidRole           = errors.New("invalid role")
)

const (
	emailVerificationTTL = 48 * time.Hour
	passwordResetTTL     = 1 * time.Hour
)

type AuthService struct {
	userRepo      *repository.UserRepo
	jwtManager    *auth.JWTManager
	notifications *email.NotificationService
}

func NewAuthService(userRepo *repository.UserRepo, jwtManager *auth.JWTManager, notifications *email.NotificationService) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		jwtManager:    jwtManager,
		notifications: notifications,
	}
}

type RegisterResult struct {
	User         *model.User
	AccessToken  string
	RefreshToken string
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
}

// Register crée le compte, stocke un token de vérification d'email et
// enregistre la session (refresh token révocable).
func (s *AuthService) Register(ctx context.Context, input model.RegisterInput) (*RegisterResult, error) {
	// Validation des rôles avant toute écriture en base.
	for _, role := range input.Roles {
		if !model.ValidRole(role) {
			return nil, ErrInvalidRole
		}
	}

	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.Create(ctx, input, hash)
	if err != nil {
		return nil, ErrUserAlreadyExists
	}

	// Auto-inscription : les rôles demandés sont accordés immédiatement
	// (cumulables).
	if err := s.grantRoles(ctx, user.ID, input.Roles); err != nil {
		return nil, err
	}

	// Vérification d'email : token opaque stocké en hash + email (si configuré)
	s.issueEmailVerification(ctx, user.ID, user.Email)

	roles := s.loadRoles(ctx, user.ID)

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.IsAdmin, roles)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.issueRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &RegisterResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Login valide le mot de passe (avec verrouillage après échecs répétés) puis
// enregistre une nouvelle session.
func (s *AuthService) Login(ctx context.Context, input model.LoginInput) (*LoginResult, error) {
	user, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, ErrAccountLocked
	}

	if !auth.CheckPassword(input.Password, user.PasswordHash) {
		if err := s.userRepo.IncrementFailedAttempts(ctx, user.ID); err == nil {
			current, _ := s.userRepo.FindByEmail(ctx, input.Email)
			if current != nil && current.FailedLoginAttempts >= 4 {
				lockUntil := time.Now().Add(15 * time.Minute)
				_ = s.userRepo.LockAccount(ctx, user.ID, lockUntil.Format(time.RFC3339))
			}
		}
		return nil, ErrInvalidCredentials
	}

	_ = s.userRepo.ResetFailedAttempts(ctx, user.ID)

	roles := s.loadRoles(ctx, user.ID)

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.IsAdmin, roles)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.issueRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// RefreshToken valide le refresh token présenté, révoque l'ancien et émet une
// nouvelle paire (rotation de session).
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResult, error) {
	claims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, auth.ErrInvalidToken
	}

	stored, err := s.userRepo.FindRefreshToken(ctx, auth.HashToken(refreshToken))
	if err != nil || stored.RevokedAt != nil {
		return nil, auth.ErrInvalidToken
	}
	if stored.UserID != claims.UserID {
		return nil, auth.ErrInvalidToken
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, auth.ErrInvalidToken
	}

	// Rotation : on révoque l'ancien refresh token, on en émet un nouveau.
	_ = s.userRepo.RevokeRefreshToken(ctx, auth.HashToken(refreshToken))

	roles := s.loadRoles(ctx, user.ID)

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.IsAdmin, roles)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := s.issueRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

// Logout révoque le refresh token présenté : la session ne peut plus être renouvelée.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return s.userRepo.RevokeRefreshToken(ctx, auth.HashToken(refreshToken))
}

// Me retourne l'utilisateur avec ses rôles (client implicite, rôles
// cumulables + is_admin). Utilisé par le frontend pour restaurer la session.
func (s *AuthService) Me(ctx context.Context, userID string) (*model.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.Roles = s.loadRoles(ctx, userID)
	return user, nil
}

// VerifyEmail marque l'email comme vérifié si le token est valide et non expiré.
func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	v, err := s.userRepo.FindEmailVerification(ctx, auth.HashToken(token))
	if err != nil || v.UsedAt != nil || time.Now().After(v.ExpiresAt) {
		return ErrInvalidOrExpiredToken
	}

	if err := s.userRepo.VerifyEmail(ctx, v.UserID); err != nil {
		return err
	}
	return s.userRepo.MarkEmailVerificationUsed(ctx, v.ID)
}

// ForgotPassword émet un token de réinitialisation et envoie le lien par email.
// La réponse est toujours la même pour ne pas révéler l'existence d'un compte.
func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil
	}

	token, err := auth.GenerateToken()
	if err != nil {
		return err
	}
	if err := s.userRepo.CreatePasswordReset(ctx, user.ID, auth.HashToken(token), time.Now().Add(passwordResetTTL)); err != nil {
		return err
	}

	if s.notifications != nil {
		go s.notifications.SendPasswordReset(context.Background(), user.Email, token)
	}
	return nil
}

// ResetPassword valide le token et remplace le mot de passe.
// Toutes les sessions existantes sont révoquées (mesure de sécurité).
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	p, err := s.userRepo.FindPasswordReset(ctx, auth.HashToken(token))
	if err != nil || p.UsedAt != nil || time.Now().After(p.ExpiresAt) {
		return ErrInvalidOrExpiredToken
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := s.userRepo.UpdatePassword(ctx, p.UserID, hash); err != nil {
		return err
	}
	if err := s.userRepo.MarkPasswordResetUsed(ctx, p.ID); err != nil {
		return err
	}
	return s.userRepo.RevokeAllUserRefreshTokens(ctx, p.UserID)
}

// loadRoles charge les rôles cumulables de l'utilisateur. Une erreur de
// lecture ne bloque pas l'authentification : on retombe sur aucun rôle.
func (s *AuthService) loadRoles(ctx context.Context, userID string) []string {
	roles, err := s.userRepo.ListRoles(ctx, userID)
	if err != nil {
		return []string{}
	}
	return roles
}

// grantRoles valide et attribue les rôles demandés à l'inscription.
func (s *AuthService) grantRoles(ctx context.Context, userID string, roles []string) error {
	for _, role := range roles {
		if !model.ValidRole(role) {
			return ErrInvalidRole
		}
		if err := s.userRepo.AddRole(ctx, userID, role); err != nil {
			return err
		}
	}
	return nil
}

// issueEmailVerification génère et stocke un token de vérification d'email,
// puis envoie le lien de vérification en arrière-plan si les emails sont configurés.
func (s *AuthService) issueEmailVerification(ctx context.Context, userID, userEmail string) {
	token, err := auth.GenerateToken()
	if err != nil {
		return
	}
	if err := s.userRepo.CreateEmailVerification(ctx, userID, auth.HashToken(token), time.Now().Add(emailVerificationTTL)); err != nil {
		return
	}
	if s.notifications != nil && userEmail != "" {
		go s.notifications.SendEmailVerification(context.Background(), userEmail, token)
	}
}

// issueRefreshToken génère un refresh token JWT et stocke son hash en base
// pour pouvoir le révoquer.
func (s *AuthService) issueRefreshToken(ctx context.Context, userID string) (string, error) {
	token, err := s.jwtManager.GenerateRefreshToken(userID)
	if err != nil {
		return "", err
	}
	if err := s.userRepo.CreateRefreshToken(ctx, userID, auth.HashToken(token), time.Now().Add(s.jwtManager.RefreshTTL())); err != nil {
		return "", err
	}
	return token, nil
}
