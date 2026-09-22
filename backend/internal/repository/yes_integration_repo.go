package repository

import (
	"context"
	"errors"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrYesClientNotFound = errors.New("yes api client not found")
	ErrSessionNotFound   = errors.New("conversational session not found")
)

type YesIntegrationRepo struct {
	pool *pgxpool.Pool
}

func NewYesIntegrationRepo(pool *pgxpool.Pool) *YesIntegrationRepo {
	return &YesIntegrationRepo{pool: pool}
}

// --- Clients API (auth HMAC) -------------------------------------------------

// CreateClient — apiKey en clair (identifiant public, pas secret en soi —
// voir middleware.RequireYesClient) ; apiSecretHash est le HASH du secret
// HMAC, jamais le secret brut (généré une seule fois côté handler admin,
// voir AdminHandler.CreateYesClient).
func (r *YesIntegrationRepo) CreateClient(ctx context.Context, name, apiKey, apiSecretHash string) (*model.YesAPIClient, error) {
	c := &model.YesAPIClient{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO yes_api_clients (name, api_key, api_secret_hash) VALUES ($1, $2, $3)
		 RETURNING id, name, api_key, api_secret_hash, is_active, created_at`,
		name, apiKey, apiSecretHash,
	).Scan(&c.ID, &c.Name, &c.APIKey, &c.APISecretHash, &c.IsActive, &c.CreatedAt)
	return c, err
}

// FindClientByAPIKey — l'API key est en clair (identifiant, pas secret en
// soi) ; le secret HMAC lui reste hashé, comparé côté middleware après
// recalcul de la signature. Ne retourne que les clients actifs.
func (r *YesIntegrationRepo) FindClientByAPIKey(ctx context.Context, apiKey string) (*model.YesAPIClient, error) {
	c := &model.YesAPIClient{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, api_key, api_secret_hash, is_active, created_at
		 FROM yes_api_clients WHERE api_key = $1 AND is_active = TRUE`,
		apiKey,
	).Scan(&c.ID, &c.Name, &c.APIKey, &c.APISecretHash, &c.IsActive, &c.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrYesClientNotFound
		}
		return nil, err
	}
	return c, nil
}

// --- Sessions conversationnelles --------------------------------------------

const sessionColumns = `id, yes_conversation_id, product_id, buyer_id, seller_id, referral_link_id, micro_ticket_sale_id, sale_id, status, created_at, updated_at`

func scanSession(row pgx.Row) (*model.ConversationalSession, error) {
	s := &model.ConversationalSession{}
	err := row.Scan(&s.ID, &s.YesConversationID, &s.ProductID, &s.BuyerID, &s.SellerID,
		&s.ReferralLinkID, &s.MicroTicketSaleID, &s.SaleID, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	return s, nil
}

// CreateSession — posée juste après confirmation du paiement du micro-ticket
// (voir YesHandler.InitiateSession) : micro_ticket_sale_id est déjà connu à
// la création, sale_id (le solde) reste NULL jusqu'au checkout in-chat.
func (r *YesIntegrationRepo) CreateSession(ctx context.Context, s *model.ConversationalSession) (*model.ConversationalSession, error) {
	return scanSession(r.pool.QueryRow(ctx,
		`INSERT INTO conversational_sessions
		   (yes_conversation_id, product_id, buyer_id, seller_id, referral_link_id, micro_ticket_sale_id, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING `+sessionColumns,
		s.YesConversationID, s.ProductID, s.BuyerID, s.SellerID, s.ReferralLinkID, s.MicroTicketSaleID, s.Status,
	))
}

func (r *YesIntegrationRepo) FindSessionByID(ctx context.Context, id string) (*model.ConversationalSession, error) {
	return scanSession(r.pool.QueryRow(ctx,
		`SELECT `+sessionColumns+` FROM conversational_sessions WHERE id = $1`, id))
}

// FindSessionBySaleID — retrouve la session à partir de la vente du SOLDE
// (sale_id), utilisé par YesHandler.NotifyDelivery pour savoir si une vente
// confirmée payée provient du flux conversationnel (et notifier YES si
// c'est le cas). Ne matche jamais sur micro_ticket_sale_id : le micro-ticket
// ne déclenche jamais de livraison.
func (r *YesIntegrationRepo) FindSessionBySaleID(ctx context.Context, saleID string) (*model.ConversationalSession, error) {
	return scanSession(r.pool.QueryRow(ctx,
		`SELECT `+sessionColumns+` FROM conversational_sessions WHERE sale_id = $1`, saleID))
}

// CompleteSession — passe la session à "completed" avec la vente du solde
// (in-chat checkout réussi). Idempotent au niveau appelant, comme
// ConfirmPaidSale pour les ventes classiques : ne rien refaire si déjà
// completed.
func (r *YesIntegrationRepo) CompleteSession(ctx context.Context, sessionID, saleID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversational_sessions SET sale_id = $2, status = 'completed', updated_at = now() WHERE id = $1`,
		sessionID, saleID)
	return err
}

// --- Avis vendeur -------------------------------------------------------------

// CreateReview — un seul avis par session (contrainte UNIQUE session_id en
// base, voir migrations/037_yes_integration.sql) ; une seconde tentative
// pour la même session échoue avec une violation unique, à traiter côté
// handler comme "avis déjà soumis" plutôt qu'une erreur serveur.
func (r *YesIntegrationRepo) CreateReview(ctx context.Context, rev *model.VendorReview) (*model.VendorReview, error) {
	out := &model.VendorReview{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO vendor_reviews (session_id, vendor_id, buyer_id, rating, comment)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, session_id, vendor_id, buyer_id, rating, comment, created_at`,
		rev.SessionID, rev.VendorID, rev.BuyerID, rev.Rating, rev.Comment,
	).Scan(&out.ID, &out.SessionID, &out.VendorID, &out.BuyerID, &out.Rating, &out.Comment, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// VendorAverageRating — note moyenne + nombre d'avis d'un vendeur, calculée à
// la volée (pas de colonne dénormalisée à maintenir — le volume d'avis reste
// largement dans les capacités d'un COUNT/AVG direct).
func (r *YesIntegrationRepo) VendorAverageRating(ctx context.Context, vendorID string) (avg float64, count int, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(AVG(rating), 0), COUNT(*) FROM vendor_reviews WHERE vendor_id = $1`,
		vendorID,
	).Scan(&avg, &count)
	return avg, count, err
}
