package service

import "testing"

func TestCalculateNoCloser(t *testing.T) {
	c := NewCommissionService()
	res := c.Calculate(10000, DefaultPlatformFeePct)

	if res.PlatformFeeCFA != 1500 {
		t.Fatalf("PlatformFeeCFA = %d, want 1500", res.PlatformFeeCFA)
	}
	if res.VendorAmountCFA != 8500 {
		t.Fatalf("VendorAmountCFA = %d, want 8500", res.VendorAmountCFA)
	}
	if res.CloserCommissionCFA != 0 {
		t.Fatalf("CloserCommissionCFA = %d, want 0", res.CloserCommissionCFA)
	}
}

func TestCalculateWithCloser(t *testing.T) {
	c := NewCommissionService()

	// 10 000 FCFA, commission closer 20% => plateforme 1500, closer 2000, vendeur 6500.
	res := c.CalculateWithCloser(10000, DefaultPlatformFeePct, 20)
	if res.PlatformFeeCFA != 1500 {
		t.Fatalf("PlatformFeeCFA = %d, want 1500", res.PlatformFeeCFA)
	}
	if res.CloserCommissionCFA != 2000 {
		t.Fatalf("CloserCommissionCFA = %d, want 2000", res.CloserCommissionCFA)
	}
	if res.VendorAmountCFA != 6500 {
		t.Fatalf("VendorAmountCFA = %d, want 6500", res.VendorAmountCFA)
	}
}

func TestCalculateHighValueTier(t *testing.T) {
	c := NewCommissionService()

	// En dessous du seuil : taux normal (15%).
	belowAmount := HighValueThresholdCFA - 1
	below := c.Calculate(belowAmount, DefaultPlatformFeePct)
	wantBelow := int(float64(belowAmount) * DefaultPlatformFeePct / 100.0)
	if below.PlatformFeeCFA != wantBelow {
		t.Fatalf("below threshold: PlatformFeeCFA = %d, want %d", below.PlatformFeeCFA, wantBelow)
	}

	// À partir du seuil (1 000 000 FCFA) : taux réduit (10%).
	at := c.Calculate(HighValueThresholdCFA, DefaultPlatformFeePct)
	if at.PlatformFeeCFA != 100000 {
		t.Fatalf("at threshold: PlatformFeeCFA = %d, want 100000", at.PlatformFeeCFA)
	}
	if at.VendorAmountCFA != 900000 {
		t.Fatalf("at threshold: VendorAmountCFA = %d, want 900000", at.VendorAmountCFA)
	}

	// Au-dessus du seuil, avec un taux admin différent : le palier prime.
	above := c.Calculate(2_000_000, 20)
	if above.PlatformFeeCFA != 200000 {
		t.Fatalf("above threshold: PlatformFeeCFA = %d, want 200000", above.PlatformFeeCFA)
	}
}

func TestCalculateWithCloserCapped(t *testing.T) {
	c := NewCommissionService()

	// Une commission trop élevée ne doit jamais dépasser la part restante
	// après la plateforme.
	res := c.CalculateWithCloser(10000, DefaultPlatformFeePct, 90)
	if res.CloserCommissionCFA+res.PlatformFeeCFA > 10000 {
		t.Fatalf("closer+platform = %d, exceeds amount", res.CloserCommissionCFA+res.PlatformFeeCFA)
	}
	if res.VendorAmountCFA < 0 {
		t.Fatalf("VendorAmountCFA = %d, negative", res.VendorAmountCFA)
	}
	if res.PlatformFeeCFA+res.CloserCommissionCFA+res.VendorAmountCFA != 10000 {
		t.Fatalf("split %d+%d+%d != 10000",
			res.PlatformFeeCFA, res.CloserCommissionCFA, res.VendorAmountCFA)
	}
}
