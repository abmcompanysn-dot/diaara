package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/diarra/backend/internal/auth"
	"github.com/diarra/backend/internal/eventfile"
	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

// EventTicketHandler génère le billet PDF (avec QR de vérification) d'une
// vente d'offre d'événement payante, et gère le scan à l'entrée. Le PDF est
// généré à la demande (jamais au moment du webhook de paiement — voir la
// décision prise avec l'utilisateur : ne pas toucher au chemin critique de
// confirmation de paiement, déjà fragile par le passé).
type EventTicketHandler struct {
	saleRepo    *repository.SaleRepo
	productRepo *repository.ProductRepo
	eventRepo   *repository.EventRepo
	ticketRepo  *repository.EventTicketRepo
	frontendURL string
}

func NewEventTicketHandler(saleRepo *repository.SaleRepo, productRepo *repository.ProductRepo, eventRepo *repository.EventRepo, ticketRepo *repository.EventTicketRepo, frontendURL string) *EventTicketHandler {
	return &EventTicketHandler{
		saleRepo:    saleRepo,
		productRepo: productRepo,
		eventRepo:   eventRepo,
		ticketRepo:  ticketRepo,
		frontendURL: frontendURL,
	}
}

// PDF — GET /api/orders/{token}/ticket.pdf (public, par checkout_token —
// même régime d'accès que SaleHandler.CheckoutStatus : un acheteur invité
// peut récupérer son billet sans compte). Ne sert que les ventes payées
// d'une offre d'événement ; toute autre vente renvoie 404 (pas de billet).
func (h *EventTicketHandler) PDF(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	sale, err := h.saleRepo.FindByCheckoutToken(r.Context(), token)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if sale.Status != string(model.SalePaid) && sale.Status != string(model.SaleDelivered) {
		http.Error(w, `{"error":"not_paid"}`, http.StatusForbidden)
		return
	}

	product, err := h.productRepo.FindByID(r.Context(), sale.ProductID)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	event, offer, err := h.eventRepo.EventAndOfferByProductID(r.Context(), product.ID)
	if err != nil {
		// Ce produit n'est pas une offre d'événement — pas de billet à générer.
		http.Error(w, `{"error":"not_a_ticket"}`, http.StatusNotFound)
		return
	}

	checkinToken, err := auth.GenerateToken()
	if err != nil {
		http.Error(w, `{"error":"token_generation_failed"}`, http.StatusInternalServerError)
		return
	}
	checkin, err := h.ticketRepo.FindOrCreateBySale(r.Context(), sale.ID, checkinToken)
	if err != nil {
		http.Error(w, `{"error":"checkin_failed"}`, http.StatusInternalServerError)
		return
	}

	checkInURL := fmt.Sprintf("%s/scan?token=%s", strings.TrimSuffix(h.frontendURL, "/"), checkin.CheckInToken)
	pdfBytes, err := eventfile.BuildPDF(eventfile.Ticket{
		EventTitle: event.Title,
		OfferTitle: offer.Title,
		BuyerName:  sale.BuyerName,
		PriceCFA:   sale.AmountCFA,
		CheckInURL: checkInURL,
	})
	if err != nil {
		http.Error(w, `{"error":"pdf_generation_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="billet.pdf"`)
	w.Write(pdfBytes)
}

// ScanStatus — GET /api/admin/events/scan?token=... (admin, ou vendeur
// propriétaire de l'événement — vérifié via event.VendorID). Renvoie l'état
// du billet sans le marquer utilisé, pour que la page de scan affiche
// d'abord "Confirmer l'entrée ?" avant validation.
func (h *EventTicketHandler) ScanStatus(w http.ResponseWriter, r *http.Request) {
	h.scan(w, r, false)
}

// ScanConfirm — POST /api/admin/events/scan (admin ou vendeur). Marque le
// billet comme utilisé (premier scan gagne).
func (h *EventTicketHandler) ScanConfirm(w http.ResponseWriter, r *http.Request) {
	h.scan(w, r, true)
}

func (h *EventTicketHandler) scan(w http.ResponseWriter, r *http.Request, confirm bool) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"error":"token_required"}`, http.StatusBadRequest)
		return
	}

	checkin, err := h.ticketRepo.FindByToken(r.Context(), token)
	if err != nil {
		http.Error(w, `{"error":"invalid_ticket"}`, http.StatusNotFound)
		return
	}
	sale, err := h.saleRepo.FindByID(r.Context(), checkin.SaleID)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	product, err := h.productRepo.FindByID(r.Context(), sale.ProductID)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	event, offer, err := h.eventRepo.EventAndOfferByProductID(r.Context(), product.ID)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}

	// Accès : tout admin (RequireAdmin, déjà vérifié par le middleware de la
	// route), OU le vendeur propriétaire de l'événement — voir la décision
	// prise avec l'utilisateur ("le vendeur propriétaire aussi").
	isAdmin := middleware.GetIsAdmin(r.Context())
	if !isAdmin && event.VendorID != userID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	if confirm {
		updated, alreadyUsed, err := h.ticketRepo.CheckIn(r.Context(), token, userID)
		if err != nil {
			http.Error(w, `{"error":"checkin_failed"}`, http.StatusInternalServerError)
			return
		}
		checkin = updated
		writeJSON(w, map[string]interface{}{
			"buyer_name":    sale.BuyerName,
			"event_title":   event.Title,
			"offer_title":   offer.Title,
			"already_used":  alreadyUsed,
			"checked_in_at": checkin.CheckedInAt,
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"buyer_name":    sale.BuyerName,
		"event_title":   event.Title,
		"offer_title":   offer.Title,
		"already_used":  checkin.CheckedInAt != nil,
		"checked_in_at": checkin.CheckedInAt,
	})
}
