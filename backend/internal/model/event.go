package model

import "time"

// Event — événement créé par un vendeur (webinaire, atelier, lancement...),
// avec 1 à 3 offres (EventOffer), chacune payante ou gratuite. Voir
// migration 034_events.sql.
type Event struct {
	ID               string     `json:"id"`
	VendorID         string     `json:"vendor_id"`
	Title            string     `json:"title"`
	Slug             string     `json:"slug"`
	Description      *string    `json:"description,omitempty"`
	CoverImageKey    *string    `json:"cover_image_key,omitempty"`
	EventDate        *time.Time `json:"event_date,omitempty"`
	MeetingLink      *string    `json:"meeting_link,omitempty"`
	// Personnalisation de la page publique (voir migration 045).
	AccentColor   *string `json:"accent_color,omitempty"`
	VenueName     *string `json:"venue_name,omitempty"`
	VenueAddress  *string `json:"venue_address,omitempty"`
	VenueMapURL   *string `json:"venue_map_url,omitempty"`
	// Mini-site événement (voir migration 046) : logo remplaçant celui de
	// DIARRA sur cette page (qui n'affiche déjà plus le header/footer du
	// site), et présentation libre de l'organisateur.
	LogoKey        *string `json:"logo_key,omitempty"`
	AboutOrganizer *string `json:"about_organizer,omitempty"`
	ModerationStatus string     `json:"moderation_status"`
	ModerationNote   *string    `json:"moderation_note,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// EventGalleryImage — une photo de la galerie d'un événement (en plus de sa
// cover_image_key), affichée dans l'ordre sort_order.
type EventGalleryImage struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	FileKey   string    `json:"file_key"`
	SortOrder int16     `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

// EventScheduleItem — un créneau du programme/planning d'un événement.
// TimeLabel est du texte libre ("14h00", "Jour 2 — matin") plutôt qu'une
// heure structurée : reste affichable même si la date exacte de l'événement
// n'est pas encore fixée.
type EventScheduleItem struct {
	ID          string    `json:"id"`
	EventID     string    `json:"event_id"`
	TimeLabel   string    `json:"time_label"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	SortOrder   int16     `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
}

// EventHeroImage — une image du carrousel hero (max 5, distinct de la
// galerie photo affichée plus bas sur la page).
type EventHeroImage struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	FileKey   string    `json:"file_key"`
	SortOrder int16     `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

// MaxEventHeroImages — plafond du carrousel hero (voir EventHandler.AddHeroImage).
const MaxEventHeroImages = 5

type AddEventGalleryImageInput struct {
	FileKey string `json:"file_key"`
}

type AddEventHeroImageInput struct {
	FileKey string `json:"file_key"`
}

type AddEventScheduleItemInput struct {
	TimeLabel   string  `json:"time_label"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
}

type UpdateEventScheduleItemInput struct {
	TimeLabel   *string `json:"time_label,omitempty"`
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

// EventOffer — un palier d'inscription pour un événement (ex: "Standard",
// "VIP"). IsFree==true : inscription directe (EventRegistration), pas de
// paiement. IsFree==false : ProductID pointe vers le Product catalogue qui
// porte le prix et le fichier de confirmation — achat via le checkout normal.
type EventOffer struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	Title     string    `json:"title"`
	IsFree    bool      `json:"is_free"`
	ProductID *string   `json:"product_id,omitempty"`
	SortOrder int16     `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

// EventOfferWithProduct — offre enrichie du produit catalogue associé
// (prix, slug) quand elle est payante, pour l'affichage public.
type EventOfferWithProduct struct {
	EventOffer
	ProductSlug  *string `json:"product_slug,omitempty"`
	ProductPrice *int    `json:"product_price_cfa,omitempty"`
}

// EventRegistration — inscription à une offre GRATUITE d'un événement.
type EventRegistration struct {
	ID           string    `json:"id"`
	EventOfferID string    `json:"event_offer_id"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	PhoneNumber  string    `json:"phone_number"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateEventInput struct {
	Title         string     `json:"title"`
	Description   *string    `json:"description,omitempty"`
	CoverImageKey *string    `json:"cover_image_key,omitempty"`
	EventDate     *time.Time `json:"event_date,omitempty"`
	MeetingLink   *string    `json:"meeting_link,omitempty"`
	AccentColor   *string    `json:"accent_color,omitempty"`
	VenueName     *string    `json:"venue_name,omitempty"`
	VenueAddress  *string    `json:"venue_address,omitempty"`
	VenueMapURL   *string    `json:"venue_map_url,omitempty"`
	LogoKey        *string `json:"logo_key,omitempty"`
	AboutOrganizer *string `json:"about_organizer,omitempty"`
	// Offers : 1 à 3 entrées, validées côté handler (voir EventHandler.Create).
	Offers []CreateEventOfferInput `json:"offers"`
}

type CreateEventOfferInput struct {
	Title    string `json:"title"`
	IsFree   bool   `json:"is_free"`
	PriceCFA int    `json:"price_cfa,omitempty"` // ignoré si IsFree
}

type UpdateEventInput struct {
	Title         *string    `json:"title,omitempty"`
	Description   *string    `json:"description,omitempty"`
	CoverImageKey *string    `json:"cover_image_key,omitempty"`
	EventDate     *time.Time `json:"event_date,omitempty"`
	MeetingLink   *string    `json:"meeting_link,omitempty"`
	AccentColor   *string    `json:"accent_color,omitempty"`
	VenueName     *string    `json:"venue_name,omitempty"`
	VenueAddress  *string    `json:"venue_address,omitempty"`
	VenueMapURL   *string    `json:"venue_map_url,omitempty"`
	LogoKey        *string `json:"logo_key,omitempty"`
	AboutOrganizer *string `json:"about_organizer,omitempty"`
}

type CreateEventRegistrationInput struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}
