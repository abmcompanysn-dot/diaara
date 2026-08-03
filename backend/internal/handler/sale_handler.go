package handler

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

func uuidString() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

type SaleHandler struct {
	saleRepo      *repository.SaleRepo
	productRepo   *repository.ProductRepo
	referralRepo  *repository.ReferralRepo
	paydunya      *payment.PayDunyaClient
	commissionSvc *service.CommissionService
}

func NewSaleHandler(
	saleRepo *repository.SaleRepo,
	productRepo *repository.ProductRepo,
	referralRepo *repository.ReferralRepo,
	paydunya *payment.PayDunyaClient,
) *SaleHandler {
	return &SaleHandler{
		saleRepo:      saleRepo,
		productRepo:   productRepo,
		referralRepo:  referralRepo,
		paydunya:      paydunya,
		commissionSvc: service.NewCommissionService(),
	}
}

// Create — l'acheteur crée une commande et initie le paiement PayDunya
func (h *SaleHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var input model.CreateOrderInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	product, err := h.productRepo.FindByID(r.Context(), input.ProductID)
	if err != nil {
		http.Error(w, `{"error":"product_not_found"}`, http.StatusNotFound)
		return
	}

	if product.ModerationStatus != "approved" {
		http.Error(w, `{"error":"product_not_available"}`, http.StatusBadRequest)
		return
	}

	// Commission : avec ou sans lien d'affiliation (closer).
	comm := h.commissionSvc.Calculate(product.PriceCFA)
	sale := &model.Sale{
		ProductID:       product.ID,
		BuyerID:         userID,
		AmountCFA:       product.PriceCFA,
		PlatformFeeCFA:  comm.PlatformFeeCFA,
		VendorAmountCFA: comm.VendorAmountCFA,
		PaymentProvider: "paydunya",
		Status:          string(model.SalePending),
	}

	if input.ReferralLinkID != nil && *input.ReferralLinkID != "" {
		link, err := h.referralRepo.FindByID(r.Context(), *input.ReferralLinkID)
		if err != nil {
			http.Error(w, `{"error":"referral_link_not_found"}`, http.StatusNotFound)
			return
		}
		// Le lien doit concerner le produit acheté.
		if link.ProductID != product.ID {
			http.Error(w, `{"error":"referral_link_mismatch"}`, http.StatusBadRequest)
			return
		}
		// Anti-fraude : le closer ne peut pas s'acheter via son propre lien.
		if link.CloserID == userID {
			http.Error(w, `{"error":"self_referral_forbidden"}`, http.StatusBadRequest)
			return
		}
		// Le closer ne peut pas promouvoir son propre produit (voir CreateLink).
		if link.CloserID == product.VendorID {
			http.Error(w, `{"error":"self_referral_forbidden"}`, http.StatusBadRequest)
			return
		}

		commCloser := h.commissionSvc.CalculateWithCloser(product.PriceCFA, link.CommissionPct)
		sale.ReferralLinkID = &link.ID
		sale.CloserCommissionCFA = commCloser.CloserCommissionCFA
		sale.VendorAmountCFA = commCloser.VendorAmountCFA
	}

	// payment_reference doit être unique : on crée un identifiant provisoire puis on met à jour
	// avec le token PayDunya après création de la facture.
	sale.PaymentReference = newUUID()

	created, err := h.saleRepo.Create(r.Context(), sale)
	if err != nil {
		http.Error(w, `{"error":"order_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	// Initier la facture PayDunya
	invoice, err := h.createInvoice(r.Context(), created, product)
	if err != nil || invoice == nil {
		h.saleRepo.UpdateStatus(r.Context(), created.ID, string(model.SaleFailed))
		http.Error(w, `{"error":"payment_init_failed"}`, http.StatusBadGateway)
		return
	}

	// Enregistrer le token PayDunya comme référence de paiement (unique)
	if invoice.Token != "" {
		if err := h.saleRepo.UpdatePaymentReference(r.Context(), created.ID, invoice.Token); err != nil {
			h.saleRepo.UpdateStatus(r.Context(), created.ID, string(model.SaleFailed))
			http.Error(w, `{"error":"payment_init_failed"}`, http.StatusBadGateway)
			return
		}
		created.PaymentReference = invoice.Token
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order":       created,
		"payment_url": invoice.InvoiceURL,
	})
}

// Get — statut d'une commande (propriétaire ou admin)
func (h *SaleHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.GetIsAdmin(r.Context())
	id := chi.URLParam(r, "id")

	sale, err := h.saleRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}

	if sale.BuyerID != userID && !isAdmin {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"order": sale})
}

// List — historique des achats de l'utilisateur connecté
func (h *SaleHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	sales, err := h.saleRepo.ListByBuyer(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"orders": sales})
}

func newUUID() string {
	return uuidString()
}

// createInvoice initie la facture PayDunya pour une vente.
func (h *SaleHandler) createInvoice(ctx context.Context, sale *model.Sale, product *model.Product) (*payment.InvoiceResponse, error) {
	if h.paydunya == nil {
		return nil, errors.New("payment non configuré")
	}
	req := payment.InvoiceRequest{}
	req.Invoice.TotalAmount = sale.AmountCFA
	req.Invoice.Description = product.Title
	req.CustomData = map[string]string{
		"sale_id": sale.ID,
	}
	return h.paydunya.CreateInvoice(ctx, req)
}
