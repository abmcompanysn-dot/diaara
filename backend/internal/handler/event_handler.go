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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"event": event, "offers": offers})
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
