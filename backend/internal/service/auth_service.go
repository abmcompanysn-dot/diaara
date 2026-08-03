package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/diarra/backend/internal/auth"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountLocked     = errors.New("account is locked")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type AuthService struct {
	userRepo    *repository.UserRepo
	jwtManager  *auth.JWTManager
}

func NewAuthService(userRepo *repository.UserRepo, jwtManager *auth.JWTManager) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

type RegisterResult struct {
	User         *model.User
	AccessToken  string
	RefreshToken string
}

func (s *AuthService) Register(ctx context.Context, input model.RegisterInput) (*RegisterResult, error) {
	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.Create(ctx, input, hash)
	if err != nil {
		return nil, ErrUserAlreadyExists
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.IsAdmin)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &RegisterResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
}

func (s *AuthService) Login(ctx context.Context, input model.LoginInput) (*LoginResult, error) {
	user, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, ErrAccountLocked
	}

	if !auth.CheckPassword(input.Password, user.PasswordHash) {
		s.userRepo.IncrementFailedAttempts(ctx, user.ID)
		
		user, _ := s.userRepo.FindByEmail(ctx, input.Email)
		if user != nil && user.FailedLoginAttempts >= 4 {
			lockUntil := time.Now().Add(15 * time.Minute).Format(time.RFC3339)
			s.userRepo.LockAccount(ctx, user.ID, lockUntil)
		}
		
		return nil, ErrInvalidCredentials
	}

	s.userRepo.ResetFailedAttempts(ctx, user.ID)

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.IsAdmin)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResult, error) {
	claims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, auth.ErrInvalidToken
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.IsAdmin)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, userID, token string) error {
	return s.userRepo.VerifyEmail(ctx, userID)
}

func (s *AuthService) ResetPassword(ctx context.Context, userID, newPassword string) error {
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.userRepo.UpdatePassword(ctx, userID, hash)
}

func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
