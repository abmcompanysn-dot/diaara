package handler

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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
}

func NewSaleHandler(
	saleRepo *repository.SaleRepo,
	productRepo *repository.ProductRepo,
	referralRepo *repository.ReferralRepo,
	userRepo *repository.UserRepo,
	pawapay *payment.PawaPayClient,
) *SaleHandler {
	return &SaleHandler{
		saleRepo:      saleRepo,
		productRepo:   productRepo,
		referralRepo:  referralRepo,
		userRepo:      userRepo,
		pawapay:       pawapay,
		commissionSvc: service.NewCommissionService(),
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

	// PawaPay exige toujours le téléphone mobile money, l'opérateur et le pays.
	if input.Phone == "" || input.Operator == "" || input.Country == "" {
		http.Error(w, `{"error":"payment_details_required"}`, http.StatusBadRequest)
		return
	}
	op, err := payment.ResolveOperator(input.Country, input.Operator)
	if err != nil {
		http.Error(w, `{"error":"unsupported_operator"}`, http.StatusBadRequest)
		return
	}
	msisdn, err := payment.NormalizePhone(op.DialCode, input.Phone)
	if err != nil {
		http.Error(w, `{"error":"invalid_phone_number"}`, http.StatusBadRequest)
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

	// Initier le dépôt mobile money PawaPay (asynchrone).
	deposit, err := h.initiateDeposit(r.Context(), created, product, op.Provider, msisdn)
	if err != nil || deposit == nil {
		h.saleRepo.UpdateStatus(r.Context(), created.ID, string(model.SaleFailed))
		http.Error(w, `{"error":"payment_init_failed"}`, http.StatusBadGateway)
		return
	}
	if deposit.Status != "ACCEPTED" {
		h.saleRepo.UpdateStatus(r.Context(), created.ID, string(model.SaleFailed))
		code := "rejected"
		if deposit.FailureReason != nil {
			code = deposit.FailureReason.FailureCode
		}
		http.Error(w, fmt.Sprintf(`{"error":"payment_rejected","reason":"%s"}`, code), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order": created,
		"checkout": map[string]string{
			"deposit_id": created.PaymentReference,
			"status":     deposit.Status,
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

// initiateDeposit initie un dépôt mobile money PawaPay pour une vente.
func (h *SaleHandler) initiateDeposit(ctx context.Context, sale *model.Sale, product *model.Product, provider, msisdn string) (*payment.DepositInitiationResponse, error) {
	if h.pawapay == nil {
		return nil, errors.New("payment non configuré")
	}
	req := payment.DepositRequest{
		DepositId: sale.PaymentReference,
		Payer: payment.Payer{
			Type: "MMO",
			AccountDetails: payment.AccountDetails{
				PhoneNumber: msisdn,
				Provider:    provider,
			},
		},
		Amount:            fmt.Sprintf("%d", sale.AmountCFA),
		Currency:          "XOF",
		ClientReferenceId: sale.ID,
		CustomerMessage:   "PAIEMENT DIARRA",
		Metadata: []payment.MetadataItem{
			{"saleId": sale.ID},
			{"product": product.Title},
		},
	}
	return h.pawapay.InitiateDeposit(ctx, req)
}
