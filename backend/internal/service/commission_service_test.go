package service

import "testing"

func TestCalculateNoCloser(t *testing.T) {
	c := NewCommissionService()
	res := c.Calculate(10000)

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
	res := c.CalculateWithCloser(10000, 20)
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

func TestCalculateWithCloserCapped(t *testing.T) {
	c := NewCommissionService()

	// Une commission trop élevée ne doit jamais dépasser la part restante
	// après la plateforme.
	res := c.CalculateWithCloser(10000, 90)
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
