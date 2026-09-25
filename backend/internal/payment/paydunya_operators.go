package payment

// PayDunyaOperator décrit un opérateur mobile money PayDunya : contrairement
// à PawaPay (une seule API pour tous), chaque opérateur a son propre
// endpoint SoftPay ET ses propres noms de champs JSON (voir
// developers.paydunya.com/doc/FR/softpay, relevé 2026-09-25) — pas
// unifiable dans un struct Go commun, d'où BuildPayload ci-dessous.
type PayDunyaOperator struct {
	Label    string // affiché à l'acheteur
	Provider string // code interne DIARRA, ex "ORANGE_SN" — distinct des codes PawaPay
	Country  string // ISO 3166-1 alpha-3
	DialCode string
	Endpoint string // suffixe après /softpay/ (dépôt)
	// TokenField — nom du champ JSON portant le token de facture pour cet
	// opérateur ("invoice_token" ou "payment_token" selon l'opérateur, voir
	// doc — aucune règle fixe par pays).
	TokenField string
	// BuildPayload construit le corps JSON exact attendu par cet endpoint
	// (noms de champs spécifiques à l'opérateur) à partir des infos communes.
	BuildPayload func(name, email, phone, token string) map[string]interface{}
	// WithdrawMode — code PayDunya pour l'API de déboursement (versement
	// vendeur), voir developers.paydunya.com/doc/FR/api_deboursement —
	// distinct de Endpoint (dépôt) : par exemple Endpoint="new-orange-money-senegal"
	// mais WithdrawMode="orange-money-senegal".
	WithdrawMode string
	// PawaPayCode — code PawaPay équivalent (voir XOFOperators), ex
	// "ORANGE_SEN". Le vendeur choisit son opérateur via ce code (moyen de
	// retrait, voir PAYOUT_COUNTRIES côté frontend) quel que soit le
	// prestataire réellement utilisé pour le versement — un seul "moyen de
	// retrait" par vendeur, indépendant du routage admin par opérateur (voir
	// FindPayDunyaOperatorByPawaPayCode, utilisé par le versement).
	PawaPayCode string
}

// PayDunyaOperators couvre les pays/opérateurs mobile money PayDunya (hors
// carte bancaire, hors OTP/preauth spécifiques comme Orange CI/BFA qui
// exigent un code reçu par SMS avant l'appel — non supportés ici, flux en
// une étape uniquement). Miroir prévu côté frontend
// (frontend/src/lib/operators.ts, PAYDUNYA_COUNTRIES).
var PayDunyaOperators = []PayDunyaOperator{
	// Sénégal
	{
		Label: "Orange Money", Provider: "ORANGE_SN", Country: "SEN", DialCode: "221", PawaPayCode: "ORANGE_SEN",
		Endpoint: "new-orange-money-senegal", TokenField: "invoice_token", WithdrawMode: "orange-money-senegal",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"customer_name": name, "customer_email": email, "phone_number": phone, "invoice_token": token}
		},
	},
	{
		Label: "Wave", Provider: "WAVE_SN", Country: "SEN", DialCode: "221", PawaPayCode: "WAVE_SEN",
		Endpoint: "wave-senegal", TokenField: "wave_senegal_payment_token", WithdrawMode: "wave-senegal",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"wave_senegal_fullName": name, "wave_senegal_email": email, "wave_senegal_phone": phone, "wave_senegal_payment_token": token}
		},
	},
	{
		Label: "Free Money", Provider: "FREE_SN", Country: "SEN", DialCode: "221", PawaPayCode: "FREE_SEN",
		Endpoint: "free-money-senegal", TokenField: "payment_token", WithdrawMode: "free-money-senegal",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"customer_name": name, "customer_email": email, "phone_number": phone, "payment_token": token}
		},
	},
	{
		Label: "Expresso", Provider: "EXPRESSO_SN", Country: "SEN", DialCode: "221",
		Endpoint: "expresso-senegal", TokenField: "payment_token", WithdrawMode: "expresso-senegal",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"expresso_sn_fullName": name, "expresso_sn_email": email, "expresso_sn_phone": phone, "payment_token": token}
		},
	},
	{
		Label: "Djamo", Provider: "DJAMO_SN", Country: "SEN", DialCode: "221",
		Endpoint: "djamo", TokenField: "djamo_payment_token", WithdrawMode: "djamo-sn",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"djamo_fullName": name, "djamo_email": email, "djamo_phone": phone, "code_country": "sn", "djamo_payment_token": token}
		},
	},
	// Bénin
	{
		Label: "MTN MoMo", Provider: "MTN_BJ", Country: "BEN", DialCode: "229", PawaPayCode: "MTN_MOMO_BEN",
		Endpoint: "mtn-benin", TokenField: "payment_token", WithdrawMode: "mtn-benin",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"mtn_benin_customer_fullname": name, "mtn_benin_email": email, "mtn_benin_phone_number": phone, "mtn_benin_wallet_provider": "MTNBENIN", "payment_token": token}
		},
	},
	{
		Label: "Moov Money", Provider: "MOOV_BJ", Country: "BEN", DialCode: "229", PawaPayCode: "MOOV_BEN",
		Endpoint: "moov-benin", TokenField: "payment_token", WithdrawMode: "moov-benin",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"moov_benin_customer_fullname": name, "moov_benin_email": email, "moov_benin_phone_number": phone, "payment_token": token}
		},
	},
	{
		Label: "Celtiis Cash", Provider: "CELTIIS_BJ", Country: "BEN", DialCode: "229",
		Endpoint: "celtiis-cash", TokenField: "payment_token", WithdrawMode: "celtiis-cash",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"celtiis_cash_customer_fullname": name, "celtiis_cash_customer_email": email, "celtiis_cash_phone_number": phone, "payment_token": token}
		},
	},
	// Côte d'Ivoire — Orange CI exclu du DÉPÔT : exige un OTP SMS obtenu
	// AVANT l'appel SoftPay (orange_money_ci_otp), flux en une étape non
	// supporté par DIARRA aujourd'hui. Reste disponible en WithdrawMode pour
	// le versement vendeur (pas cette contrainte côté déboursement) — voir
	// PayDunyaWithdrawOnlyOperators plus bas si besoin un jour.
	{
		Label: "MTN MoMo", Provider: "MTN_CI", Country: "CIV", DialCode: "225", PawaPayCode: "MTN_MOMO_CIV",
		Endpoint: "mtn-ci", TokenField: "payment_token", WithdrawMode: "mtn-ci",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"mtn_ci_customer_fullname": name, "mtn_ci_email": email, "mtn_ci_phone_number": phone, "mtn_ci_wallet_provider": "MTNCI", "payment_token": token}
		},
	},
	{
		Label: "Moov Money", Provider: "MOOV_CI", Country: "CIV", DialCode: "225",
		Endpoint: "moov-ci", TokenField: "payment_token", WithdrawMode: "moov-ci",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"moov_ci_customer_fullname": name, "moov_ci_email": email, "moov_ci_phone_number": phone, "payment_token": token}
		},
	},
	{
		Label: "Wave", Provider: "WAVE_CI", Country: "CIV", DialCode: "225", PawaPayCode: "WAVE_CIV",
		Endpoint: "wave-ci", TokenField: "wave_ci_payment_token", WithdrawMode: "wave-ci",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"wave_ci_fullName": name, "wave_ci_email": email, "wave_ci_phone": phone, "wave_ci_payment_token": token}
		},
	},
	{
		Label: "Djamo", Provider: "DJAMO_CI", Country: "CIV", DialCode: "225",
		Endpoint: "djamo", TokenField: "djamo_payment_token", WithdrawMode: "djamo-ci",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"djamo_fullName": name, "djamo_email": email, "djamo_phone": phone, "code_country": "ci", "djamo_payment_token": token}
		},
	},
	// Togo
	{
		Label: "T-Money", Provider: "TMONEY_TG", Country: "TGO", DialCode: "228",
		Endpoint: "t-money-togo", TokenField: "payment_token", WithdrawMode: "t-money-togo",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"name_t_money": name, "email_t_money": email, "phone_t_money": phone, "payment_token": token}
		},
	},
	// Mali
	{
		Label: "Orange Money", Provider: "ORANGE_ML", Country: "MLI", DialCode: "223",
		Endpoint: "orange-money-mali", TokenField: "payment_token", WithdrawMode: "orange-money-mali",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"orange_money_mali_customer_fullname": name, "orange_money_mali_email": email, "orange_money_mali_phone_number": phone, "payment_token": token}
		},
	},
	{
		Label: "Moov Money", Provider: "MOOV_ML", Country: "MLI", DialCode: "223",
		Endpoint: "moov-mali", TokenField: "payment_token", WithdrawMode: "moov-mali",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"moov_ml_customer_fullname": name, "moov_ml_email": email, "moov_ml_phone_number": phone, "payment_token": token}
		},
	},
	// Burkina Faso — Orange BFA exclu du DÉPÔT : même contrainte OTP
	// qu'Orange CI (voir orange-money-burkina).
	{
		Label: "Moov Money", Provider: "MOOV_BF", Country: "BFA", DialCode: "226", PawaPayCode: "MOOV_BFA",
		Endpoint: "moov-burkina", TokenField: "moov_burkina_faso_payment_token", WithdrawMode: "moov-burkina-faso",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"moov_burkina_faso_fullName": name, "moov_burkina_faso_email": email, "moov_burkina_faso_phone_number": phone, "moov_burkina_faso_payment_token": token}
		},
	},
	// Cameroun
	{
		Label: "MTN MoMo", Provider: "MTN_CM", Country: "CMR", DialCode: "237", PawaPayCode: "MTN_MOMO_CMR",
		Endpoint: "mtn-cameroun", TokenField: "payment_token", WithdrawMode: "mtn-cameroun",
		BuildPayload: func(name, email, phone, token string) map[string]interface{} {
			return map[string]interface{}{"mtn_cameroun_customer_fullname": name, "mtn_cameroun_email": email, "mtn_cameroun_phone_number": phone, "mtn_cameroun_wallet_provider": "MTNCAMEROUN", "payment_token": token}
		},
	},
}

// FindPayDunyaOperator retrouve un opérateur par son code Provider.
func FindPayDunyaOperator(provider string) (PayDunyaOperator, bool) {
	for _, op := range PayDunyaOperators {
		if op.Provider == provider {
			return op, true
		}
	}
	return PayDunyaOperator{}, false
}

// FindPayDunyaOperatorByPawaPayCode retrouve l'équivalent PayDunya d'un
// opérateur identifié par son code PawaPay (voir PawaPayOperator ci-dessus)
// — utilisé pour le versement vendeur : le vendeur choisit son "moyen de
// retrait" en codes PawaPay (voir PAYOUT_COUNTRIES côté frontend), quel que
// soit le prestataire réellement routé pour l'opérateur (choix admin, voir
// AdminHandler.initiatePayoutWithProvider). Un opérateur sans équivalent
// PayDunya (PawaPayCode vide) ne matche jamais.
func FindPayDunyaOperatorByPawaPayCode(pawaPayCode string) (PayDunyaOperator, bool) {
	if pawaPayCode == "" {
		return PayDunyaOperator{}, false
	}
	for _, op := range PayDunyaOperators {
		if op.PawaPayCode == pawaPayCode {
			return op, true
		}
	}
	return PayDunyaOperator{}, false
}

// PayDunyaCountryCurrency — devise de facturation par pays (PayDunya facture
// dans la devise locale, comme PawaPay). Miroir de CountryCurrency
// (pawapay.go) pour les pays couverts ici, XOF pour la zone UEMOA + Cameroun
// en XAF.
var PayDunyaCountryCurrency = map[string]string{
	"SEN": "XOF", "BEN": "XOF", "CIV": "XOF", "TGO": "XOF", "MLI": "XOF", "BFA": "XOF",
	"CMR": "XAF",
}

// PayDunyaCountries — codes ISO3 couverts par PayDunya (clés de
// PayDunyaCountryCurrency), pour les points d'appel qui ont juste besoin de
// la liste (voir SaleHandler.CheckoutConfig).
func PayDunyaCountries() []string {
	out := make([]string, 0, len(PayDunyaCountryCurrency))
	for iso3 := range PayDunyaCountryCurrency {
		out = append(out, iso3)
	}
	return out
}
