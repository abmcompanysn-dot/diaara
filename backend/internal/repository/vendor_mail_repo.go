package repository

import (
	"context"
	"errors"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrVendorMailThreadNotFound = errors.New("vendor mail thread not found")

type VendorMailRepo struct {
	pool *pgxpool.Pool
}

func NewVendorMailRepo(pool *pgxpool.Pool) *VendorMailRepo {
	return &VendorMailRepo{pool: pool}
}

const vendorMailThreadColumns = `id, vendor_id, subject, last_message_id, created_at, updated_at`

func scanVendorMailThread(row pgx.Row) (*model.VendorMailThread, error) {
	t := &model.VendorMailThread{}
	err := row.Scan(&t.ID, &t.VendorID, &t.Subject, &t.LastMessageID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVendorMailThreadNotFound
		}
		return nil, err
	}
	return t, nil
}

const vendorMailMessageColumns = `id, thread_id, direction, status, subject, body, message_id, in_reply_to, approved_by, approved_at, sent_at, created_at`

func scanVendorMailMessage(row pgx.Row) (*model.VendorMailMessage, error) {
	m := &model.VendorMailMessage{}
	err := row.Scan(&m.ID, &m.ThreadID, &m.Direction, &m.Status, &m.Subject, &m.Body,
		&m.MessageID, &m.InReplyTo, &m.ApprovedBy, &m.ApprovedAt, &m.SentAt, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// FindOrCreateThread renvoie le fil existant du vendeur, ou en crée un (une
// seule conversation par vendeur — voir la contrainte UNIQUE(vendor_id)).
func (r *VendorMailRepo) FindOrCreateThread(ctx context.Context, vendorID, subject string) (*model.VendorMailThread, error) {
	thread, err := scanVendorMailThread(r.pool.QueryRow(ctx,
		`SELECT `+vendorMailThreadColumns+` FROM vendor_mail_threads WHERE vendor_id = $1`, vendorID))
	if err == nil {
		return thread, nil
	}
	if !errors.Is(err, ErrVendorMailThreadNotFound) {
		return nil, err
	}
	return scanVendorMailThread(r.pool.QueryRow(ctx,
		`INSERT INTO vendor_mail_threads (vendor_id, subject) VALUES ($1, $2)
		 ON CONFLICT (vendor_id) DO UPDATE SET subject = vendor_mail_threads.subject
		 RETURNING `+vendorMailThreadColumns, vendorID, subject))
}

func (r *VendorMailRepo) FindThreadByVendor(ctx context.Context, vendorID string) (*model.VendorMailThread, error) {
	return scanVendorMailThread(r.pool.QueryRow(ctx,
		`SELECT `+vendorMailThreadColumns+` FROM vendor_mail_threads WHERE vendor_id = $1`, vendorID))
}

func (r *VendorMailRepo) FindThreadByID(ctx context.Context, id string) (*model.VendorMailThread, error) {
	return scanVendorMailThread(r.pool.QueryRow(ctx,
		`SELECT `+vendorMailThreadColumns+` FROM vendor_mail_threads WHERE id = $1`, id))
}

// ListThreads renvoie tous les fils, triés par dernière activité, avec
// l'email et la boutique du vendeur pour l'affichage admin.
func (r *VendorMailRepo) ListThreads(ctx context.Context) ([]*model.VendorMailThread, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT t.id, t.vendor_id, t.subject, t.last_message_id, t.created_at, t.updated_at,
		        u.email, COALESCE(u.shop_name, '')
		 FROM vendor_mail_threads t
		 JOIN users u ON u.id = t.vendor_id
		 ORDER BY t.updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []*model.VendorMailThread
	for rows.Next() {
		t := &model.VendorMailThread{}
		if err := rows.Scan(&t.ID, &t.VendorID, &t.Subject, &t.LastMessageID, &t.CreatedAt, &t.UpdatedAt,
			&t.VendorEmail, &t.VendorShop); err != nil {
			return nil, err
		}
		threads = append(threads, t)
	}
	return threads, rows.Err()
}

func (r *VendorMailRepo) TouchThread(ctx context.Context, id, lastMessageID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE vendor_mail_threads SET last_message_id = $2, updated_at = NOW() WHERE id = $1`,
		id, lastMessageID)
	return err
}

func (r *VendorMailRepo) ListMessages(ctx context.Context, threadID string) ([]*model.VendorMailMessage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+vendorMailMessageColumns+` FROM vendor_mail_messages
		 WHERE thread_id = $1 ORDER BY created_at ASC`, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*model.VendorMailMessage
	for rows.Next() {
		m, err := scanVendorMailMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// ListDraftMessages renvoie tous les brouillons sortants en attente de
// validation admin, toutes conversations confondues — c'est la file
// affichée dans /admin/vendor-mail.
func (r *VendorMailRepo) ListDraftMessages(ctx context.Context) ([]*model.VendorMailMessage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+vendorMailMessageColumns+` FROM vendor_mail_messages
		 WHERE status = 'draft' ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*model.VendorMailMessage
	for rows.Next() {
		m, err := scanVendorMailMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (r *VendorMailRepo) CreateDraft(ctx context.Context, threadID, subject, body string, inReplyTo *string) (*model.VendorMailMessage, error) {
	return scanVendorMailMessage(r.pool.QueryRow(ctx,
		`INSERT INTO vendor_mail_messages (thread_id, direction, status, subject, body, in_reply_to)
		 VALUES ($1, 'outbound', 'draft', $2, $3, $4)
		 RETURNING `+vendorMailMessageColumns,
		threadID, subject, body, inReplyTo))
}

func (r *VendorMailRepo) CreateInbound(ctx context.Context, threadID, subject, body, messageID string, inReplyTo *string) (*model.VendorMailMessage, error) {
	return scanVendorMailMessage(r.pool.QueryRow(ctx,
		`INSERT INTO vendor_mail_messages (thread_id, direction, status, subject, body, message_id, in_reply_to)
		 VALUES ($1, 'inbound', 'received', $2, $3, $4, $5)
		 ON CONFLICT (message_id) WHERE message_id IS NOT NULL DO NOTHING
		 RETURNING `+vendorMailMessageColumns,
		threadID, subject, body, messageID, inReplyTo))
}

func (r *VendorMailRepo) FindMessageByID(ctx context.Context, id string) (*model.VendorMailMessage, error) {
	return scanVendorMailMessage(r.pool.QueryRow(ctx,
		`SELECT `+vendorMailMessageColumns+` FROM vendor_mail_messages WHERE id = $1`, id))
}

// UpdateDraftBody : l'admin corrige le texte d'un brouillon avant validation.
func (r *VendorMailRepo) UpdateDraftBody(ctx context.Context, id, body string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE vendor_mail_messages SET body = $2 WHERE id = $1 AND status = 'draft'`, id, body)
	return err
}

func (r *VendorMailRepo) ApproveDraft(ctx context.Context, id, adminID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE vendor_mail_messages SET status = 'approved', approved_by = $2, approved_at = NOW()
		 WHERE id = $1 AND status = 'draft'`, id, adminID)
	return err
}

func (r *VendorMailRepo) RejectDraft(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE vendor_mail_messages SET status = 'rejected' WHERE id = $1 AND status = 'draft'`, id)
	return err
}

func (r *VendorMailRepo) MarkSent(ctx context.Context, id, messageID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE vendor_mail_messages SET status = 'sent', sent_at = NOW(), message_id = $2
		 WHERE id = $1 AND status = 'approved'`, id, messageID)
	return err
}

// ListApprovedPendingSend : brouillons déjà validés par l'admin mais pas
// encore réellement envoyés (le job d'envoi les traite puis appelle MarkSent).
func (r *VendorMailRepo) ListApprovedPendingSend(ctx context.Context) ([]*model.VendorMailMessage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+vendorMailMessageColumns+` FROM vendor_mail_messages
		 WHERE status = 'approved' ORDER BY approved_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*model.VendorMailMessage
	for rows.Next() {
		m, err := scanVendorMailMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// VendorMailRecipient — un destinataire potentiel pour une diffusion
// (ListByRole), avec son nom d'affichage préféré (shop_name, sinon
// display_name, sinon vide).
type VendorMailRecipient struct {
	UserID string
	Email  string
	Name   string
}

// ListByRole renvoie les comptes distincts ayant ce rôle ("vendeur" ou
// "closer"), avec email non vide — utilisé par la diffusion groupée
// (AdminHandler ou équivalent, voir /admin/vendor-mail "Diffuser à tous").
func (r *VendorMailRepo) ListByRole(ctx context.Context, role string) ([]VendorMailRecipient, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT u.id, u.email, COALESCE(NULLIF(TRIM(u.shop_name), ''), NULLIF(TRIM(u.display_name), ''), '')
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role = $1 AND u.email IS NOT NULL AND u.email != ''`, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []VendorMailRecipient
	for rows.Next() {
		var rec VendorMailRecipient
		if err := rows.Scan(&rec.UserID, &rec.Email, &rec.Name); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *VendorMailRepo) GetLastUID(ctx context.Context, mailbox string) (uint32, error) {
	var uid uint32
	err := r.pool.QueryRow(ctx,
		`INSERT INTO vendor_mail_imap_state (mailbox, last_uid) VALUES ($1, 0)
		 ON CONFLICT (mailbox) DO UPDATE SET mailbox = vendor_mail_imap_state.mailbox
		 RETURNING last_uid`, mailbox).Scan(&uid)
	return uid, err
}

func (r *VendorMailRepo) SetLastUID(ctx context.Context, mailbox string, uid uint32) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE vendor_mail_imap_state SET last_uid = $2, updated_at = NOW() WHERE mailbox = $1`,
		mailbox, uid)
	return err
}
