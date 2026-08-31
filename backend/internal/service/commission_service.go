package service

// DefaultPlatformFeePct est la commission plateforme par défaut (15%),
// utilisée si aucun réglage "commission_rate_pct" n'est configuré.
const DefaultPlatformFeePct = 15

// HighValueThresholdCFA est le seuil à partir duquel la commission plateforme
// est réduite à HighValuePlatformFeePct, quel que soit le taux configuré.
const HighValueThresholdCFA = 1_000_000

// HighValuePlatformFeePct est la commission plateforme appliquée aux ventes
// dont le montant atteint HighValueThresholdCFA (10% au lieu de 15%).
const HighValuePlatformFeePct = 10

type CommissionResult struct {
	PlatformFeeCFA      int
	VendorAmountCFA     int
	CloserCommissionCFA int
}

// CommissionService calcule la répartition des montants de vente. Le taux
// plateforme est passé à chaque appel (configurable depuis l'admin via la
// table settings) plutôt que codé en dur, pour rester à jour sans redéploiement.
type CommissionService struct{}

func NewCommissionService() *CommissionService {
	return &CommissionService{}
}

// EffectivePlatformFeePct applique le palier de commission réduite : à partir
// de HighValueThresholdCFA, le taux plateforme passe à HighValuePlatformFeePct
// même si un taux différent est configuré par l'admin.
func EffectivePlatformFeePct(amountCFA int, baseRatePct float64) float64 {
	if amountCFA >= HighValueThresholdCFA {
		return HighValuePlatformFeePct
	}
	return baseRatePct
}

// Calculate répartit le montant d'une vente sans closer selon platformFeePct,
// avec application du palier de commission réduite au-delà de HighValueThresholdCFA.
func (s *CommissionService) Calculate(amountCFA int, platformFeePct float64) CommissionResult {
	rate := EffectivePlatformFeePct(amountCFA, platformFeePct)
	platformFee := int(float64(amountCFA) * rate / 100.0)
	return CommissionResult{
		PlatformFeeCFA:  platformFee,
		VendorAmountCFA: amountCFA - platformFee,
	}
}

// CalculateWithCloser répartit le montant d'une vente réalisée via un lien
// d'affiliation : platformFeePct% plateforme (réduit au palier au-delà de
// HighValueThresholdCFA), closerPct% au closer, le reste au vendeur.
func (s *CommissionService) CalculateWithCloser(amountCFA int, platformFeePct, closerPct float64) CommissionResult {
	rate := EffectivePlatformFeePct(amountCFA, platformFeePct)
	platformFee := int(float64(amountCFA) * rate / 100.0)
	closerFee := int(float64(amountCFA) * closerPct / 100.0)
	rest := amountCFA - platformFee
	if closerFee > rest {
		closerFee = rest
	}
	return CommissionResult{
		PlatformFeeCFA:      platformFee,
		CloserCommissionCFA: closerFee,
		VendorAmountCFA:     rest - closerFee,
	}
}
