package model

// Clés de réglages connues (settings.key). D'autres clés peuvent exister
// sans y être listées ; ceci documente celles utilisées par le code.
const (
	SettingCommissionRatePct = "commission_rate_pct"
	SettingGatewayOrange     = "gateway_orange_money"
	SettingGatewayWave       = "gateway_wave"
	SettingGatewayMTN        = "gateway_mtn_momo"
	SettingGatewayFree       = "gateway_free_money"
	SettingGatewayMoov       = "gateway_moov_money"
	// SettingAutomationAPIKey : clé secrète pour l'endpoint de création de
	// produit automatisée (voir middleware.RequireAutomation).
	SettingAutomationAPIKey = "automation_api_key"
)

// UpdateSettingsInput — mise à jour partielle (seules les clés présentes sont modifiées).
type UpdateSettingsInput map[string]string
