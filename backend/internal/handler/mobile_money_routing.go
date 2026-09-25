package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/repository"
)

// resolveMobileMoneyProvider détermine le prestataire mobile money
// ("pawapay" | "paydunya") pour un opérateur donné. Deux niveaux (voir
// model.GatewayOperatorSettingKey/CheckoutProviderSettingKey) :
//  1. Interrupteur général par PAYS (CheckoutProviderSettingKey) : le
//     prestataire par défaut si aucun réglage plus précis n'existe.
//  2. Réglage par OPÉRATEUR EXACT (GatewayOperatorSettingKey, code logique —
//     voir payment.LogicalOperator), prioritaire — permet ex. "Wave Sénégal
//     → PayDunya" alors que le Sénégal est par ailleurs sur PawaPay. "off"
//     bloque l'opérateur (voir isOperatorBlocked, vérifié à part).
//
// Partagée par SaleHandler (checkout classique) et YesHandler (achat
// conversationnel) — les deux flux passent par le même formulaire
// opérateur/téléphone côté frontend (voir checkout-view.tsx,
// vendor-chat-yes.tsx) et doivent router identiquement.
func resolveMobileMoneyProvider(ctx context.Context, settingsRepo *repository.SettingsRepo, country, operatorCode string) string {
	defaultProvider := settingsRepo.Get(ctx, model.CheckoutProviderSettingKey(country), "pawapay")
	resolved := defaultProvider
	if operatorCode != "" {
		if v := settingsRepo.Get(ctx, model.GatewayOperatorSettingKey(operatorCode), ""); v != "" {
			resolved = v // "off" | "pawapay" | "paydunya" — "off" traité par isOperatorBlocked
		}
	}
	if resolved == "off" {
		return resolved
	}
	// Le réglage résolu (explicite ou défaut du pays) ne vaut que pour un
	// prestataire qui couvre réellement cet opérateur — un opérateur
	// PayDunya-only (Mali/Togo entiers, Djamo, Expresso, Celtiis Cash : pas
	// de PawaPayCode) ne peut jamais être routé vers "pawapay", même en
	// l'absence de tout réglage (voir SaleHandler.CheckoutConfig, même
	// correctif). Sans opérateur connu (operatorCode vide ou logique
	// introuvable), on fait confiance au réglage résolu tel quel.
	logicalOp, ok := payment.FindLogicalOperator(operatorCode)
	if !ok {
		return resolved
	}
	for _, p := range logicalOp.AvailableProviders() {
		if p == resolved {
			return resolved
		}
	}
	if available := logicalOp.AvailableProviders(); len(available) > 0 {
		return available[0]
	}
	return resolved
}

// isOperatorBlocked — vrai si l'admin a explicitement désactivé cet
// opérateur (réglage "off", voir model.GatewayOperatorSettingKey).
func isOperatorBlocked(ctx context.Context, settingsRepo *repository.SettingsRepo, operatorCode string) bool {
	if operatorCode == "" {
		return false
	}
	return settingsRepo.Get(ctx, model.GatewayOperatorSettingKey(operatorCode), "") == "off"
}

// initiateMobileMoneyDeposit — dépôt mobile money de bout en bout vers le
// prestataire déjà résolu (providerName, voir resolveMobileMoneyProvider),
// en traduisant operatorCode (code LOGIQUE choisi par l'acheteur, voir
// payment.LogicalOperator.Code) vers le code spécifique attendu par le
// prestataire réel. Partagée par SaleHandler et YesHandler — les deux
// encaissent en mobile money direct avec le même formulaire opérateur/
// téléphone.
func initiateMobileMoneyDeposit(
	ctx context.Context,
	pawapay *payment.PawaPayClient,
	paydunya *payment.PayDunyaClient,
	saleRepo *repository.SaleRepo,
	saleID, paymentReference, productTitle, buyerName, buyerEmail string,
	amountCFA int,
	providerName, operatorCode, phone, returnURL, otp string,
) (string, error) {
	logicalOp, hasLogicalOp := payment.FindLogicalOperator(operatorCode)

	if providerName == "paydunya" {
		if paydunya == nil {
			return "", errors.New("payment non configuré")
		}
		payDunyaCode := operatorCode
		if hasLogicalOp {
			payDunyaCode = logicalOp.PayDunyaCode
		}
		op, ok := payment.FindPayDunyaOperator(payDunyaCode)
		if !ok {
			return "", fmt.Errorf("opérateur PayDunya inconnu: %s", payDunyaCode)
		}
		token, redirectURL, err := paydunya.InitiateDirectDeposit(ctx, payment.DirectDepositRequest{
			ProductTitle: productTitle,
			BuyerName:    buyerName,
			BuyerEmail:   buyerEmail,
			AmountCFA:    amountCFA,
			Operator:     op,
			Phone:        phone,
			ReturnURL:    returnURL,
			OTP:          otp,
		})
		if token != "" {
			saleRepo.SetPaymentReference(ctx, saleID, token)
		}
		return redirectURL, err
	}

	if pawapay == nil {
		return "", errors.New("payment non configuré")
	}
	pawaPayCode := operatorCode
	if hasLogicalOp {
		pawaPayCode = logicalOp.PawaPayCode
	}
	if pawaPayCode == "" {
		return "", fmt.Errorf("opérateur PawaPay inconnu: %s", operatorCode)
	}
	var country string
	if hasLogicalOp {
		country = logicalOp.Country
	}
	return pawapay.InitiateDirectDeposit(ctx, paymentReference, country, phone, pawaPayCode, amountCFA, returnURL)
}
