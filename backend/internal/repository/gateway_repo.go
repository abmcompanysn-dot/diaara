package repository

import (
	"context"
	"errors"
	"time"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrGatewayClientNotFound = errors.New("gateway client not found")
	ErrGatewayTxNotFound     = errors.New("gateway transaction not found")
)

type GatewayRepo struct {
	pool *pgxpool.Pool
}

func NewGatewayRepo(pool *pgxpool.Pool) *GatewayRepo {
	return &GatewayRepo{pool: pool}
}

// --- Clients -----------------------------------------------------------

const gatewayClientColumns = `id, name, api_key_hash, hmac_secret_hash, default_callback_url, is_active, created_at, updated_at`

// CreateClient — hmacSecretHash peut être vide ("") pour un client créé avant
// la migration 030 ou qui n'a pas encore fait tourner sa clé HMAC ; dans ce
// cas RequireGatewayClient refuse toute requête signée pour lui tant qu'un
// secret n'a pas été défini (voir SetClientHMACSecret).
func (r *GatewayRepo) CreateClient(ctx context.Context, name, apiKeyHash, hmacSecretHash string, defaultCallbackURL *string) (*model.GatewayClient, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO gateway_clients (name, api_key_hash, hmac_secret_hash, default_callback_url)
		 VALUES ($1, $2, $3, $4)
		 RETURNING `+gatewayClientColumns,
		name, apiKeyHash, hmacSecretHash, defaultCallbackURL)
	return scanGatewayClient(row)
}

// FindClientByKeyHash — résout la clé API envoyée par un client externe
// (header X-Gateway-Key, déjà hashée par l'appelant) vers son identité.
// Ignore les clients désactivés (is_active = false).
func (r *GatewayRepo) FindClientByKeyHash(ctx context.Context, keyHash string) (*model.GatewayClient, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+gatewayClientColumns+`
		 FROM gateway_clients WHERE api_key_hash = $1 AND is_active = TRUE`, keyHash)
	return scanGatewayClient(row)
}

// FindClientByID — utilisé par le relais de webhook (gateway_relay.go) pour
// retrouver le secret HMAC du client au moment de signer un callback sortant.
// Contrairement à FindClientByKeyHash, ne filtre pas sur is_active : un
// client désactivé APRÈS la création d'une transaction doit quand même
// pouvoir recevoir le callback qui la conclut (statut final d'un paiement en
// cours), la désactivation bloque seulement de NOUVELLES requêtes entrantes.
func (r *GatewayRepo) FindClientByID(ctx context.Context, id string) (*model.GatewayClient, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+gatewayClientColumns+` FROM gateway_clients WHERE id = $1`, id)
	return scanGatewayClient(row)
}

func (r *GatewayRepo) ListClients(ctx context.Context) ([]*model.GatewayClient, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+gatewayClientColumns+` FROM gateway_clients ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.GatewayClient{}
	for rows.Next() {
		c, err := scanGatewayClientRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *GatewayRepo) SetClientActive(ctx context.Context, id string, active bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE gateway_clients SET is_active = $2, updated_at = now() WHERE id = $1`, id, active)
	return err
}

func scanGatewayClient(row pgx.Row) (*model.GatewayClient, error) {
	c := &model.GatewayClient{}
	err := row.Scan(&c.ID, &c.Name, &c.APIKeyHash, &c.HMACSecretHash, &c.DefaultCallbackURL, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrGatewayClientNotFound
		}
		return nil, err
	}
	return c, nil
}

func scanGatewayClientRow(rows pgx.Rows) (*model.GatewayClient, error) {
	c := &model.GatewayClient{}
	err := rows.Scan(&c.ID, &c.Name, &c.APIKeyHash, &c.HMACSecretHash, &c.DefaultCallbackURL, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

// --- Transactions --------------------------------------------------------

const gatewayTxColumns = `id, client_id, client_ref, type, provider, provider_ref, related_deposit_ref,
	status, failure_reason, amount_cfa, currency, recipient_phone, recipient_operator, country,
	description, callback_url, created_at, updated_at`

func scanGatewayTx(row pgx.Row) (*model.GatewayTransaction, error) {
	t := &model.GatewayTransaction{}
	err := row.Scan(&t.ID, &t.ClientID, &t.ClientRef, &t.Type, &t.Provider, &t.ProviderRef, &t.RelatedDepositRef,
		&t.Status, &t.FailureReason, &t.AmountCFA, &t.Currency, &t.RecipientPhone, &t.RecipientOperator, &t.Country,
		&t.Description, &t.CallbackURL, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrGatewayTxNotFound
		}
		return nil, err
	}
	return t, nil
}

type CreateGatewayTxParams struct {
	ClientID          string
	ClientRef         string
	Type              string
	Provider          string
	AmountCFA         int
	Currency          string
	RecipientPhone    *string
	RecipientOperator *string
	Country           *string
	Description       *string
	CallbackURL       *string
	RelatedDepositRef *string
}

func (r *GatewayRepo) CreateTransaction(ctx context.Context, p CreateGatewayTxParams) (*model.GatewayTransaction, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO gateway_transactions
			(client_id, client_ref, type, provider, amount_cfa, currency, recipient_phone,
			 recipient_operator, country, description, callback_url, related_deposit_ref)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING `+gatewayTxColumns,
		p.ClientID, p.ClientRef, p.Type, p.Provider, p.AmountCFA, p.Currency, p.RecipientPhone,
		p.RecipientOperator, p.Country, p.Description, p.CallbackURL, p.RelatedDepositRef)
	return scanGatewayTx(row)
}

// FindByClientRef — un client externe interroge toujours par SON identifiant
// (client_ref), jamais par l'UUID interne DIARRA qu'il ne connaît qu'après
// coup (renvoyé à la création, mais on ne l'oblige pas à le stocker).
func (r *GatewayRepo) FindByClientRef(ctx context.Context, clientID, clientRef, txType string) (*model.GatewayTransaction, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+gatewayTxColumns+` FROM gateway_transactions
		 WHERE client_id = $1 AND client_ref = $2 AND type = $3`, clientID, clientRef, txType)
	return scanGatewayTx(row)
}

// FindByProviderRef — utilisé par le relais de webhook : retrouve la
// transaction gateway correspondant à un depositId/payoutId/refundId reçu
// dans un callback agrégateur, pour la distinguer d'une sale/payout interne
// DIARRA (voir webhook_handler.go).
func (r *GatewayRepo) FindByProviderRef(ctx context.Context, provider, providerRef string) (*model.GatewayTransaction, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+gatewayTxColumns+` FROM gateway_transactions
		 WHERE provider = $1 AND provider_ref = $2`, provider, providerRef)
	return scanGatewayTx(row)
}

func (r *GatewayRepo) SetProviderRef(ctx context.Context, id, providerRef string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE gateway_transactions SET provider_ref = $2, updated_at = now() WHERE id = $1`, id, providerRef)
	return err
}

func (r *GatewayRepo) UpdateStatus(ctx context.Context, id, status string, failureReason *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE gateway_transactions SET status = $2, failure_reason = $3, updated_at = now() WHERE id = $1`,
		id, status, failureReason)
	return err
}

// ListPendingByProvider — transactions gateway encore "pending" chez un
// provider donné, plus récentes que maxAge, ayant un provider_ref (donc
// réellement soumises à l'agrégateur). Sert à la réconciliation quand un
// webhook agrégateur n'est jamais arrivé (voir RunDepositReconcileLoop).
func (r *GatewayRepo) ListPendingByProvider(ctx context.Context, provider string, maxAge time.Duration) ([]*model.GatewayTransaction, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+gatewayTxColumns+` FROM gateway_transactions
		 WHERE provider = $1 AND status = 'pending' AND provider_ref IS NOT NULL
		   AND created_at > now() - $2::interval
		 ORDER BY created_at`,
		provider, maxAge.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.GatewayTransaction{}
	for rows.Next() {
		t := &model.GatewayTransaction{}
		if err := rows.Scan(&t.ID, &t.ClientID, &t.ClientRef, &t.Type, &t.Provider, &t.ProviderRef, &t.RelatedDepositRef,
			&t.Status, &t.FailureReason, &t.AmountCFA, &t.Currency, &t.RecipientPhone, &t.RecipientOperator, &t.Country,
			&t.Description, &t.CallbackURL, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// FindByID — lecture d'une transaction gateway par son id (bouton admin
// "vérifier chez le prestataire").
func (r *GatewayRepo) FindByID(ctx context.Context, id string) (*model.GatewayTransaction, error) {
	return scanGatewayTx(r.pool.QueryRow(ctx,
		`SELECT `+gatewayTxColumns+` FROM gateway_transactions WHERE id = $1`, id))
}
