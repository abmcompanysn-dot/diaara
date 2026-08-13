package handler

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/diarra/backend/internal/auth"
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
	userRepo      *repository.UserRepo
	pawapay       *payment.PawaPayClient
	commissionSvc *service.CommissionService
	frontendURL   string
}

func NewSaleHandler(
	saleRepo *repository.SaleRepo,
	productRepo *repository.ProductRepo,
	referralRepo *repository.ReferralRepo,
	userRepo *repository.UserRepo,
	pawapay *payment.PawaPayClient,
	frontendURL string,
) *SaleHandler {
	return &SaleHandler{
		saleRepo:      saleRepo,
		productRepo:   productRepo,
		referralRepo:  referralRepo,
		userRepo:      userRepo,
		pawapay:       pawapay,
		commissionSvc: service.NewCommissionService(),
		frontendURL:   strings.TrimSuffix(frontendURL, "/"),
	}
}

// Create — l'acheteur crée une commande et initie un dépôt mobile money PawaPay
func (h *SaleHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var input model.CreateOrderInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if input.BuyerName == "" {
		http.Error(w, `{"error":"name_required"}`, http.StatusBadRequest)
		return
	}

	// Guest Checkout : si non connecté, on exige l'email
	if userID == "" {
		if input.BuyerEmail == nil || *input.BuyerEmail == "" {
			http.Error(w, `{"error":"email_required_for_guest_checkout"}`, http.StatusBadRequest)
			return
		}

		// Si l'email possède déjà un compte, on le réutilise (l'acheteur pourra
		// retrouver ses commandes en se connectant). Sinon on crée un compte invité.
		user, err := h.userRepo.FindByEmail(r.Context(), *input.BuyerEmail)
		if err != nil {
			guestPassword := "GUEST_PASSWORD_" + uuidString()
			hash, hashErr := auth.HashPassword(guestPassword)
			if hashErr != nil {
				http.Error(w, `{"error":"guest_account_creation_failed"}`, http.StatusInternalServerError)
				return
			}
			user, err = h.userRepo.Create(r.Context(), model.RegisterInput{
				Email:    *input.BuyerEmail,
				Password: guestPassword,
			}, hash)
			if err != nil {
				http.Error(w, `{"error":"guest_account_creation_failed"}`, http.StatusInternalServerError)
				return
			}
		}
		userID = user.ID
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
		BuyerName:       input.BuyerName,
		AmountCFA:       product.PriceCFA,
		PlatformFeeCFA:  comm.PlatformFeeCFA,
		VendorAmountCFA: comm.VendorAmountCFA,
		PaymentProvider: "pawapay",
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

	// payment_reference = depositId PawaPay (UUID fourni par nous, unique).
	// checkout_token = jeton secret pour suivre la commande sans être connecté.
	checkoutToken := newUUID()
	sale.PaymentReference = newUUID()
	sale.CheckoutToken = &checkoutToken

	created, err := h.saleRepo.Create(r.Context(), sale)
	if err != nil {
		http.Error(w, `{"error":"order_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	// Page de paiement PawaPay hébergée : l'acheteur y choisit lui-même son
	// opérateur mobile money, PawaPay le redirige ensuite vers ReturnUrl.
	page, err := h.initiatePaymentPage(r.Context(), created, product)
	if err != nil || page == nil || page.RedirectUrl == "" {
		log.Printf("payment_init_failed sale=%s: %v", created.ID, err)
		h.saleRepo.UpdateStatus(r.Context(), created.ID, string(model.SaleFailed))
		http.Error(w, `{"error":"payment_init_failed"}`, http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order": created,
		"checkout": map[string]string{
			"deposit_id":   created.PaymentReference,
			"redirect_url": page.RedirectUrl,
		},
	})
}

// CheckoutStatus — suivi public d'une commande par checkout_token
// (permet à un acheteur invité de connaître l'état de son paiement sans compte).
func (h *SaleHandler) CheckoutStatus(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"error":"token_required"}`, http.StatusBadRequest)
		return
	}
	sale, err := h.saleRepo.FindByCheckoutToken(r.Context(), token)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}

	status := sale.Status
	// Si le webhook n'est pas (encore) arrivé, on demande le statut frais à PawaPay.
	if (status == string(model.SalePending)) && h.pawapay != nil {
		if res, err := h.pawapay.GetDepositStatus(r.Context(), sale.PaymentReference); err == nil && res.Data != nil {
			switch res.Data.Status {
			case "COMPLETED":
				status = string(model.SalePaid)
			case "FAILED":
				status = string(model.SaleFailed)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order": map[string]interface{}{
			"id":          sale.ID,
			"status":      status,
			"amount_cfa":  sale.AmountCFA,
			"product_id":  sale.ProductID,
			"created_at":  sale.CreatedAt,
			"delivered_at": sale.DeliveredAt,
		},
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

// initiatePaymentPage crée une page de paiement PawaPay hébergée pour une vente :
// l'acheteur y choisit lui-même son opérateur mobile money et son téléphone.
func (h *SaleHandler) initiatePaymentPage(ctx context.Context, sale *model.Sale, product *model.Product) (*payment.PaymentPageResponse, error) {
	if h.pawapay == nil {
		return nil, errors.New("payment non configuré")
	}
	returnURL := h.frontendURL + "/checkout/return?token=" + *sale.CheckoutToken
	reason := product.Title
	if runes := []rune(reason); len(runes) > 50 {
		reason = string(runes[:47]) + "..."
	}
	req := payment.PaymentPageRequest{
		DepositId: sale.PaymentReference,
		ReturnUrl: returnURL,
		AmountDetails: payment.AmountDetails{
			Amount:   fmt.Sprintf("%d", sale.AmountCFA),
			Currency: "XOF",
		},
		Reason:          reason,
		CustomerMessage: "PAIEMENT DIARRA",
		Language:        "FR",
		Metadata: []payment.MetadataItem{
			{"saleId": sale.ID},
			{"product": product.Title},
		},
	}
	return h.pawapay.CreatePaymentPage(ctx, req)
}
