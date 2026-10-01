package handler

import "testing"

func TestSplitAmount(t *testing.T) {
	commission, budget := splitAmount(10000, 20)
	if commission != 2000 || budget != 8000 {
		t.Fatalf("10000 à 20%% -> %d / %d", commission, budget)
	}
	// Commission arrondie au FCFA supérieur, jamais de budget gonflé.
	commission, budget = splitAmount(10001, 20)
	if commission != 2001 || budget != 8000 || commission+budget != 10001 {
		t.Fatalf("10001 à 20%% -> %d / %d", commission, budget)
	}
}

func TestMinAmountFor(t *testing.T) {
	// 7 jours × 1000 FCFA/jour de budget pub, 20 % de commission -> 8750 payés.
	min := minAmountFor(7, 1000, 20)
	if min != 8750 {
		t.Fatalf("min = %d", min)
	}
	if _, budget := splitAmount(min, 20); budget < 7*1000 {
		t.Fatalf("le minimum ne couvre pas le budget journalier : %d", budget)
	}
}
