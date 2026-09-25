package model

import "strings"

// Clés de réglages connues (settings.key). D'autres clés peuvent exister
// sans y être listées ; ceci documente celles utilisées par le code.
const (
	SettingCommissionRatePct = "commission_rate_pct"
	// Dépréciées (voir GatewayOperatorSettingKey) :
	// regroupaient par MARQUE (ex. gateway_mtn_momo couvrait tous les pays
	// à la fois), trop grossier pour router PawaPay/PayDunya opérateur par
	// opérateur. Gardées pour compatibilité arrière, plus lues nulle part.
	SettingGatewayOrange = "gateway_orange_money"
	SettingGatewayWave   = "gateway_wave"
	SettingGatewayMTN    = "gateway_mtn_momo"
	SettingGatewayFree   = "gateway_free_money"
	SettingGatewayMoov   = "gateway_moov_money"
	// SettingAutomationAPIKey : clé secrète pour l'endpoint de création de
	// produit automatisée (voir middleware.RequireAutomation).
	SettingAutomationAPIKey = "automation_api_key"
	// Programme de reversement automatique ("Fidélisation") — voir
	// service.DonationService pour les valeurs par défaut et la logique.
	SettingDonationSharePct     = "donation_share_pct"
	SettingDonationThresholdCFA = "donation_threshold_cfa"
	SettingDonationEnabled      = "donation_program_enabled"

	// Communauté WhatsApp : lien d'invitation général (chat.whatsapp.com/... ou
	// lien de communauté). Inséré dans les emails de bienvenue, de passage
	// vendeur, les broadcasts et les messages admin individuels. Un lien par
	// pays peut le surcharger (voir WhatsAppCommunitySettingKey) ; à défaut on
	// retombe sur ce lien général.
	SettingWhatsAppCommunityURL = "whatsapp_community_url"

	// SettingCardPaymentEnabled — interrupteur admin pour le paiement carte
	// bancaire/PayPal au checkout (valeurs "true"/"false", "true" par défaut
	// si absent). Coupe-circuit indépendant de PAYPAL_CLIENT_ID/SECRET : sert
	// à désactiver ce flux à la volée (ex. souci PayPal) sans toucher au
	// serveur, en repliant le checkout sur PawaPay (mobile money) seul — voir
	// SaleHandler.resolveCheckoutProvider et SaleHandler.CheckoutConfig.
	SettingCardPaymentEnabled = "card_payment_enabled"

	// SettingYesMicroTicketAmountCFA — montant du "ticket d'entrée" (achat
	// conversationnel YES Business, bêta) qui ouvre la discussion avec le
	// vendeur, en FCFA. Défaut 600 si absent (voir
	// model.DefaultYesMicroTicketAmountCFA) — voir YesHandler.OpenConversation.
	SettingYesMicroTicketAmountCFA = "yes_micro_ticket_amount_cfa"
)

// WhatsAppCommunitySettingKey — lien communauté WhatsApp spécifique à un pays
// (ISO 3166-1 alpha-3), ex. "whatsapp_community_url_sen". Optionnel : si le
// réglage est vide, l'appelant retombe sur SettingWhatsAppCommunityURL.
func WhatsAppCommunitySettingKey(countryISO3 string) string {
	return "whatsapp_community_url_" + strings.ToLower(countryISO3)
}

// GatewayOperatorSettingKey — réglage par OPÉRATEUR LOGIQUE exact (voir
// payment.LogicalOperator.Code, ex. "WAVE_SEN"), valeur à 3 états : "off" |
// "pawapay" | "paydunya". Pilote À LA FOIS le versement vendeur ET l'achat
// pour cet opérateur (un seul tableau admin "Mobile Money" pour les deux) —
// utilisé en priorité sur CheckoutProviderSettingKey (niveau 2, prioritaire
// sur le niveau 1). "off" bloque l'opérateur (versement refusé, et il
// n'apparaît plus au checkout). Une ligne sans réglage retombe sur
// l'interrupteur général (voir CheckoutProviderSettingKey) — voir
// payment.ResolveOperatorProvider.
func GatewayOperatorSettingKey(operatorCode string) string {
	return "gateway_op_" + strings.ToLower(operatorCode)
}

// CheckoutProviderSettingKey — interrupteur général PAR PAYS (ISO 3166-1
// alpha-3, "niveau 1"), valeur à 2 états : "pawapay" | "paydunya". S'applique
// à tout opérateur de ce pays qui n'a pas de réglage propre via
// GatewayOperatorSettingKey ("niveau 2", prioritaire) — voir
// payment.ResolveOperatorProvider.
func CheckoutProviderSettingKey(countryISO3 string) string {
	return "checkout_provider_" + strings.ToLower(countryISO3)
}

// UpdateSettingsInput — mise à jour partielle (seules les clés présentes sont modifiées).
type UpdateSettingsInput map[string]string
