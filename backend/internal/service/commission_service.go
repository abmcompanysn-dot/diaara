package service

// DefaultPlatformFeePct est la commission plateforme par défaut (15%),
// utilisée si aucun réglage "commission_rate_pct" n'est configuré.
const DefaultPlatformFeePct = 15

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

// Calculate répartit le montant d'une vente sans closer selon platformFeePct.
func (s *CommissionService) Calculate(amountCFA int, platformFeePct float64) CommissionResult {
	platformFee := int(float64(amountCFA) * platformFeePct / 100.0)
	return CommissionResult{
		PlatformFeeCFA:  platformFee,
		VendorAmountCFA: amountCFA - platformFee,
	}
}

// CalculateWithCloser répartit le montant d'une vente réalisée via un lien
// d'affiliation : platformFeePct% plateforme, closerPct% au closer, le reste au vendeur.
func (s *CommissionService) CalculateWithCloser(amountCFA int, platformFeePct, closerPct float64) CommissionResult {
	platformFee := int(float64(amountCFA) * platformFeePct / 100.0)
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
