package repository

import (
	"context"
	"errors"
	"time"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrMetaConnectionNotFound = errors.New("meta connection not found")

// MetaConnectionRepo — compte Facebook connecté par chaque vendeur (table
// vendor_meta_connections, migration 048). Le jeton est stocké DÉJÀ chiffré
// par l'appelant (internal/secretbox) : ce repo ne voit jamais le jeton en
// clair.
type MetaConnectionRepo struct {
	pool *pgxpool.Pool
}

func NewMetaConnectionRepo(pool *pgxpool.Pool) *MetaConnectionRepo {
	return &MetaConnectionRepo{pool: pool}
}

func (r *MetaConnectionRepo) Get(ctx context.Context, vendorID string) (*model.VendorMetaConnection, error) {
	c := &model.VendorMetaConnection{}
	err := r.pool.QueryRow(ctx,
		`SELECT vendor_id, fb_user_id, fb_user_name, access_token, token_expires_at, page_id, page_name,
			ad_account_id, ad_account_name, currency, needs_reconnect, created_at, updated_at
		 FROM vendor_meta_connections WHERE vendor_id = $1`, vendorID).Scan(
		&c.VendorID, &c.FBUserID, &c.FBUserName, &c.AccessTokenEnc, &c.TokenExpiresAt, &c.PageID, &c.PageName,
		&c.AdAccountID, &c.AdAccountName, &c.Currency, &c.NeedsReconnect, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMetaConnectionNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Upsert enregistre (ou renouvelle) la connexion après le retour OAuth.
// Reconnexion avec le MÊME compte Facebook : la page et le compte pub choisis
// sont conservés ; avec un AUTRE compte Facebook, ils sont effacés (ils
// appartenaient à l'ancien compte). needs_reconnect repasse à false.
// Dans ON CONFLICT DO UPDATE, "vendor_meta_connections.x" désigne la ligne
// existante et EXCLUDED.x la nouvelle : l'ordre des affectations est sans
// effet (toutes lisent l'ancienne ligne).
func (r *MetaConnectionRepo) Upsert(ctx context.Context, vendorID, fbUserID, fbUserName, accessTokenEnc string, expiresAt *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO vendor_meta_connections (vendor_id, fb_user_id, fb_user_name, access_token, token_expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (vendor_id) DO UPDATE SET
			page_id = CASE WHEN vendor_meta_connections.fb_user_id = EXCLUDED.fb_user_id THEN vendor_meta_connections.page_id END,
			page_name = CASE WHEN vendor_meta_connections.fb_user_id = EXCLUDED.fb_user_id THEN vendor_meta_connections.page_name END,
			ad_account_id = CASE WHEN vendor_meta_connections.fb_user_id = EXCLUDED.fb_user_id THEN vendor_meta_connections.ad_account_id END,
			ad_account_name = CASE WHEN vendor_meta_connections.fb_user_id = EXCLUDED.fb_user_id THEN vendor_meta_connections.ad_account_name END,
			currency = CASE WHEN vendor_meta_connections.fb_user_id = EXCLUDED.fb_user_id THEN vendor_meta_connections.currency END,
			fb_user_id = EXCLUDED.fb_user_id,
			fb_user_name = EXCLUDED.fb_user_name,
			access_token = EXCLUDED.access_token,
			token_expires_at = EXCLUDED.token_expires_at,
			needs_reconnect = false,
			updated_at = now()`,
		vendorID, fbUserID, fbUserName, accessTokenEnc, expiresAt)
	return err
}

// SetSelection enregistre la page et le compte publicitaire choisis (déjà
// vérifiés auprès de Meta par l'appelant).
func (r *MetaConnectionRepo) SetSelection(ctx context.Context, vendorID, pageID, pageName, adAccountID, adAccountName, currency string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE vendor_meta_connections SET page_id = $2, page_name = $3, ad_account_id = $4,
			ad_account_name = $5, currency = $6, updated_at = now()
		 WHERE vendor_id = $1`,
		vendorID, pageID, pageName, adAccountID, adAccountName, currency)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMetaConnectionNotFound
	}
	return nil
}

// MarkNeedsReconnect — jeton expiré/révoqué. true uniquement au premier
// passage (pour ne notifier le vendeur qu'une fois).
func (r *MetaConnectionRepo) MarkNeedsReconnect(ctx context.Context, vendorID string) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE vendor_meta_connections SET needs_reconnect = true, updated_at = now()
		 WHERE vendor_id = $1 AND NOT needs_reconnect`, vendorID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// Delete — déconnexion : le jeton est effacé de la base.
func (r *MetaConnectionRepo) Delete(ctx context.Context, vendorID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM vendor_meta_connections WHERE vendor_id = $1`, vendorID)
	return err
}
