package service

// PlatformFeePct est la commission plateforme DIARRA (15%).
const PlatformFeePct = 15

type CommissionResult struct {
	PlatformFeeCFA      int
	VendorAmountCFA     int
	CloserCommissionCFA int
}

// CommissionService calcule la répartition des montants de vente.
type CommissionService struct{}

func NewCommissionService() *CommissionService {
	return &CommissionService{}
}

// Calculate répartit le montant d'une vente sans closer :
// 15% plateforme, 85% vendeur.
func (s *CommissionService) Calculate(amountCFA int) CommissionResult {
	platformFee := amountCFA * PlatformFeePct / 100
	return CommissionResult{
		PlatformFeeCFA:  platformFee,
		VendorAmountCFA: amountCFA - platformFee,
	}
}

// CalculateWithCloser répartit le montant d'une vente réalisée via un lien
// d'affiliation : 15% plateforme, closerPct% au closer, le reste au vendeur.
func (s *CommissionService) CalculateWithCloser(amountCFA int, closerPct float64) CommissionResult {
	platformFee := amountCFA * PlatformFeePct / 100
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
