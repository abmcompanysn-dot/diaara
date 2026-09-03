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
	kpay          *payment.KPayClient
	paypal        *payment.PayPalClient
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
	kpay *payment.KPayClient,
	paypal *payment.PayPalClient,
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
		kpay:          kpay,
		paypal:        paypal,
		commissionSvc: service.NewCommissionService(),
		notifications: notifications,
		frontendURL:   strings.TrimSuffix(frontendURL, "/"),
	}
}

// resolveDepositProvider retourne l'adaptateur PaymentProvider (statut/
// remboursement, voir payment.PaymentProvider) correspondant au fournisseur
// déjà enregistré sur une vente — jamais recalculé après coup, contrairement
// à la sélection faite à l'initiation (resolveCheckoutProvider).
func (h *SaleHandler) resolveDepositProvider(providerName string) payment.PaymentProvider {
	switch providerName {
	case "kpay":
		if h.kpay != nil {
			return h.kpay.AsProvider()
		}
	case "paypal":
		if h.paypal != nil {
			return h.paypal.AsProvider()
		}
	}
	return h.pawapay.AsProvider()
}

// resolveCheckoutProvider détermine le prestataire à utiliser pour un NOUVEAU
// paiement. Carte/PayPal forcent PayPal (route de capacité : PawaPay n'a pas
// carte/PayPal, ce n'est pas un choix — KPay a été désactivé de ce flux le
// 2026-09-03, voir kpay.go) ; sinon (mobile money) on lit le réglage admin
// par pays (model.CheckoutProviderSettingKey) — le checkout en mode GATEWAY
// ne connaît que le pays de l'acheteur, jamais l'opérateur exact (choisi
// ensuite sur la page hébergée), contrairement aux versements vendeur qui,
// eux, routent par opérateur exact (voir PayoutHandler).
//
// Verrou d'intégration : KPay n'est pas prêt pour le checkout mobile money.
// Si le réglage en base vaut malgré tout "kpay" (valeur posée avant qu'on
// sache l'intégration incomplète, ou modifiée directement en base), on
// l'ignore et on retombe sur pawapay — AdminHandler.UpdateSettings refuse déjà
// d'écrire "kpay" pour ce réglage, ceci est la deuxième barrière côté lecture.
func (h *SaleHandler) resolveCheckoutProvider(ctx context.Context, country, paymentMethod string) string {
	if paymentMethod == "card" || paymentMethod == "paypal" {
		return "paypal"
	}
	provider := h.settingsRepo.Get(ctx, model.CheckoutProviderSettingKey(country), "pawapay")
	if provider == "kpay" {
		return "pawapay"
	}
	return provider
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
	// Palier réduit : à partir de 1 000 000 FCFA, la commission passe à 10%
	// quel que soit le taux configuré (voir service.HighValueThresholdCFA).
	rate := h.settingsRepo.GetFloat(r.Context(), model.SettingCommissionRatePct, service.DefaultPlatformFeePct)
	rate = service.EffectivePlatformFeePct(amount, rate)
	platformFee := int(float64(amount) * rate / 100.0)
	providerName := h.resolveCheckoutProvider(r.Context(), input.Country, input.PaymentMethod)
	sale := &model.Sale{
		ProductID:       product.ID,
		BuyerID:         userID,
		BuyerName:       input.BuyerName,
		Country:         &input.Country,
		AmountCFA:       amount,
		PlatformFeeCFA:  platformFee,
		VendorAmountCFA: amount - platformFee,
		PaymentProvider: providerName,
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

	// Page de paiement hébergée (PawaPay ou KPay selon providerName) :
	// l'acheteur y choisit lui-même son opérateur mobile money/carte/PayPal,
	// le prestataire le redirige ensuite vers ReturnUrl.
	redirectURL, err := h.initiateCheckout(r.Context(), created, product, input.Country, providerName)
	if err != nil || redirectURL == "" {
		log.Printf("payment_init_failed sale=%s provider=%s: %v", created.ID, providerName, err)
		h.saleRepo.UpdateStatus(r.Context(), created.ID, string(model.SaleFailed))
		http.Error(w, `{"error":"payment_init_failed"}`, http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order": created,
		"checkout": map[string]string{
			"deposit_id":   created.PaymentReference,
			"redirect_url": redirectURL,
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
	// Si le webhook n'est pas (encore) arrivé, on demande le statut frais au
	// prestataire qui a réellement traité cette vente (sale.PaymentProvider).
	providerConfigured := (sale.PaymentProvider == "kpay" && h.kpay != nil) ||
		(sale.PaymentProvider == "paypal" && h.paypal != nil) ||
		(sale.PaymentProvider != "kpay" && sale.PaymentProvider != "paypal" && h.pawapay != nil)
	if status == string(model.SalePending) && providerConfigured {
		ref := sale.PaymentReference
		if (sale.PaymentProvider == "kpay" || sale.PaymentProvider == "paypal") && sale.ProviderTransactionID != nil {
			ref = *sale.ProviderTransactionID
		}
		if outcome, err := h.resolveDepositProvider(sale.PaymentProvider).GetDepositStatus(r.Context(), ref); err == nil {
			// PayPal seulement : l'order ID de départ devient l'ID de CAPTURE
			// une fois la commande capturée — nécessaire pour un remboursement
			// ultérieur (voir DepositOutcome.UpdatedProviderRef).
			if outcome.UpdatedProviderRef != "" {
				h.saleRepo.SetProviderTransactionID(r.Context(), sale.ID, outcome.UpdatedProviderRef)
			}
			switch outcome.Status {
			case "completed":
				status = string(model.SalePaid)
			case "failed", "cancelled":
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

// reminderMinInterval — délai minimal entre deux relances manuelles d'une même
// commande (anti-spam ; le cron auto a sa propre cadence 1h/24h).
const reminderMinInterval = 20 * time.Hour

// sendPendingReminder envoie l'email de relance pour une commande "pending" et
// enregistre l'envoi. Partagé par la relance vendeur, la relance admin et le
// cron. Renvoie une erreur "explicative" destinée à l'API (déjà à jour, trop
// tôt, email manquant...).
func sendPendingReminder(ctx context.Context, saleRepo *repository.SaleRepo, notifications *email.NotificationService, v *repository.PendingSaleView, enforceCooldown bool) error {
	if notifications == nil {
		return errors.New("email_not_configured")
	}
	if v.Status != string(model.SalePending) {
		return errors.New("sale_not_pending")
	}
	if v.CheckoutToken == nil || *v.CheckoutToken == "" {
		return errors.New("no_checkout_token")
	}
	if v.BuyerEmail == "" {
		return errors.New("buyer_has_no_email")
	}
	if enforceCooldown && v.RemindedAt != nil && time.Since(*v.RemindedAt) < reminderMinInterval {
		return errors.New("reminded_too_recently")
	}
	if err := notifications.SendCartReminder(ctx, v.BuyerEmail, v.BuyerName, v.ProductTitle, v.AmountCFA, *v.CheckoutToken); err != nil {
		return err
	}
	return saleRepo.MarkReminded(ctx, v.ID)
}

// Cadence de relance automatique des commandes "pending" : 1ʳᵉ relance 1h après
// la création, 2ᵉ 24h après, puis plus rien ; au-delà de 7 jours on abandonne.
const (
	reminderFirstAfter  = 1 * time.Hour
	reminderSecondAfter = 24 * time.Hour
	reminderMaxAge      = 7 * 24 * time.Hour
	reminderScanEvery   = 15 * time.Minute
)

// RunReminderLoop relance automatiquement, en tâche de fond, les acheteurs dont
// la commande est restée "pending" (voir SaleRepo.ListRemindable pour la
// cadence). Boucle jusqu'à annulation du contexte. À lancer une fois au
// démarrage : go saleHandler.RunReminderLoop(ctx).
func (h *SaleHandler) RunReminderLoop(ctx context.Context) {
	if h.notifications == nil {
		log.Printf("relance auto: désactivée (aucun fournisseur email configuré)")
		return
	}
	ticker := time.NewTicker(reminderScanEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.runReminderPass(ctx)
		}
	}
}

func (h *SaleHandler) runReminderPass(ctx context.Context) {
	views, err := h.saleRepo.ListRemindable(ctx, reminderFirstAfter, reminderSecondAfter, reminderMaxAge)
	if err != nil {
		log.Printf("relance auto: échec de la lecture des commandes: %v", err)
		return
	}
	sent := 0
	for _, v := range views {
		// enforceCooldown=false : la fenêtre est déjà décidée par la requête SQL
		// (1h / 24h selon reminder_count).
		if err := sendPendingReminder(ctx, h.saleRepo, h.notifications, v, false); err != nil {
			log.Printf("relance auto: sale=%s ignorée: %v", v.ID, err)
			continue
		}
		sent++
	}
	if sent > 0 {
		log.Printf("relance auto: %d relance(s) envoyée(s) sur %d commande(s) candidate(s)", sent, len(views))
	}
}

// RemindVendor — POST /api/vendor/sales/{id}/remind : le vendeur relance par
// email un acheteur dont la commande d'un de SES produits est restée "pending".
func (h *SaleHandler) RemindVendor(w http.ResponseWriter, r *http.Request) {
	vendorID := middleware.GetUserID(r.Context())
	if vendorID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	saleID := chi.URLParam(r, "id")

	view, err := h.saleRepo.FindPendingViewForVendor(r.Context(), saleID, vendorID)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if err := sendPendingReminder(r.Context(), h.saleRepo, h.notifications, view, true); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func newUUID() string {
	return uuidString()
}

// initiateCheckout distribue vers PawaPay ou KPay selon providerName (déjà
// résolu par resolveCheckoutProvider et persisté sur sale.PaymentProvider),
// et renvoie l'URL de redirection vers la page de paiement hébergée.
func (h *SaleHandler) initiateCheckout(ctx context.Context, sale *model.Sale, product *model.Product, country, providerName string) (string, error) {
	switch providerName {
	case "kpay":
		return h.initiateKPayCheckout(ctx, sale, product)
	case "paypal":
		return h.initiatePayPalCheckout(ctx, sale, product)
	}
	page, err := h.initiatePaymentPage(ctx, sale, product, country)
	if err != nil || page == nil {
		return "", err
	}
	return page.RedirectUrl, nil
}

// initiatePayPalCheckout — mode "hosted redirect" (comme KPay/PawaPay) :
// l'acheteur est redirigé vers la page PayPal pour payer par carte ou son
// compte PayPal, puis revient sur ReturnUrl où le statut est vérifié/capturé
// (voir CheckoutStatus + paypalAdapter.GetDepositStatus).
func (h *SaleHandler) initiatePayPalCheckout(ctx context.Context, sale *model.Sale, product *model.Product) (string, error) {
	if h.paypal == nil {
		return "", errors.New("PayPal non configuré")
	}
	returnURL := h.frontendURL + "/checkout/return?token=" + *sale.CheckoutToken
	cancelURL := h.frontendURL + "/checkout/return?token=" + *sale.CheckoutToken + "&cancelled=1"

	resp, err := h.paypal.CreateOrder(ctx, payment.PayPalOrderRequest{
		SaleID:      sale.ID,
		AmountXOF:   sale.AmountCFA,
		ReturnURL:   returnURL,
		CancelURL:   cancelURL,
		Description: product.Title,
	})
	if err != nil {
		return "", err
	}
	if resp.ID != "" {
		if err := h.saleRepo.SetProviderTransactionID(ctx, sale.ID, resp.ID); err != nil {
			return "", err
		}
	}
	return resp.ApproveURL, nil
}

// initiateKPayCheckout — mode GATEWAY (page hébergée par KPay) : pas de
// provider/phoneNumber envoyés, l'acheteur choisit lui-même son opérateur
// mobile money, sa carte ou PayPal sur la page KPay. paymentMethod
// "card"/"paypal" force ce mode même si le pays serait normalement routé
// vers PawaPay (voir resolveCheckoutProvider).
//
// Devise : XOF par défaut (même devise que le catalogue) — à ajuster si la
// passerelle carte de KPay l'exige autrement (à vérifier en sandbox).
func (h *SaleHandler) initiateKPayCheckout(ctx context.Context, sale *model.Sale, product *model.Product) (string, error) {
	if h.kpay == nil {
		return "", errors.New("KPay non configuré")
	}
	returnURL := h.frontendURL + "/checkout/return?token=" + *sale.CheckoutToken

	req := payment.PaymentInitRequest{
		Amount:      fmt.Sprintf("%d", sale.AmountCFA),
		Currency:    "XOF",
		ExternalId:  sale.PaymentReference,
		ReturnUrl:   returnURL,
		Description: product.Title,
		Metadata:    map[string]string{"saleId": sale.ID},
	}
	resp, err := h.kpay.InitiatePayment(ctx, req)
	if err != nil {
		return "", err
	}
	if resp.ID != "" {
		if err := h.saleRepo.SetProviderTransactionID(ctx, sale.ID, resp.ID); err != nil {
			return "", err
		}
	}
	return resp.GatewayUrl, nil
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
		// Sans callbackUrl, PawaPay ne pousse jamais le statut final : la vente
		// reste "pending" même après paiement (voir PaymentPageRequest.CallbackUrl).
		CallbackUrl: h.pawapay.CallbackURL(),
		Metadata: []payment.MetadataItem{
			{"saleId": sale.ID},
			{"product": product.Title},
		},
	}
	return h.pawapay.CreatePaymentPage(ctx, req)
}
