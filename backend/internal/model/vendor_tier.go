package model

// Paliers de trophée vendeur, basés sur le chiffre d'affaires cumulé depuis
// toujours (somme des vendor_amount_cfa des ventes payées/livrées). Affiché
// comme repère de confiance pour les acheteurs sur la boutique publique, et
// comme progression pour le vendeur sur son tableau de bord.
const (
	VendorTierBronzeCFA  = 50_000
	VendorTierGoldCFA    = 500_000
	VendorTierDiamondCFA = 5_000_000
)

// VendorTier retourne le palier atteint ("bronze", "or", "diamant") ou ""
// si le vendeur n'a pas encore atteint le premier palier.
func VendorTier(totalEarnedCFA int) string {
	switch {
	case totalEarnedCFA >= VendorTierDiamondCFA:
		return "diamant"
	case totalEarnedCFA >= VendorTierGoldCFA:
		return "or"
	case totalEarnedCFA >= VendorTierBronzeCFA:
		return "bronze"
	default:
		return ""
	}
}
