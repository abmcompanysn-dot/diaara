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
	"time"

	"github.com/diarra/backend/internal/auth"
	"github.com/diarra/backend/internal/email"
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
	settingsRepo  *repository.SettingsRepo
	pawapay       *payment.PawaPayClient
	commissionSvc *service.CommissionService
	notifications *email.NotificationService
	frontendURL   string
}

func NewSaleHandler(
	saleRepo *repository.SaleRepo,
	productRepo *repository.ProductRepo,
	referralRepo *repository.ReferralRepo,
	userRepo *repository.UserRepo,
	settingsRepo *repository.SettingsRepo,
	pawapay *payment.PawaPayClient,
	notifications *email.NotificationService,
	frontendURL string,
) *SaleHandler {
	return &SaleHandler{
		saleRepo:      saleRepo,
		productRepo:   productRepo,
		referralRepo:  referralRepo,
		userRepo:      userRepo,
		settingsRepo:  settingsRepo,
		pawapay:       pawapay,
		commissionSvc: service.NewCommissionService(),
		notifications: notifications,
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
	if input.Country == "" {
		http.Error(w, `{"error":"country_required"}`, http.StatusBadRequest)
		return
	}
	if payment.UnavailableCountries[input.Country] {
		http.Error(w, `{"error":"country_not_available"}`, http.StatusBadRequest)
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

			// Le mot de passe généré ci-dessus n'est jamais communiqué à l'acheteur :
			// on lui envoie un lien "définir mon mot de passe" (même flux que
			// ForgotPassword) pour qu'il puisse se reconnecter plus tard.
			if h.notifications != nil {
				if resetToken, tokenErr := auth.GenerateToken(); tokenErr == nil {
					if createErr := h.userRepo.CreatePasswordReset(r.Context(), user.ID, auth.HashToken(resetToken), time.Now().Add(1*time.Hour)); createErr == nil {
						go h.notifications.SendPasswordReset(context.Background(), user.Email, resetToken)
					}
				}
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

	// Montant de la vente : pour un produit à prix fixe (comportement par
	// défaut, inchangé), on ignore tout montant envoyé par le client et on
	// utilise product.PriceCFA — ne jamais faire confiance à un montant
	// client pour un produit à prix fixe. Seul un produit explicitement en
	// price_mode "flexible" autorise l'acheteur à choisir son montant, et
	// uniquement au-dessus du minimum fixé par le vendeur.
	amount := product.PriceCFA
	if product.PriceMode == "flexible" {
		if input.AmountCFA == nil || *input.AmountCFA <= 0 {
			http.Error(w, `{"error":"amount_required_for_flexible_pricing"}`, http.StatusBadRequest)
			return
		}
		minAmount := 0
		if product.MinPriceCFA != nil {
			minAmount = *product.MinPriceCFA
		}
		if *input.AmountCFA < minAmount {
			http.Error(w, `{"error":"amount_below_minimum"}`, http.StatusBadRequest)
			return
		}
		amount = *input.AmountCFA
	}

	// Commission : taux configurable depuis l'admin (settings.commission_rate_pct,
	// 15% par défaut). Calculé ici plutôt que via CommissionService (qui reste
	// au taux fixe pour ses autres usages) pour ne dépendre que de ce réglage.
	rate := h.settingsRepo.GetFloat(r.Context(), model.SettingCommissionRatePct, service.DefaultPlatformFeePct)
	platformFee := int(float64(amount) * rate / 100.0)
	sale := &model.Sale{
		ProductID:       product.ID,
		BuyerID:         userID,
		BuyerName:       input.BuyerName,
		Country:         &input.Country,
		AmountCFA:       amount,
		PlatformFeeCFA:  platformFee,
		VendorAmountCFA: amount - platformFee,
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

		closerFee := int(float64(amount) * link.CommissionPct / 100.0)
		rest := amount - platformFee
		if closerFee > rest {
			closerFee = rest
		}
		sale.ReferralLinkID = &link.ID
		sale.CloserCommissionCFA = closerFee
		sale.VendorAmountCFA = rest - closerFee
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
	page, err := h.initiatePaymentPage(r.Context(), created, product, input.Country)
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

// ListVendor — GET /api/vendor/sales : les ventes des produits du vendeur
// connecté, avec les coordonnées de l'acheteur (nom, email, pays).
func (h *SaleHandler) ListVendor(w http.ResponseWriter, r *http.Request) {
	vendorID := middleware.GetUserID(r.Context())
	if vendorID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	sales, err := h.saleRepo.ListByVendor(r.Context(), vendorID)
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"sales": sales})
}

func newUUID() string {
	return uuidString()
}

// initiatePaymentPage crée une page de paiement PawaPay hébergée pour une vente :
// l'acheteur y choisit lui-même son opérateur mobile money et son téléphone.
// PawaPay exige un pays (ou un téléphone) dès qu'un montant est fourni — on ne
// demande que le pays, l'opérateur reste au choix de l'acheteur sur leur page.
func (h *SaleHandler) initiatePaymentPage(ctx context.Context, sale *model.Sale, product *model.Product, country string) (*payment.PaymentPageResponse, error) {
	if h.pawapay == nil {
		return nil, errors.New("payment non configuré")
	}
	returnURL := h.frontendURL + "/checkout/return?token=" + *sale.CheckoutToken
	// PawaPay a rejeté en production des titres contenant un emoji ou un
	// tiret cadratin (INVALID_PARAMETER sur "reason", incident 2026-08-27)
	// alors que la longueur était valide — voir SanitizePaymentReason.
	reason := payment.SanitizePaymentReason(product.Title, 50)

	// Le catalogue est tarifé en XOF ; on convertit vers la devise locale du
	// pays choisi (PawaPay exige le montant dans la devise du pays/opérateur).
	currency := payment.CountryCurrency[country]
	if currency == "" {
		currency = "XOF"
	}
	amount, err := payment.ConvertFromXOF(sale.AmountCFA, currency)
	if err != nil {
		return nil, err
	}

	req := payment.PaymentPageRequest{
		DepositId: sale.PaymentReference,
		ReturnUrl: returnURL,
		AmountDetails: payment.AmountDetails{
			Amount:   amount,
			Currency: currency,
		},
		Country:         country,
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
