package repository

import (
	"context"
	"errors"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserExists = errors.New("user already exists")

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, input model.RegisterInput, hash string) (*model.User, error) {
	user := &model.User{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, phone)
		 VALUES ($1, $2, $3)
		 RETURNING id, email, phone, is_admin, email_verified_at, phone_verified_at, failed_login_attempts, locked_until, created_at, updated_at`,
		input.Email, hash, input.Phone,
	).Scan(&user.ID, &user.Email, &user.Phone, &user.IsAdmin, &user.EmailVerifiedAt, &user.PhoneVerifiedAt, &user.FailedLoginAttempts, &user.LockedUntil, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, phone, password_hash, is_admin, email_verified_at, phone_verified_at, failed_login_attempts, locked_until, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.Phone, &user.PasswordHash, &user.IsAdmin, &user.EmailVerifiedAt, &user.PhoneVerifiedAt, &user.FailedLoginAttempts, &user.LockedUntil, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	user := &model.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, phone, password_hash, is_admin, email_verified_at, phone_verified_at, failed_login_attempts, locked_until, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Email, &user.Phone, &user.PasswordHash, &user.IsAdmin, &user.EmailVerifiedAt, &user.PhoneVerifiedAt, &user.FailedLoginAttempts, &user.LockedUntil, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (r *UserRepo) IncrementFailedAttempts(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET failed_login_attempts = failed_login_attempts + 1 WHERE id = $1`,
		userID,
	)
	return err
}

func (r *UserRepo) ResetFailedAttempts(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET failed_login_attempts = 0, locked_until = NULL WHERE id = $1`,
		userID,
	)
	return err
}

func (r *UserRepo) LockAccount(ctx context.Context, userID string, until string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET locked_until = $2 WHERE id = $1`,
		userID, until,
	)
	return err
}

func (r *UserRepo) VerifyEmail(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET email_verified_at = NOW() WHERE id = $1`,
		userID,
	)
	return err
}

func (r *UserRepo) UpdatePassword(ctx context.Context, userID, hash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`,
		userID, hash,
	)
	return err
}

func (r *UserRepo) AddRole(ctx context.Context, userID, role string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, role,
	)
	return err
}

func (r *UserRepo) HasRole(ctx context.Context, userID, role string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_roles WHERE user_id = $1 AND role = $2)`,
		userID, role,
	).Scan(&exists)
	return exists, err
}

// ListAllUsers retourne tous les utilisateurs (pour l'admin).
func (r *UserRepo) ListAllUsers(ctx context.Context) ([]*model.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, email, phone, password_hash, is_admin, email_verified_at, phone_verified_at,
			failed_login_attempts, locked_until, created_at, updated_at
		 FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*model.User{}
	for rows.Next() {
		user := &model.User{}
		err := rows.Scan(&user.ID, &user.Email, &user.Phone, &user.PasswordHash, &user.IsAdmin,
			&user.EmailVerifiedAt, &user.PhoneVerifiedAt, &user.FailedLoginAttempts,
			&user.LockedUntil, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// CountAllUsers compte les utilisateurs (pour les stats admin).
func (r *UserRepo) CountAllUsers(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}
