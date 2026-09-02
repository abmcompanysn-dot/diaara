package repository

import (
	"context"
	"errors"
	"time"

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
		`INSERT INTO users (email, password_hash, phone, display_name, shop_name)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, email, phone, display_name, shop_name, is_admin, email_verified_at, phone_verified_at, failed_login_attempts, locked_until, created_at, updated_at`,
		input.Email, hash, input.Phone, input.DisplayName, input.ShopName,
	).Scan(&user.ID, &user.Email, &user.Phone, &user.DisplayName, &user.ShopName, &user.IsAdmin, &user.EmailVerifiedAt, &user.PhoneVerifiedAt, &user.FailedLoginAttempts, &user.LockedUntil, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, phone, display_name, shop_name, password_hash, is_admin, email_verified_at, phone_verified_at, failed_login_attempts, locked_until, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.Phone, &user.DisplayName, &user.ShopName, &user.PasswordHash, &user.IsAdmin, &user.EmailVerifiedAt, &user.PhoneVerifiedAt, &user.FailedLoginAttempts, &user.LockedUntil, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// SetProfile enregistre/modifie le nom affiché et le nom de boutique d'un
// utilisateur (ex: formulaire "devenir vendeur").
func (r *UserRepo) SetProfile(ctx context.Context, userID string, displayName, shopName *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET display_name = $2, shop_name = $3 WHERE id = $1`,
		userID, displayName, shopName)
	return err
}

// GetPayoutMethod retourne le moyen de versement enregistré du vendeur
// (nil si jamais renseigné).
func (r *UserRepo) GetPayoutMethod(ctx context.Context, userID string) (phone, operator, country *string, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT payout_phone, payout_operator, payout_country FROM users WHERE id = $1`, userID,
	).Scan(&phone, &operator, &country)
	return
}

// SetPayoutMethod enregistre/remplace le moyen de versement du vendeur.
// phone et operator sont déjà normalisés/validés par l'appelant.
func (r *UserRepo) SetPayoutMethod(ctx context.Context, userID, phone, operator, country string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET payout_phone = $2, payout_operator = $3, payout_country = $4 WHERE id = $1`,
		userID, phone, operator, country)
	return err
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	user := &model.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, phone, display_name, shop_name, facebook_pixel_id, google_tag_id, password_hash, is_admin, email_verified_at, phone_verified_at, failed_login_attempts, locked_until, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Email, &user.Phone, &user.DisplayName, &user.ShopName, &user.FacebookPixelID, &user.GoogleTagID, &user.PasswordHash, &user.IsAdmin, &user.EmailVerifiedAt, &user.PhoneVerifiedAt, &user.FailedLoginAttempts, &user.LockedUntil, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// SetAdTracking enregistre le Facebook Pixel / Google Tag propres au
// vendeur (publicité qu'il gère lui-même). Mise à jour partielle : un champ
// nil dans l'input n'écrase pas la valeur existante (contrairement à
// SetProfile, qui remplace toujours les deux colonnes).
func (r *UserRepo) SetAdTracking(ctx context.Context, userID string, input model.UpdateAdTrackingInput) error {
	sets := []string{}
	args := []interface{}{}
	argIdx := 1

	if input.FacebookPixelID != nil {
		sets = append(sets, `facebook_pixel_id = NULLIF($`+itoa(argIdx)+`, '')`)
		args = append(args, *input.FacebookPixelID)
		argIdx++
	}
	if input.GoogleTagID != nil {
		sets = append(sets, `google_tag_id = NULLIF($`+itoa(argIdx)+`, '')`)
		args = append(args, *input.GoogleTagID)
		argIdx++
	}
	if len(sets) == 0 {
		return nil
	}

	query := `UPDATE users SET ` + join(sets, ", ") + ` WHERE id = $` + itoa(argIdx)
	args = append(args, userID)
	_, err := r.pool.Exec(ctx, query, args...)
	return err
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
		`UPDATE users SET email_verified_at = NOW(), updated_at = NOW() WHERE id = $1`,
		userID,
	)
	return err
}

func (r *UserRepo) VerifyPhone(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET phone_verified_at = NOW(), updated_at = NOW() WHERE id = $1`,
		userID,
	)
	return err
}

// SetVerifiedPhone enregistre le numéro confirmé par Firebase Phone Auth et
// le marque vérifié en une seule opération — le numéro E.164 renvoyé par
// Firebase remplace celui saisi librement à l'inscription (potentiellement
// mal formaté), puisque Firebase vient de prouver que l'utilisateur le
// possède réellement.
func (r *UserRepo) SetVerifiedPhone(ctx context.Context, userID, phone string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET phone = $2, phone_verified_at = NOW(), updated_at = NOW() WHERE id = $1`,
		userID, phone,
	)
	return err
}

func (r *UserRepo) CreateEmailVerification(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_verifications (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

func (r *UserRepo) FindEmailVerification(ctx context.Context, tokenHash string) (*model.EmailVerification, error) {
	v := &model.EmailVerification{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, used_at, created_at
		 FROM email_verifications WHERE token_hash = $1`,
		tokenHash,
	).Scan(&v.ID, &v.UserID, &v.TokenHash, &v.ExpiresAt, &v.UsedAt, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (r *UserRepo) MarkEmailVerificationUsed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_verifications SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`,
		id,
	)
	return err
}

func (r *UserRepo) CreatePasswordReset(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO password_resets (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

func (r *UserRepo) FindPasswordReset(ctx context.Context, tokenHash string) (*model.PasswordReset, error) {
	p := &model.PasswordReset{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, used_at, created_at
		 FROM password_resets WHERE token_hash = $1`,
		tokenHash,
	).Scan(&p.ID, &p.UserID, &p.TokenHash, &p.ExpiresAt, &p.UsedAt, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *UserRepo) MarkPasswordResetUsed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE password_resets SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`,
		id,
	)
	return err
}

func (r *UserRepo) CreateRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

func (r *UserRepo) FindRefreshToken(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	t := &model.RefreshToken{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		 FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *UserRepo) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL`,
		tokenHash,
	)
	return err
}

func (r *UserRepo) RevokeAllUserRefreshTokens(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`,
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

func (r *UserRepo) RemoveRole(ctx context.Context, userID, role string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM user_roles WHERE user_id = $1 AND role = $2`,
		userID, role,
	)
	return err
}

// ListRoles retourne les rôles cumulables de l'utilisateur (hors admin, qui
// est un booléen sur users).
func (r *UserRepo) ListRoles(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT role FROM user_roles WHERE user_id = $1 ORDER BY role`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := []string{}
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *UserRepo) HasRole(ctx context.Context, userID, role string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_roles WHERE user_id = $1 AND role = $2)`,
		userID, role,
	).Scan(&exists)
	return exists, err
}

// ListAllUsers retourne tous les utilisateurs avec leurs rôles (pour l'admin).
func (r *UserRepo) ListAllUsers(ctx context.Context) ([]*model.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT u.id, u.email, u.phone, u.password_hash, u.is_admin, u.email_verified_at, u.phone_verified_at,
			u.failed_login_attempts, u.locked_until, u.created_at, u.updated_at,
			COALESCE(ARRAY_AGG(ur.role) FILTER (WHERE ur.role IS NOT NULL), ARRAY[]::text[])
		 FROM users u
		 LEFT JOIN user_roles ur ON ur.user_id = u.id
		 GROUP BY u.id ORDER BY u.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*model.User{}
	for rows.Next() {
		user := &model.User{}
		err := rows.Scan(&user.ID, &user.Email, &user.Phone, &user.PasswordHash, &user.IsAdmin,
			&user.EmailVerifiedAt, &user.PhoneVerifiedAt, &user.FailedLoginAttempts,
			&user.LockedUntil, &user.CreatedAt, &user.UpdatedAt, &user.Roles)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// UserWithStats — utilisateur enrichi de ses statistiques de vente (vue admin).
// Country / CountryLabel sont dérivés de l'indicatif du téléphone par le
// handler (voir payment.CountryFromPhone) — pas stockés en base.
type UserWithStats struct {
	model.User
	ProductsSold        int    `json:"products_sold"`
	RevenueGeneratedCFA int    `json:"revenue_generated_cfa"`
	Country             string `json:"country,omitempty"`       // ISO3, "" si inconnu
	CountryLabel        string `json:"country_label,omitempty"` // nom lisible ou "—"
}

// ListAllUsersWithStats — tous les utilisateurs avec, pour chacun, le nombre
// de ventes abouties de ses produits et le revenu (part vendeur) généré.
func (r *UserRepo) ListAllUsersWithStats(ctx context.Context) ([]*UserWithStats, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.email, u.phone, u.is_admin, u.email_verified_at, u.phone_verified_at,
		       u.locked_until, u.created_at, u.updated_at,
		       COALESCE(ARRAY_AGG(DISTINCT ur.role) FILTER (WHERE ur.role IS NOT NULL), ARRAY[]::text[]),
		       COALESCE(stats.products_sold, 0), COALESCE(stats.revenue_generated, 0)
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		LEFT JOIN (
			SELECT p.vendor_id, COUNT(s.id) AS products_sold,
			       COALESCE(SUM(s.vendor_amount_cfa), 0) AS revenue_generated
			FROM sales s JOIN products p ON p.id = s.product_id
			WHERE s.status IN ('paid', 'delivered')
			GROUP BY p.vendor_id
		) stats ON stats.vendor_id = u.id
		GROUP BY u.id, stats.products_sold, stats.revenue_generated
		ORDER BY u.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*UserWithStats{}
	for rows.Next() {
		u := &UserWithStats{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Phone, &u.IsAdmin, &u.EmailVerifiedAt, &u.PhoneVerifiedAt,
			&u.LockedUntil, &u.CreatedAt, &u.UpdatedAt, &u.Roles, &u.ProductsSold, &u.RevenueGeneratedCFA); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// UnlockAccount — réactive un compte suspendu (retire locked_until).
func (r *UserRepo) UnlockAccount(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET locked_until = NULL, failed_login_attempts = 0 WHERE id = $1`, userID)
	return err
}

// SetAdminStatus promeut ou rétrograde un utilisateur au statut administrateur.
func (r *UserRepo) SetAdminStatus(ctx context.Context, userID string, isAdmin bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET is_admin = $2, updated_at = NOW() WHERE id = $1`,
		userID, isAdmin,
	)
	return err
}

// ListAdminIDs — identifiants de tous les administrateurs, pour les
// notifications qui concernent l'équipe admin dans son ensemble (ex: échec
// d'un versement de don, voir DonationService.notifyAdmins).
func (r *UserRepo) ListAdminIDs(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM users WHERE is_admin = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListAdminEmails — emails de tous les administrateurs (notifications par
// email, ex: nouvelle inscription — voir AuthService.notifyAdminsOfSignup).
func (r *UserRepo) ListAdminEmails(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT email FROM users WHERE is_admin = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emails := []string{}
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	return emails, rows.Err()
}

// ListAllEmails — emails de tous les comptes utilisateurs, pour la diffusion
// admin (voir AdminHandler.SendBroadcast). Ne renvoie que les emails, pour
// limiter la mémoire même avec un grand nombre de comptes.
func (r *UserRepo) ListAllEmails(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT email FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emails := []string{}
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	return emails, rows.Err()
}

// EmailWithPhone — un destinataire pour la diffusion admin ciblée par pays :
// l'email (pour l'envoi) et le téléphone (pour déduire le pays côté handler,
// via payment.CountryFromPhone).
type EmailWithPhone struct {
	Email string
	Phone *string
}

// ListEmailsWithPhone — comme ListAllEmails, mais avec le téléphone de chaque
// compte, pour permettre au handler de filtrer la diffusion par pays.
func (r *UserRepo) ListEmailsWithPhone(ctx context.Context) ([]EmailWithPhone, error) {
	rows, err := r.pool.Query(ctx, `SELECT email, phone FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []EmailWithPhone{}
	for rows.Next() {
		var e EmailWithPhone
		if err := rows.Scan(&e.Email, &e.Phone); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// UserRoleCounts — répartition des utilisateurs par rôle (un compte peut
// cumuler plusieurs rôles, donc la somme peut dépasser le total).
type UserRoleCounts struct {
	Total   int `json:"total"`
	Vendors int `json:"vendors"`
	Closers int `json:"closers"`
	Admins  int `json:"admins"`
}

func (r *UserRepo) CountByRole(ctx context.Context) (UserRoleCounts, error) {
	var c UserRoleCounts
	err := r.pool.QueryRow(ctx, `SELECT
		(SELECT COUNT(*) FROM users),
		(SELECT COUNT(DISTINCT user_id) FROM user_roles WHERE role = 'vendeur'),
		(SELECT COUNT(DISTINCT user_id) FROM user_roles WHERE role = 'closer'),
		(SELECT COUNT(*) FROM users WHERE is_admin)`,
	).Scan(&c.Total, &c.Vendors, &c.Closers, &c.Admins)
	return c, err
}

// CountAllUsers compte les utilisateurs (pour les stats admin).
func (r *UserRepo) CountAllUsers(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}
