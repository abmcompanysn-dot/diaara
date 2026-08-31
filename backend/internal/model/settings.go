package model

import "strings"

// Clés de réglages connues (settings.key). D'autres clés peuvent exister
// sans y être listées ; ceci documente celles utilisées par le code.
const (
	SettingCommissionRatePct = "commission_rate_pct"
	// Dépréciées depuis l'ajout de KPay (voir GatewayOperatorSettingKey) :
	// regroupaient par MARQUE (ex. gateway_mtn_momo couvrait tous les pays
	// à la fois), trop grossier pour router PawaPay/KPay opérateur par
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
)

// GatewayOperatorSettingKey — réglage par code opérateur EXACT (ex.
// "MTN_MOMO_BEN"), valeur à 3 états : "off" | "pawapay" | "kpay". Pilote
// à la fois l'activation d'un opérateur ET le prestataire qui le traite —
// un seul champ pour éviter les états contradictoires (ex. désactivé mais
// assigné à kpay). Voir payment.GatewaySettingKey pour l'ancien
// regroupement par marque (déprécié).
func GatewayOperatorSettingKey(providerCode string) string {
	return "gateway_op_" + strings.ToLower(providerCode)
}

// CheckoutProviderSettingKey — réglage par PAYS (ISO 3166-1 alpha-3), valeur
// "pawapay" | "kpay". Le checkout (mode GATEWAY, page hébergée) ne connaît
// que le pays de l'acheteur au moment de la redirection, jamais l'opérateur
// exact — contrairement aux versements vendeur, routés par
// GatewayOperatorSettingKey.
func CheckoutProviderSettingKey(countryISO3 string) string {
	return "checkout_provider_" + strings.ToLower(countryISO3)
}

// UpdateSettingsInput — mise à jour partielle (seules les clés présentes sont modifiées).
type UpdateSettingsInput map[string]string
