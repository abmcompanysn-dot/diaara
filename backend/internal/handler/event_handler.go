package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/diarra/backend/internal/email"
	"github.com/diarra/backend/internal/eventfile"
	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/storage"
	"github.com/go-chi/chi/v5"
)

// maxEventOffers — un événement a entre 1 et 3 offres (voir la décision
// prise avec l'utilisateur : "1 à 3 offres, comme le Summit").
const maxEventOffers = 3

// maxEventFieldLen borne la taille des champs du formulaire d'inscription
// publique gratuite (endpoint non authentifié).
const maxEventFieldLen = 200

type EventHandler struct {
	eventRepo     *repository.EventRepo
	productRepo   *repository.ProductRepo
	storage       StorageService
	notifications *email.NotificationService
	frontendURL   string
}

func NewEventHandler(eventRepo *repository.EventRepo, productRepo *repository.ProductRepo, storage StorageService, notifications *email.NotificationService, frontendURL string) *EventHandler {
	return &EventHandler{
		eventRepo:     eventRepo,
		productRepo:   productRepo,
		storage:       storage,
		notifications: notifications,
		frontendURL:   frontendURL,
	}
}

// Create — POST /api/vendor/events (vendeur authentifié). Crée l'événement
// puis ses 1 à 3 offres : une offre payante crée un Product catalogue
// (catégorie "event", avec un fichier de confirmation généré — voir
// eventfile.BuildHTML) ; une offre gratuite n'a besoin de rien de plus,
// l'inscription passera par RegisterFree.
func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var input model.CreateEventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		http.Error(w, `{"error":"title_required"}`, http.StatusBadRequest)
		return
	}
	if len(input.Offers) == 0 || len(input.Offers) > maxEventOffers {
		http.Error(w, `{"error":"offers_count_invalid"}`, http.StatusBadRequest)
		return
	}
	needsFreeLink := false
	for i, offer := range input.Offers {
		input.Offers[i].Title = strings.TrimSpace(offer.Title)
		if input.Offers[i].Title == "" {
			http.Error(w, `{"error":"offer_title_required"}`, http.StatusBadRequest)
			return
		}
		if offer.IsFree {
			needsFreeLink = true
		} else if offer.PriceCFA <= 0 {
			http.Error(w, `{"error":"offer_price_required"}`, http.StatusBadRequest)
			return
		}
	}
	if needsFreeLink && (input.MeetingLink == nil || strings.TrimSpace(*input.MeetingLink) == "") {
		http.Error(w, `{"error":"meeting_link_required_for_free_offer"}`, http.StatusBadRequest)
		return
	}
	if input.CoverImageKey != nil {
		if err := validateCoverImageKey(r.Context(), h.storage, *input.CoverImageKey); err != nil {
			http.Error(w, `{"error":"invalid_cover_image"}`, http.StatusBadRequest)
			return
		}
	}
	if input.AccentColor != nil && !validHexColor(*input.AccentColor) {
		http.Error(w, `{"error":"invalid_accent_color"}`, http.StatusBadRequest)
		return
	}

	event, err := h.eventRepo.Create(r.Context(), input, userID)
	if err != nil {
		http.Error(w, `{"error":"creation_failed"}`, http.StatusInternalServerError)
		return
	}

	// Pas de transaction unique couvrant event + offres + Products (créés via
	// ProductRepo, connexion séparée) : si une offre échoue à mi-chemin, on
	// supprime l'événement plutôt que de laisser un événement à moitié créé
	// (1 offre sur 3, par ex.) que le vendeur ne peut ni réparer ni retenter
	// proprement — voir EventRepo.Delete (CASCADE sur ses offres déjà créées).
	for i, offerInput := range input.Offers {
		if offerInput.IsFree {
			if _, err := h.eventRepo.CreateOffer(r.Context(), event.ID, offerInput.Title, true, nil, i); err != nil {
				h.eventRepo.Delete(r.Context(), event.ID)
				http.Error(w, `{"error":"offer_creation_failed"}`, http.StatusInternalServerError)
				return
			}
			continue
		}

		productID, err := h.createOfferProduct(r.Context(), event, offerInput, userID)
		if err != nil {
			h.eventRepo.Delete(r.Context(), event.ID)
			http.Error(w, `{"error":"offer_product_failed"}`, http.StatusInternalServerError)
			return
		}
		if _, err := h.eventRepo.CreateOffer(r.Context(), event.ID, offerInput.Title, false, &productID, i); err != nil {
			h.eventRepo.Delete(r.Context(), event.ID)
			http.Error(w, `{"error":"offer_creation_failed"}`, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

// createOfferProduct crée le Product catalogue derrière une offre payante :
// même mécanique que cmd/seed_summit_tickets (fichier de confirmation
// généré, catégorie "event"), mais déclenchée à la volée pour n'importe quel
// vendeur plutôt qu'exécutée une fois en script pour le Summit seul.
func (h *EventHandler) createOfferProduct(ctx context.Context, event *model.Event, offer model.CreateEventOfferInput, vendorID string) (string, error) {
	if h.storage == nil {
		return "", fmt.Errorf("stockage objet non configuré")
	}

	title := fmt.Sprintf("%s — %s", event.Title, offer.Title)
	confirmationHTML := eventfile.BuildHTML(eventfile.Confirmation{
		Title:      title,
		PriceCFA:   offer.PriceCFA,
		Inclusions: []string{fmt.Sprintf("Accès à l'événement « %s »", event.Title)},
		Note:       "Le vendeur vous recontactera avec les détails pratiques (lien de connexion, programme) avant l'événement.",
	})
	fileKey := storage.NewFileKey(vendorID, fmt.Sprintf("evenement-%s.html", offer.Title))
	if err := h.storage.Upload(ctx, fileKey, []byte(confirmationHTML)); err != nil {
		return "", err
	}

	desc := fmt.Sprintf("Offre « %s » pour l'événement « %s ».", offer.Title, event.Title)
	product, err := h.productRepo.Create(ctx, model.CreateProductInput{
		Title:       title,
		Description: &desc,
		PriceCFA:    offer.PriceCFA,
		PriceMode:   "fixed",
		Category:    "event",
		FileKey:     fileKey,
	}, vendorID)
	if err != nil {
		return "", err
	}
	return product.ID, nil
}

// Update — PUT /api/vendor/events/{id} (propriétaire uniquement).
func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")

	var input model.UpdateEventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.CoverImageKey != nil {
		if err := validateCoverImageKey(r.Context(), h.storage, *input.CoverImageKey); err != nil {
			http.Error(w, `{"error":"invalid_cover_image"}`, http.StatusBadRequest)
			return
		}
	}
	if input.AccentColor != nil && !validHexColor(*input.AccentColor) {
		http.Error(w, `{"error":"invalid_accent_color"}`, http.StatusBadRequest)
		return
	}

	event, err := h.eventRepo.Update(r.Context(), id, userID, input)
	if err != nil {
		if err == repository.ErrEventNotFound {
			http.Error(w, `{"error":"not_found_or_forbidden"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// Delete — DELETE /api/vendor/events/{id} (propriétaire uniquement).
func (h *EventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.eventRepo.DeleteOwned(r.Context(), id, userID); err != nil {
		if err == repository.ErrEventNotFound {
			http.Error(w, `{"error":"not_found_or_forbidden"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"delete_failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ownedOffer vérifie que l'offre offerID appartient bien à un événement du
// vendeur userID, et renvoie l'événement + l'offre si c'est le cas. Utilisé
// par UpdateOffer/DeleteOffer pour ne jamais laisser un vendeur toucher à
// l'offre d'un autre (l'ID d'offre seul ne porte aucune info de
// propriétaire, il faut remonter à l'événement).
func (h *EventHandler) ownedOffer(ctx context.Context, offerID, userID string) (*model.Event, *model.EventOffer, error) {
	offer, err := h.eventRepo.FindOfferByID(ctx, offerID)
	if err != nil {
		return nil, nil, repository.ErrEventOfferNotFound
	}
	event, err := h.eventRepo.FindByID(ctx, offer.EventID)
	if err != nil || event.VendorID != userID {
		return nil, nil, repository.ErrEventOfferNotFound
	}
	return event, offer, nil
}

// UpdateOffer — PUT /api/vendor/events/offers/{offerId} (propriétaire de
// l'événement uniquement). Le titre se modifie directement ; le prix d'une
// offre payante passe par le Product lié (voir eventOfferOwnedProduct ci-
// dessous) — is_free et product_id restent figés après création (voir
// EventRepo.UpdateOfferTitle).
func (h *EventHandler) UpdateOffer(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	offerID := chi.URLParam(r, "offerId")
	_, offer, err := h.ownedOffer(r.Context(), offerID, userID)
	if err != nil {
		http.Error(w, `{"error":"not_found_or_forbidden"}`, http.StatusNotFound)
		return
	}

	var input struct {
		Title    *string `json:"title,omitempty"`
		PriceCFA *int    `json:"price_cfa,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if input.Title != nil && strings.TrimSpace(*input.Title) != "" {
		if _, err := h.eventRepo.UpdateOfferTitle(r.Context(), offerID, strings.TrimSpace(*input.Title)); err != nil {
			http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
			return
		}
	}
	if input.PriceCFA != nil {
		if offer.IsFree || offer.ProductID == nil {
			http.Error(w, `{"error":"offer_is_free"}`, http.StatusBadRequest)
			return
		}
		if *input.PriceCFA <= 0 {
			http.Error(w, `{"error":"invalid_price"}`, http.StatusBadRequest)
			return
		}
		price := *input.PriceCFA
		if _, err := h.productRepo.Update(r.Context(), *offer.ProductID, model.UpdateProductInput{PriceCFA: &price}); err != nil {
			http.Error(w, `{"error":"price_update_failed"}`, http.StatusInternalServerError)
			return
		}
	}

	updated, err := h.eventRepo.FindOfferByID(r.Context(), offerID)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// AddOffer — POST /api/vendor/events/{id}/offers (propriétaire uniquement).
// Ajoute une offre à un événement existant, jusqu'à maxEventOffers au total
// — même validation que Create pour une offre payante (Product généré).
func (h *EventHandler) AddOffer(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	eventID := chi.URLParam(r, "id")
	event, err := h.eventRepo.FindByID(r.Context(), eventID)
	if err != nil || event.VendorID != userID {
		http.Error(w, `{"error":"not_found_or_forbidden"}`, http.StatusNotFound)
		return
	}

	count, err := h.eventRepo.CountOffers(r.Context(), event.ID)
	if err != nil {
		http.Error(w, `{"error":"count_failed"}`, http.StatusInternalServerError)
		return
	}
	if count >= maxEventOffers {
		http.Error(w, `{"error":"max_offers_reached"}`, http.StatusBadRequest)
		return
	}

	var input model.CreateEventOfferInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		http.Error(w, `{"error":"offer_title_required"}`, http.StatusBadRequest)
		return
	}
	if input.IsFree {
		if event.MeetingLink == nil || strings.TrimSpace(*event.MeetingLink) == "" {
			http.Error(w, `{"error":"meeting_link_required_for_free_offer"}`, http.StatusBadRequest)
			return
		}
		offer, err := h.eventRepo.CreateOffer(r.Context(), event.ID, input.Title, true, nil, count)
		if err != nil {
			http.Error(w, `{"error":"offer_creation_failed"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(offer)
		return
	}

	if input.PriceCFA <= 0 {
		http.Error(w, `{"error":"offer_price_required"}`, http.StatusBadRequest)
		return
	}
	productID, err := h.createOfferProduct(r.Context(), event, input, userID)
	if err != nil {
		http.Error(w, `{"error":"offer_product_failed"}`, http.StatusInternalServerError)
		return
	}
	offer, err := h.eventRepo.CreateOffer(r.Context(), event.ID, input.Title, false, &productID, count)
	if err != nil {
		http.Error(w, `{"error":"offer_creation_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(offer)
}

// DeleteOffer — DELETE /api/vendor/events/offers/{offerId} (propriétaire
// uniquement). Refuse de supprimer la dernière offre restante : un
// événement publié doit toujours avoir au moins une façon de s'inscrire.
func (h *EventHandler) DeleteOffer(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	offerID := chi.URLParam(r, "offerId")
	event, _, err := h.ownedOffer(r.Context(), offerID, userID)
	if err != nil {
		http.Error(w, `{"error":"not_found_or_forbidden"}`, http.StatusNotFound)
		return
	}
	count, err := h.eventRepo.CountOffers(r.Context(), event.ID)
	if err != nil {
		http.Error(w, `{"error":"count_failed"}`, http.StatusInternalServerError)
		return
	}
	if count <= 1 {
		http.Error(w, `{"error":"cannot_delete_last_offer"}`, http.StatusBadRequest)
		return
	}
	if err := h.eventRepo.DeleteOffer(r.Context(), offerID); err != nil {
		http.Error(w, `{"error":"delete_failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListVendor — GET /api/vendor/events (événements du vendeur connecté).
func (h *EventHandler) ListVendor(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	events, err := h.eventRepo.ListByVendor(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"events": events})
}

// Get — GET /api/events/{id} (public, id ou slug). N'expose que les
// événements approuvés, sauf pour le vendeur propriétaire (aperçu avant/
// pendant modération) — même logique que ProductHandler.Get.
func (h *EventHandler) Get(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "id")
	event, err := h.eventRepo.FindByID(r.Context(), idOrSlug)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	userID := middleware.GetUserID(r.Context())
	if event.ModerationStatus != "approved" && event.VendorID != userID {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	offers, err := h.eventRepo.ListOffersWithProduct(r.Context(), event.ID)
	if err != nil {
		http.Error(w, `{"error":"offers_failed"}`, http.StatusInternalServerError)
		return
	}
	gallery, err := h.eventRepo.ListGalleryImages(r.Context(), event.ID)
	if err != nil {
		http.Error(w, `{"error":"gallery_failed"}`, http.StatusInternalServerError)
		return
	}
	schedule, err := h.eventRepo.ListScheduleItems(r.Context(), event.ID)
	if err != nil {
		http.Error(w, `{"error":"schedule_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"event": event, "offers": offers, "gallery": gallery, "schedule": schedule,
	})
}

// Cover — GET /api/events/{id}/cover (public). Sert l'image de couverture de
// l'événement en la téléchargeant depuis le stockage objet — même principe
// que ProductHandler.Cover (le file_key n'est jamais une URL publique
// directe). Ne vérifie pas le statut de modération : une image seule ne
// révèle rien de sensible, et la fiche non approuvée reste bloquée par Get.
func (h *EventHandler) Cover(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "id")
	event, err := h.eventRepo.FindByID(r.Context(), idOrSlug)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if event.CoverImageKey == nil || *event.CoverImageKey == "" {
		http.Error(w, `{"error":"no_cover"}`, http.StatusNotFound)
		return
	}
	if h.storage == nil {
		http.Error(w, `{"error":"storage_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	data, err := h.storage.Download(r.Context(), *event.CoverImageKey)
	if err != nil {
		http.Error(w, `{"error":"cover_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", imageContentType(*event.CoverImageKey, data))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(data)
}

// validHexColor accepte "" (effacer/pas de couleur) ou "#RRGGBB". Rejette
// les noms de couleur CSS ou rgb() : un format unique simplifie l'injection
// telle quelle dans un style inline côté frontend, sans risque XSS (six
// chiffres hexadécimaux, rien d'autre n'est syntaxiquement possible).
func validHexColor(s string) bool {
	if s == "" {
		return true
	}
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, c := range s[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// --- Galerie photo ---

// AddGalleryImage — POST /api/vendor/events/{id}/gallery (propriétaire).
// L'image doit déjà être uploadée (voir ProductHandler.Upload, même endpoint
// générique) : le body ne porte que la clé de stockage résultante.
func (h *EventHandler) AddGalleryImage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	eventID := chi.URLParam(r, "id")

	var input model.AddEventGalleryImageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.FileKey == "" {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if err := validateCoverImageKey(r.Context(), h.storage, input.FileKey); err != nil {
		http.Error(w, `{"error":"invalid_image"}`, http.StatusBadRequest)
		return
	}

	// Vérifie la propriété de l'événement avant d'ajouter (FindByID ne filtre
	// pas par vendor_id, contrairement aux méthodes *Owned).
	event, err := h.eventRepo.FindByID(r.Context(), eventID)
	if err != nil || event.VendorID != userID {
		http.Error(w, `{"error":"not_found_or_forbidden"}`, http.StatusNotFound)
		return
	}

	img, err := h.eventRepo.AddGalleryImage(r.Context(), eventID, input.FileKey)
	if err != nil {
		http.Error(w, `{"error":"add_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"image": img})
}

// DeleteGalleryImage — DELETE /api/vendor/events/gallery/{imageId} (propriétaire).
func (h *EventHandler) DeleteGalleryImage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	imageID := chi.URLParam(r, "imageId")
	if err := h.eventRepo.DeleteGalleryImageOwned(r.Context(), imageID, userID); err != nil {
		if err == repository.ErrEventNotFound {
			http.Error(w, `{"error":"not_found_or_forbidden"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"delete_failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GalleryImage — GET /api/events/gallery/{imageId}/file (public). Même
// principe que EventHandler.Cover : le fichier n'est jamais servi via une URL
// de stockage publique directe.
func (h *EventHandler) GalleryImage(w http.ResponseWriter, r *http.Request) {
	imageID := chi.URLParam(r, "imageId")
	img, err := h.eventRepo.FindGalleryImageByID(r.Context(), imageID)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if h.storage == nil {
		http.Error(w, `{"error":"storage_not_configured"}`, http.StatusServiceUnavailable)
		return
	}
	data, err := h.storage.Download(r.Context(), img.FileKey)
	if err != nil {
		http.Error(w, `{"error":"image_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", imageContentType(img.FileKey, data))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(data)
}

// --- Programme / planning ---

// AddScheduleItem — POST /api/vendor/events/{id}/schedule (propriétaire).
func (h *EventHandler) AddScheduleItem(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	eventID := chi.URLParam(r, "id")

	var input model.AddEventScheduleItemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	input.TimeLabel = strings.TrimSpace(input.TimeLabel)
	input.Title = strings.TrimSpace(input.Title)
	if input.TimeLabel == "" || input.Title == "" {
		http.Error(w, `{"error":"time_label_and_title_required"}`, http.StatusBadRequest)
		return
	}

	event, err := h.eventRepo.FindByID(r.Context(), eventID)
	if err != nil || event.VendorID != userID {
		http.Error(w, `{"error":"not_found_or_forbidden"}`, http.StatusNotFound)
		return
	}

	item, err := h.eventRepo.AddScheduleItem(r.Context(), eventID, input)
	if err != nil {
		http.Error(w, `{"error":"add_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"item": item})
}

// UpdateScheduleItem — PUT /api/vendor/events/schedule/{itemId} (propriétaire).
func (h *EventHandler) UpdateScheduleItem(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	itemID := chi.URLParam(r, "itemId")

	if _, err := h.eventRepo.FindScheduleItemOwned(r.Context(), itemID, userID); err != nil {
		http.Error(w, `{"error":"not_found_or_forbidden"}`, http.StatusNotFound)
		return
	}

	var input model.UpdateEventScheduleItemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	item, err := h.eventRepo.UpdateScheduleItem(r.Context(), itemID, input)
	if err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"item": item})
}

// DeleteScheduleItem — DELETE /api/vendor/events/schedule/{itemId} (propriétaire).
func (h *EventHandler) DeleteScheduleItem(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	itemID := chi.URLParam(r, "itemId")
	if err := h.eventRepo.DeleteScheduleItemOwned(r.Context(), itemID, userID); err != nil {
		if err == repository.ErrEventNotFound {
			http.Error(w, `{"error":"not_found_or_forbidden"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"delete_failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RegisterFree — POST /api/events/offers/{offerId}/register (public).
// Inscription à une offre GRATUITE : pas de paiement, envoie le lien de
// visio de l'événement par email.
func (h *EventHandler) RegisterFree(w http.ResponseWriter, r *http.Request) {
	offerID := chi.URLParam(r, "offerId")
	offer, err := h.eventRepo.FindOfferByID(r.Context(), offerID)
	if err != nil {
		http.Error(w, `{"error":"offer_not_found"}`, http.StatusNotFound)
		return
	}
	if !offer.IsFree {
		http.Error(w, `{"error":"offer_not_free"}`, http.StatusBadRequest)
		return
	}
	event, err := h.eventRepo.FindByID(r.Context(), offer.EventID)
	if err != nil {
		http.Error(w, `{"error":"event_not_found"}`, http.StatusNotFound)
		return
	}

	var input model.CreateEventRegistrationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	input.FullName = strings.TrimSpace(input.FullName)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.Phone = strings.TrimSpace(input.Phone)

	if input.FullName == "" || input.Email == "" || input.Phone == "" {
		http.Error(w, `{"error":"fields_required"}`, http.StatusBadRequest)
		return
	}
	if len(input.FullName) > maxEventFieldLen || len(input.Email) > maxEventFieldLen || len(input.Phone) > maxEventFieldLen {
		http.Error(w, `{"error":"field_too_long"}`, http.StatusBadRequest)
		return
	}
	if !strings.Contains(input.Email, "@") {
		http.Error(w, `{"error":"invalid_email"}`, http.StatusBadRequest)
		return
	}

	reg, err := h.eventRepo.CreateRegistration(r.Context(), offerID, input)
	if err != nil {
		if err == repository.ErrEventAlreadyRegistered {
			http.Error(w, `{"error":"already_registered"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"registration_failed"}`, http.StatusInternalServerError)
		return
	}

	// Best-effort : une panne d'email ne doit pas faire échouer une
	// inscription déjà persistée en base (même principe que Summit).
	if h.notifications != nil {
		meetingLink := ""
		if event.MeetingLink != nil {
			meetingLink = *event.MeetingLink
		}
		if err := h.notifications.SendEventRegistrationConfirmation(r.Context(), reg.Email, reg.FullName, event.Title, meetingLink); err != nil {
			logEventEmailFailure(reg.Email, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(reg)
}

// ListRegistrations — GET /api/vendor/events/{id}/registrations (propriétaire
// uniquement — vérifié via FindByID + comparaison VendorID, pas de requête
// dédiée : la liste des inscrits n'a de sens que rattachée à "son" événement).
func (h *EventHandler) ListRegistrations(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	event, err := h.eventRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if event.VendorID != userID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	regs, err := h.eventRepo.ListRegistrationsByEvent(r.Context(), event.ID)
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"registrations": regs})
}

// ListApproved — GET /api/events (public, catalogue d'événements).
func (h *EventHandler) ListApproved(w http.ResponseWriter, r *http.Request) {
	events, err := h.eventRepo.ListApproved(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"events": events})
}

// --- Admin ---

// ListPendingModeration — GET /api/admin/events/pending.
func (h *EventHandler) ListPendingModeration(w http.ResponseWriter, r *http.Request) {
	events, err := h.eventRepo.ListPendingModeration(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"events": events})
}

// Moderate — PUT /api/admin/events/{id}/moderate.
func (h *EventHandler) Moderate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input struct {
		Status string  `json:"status"`
		Note   *string `json:"note,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Status != "approved" && input.Status != "rejected" {
		http.Error(w, `{"error":"invalid_status"}`, http.StatusBadRequest)
		return
	}
	if err := h.eventRepo.UpdateModerationStatus(r.Context(), id, input.Status, input.Note); err != nil {
		http.Error(w, `{"error":"moderation_failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func logEventEmailFailure(toEmail string, err error) {
	log.Printf("WARNING: email confirmation événement échoué pour %s: %v", toEmail, err)
}
