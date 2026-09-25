package payment

// LogicalOperator représente un opérateur mobile money PHYSIQUE (ex "Wave
// Sénégal"), indépendamment du prestataire qui le traite finalement
// (PawaPay et/ou PayDunya). Le Code logique est stable et sert de clé pour
// le routage admin (voir model.OperatorRouteSettingKey) — c'est le code
// PawaPay quand il existe (déjà utilisé partout ailleurs : moyen de
// retrait vendeur, versements, historique), sinon un code PayDunya dédié
// pour un opérateur que seul PayDunya couvre.
//
// Un opérateur logique peut n'avoir qu'un seul prestataire réel
// (PawaPayCode ou PayDunyaCode vide) : la route admin reste alors figée sur
// l'unique prestataire disponible, quel que soit le réglage.
type LogicalOperator struct {
	Code         string // clé stable pour le routage admin — voir commentaire ci-dessus
	Label        string // affiché à l'acheteur, ex "Wave"
	Country      string // ISO 3166-1 alpha-3
	DialCode     string
	PawaPayCode  string // code XOFOperators, vide si PawaPay ne couvre pas cet opérateur
	PayDunyaCode string // code PayDunyaOperators, vide si PayDunya ne couvre pas cet opérateur
	// RequiresOTP — vrai si le prestataire PayDunya résolu pour cet
	// opérateur exige un code obtenu par l'acheteur AVANT de payer (ex
	// Orange Money CI/BFA — voir PayDunyaOperator.RequiresOTP). Un
	// opérateur avec PawaPayCode ET PayDunyaCode n'a cette contrainte que
	// si le routage résolu est effectivement "paydunya" — le frontend
	// n'affiche le champ OTP que dans ce cas (voir CheckoutConfig).
	RequiresOTP bool
}

// LogicalOperators fusionne XOFOperators (PawaPay) et PayDunyaOperators
// (PayDunya) en une liste unique par opérateur physique (pays + libellé),
// pour permettre au checkout de proposer UN SEUL bouton par opérateur et de
// router ensuite vers le bon prestataire (voir model.OperatorRouteSettingKey,
// ResolveOperatorProvider). Construite une fois à l'import.
var LogicalOperators = buildLogicalOperators()

func buildLogicalOperators() []LogicalOperator {
	out := make([]LogicalOperator, 0, len(XOFOperators))
	// PawaPay d'abord : donne le Code logique (déjà utilisé partout côté
	// vendeur/versement) et la position dans la liste. PawaPayCode n'est
	// rempli que si l'opérateur est réellement activé sur notre compte
	// (voir IsPawaPayOperatorActive) — sinon l'opérateur logique reste sans
	// PawaPayCode et retombe sur PayDunya s'il le couvre, ou disparaît du
	// checkout si aucun des deux ne le couvre réellement (constaté
	// 2026-09-25 : XOFOperators documente plus d'opérateurs que ce que
	// PawaPay a activé chez nous).
	for _, op := range XOFOperators {
		logical := LogicalOperator{
			Code: op.Provider, Label: op.Label, Country: op.Country, DialCode: op.DialCode,
		}
		if IsPawaPayOperatorActive(op.Provider) {
			logical.PawaPayCode = op.Provider
		}
		out = append(out, logical)
	}
	// PayDunya ensuite : complète un opérateur déjà présent (même Code
	// logique — PayDunyaOperator.PawaPayCode reste le code PawaPay
	// "canonique" même quand ce PawaPayCode n'est pas actif chez nous, donc
	// PAS out[i].PawaPayCode qui peut être vide, voir ci-dessus) ou en
	// ajoute un nouveau, PayDunya-only (ex Expresso, Djamo, T-Money,
	// Celtiis Cash, Mali/Togo entiers).
	for _, op := range PayDunyaOperators {
		if op.PawaPayCode != "" {
			for i := range out {
				if out[i].Code == op.PawaPayCode {
					out[i].PayDunyaCode = op.Provider
					out[i].RequiresOTP = op.RequiresOTP
					break
				}
			}
			continue
		}
		out = append(out, LogicalOperator{
			Code: op.Provider, Label: op.Label, Country: op.Country, DialCode: op.DialCode,
			PayDunyaCode: op.Provider, RequiresOTP: op.RequiresOTP,
		})
	}
	return out
}

// FindLogicalOperator retrouve un opérateur logique par son Code.
func FindLogicalOperator(code string) (LogicalOperator, bool) {
	for _, op := range LogicalOperators {
		if op.Code == code {
			return op, true
		}
	}
	return LogicalOperator{}, false
}

// AvailableProviders — prestataires réellement capables de traiter cet
// opérateur logique (un seul si PawaPay ou PayDunya seul le couvre).
func (o LogicalOperator) AvailableProviders() []string {
	var out []string
	if o.PawaPayCode != "" {
		out = append(out, "pawapay")
	}
	if o.PayDunyaCode != "" {
		out = append(out, "paydunya")
	}
	return out
}
