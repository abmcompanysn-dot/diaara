package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diarra/backend/internal/auth"
	"github.com/diarra/backend/internal/email"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/otp"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/sms"
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
	userRepo           *repository.UserRepo
	otpRepo            *repository.OTPRepo
	otpService         *otp.Service
	smsSender          sms.Sender
	jwtManager         *auth.JWTManager
	notifications      *email.NotificationService
	adminPermissionRepo *repository.AdminPermissionRepo
}

func NewAuthService(userRepo *repository.UserRepo, otpRepo *repository.OTPRepo, otpService *otp.Service, smsSender sms.Sender, jwtManager *auth.JWTManager, notifications *email.NotificationService, adminPermissionRepo *repository.AdminPermissionRepo) *AuthService {
	return &AuthService{
		userRepo:            userRepo,
		otpRepo:             otpRepo,
		otpService:          otpService,
		smsSender:           smsSender,
		jwtManager:          jwtManager,
		notifications:       notifications,
		adminPermissionRepo: adminPermissionRepo,
	}
}

type RegisterResult struct {
	User         *model.User
	AccessToken  string
	RefreshToken string
	EmailOTP     string
	PhoneOTP     *string
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
}

// Register crée le compte, émet un OTP email obligatoire (+ SMS si téléphone)
// et enregistre la session (refresh token révocable).
func (s *AuthService) Register(ctx context.Context, input model.RegisterInput) (*RegisterResult, error) {
	// Validation des rôles avant toute écriture en base.
	for _, role := range input.Roles {
		if !model.ValidRole(role) {
			return nil, ErrInvalidRole
		}
	}

	// Téléphone vide → NULL en base (colonne UNIQUE : "" casserait la 2e
	// inscription sans téléphone).
	if input.Phone != nil && *input.Phone == "" {
		input.Phone = nil
	}

	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.Create(ctx, input, hash)
	if err != nil {
		if repository.IsUniqueViolation(err) {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	// Auto-inscription : les rôles demandés sont accordés immédiatement
	// (cumulables).
	if err := s.grantRoles(ctx, user.ID, input.Roles); err != nil {
		return nil, err
	}

	// Émettre OTP email obligatoire
	emailCode, err := s.otpService.Issue(ctx, user.ID, model.OTPChannelEmail, model.OTPPurposeEmailVerify)
	if err != nil {
		return nil, err
	}
	if s.notifications != nil {
		go s.notifications.SendOTP(context.Background(), user.Email, emailCode, "vérification de l'email")
		go s.notifyAdmins(context.Background(), func(ctx context.Context, adminEmail string) error {
			return s.notifications.SendAdminNewSignup(ctx, adminEmail, user.Email)
		})
	}

	// Émettre OTP SMS si téléphone fourni
	var smsCode string
	if user.Phone != nil {
		smsCode, err = s.otpService.Issue(ctx, user.ID, model.OTPChannelSMS, model.OTPPurposePhoneVerify)
		if err != nil {
			return nil, err
		}
		if s.smsSender != nil {
			go s.smsSender.Send(context.Background(), *user.Phone, fmt.Sprintf("Votre code DIARRA : %s", smsCode))
		}
	}

	roles := s.loadRoles(ctx, user.ID)
	adminPerms := s.loadAdminPermissions(ctx, user.IsAdmin, user.ID)

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.IsAdmin, roles, adminPerms)
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
		EmailOTP:     emailCode,
		PhoneOTP:     &smsCode,
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

	return s.issueSession(ctx, user)
}

// issueSession émet la paire access/refresh token pour un utilisateur déjà
// authentifié par un autre moyen (mot de passe validé, Firebase, etc.).
func (s *AuthService) issueSession(ctx context.Context, user *model.User) (*LoginResult, error) {
	roles := s.loadRoles(ctx, user.ID)
	adminPerms := s.loadAdminPermissions(ctx, user.IsAdmin, user.ID)

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.IsAdmin, roles, adminPerms)
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

// LoginWithFirebase trouve-ou-crée un compte à partir d'une identité Firebase
// déjà vérifiée (voir handler.AuthHandler.GoogleLogin) et émet une session
// DIARRA classique — Firebase ne sert que de vérificateur d'identité Google,
// pas de gestionnaire de session (voir plan : le système JWT/rôles actuel
// reste la source de vérité).
func (s *AuthService) LoginWithFirebase(ctx context.Context, email string, emailVerified bool, displayName string) (*LoginResult, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// Nouveau compte : pas de mot de passe utilisable (même pattern que le
		// compte invité créé au premier achat), l'utilisateur se reconnectera
		// toujours via Google.
		randomPassword, genErr := auth.GenerateToken()
		if genErr != nil {
			return nil, genErr
		}
		hash, hashErr := auth.HashPassword(randomPassword)
		if hashErr != nil {
			return nil, hashErr
		}
		var name *string
		if displayName != "" {
			name = &displayName
		}
		user, err = s.userRepo.Create(ctx, model.RegisterInput{Email: email, Password: randomPassword, DisplayName: name}, hash)
		if err != nil {
			return nil, err
		}
		if emailVerified {
			_ = s.userRepo.VerifyEmail(ctx, user.ID)
		}
		return s.issueSession(ctx, user)
	}

	if emailVerified && user.EmailVerifiedAt == nil {
		// Le compte existait déjà mais son email n'avait jamais été vérifié —
		// il a pu être pré-enregistré par quelqu'un d'autre avec un mot de
		// passe, sans jamais prouver qu'il possède réellement cette adresse
		// (voir Register : aucune vérification n'est requise pour qu'un compte
		// devienne utilisable par mot de passe). Google vient de prouver que
		// CET utilisateur est le vrai propriétaire de l'adresse : on invalide
		// l'ancien mot de passe et on révoque toutes les sessions existantes
		// pour empêcher une prise de contrôle persistante par un tiers qui
		// aurait pré-enregistré ce compte.
		if randomPassword, genErr := auth.GenerateToken(); genErr == nil {
			if hash, hashErr := auth.HashPassword(randomPassword); hashErr == nil {
				_ = s.userRepo.UpdatePassword(ctx, user.ID, hash)
			}
		}
		_ = s.userRepo.RevokeAllUserRefreshTokens(ctx, user.ID)
		_ = s.userRepo.VerifyEmail(ctx, user.ID)
	}

	return s.issueSession(ctx, user)
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
	adminPerms := s.loadAdminPermissions(ctx, user.IsAdmin, user.ID)

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.IsAdmin, roles, adminPerms)
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

// VerifyOTP valide un code OTP pour le canal et l'usage donnés.
// Sur succès, marque l'email ou le téléphone comme vérifié.
func (s *AuthService) VerifyOTP(ctx context.Context, userID, channel, purpose, code string) error {
	if err := s.otpService.Verify(ctx, userID, channel, purpose, code); err != nil {
		return err
	}

	// Marque le champ correspondant comme vérifié
	switch purpose {
	case model.OTPPurposeEmailVerify:
		if err := s.userRepo.VerifyEmail(ctx, userID); err != nil {
			return err
		}
		if s.notifications != nil {
			if user, err := s.userRepo.FindByID(ctx, userID); err == nil {
				go s.notifyAdmins(context.Background(), func(ctx context.Context, adminEmail string) error {
					return s.notifications.SendAdminEmailVerified(ctx, adminEmail, user.Email)
				})
			}
		}
		return nil
	case model.OTPPurposePhoneVerify:
		return s.userRepo.VerifyPhone(ctx, userID)
	default:
		return nil
	}
}

// notifyAdmins envoie un email à chaque administrateur en tâche de fond —
// une erreur sur un envoi n'empêche pas les autres, et n'est jamais
// remontée à l'appelant (ces notifications sont secondaires, jamais sur le
// chemin critique d'une action utilisateur).
func (s *AuthService) notifyAdmins(ctx context.Context, send func(ctx context.Context, adminEmail string) error) {
	emails, err := s.userRepo.ListAdminEmails(ctx)
	if err != nil {
		return
	}
	for _, email := range emails {
		_ = send(ctx, email)
	}
}

// VerifyPhoneWithFirebase marque le téléphone de l'utilisateur vérifié à
// partir d'un numéro confirmé par Firebase Phone Auth (SMS envoyé et
// vérifié entièrement côté Firebase — le backend ne génère ni n'envoie
// aucun code lui-même pour ce canal). Remplace le canal SMS de l'ancien
// flux OTP maison (voir VerifyOTP, toujours utilisé pour l'email).
func (s *AuthService) VerifyPhoneWithFirebase(ctx context.Context, userID, phoneNumber string) error {
	if phoneNumber == "" {
		return ErrInvalidOrExpiredToken
	}
	return s.userRepo.SetVerifiedPhone(ctx, userID, phoneNumber)
}

// SendOTP réémet un code OTP pour le canal demandé (resend).
func (s *AuthService) SendOTP(ctx context.Context, userID, channel, purpose string) (string, error) {
	code, err := s.otpService.Issue(ctx, userID, channel, purpose)
	if err != nil {
		return "", err
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return code, nil
	}

	switch channel {
	case model.OTPChannelEmail:
		if s.notifications != nil {
			go s.notifications.SendOTP(context.Background(), user.Email, code, purposeLabel(purpose))
		}
	case model.OTPChannelSMS:
		if user.Phone != nil && s.smsSender != nil {
			go s.smsSender.Send(context.Background(), *user.Phone, fmt.Sprintf("Votre code DIARRA : %s", code))
		}
	}

	return code, nil
}

func purposeLabel(purpose string) string {
	switch purpose {
	case model.OTPPurposeEmailVerify:
		return "vérification de l'email"
	case model.OTPPurposePhoneVerify:
		return "vérification du téléphone"
	default:
		return "vérification"
	}
}

// VerifyEmail marque l'email comme vérifié si le token est valide et non expiré.
// (Ancien flux par lien — conservé pour compatibilité pendant la transition.)
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

// loadRoles charge les rôles cumulables de l'utilisateur.
func (s *AuthService) loadRoles(ctx context.Context, userID string) []string {
	roles, err := s.userRepo.ListRoles(ctx, userID)
	if err != nil {
		return []string{}
	}
	return roles
}

// loadAdminPermissions retourne les scopes admin d'un utilisateur admin
// (liste vide = accès complet). Non pertinent pour un non-admin.
func (s *AuthService) loadAdminPermissions(ctx context.Context, isAdmin bool, userID string) []string {
	if !isAdmin {
		return nil
	}
	perms, err := s.adminPermissionRepo.ListForUser(ctx, userID)
	if err != nil {
		return []string{}
	}
	return perms
}

// UpdateProfile enregistre le nom affiché et/ou le nom de boutique d'un
// utilisateur (ex: formulaire "devenir vendeur").
func (s *AuthService) UpdateProfile(ctx context.Context, userID string, input model.UpdateProfileInput) error {
	return s.userRepo.SetProfile(ctx, userID, input.DisplayName, input.ShopName)
}

// UpdateAdTracking enregistre le Facebook Pixel / Google Tag du vendeur
// (publicité gérée par le vendeur lui-même sur sa boutique DIARRA).
func (s *AuthService) UpdateAdTracking(ctx context.Context, userID string, input model.UpdateAdTrackingInput) error {
	return s.userRepo.SetAdTracking(ctx, userID, input)
}

// AddRole attribue un rôle cumulable (vendeur/closer) à un compte déjà
// existant — libre-service, sans validation admin (ex: bouton "devenir
// vendeur" depuis l'espace client).
func (s *AuthService) AddRole(ctx context.Context, userID, role string) error {
	return s.grantRoles(ctx, userID, []string{role})
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
