package repository

import (
	"context"
	"errors"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEventNotFound      = errors.New("event not found")
	ErrEventOfferNotFound = errors.New("event offer not found")
	// ErrEventAlreadyRegistered — l'email est déjà inscrit à cette offre
	// (index unique sur (event_offer_id, lower(email))).
	ErrEventAlreadyRegistered = errors.New("event: email already registered for this offer")
)

type EventRepo struct {
	pool *pgxpool.Pool
}

func NewEventRepo(pool *pgxpool.Pool) *EventRepo {
	return &EventRepo{pool: pool}
}

const eventColumns = `id, vendor_id, title, slug, description, cover_image_key, event_date, meeting_link,
	moderation_status, moderation_note, created_at, updated_at`

func scanEvent(row pgx.Row) (*model.Event, error) {
	e := &model.Event{}
	err := row.Scan(&e.ID, &e.VendorID, &e.Title, &e.Slug, &e.Description, &e.CoverImageKey, &e.EventDate,
		&e.MeetingLink, &e.ModerationStatus, &e.ModerationNote, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrEventNotFound
		}
		return nil, err
	}
	return e, nil
}

// uniqueEventSlug — même logique que ProductRepo.uniqueSlug (voir
// product_repo.go), dupliquée ici plutôt que partagée entre repos : reste
// autonome si l'un des deux évolue séparément (ex: espace de slugs distinct
// products vs events).
func (r *EventRepo) uniqueSlug(ctx context.Context, base string) (string, error) {
	candidate := base
	for i := 2; ; i++ {
		var exists bool
		if err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM events WHERE slug = $1)`, candidate).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = base + "-" + itoa(i)
	}
}

// Create insère l'événement (moderation_status "pending" par défaut, comme
// products — voir Product.Create). Les offres sont créées séparément via
// CreateOffer, un Product catalogue devant d'abord exister pour chaque offre
// payante (voir EventHandler.Create qui orchestre les deux).
func (r *EventRepo) Create(ctx context.Context, input model.CreateEventInput, vendorID string) (*model.Event, error) {
	slug, err := r.uniqueSlug(ctx, slugify(input.Title))
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO events (vendor_id, title, slug, description, cover_image_key, event_date, meeting_link)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING `+eventColumns,
		vendorID, input.Title, slug, input.Description, input.CoverImageKey, input.EventDate, input.MeetingLink,
	)
	return scanEvent(row)
}

func (r *EventRepo) FindByID(ctx context.Context, idOrSlug string) (*model.Event, error) {
	return scanEvent(r.pool.QueryRow(ctx,
		`SELECT `+eventColumns+` FROM events WHERE id::text = $1 OR slug = $1`, idOrSlug))
}

func (r *EventRepo) ListByVendor(ctx context.Context, vendorID string) ([]*model.Event, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+eventColumns+` FROM events WHERE vendor_id = $1 ORDER BY created_at DESC`, vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

// ListApproved — événements publics (modération approuvée), plus récents
// d'abord. Même filtre que ProductRepo.ListApproved.
func (r *EventRepo) ListApproved(ctx context.Context) ([]*model.Event, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+eventColumns+` FROM events WHERE moderation_status = 'approved' ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func (r *EventRepo) ListPendingModeration(ctx context.Context) ([]*model.Event, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+eventColumns+` FROM events WHERE moderation_status = 'pending' ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func scanEvents(rows pgx.Rows) ([]*model.Event, error) {
	var out []*model.Event
	for rows.Next() {
		e := &model.Event{}
		if err := rows.Scan(&e.ID, &e.VendorID, &e.Title, &e.Slug, &e.Description, &e.CoverImageKey, &e.EventDate,
			&e.MeetingLink, &e.ModerationStatus, &e.ModerationNote, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *EventRepo) Update(ctx context.Context, id, vendorID string, input model.UpdateEventInput) (*model.Event, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE events SET
			title = COALESCE($3, title),
			description = COALESCE($4, description),
			cover_image_key = COALESCE($5, cover_image_key),
			event_date = COALESCE($6, event_date),
			meeting_link = COALESCE($7, meeting_link),
			updated_at = now()
		 WHERE id = $1 AND vendor_id = $2
		 RETURNING `+eventColumns,
		id, vendorID, input.Title, input.Description, input.CoverImageKey, input.EventDate, input.MeetingLink,
	)
	return scanEvent(row)
}

// Delete supprime l'événement (CASCADE sur event_offers/event_registrations,
// voir migration 034). Utilisé en nettoyage quand la création d'une offre
// échoue à mi-chemin (voir EventHandler.Create) — sans vérification de
// propriétaire, car appelé juste après Create par le même flux (le
// vendorID n'est pas encore remis en cause à ce stade).
func (r *EventRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM events WHERE id = $1`, id)
	return err
}

// DeleteOwned — suppression exposée au vendeur (route DELETE), filtrée par
// vendor_id : renvoie ErrEventNotFound si l'événement n'existe pas OU
// n'appartient pas à ce vendeur (jamais de distinction entre les deux côté
// appelant, pour ne pas révéler l'existence d'un événement d'un tiers).
func (r *EventRepo) DeleteOwned(ctx context.Context, id, vendorID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM events WHERE id = $1 AND vendor_id = $2`, id, vendorID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrEventNotFound
	}
	return nil
}

func (r *EventRepo) UpdateModerationStatus(ctx context.Context, id, status string, note *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE events SET moderation_status = $2, moderation_note = $3, updated_at = now() WHERE id = $1`,
		id, status, note,
	)
	return err
}

// --- Offres ---

const eventOfferColumns = `id, event_id, title, is_free, product_id, sort_order, created_at`

func scanEventOffer(row pgx.Row) (*model.EventOffer, error) {
	o := &model.EventOffer{}
	err := row.Scan(&o.ID, &o.EventID, &o.Title, &o.IsFree, &o.ProductID, &o.SortOrder, &o.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrEventOfferNotFound
		}
		return nil, err
	}
	return o, nil
}

// EventAndOfferByProductID retrouve l'événement et l'offre payante qui a
// créé ce Product (voir EventHandler.createOfferProduct) — utilisé pour
// générer le billet PDF d'une vente à partir de sale.ProductID, sans avoir à
// faire porter cette information par Sale elle-même (qui n'a pas besoin de
// connaître les événements pour le reste de la marketplace).
func (r *EventRepo) EventAndOfferByProductID(ctx context.Context, productID string) (*model.Event, *model.EventOffer, error) {
	offer, err := scanEventOffer(r.pool.QueryRow(ctx,
		`SELECT `+eventOfferColumns+` FROM event_offers WHERE product_id = $1`, productID))
	if err != nil {
		return nil, nil, err
	}
	event, err := r.FindByID(ctx, offer.EventID)
	if err != nil {
		return nil, nil, err
	}
	return event, offer, nil
}

func (r *EventRepo) CreateOffer(ctx context.Context, eventID, title string, isFree bool, productID *string, sortOrder int) (*model.EventOffer, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO event_offers (event_id, title, is_free, product_id, sort_order)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+eventOfferColumns,
		eventID, title, isFree, productID, sortOrder,
	)
	return scanEventOffer(row)
}

func (r *EventRepo) FindOfferByID(ctx context.Context, id string) (*model.EventOffer, error) {
	return scanEventOffer(r.pool.QueryRow(ctx, `SELECT `+eventOfferColumns+` FROM event_offers WHERE id = $1`, id))
}

// UpdateOfferTitle — seul le titre de l'offre est modifiable ici ; le prix
// vit sur le Product lié (voir ProductRepo.Update) et is_free/product_id ne
// changent jamais après création (transformer une offre gratuite en payante
// demanderait de créer un Product a posteriori — non géré, le vendeur
// recrée l'offre à la place, voir EventHandler.AddOffer/DeleteOffer).
func (r *EventRepo) UpdateOfferTitle(ctx context.Context, id, title string) (*model.EventOffer, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE event_offers SET title = $2 WHERE id = $1 RETURNING `+eventOfferColumns,
		id, title,
	)
	return scanEventOffer(row)
}

// DeleteOffer supprime une offre (CASCADE sur ses éventuelles inscriptions
// gratuites déjà enregistrées, voir migration 034). N'affecte pas le
// Product lié d'une offre payante — laissé tel quel plutôt que supprimé,
// pour ne jamais faire disparaître un produit déjà potentiellement acheté.
func (r *EventRepo) DeleteOffer(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM event_offers WHERE id = $1`, id)
	return err
}

// CountOffers — nombre d'offres restantes pour un événement, pour empêcher
// de supprimer la dernière (un événement doit garder au moins une offre).
func (r *EventRepo) CountOffers(ctx context.Context, eventID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM event_offers WHERE event_id = $1`, eventID).Scan(&n)
	return n, err
}

// ListOffersWithProduct — offres d'un événement, enrichies du prix/slug du
// Product catalogue pour les offres payantes (affichage public).
func (r *EventRepo) ListOffersWithProduct(ctx context.Context, eventID string) ([]*model.EventOfferWithProduct, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT eo.id, eo.event_id, eo.title, eo.is_free, eo.product_id, eo.sort_order, eo.created_at,
			p.slug, p.price_cfa
		FROM event_offers eo
		LEFT JOIN products p ON p.id = eo.product_id
		WHERE eo.event_id = $1
		ORDER BY eo.sort_order`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.EventOfferWithProduct
	for rows.Next() {
		o := &model.EventOfferWithProduct{}
		if err := rows.Scan(&o.ID, &o.EventID, &o.Title, &o.IsFree, &o.ProductID, &o.SortOrder, &o.CreatedAt,
			&o.ProductSlug, &o.ProductPrice); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// --- Inscriptions (offres gratuites) ---

func (r *EventRepo) CreateRegistration(ctx context.Context, offerID string, in model.CreateEventRegistrationInput) (*model.EventRegistration, error) {
	reg := &model.EventRegistration{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO event_registrations (event_offer_id, full_name, email, phone_number)
		VALUES ($1, $2, $3, $4)
		RETURNING id, event_offer_id, full_name, email, phone_number, created_at`,
		offerID, in.FullName, in.Email, in.Phone,
	).Scan(&reg.ID, &reg.EventOfferID, &reg.FullName, &reg.Email, &reg.PhoneNumber, &reg.CreatedAt)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrEventAlreadyRegistered
		}
		return nil, err
	}
	return reg, nil
}

func (r *EventRepo) ListRegistrationsByEvent(ctx context.Context, eventID string) ([]*model.EventRegistration, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT er.id, er.event_offer_id, er.full_name, er.email, er.phone_number, er.created_at
		FROM event_registrations er
		JOIN event_offers eo ON eo.id = er.event_offer_id
		WHERE eo.event_id = $1
		ORDER BY er.created_at DESC`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.EventRegistration
	for rows.Next() {
		reg := &model.EventRegistration{}
		if err := rows.Scan(&reg.ID, &reg.EventOfferID, &reg.FullName, &reg.Email, &reg.PhoneNumber, &reg.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, reg)
	}
	return out, rows.Err()
}
