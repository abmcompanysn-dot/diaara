package service

// PlatformFeePct est la commission plateforme DIARRA (15%).
const PlatformFeePct = 15

type CommissionResult struct {
	PlatformFeeCFA  int
	VendorAmountCFA int
}

// CommissionService calcule la répartition des montants de vente.
type CommissionService struct{}

func NewCommissionService() *CommissionService {
	return &CommissionService{}
}

// Calculate répartit le montant d'une vente :
// 15% plateforme, 85% vendeur. (Closer = V2.)
func (s *CommissionService) Calculate(amountCFA int) CommissionResult {
	platformFee := amountCFA * PlatformFeePct / 100
	return CommissionResult{
		PlatformFeeCFA:  platformFee,
		VendorAmountCFA: amountCFA - platformFee,
	}
}
