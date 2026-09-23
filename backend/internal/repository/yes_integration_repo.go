package repository

import (
	"context"
	"errors"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSessionNotFound = errors.New("conversational session not found")

type YesIntegrationRepo struct {
	pool *pgxpool.Pool
}

func NewYesIntegrationRepo(pool *pgxpool.Pool) *YesIntegrationRepo {
	return &YesIntegrationRepo{pool: pool}
}

const sessionColumns = `id, yes_session_id, chat_url, product_id, buyer_id, seller_id, referral_link_id, micro_ticket_sale_id, sale_id, status, created_at, updated_at`

func scanSession(row pgx.Row) (*model.ConversationalSession, error) {
	s := &model.ConversationalSession{}
	err := row.Scan(&s.ID, &s.YesSessionID, &s.ChatURL, &s.ProductID, &s.BuyerID, &s.SellerID,
		&s.ReferralLinkID, &s.MicroTicketSaleID, &s.SaleID, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	return s, nil
}

// CreatePendingSession — posée dès OpenConversation, AVANT confirmation du
// paiement micro-ticket (statut "pending", yes_session_id/chat_url encore
// NULL). Voir OpenSessionAfterPayment pour la suite du cycle de vie.
func (r *YesIntegrationRepo) CreatePendingSession(ctx context.Context, s *model.ConversationalSession) (*model.ConversationalSession, error) {
	return scanSession(r.pool.QueryRow(ctx,
		`INSERT INTO conversational_sessions
		   (product_id, buyer_id, seller_id, referral_link_id, micro_ticket_sale_id, status)
		 VALUES ($1, $2, $3, $4, $5, 'pending')
		 RETURNING `+sessionColumns,
		s.ProductID, s.BuyerID, s.SellerID, s.ReferralLinkID, s.MicroTicketSaleID,
	))
}

func (r *YesIntegrationRepo) FindSessionByID(ctx context.Context, id string) (*model.ConversationalSession, error) {
	return scanSession(r.pool.QueryRow(ctx,
		`SELECT `+sessionColumns+` FROM conversational_sessions WHERE id = $1`, id))
}

// FindSessionByMicroTicketSaleID — retrouve la session "pending" à partir de
// la vente du MICRO-TICKET, utilisé par YesHandler.OnSaleConfirmed pour
// savoir qu'une vente confirmée payée doit déclencher l'appel YES
// session/initiate (voir migrations/038, contrainte UNIQUE sur cette colonne).
func (r *YesIntegrationRepo) FindSessionByMicroTicketSaleID(ctx context.Context, saleID string) (*model.ConversationalSession, error) {
	return scanSession(r.pool.QueryRow(ctx,
		`SELECT `+sessionColumns+` FROM conversational_sessions WHERE micro_ticket_sale_id = $1`, saleID))
}

// FindSessionBySaleID — retrouve la session à partir de la vente du SOLDE
// (sale_id), utilisé par YesHandler.OnSaleConfirmed pour savoir qu'une vente
// confirmée payée doit déclencher l'appel YES delivery/fulfill.
func (r *YesIntegrationRepo) FindSessionBySaleID(ctx context.Context, saleID string) (*model.ConversationalSession, error) {
	return scanSession(r.pool.QueryRow(ctx,
		`SELECT `+sessionColumns+` FROM conversational_sessions WHERE sale_id = $1`, saleID))
}

// OpenSessionAfterPayment — passe "pending" -> "opened" avec les identifiants
// renvoyés par YES session/initiate (appelé seulement APRÈS confirmation
// réelle du paiement micro-ticket, voir YesHandler.OnSaleConfirmed).
func (r *YesIntegrationRepo) OpenSessionAfterPayment(ctx context.Context, id, yesSessionID, chatURL string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversational_sessions SET yes_session_id = $2, chat_url = $3, status = 'opened', updated_at = now() WHERE id = $1`,
		id, yesSessionID, chatURL)
	return err
}

// SetOfferSent — passe la session à "offer_sent" après un appel YES
// send-offer réussi (voir YesHandler.SendOffer).
func (r *YesIntegrationRepo) SetOfferSent(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversational_sessions SET status = 'offer_sent', updated_at = now() WHERE id = $1`, id)
	return err
}

// SetBalanceSale — pose sale_id (la vente du SOLDE, encore "pending") dès sa
// création, pour que FindSessionBySaleID la retrouve ensuite — NE PASSE PAS
// le statut à "completed" (voir CompleteSession), qui n'a lieu qu'à la
// confirmation réelle du paiement.
func (r *YesIntegrationRepo) SetBalanceSale(ctx context.Context, sessionID, saleID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversational_sessions SET sale_id = $2, updated_at = now() WHERE id = $1`,
		sessionID, saleID)
	return err
}

// CompleteSession — passe la session à "completed" : appelé UNIQUEMENT après
// confirmation réelle du paiement du solde ET succès de l'appel YES
// delivery/fulfill (voir YesHandler.OnSaleConfirmed).
func (r *YesIntegrationRepo) CompleteSession(ctx context.Context, sessionID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversational_sessions SET status = 'completed', updated_at = now() WHERE id = $1`,
		sessionID)
	return err
}

// --- Avis vendeur -------------------------------------------------------------

// CreateReview — un seul avis par session (contrainte UNIQUE session_id en
// base) ; une seconde tentative échoue avec une violation unique, à traiter
// côté handler comme "avis déjà soumis" plutôt qu'une erreur serveur.
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
// la volée (pas de colonne dénormalisée à maintenir).
func (r *YesIntegrationRepo) VendorAverageRating(ctx context.Context, vendorID string) (avg float64, count int, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(AVG(rating), 0), COUNT(*) FROM vendor_reviews WHERE vendor_id = $1`,
		vendorID,
	).Scan(&avg, &count)
	return avg, count, err
}
