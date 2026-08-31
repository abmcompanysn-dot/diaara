package payment

import "context"

// PaymentProvider — plus petit dénominateur commun entre PawaPay et KPay,
// pour les points d'appel où les deux convergent naturellement (statut,
// versement, remboursement). L'INITIATION d'un paiement reste branchée
// explicitement dans les handlers (CreatePaymentPage vs InitiatePayment ont
// des formes trop différentes pour être unifiées sans perdre en clarté) —
// voir resolveDepositProvider dans sale_handler.go.
//
// Implémentée par pawaPayAdapter/kpayAdapter (ci-dessous), pas directement
// par *PawaPayClient/*KPayClient : ces derniers ont déjà des méthodes
// nommées GetDepositStatus/InitiatePayout/etc. avec les signatures brutes
// de chaque API — un type adaptateur séparé évite toute collision et garde
// les méthodes brutes disponibles là où leur forme complète est nécessaire
// (ex. CreatePaymentPage, qui n'a pas d'équivalent commun).
//
// Statuts normalisés en un petit vocabulaire commun (pending|processing|
// completed|failed|cancelled) pour que webhook_handler.go et
// payout_handler.go n'aient plus à connaître le vocabulaire propre à
// chaque provider (PawaPay: ACCEPTED/COMPLETED/FAILED vs KPay: PENDING/
// PROCESSING/COMPLETED/FAILED/CANCELLED).
var (
	_ PaymentProvider = pawaPayAdapter{}
	_ PaymentProvider = kpayAdapter{}
)

type PaymentProvider interface {
	Name() string // "pawapay" | "kpay"

	GetDepositStatus(ctx context.Context, ref string) (DepositOutcome, error)
	InitiatePayout(ctx context.Context, req PayoutOp) (PayoutInitOutcome, error)
	GetPayoutStatus(ctx context.Context, ref string) (PayoutOutcome, error)
	InitiateRefund(ctx context.Context, depositRef, refundExternalID string) (RefundInitOutcome, error)
}

type DepositOutcome struct {
	Status        string // pending|processing|completed|failed|cancelled
	FailureReason string
}

type PayoutOp struct {
	Phone, Operator, Amount, Currency, ClientRef string
}

type PayoutInitOutcome struct {
	Accepted      bool
	ProviderRef   string
	FailureReason string
}

type PayoutOutcome struct {
	Status        string
	FailureReason string
}

type RefundInitOutcome struct {
	Accepted      bool
	ProviderRef   string
	FailureReason string
}

// --- Adaptateur PawaPay ---------------------------------------------------

type pawaPayAdapter struct{ *PawaPayClient }

// AsProvider expose ce client via l'interface PaymentProvider commune.
func (c *PawaPayClient) AsProvider() PaymentProvider { return pawaPayAdapter{c} }

func (pawaPayAdapter) Name() string { return "pawapay" }

func normalizePawaPayStatus(s string) string {
	switch s {
	case "ACCEPTED", "ENQUEUED":
		return "pending"
	case "PROCESSING", "IN_RECONCILIATION":
		return "processing"
	case "COMPLETED":
		return "completed"
	case "FAILED":
		return "failed"
	default:
		return "pending"
	}
}

func (a pawaPayAdapter) GetDepositStatus(ctx context.Context, depositID string) (DepositOutcome, error) {
	resp, err := a.PawaPayClient.GetDepositStatus(ctx, depositID)
	if err != nil {
		return DepositOutcome{}, err
	}
	if resp.Data == nil {
		return DepositOutcome{Status: "pending"}, nil
	}
	out := DepositOutcome{Status: normalizePawaPayStatus(resp.Data.Status)}
	if resp.Data.FailureReason != nil {
		out.FailureReason = resp.Data.FailureReason.FailureMessage
	}
	return out, nil
}

func (a pawaPayAdapter) InitiatePayout(ctx context.Context, op PayoutOp) (PayoutInitOutcome, error) {
	resp, err := a.PawaPayClient.InitiatePayout(ctx, PayoutRequest{
		PayoutId: op.ClientRef,
		Recipient: Payer{
			Type:           "MMO",
			AccountDetails: AccountDetails{PhoneNumber: op.Phone, Provider: op.Operator},
		},
		Amount:            op.Amount,
		Currency:          op.Currency,
		ClientReferenceId: op.ClientRef,
		CustomerMessage:   "VERSEMENT DIARRA",
	})
	if err != nil {
		return PayoutInitOutcome{}, err
	}
	out := PayoutInitOutcome{Accepted: resp.Status == "ACCEPTED", ProviderRef: resp.PayoutId}
	if resp.FailureReason != nil {
		out.FailureReason = resp.FailureReason.FailureCode
	}
	return out, nil
}

func (a pawaPayAdapter) GetPayoutStatus(ctx context.Context, payoutID string) (PayoutOutcome, error) {
	resp, err := a.PawaPayClient.GetPayoutStatus(ctx, payoutID)
	if err != nil {
		return PayoutOutcome{}, err
	}
	if resp.Data == nil {
		return PayoutOutcome{Status: "pending"}, nil
	}
	out := PayoutOutcome{Status: normalizePawaPayStatus(resp.Data.Status)}
	if resp.Data.FailureReason != nil {
		out.FailureReason = resp.Data.FailureReason.FailureMessage
	}
	return out, nil
}

func (a pawaPayAdapter) InitiateRefund(ctx context.Context, depositID, refundExternalID string) (RefundInitOutcome, error) {
	resp, err := a.PawaPayClient.InitiateRefund(ctx, RefundRequest{
		RefundId:  refundExternalID,
		DepositId: depositID,
	})
	if err != nil {
		return RefundInitOutcome{}, err
	}
	out := RefundInitOutcome{Accepted: resp.Status == "ACCEPTED", ProviderRef: resp.RefundId}
	if resp.FailureReason != nil {
		out.FailureReason = resp.FailureReason.FailureCode
	}
	return out, nil
}

// --- Adaptateur KPay -------------------------------------------------------

type kpayAdapter struct{ *KPayClient }

// AsProvider expose ce client via l'interface PaymentProvider commune.
func (c *KPayClient) AsProvider() PaymentProvider { return kpayAdapter{c} }

func (kpayAdapter) Name() string { return "kpay" }

func normalizeKPayStatus(s string) string {
	switch s {
	case "PENDING":
		return "pending"
	case "PROCESSING":
		return "processing"
	case "COMPLETED":
		return "completed"
	case "FAILED":
		return "failed"
	case "CANCELLED":
		return "cancelled"
	default:
		return "pending"
	}
}

func (a kpayAdapter) GetDepositStatus(ctx context.Context, id string) (DepositOutcome, error) {
	resp, err := a.KPayClient.GetPaymentStatus(ctx, id)
	if err != nil {
		return DepositOutcome{}, err
	}
	return DepositOutcome{Status: normalizeKPayStatus(resp.Status), FailureReason: resp.FailureReason}, nil
}

func (a kpayAdapter) InitiatePayout(ctx context.Context, op PayoutOp) (PayoutInitOutcome, error) {
	resp, err := a.KPayClient.InitiatePayout(ctx, PayoutInitRequest{
		Amount:      op.Amount,
		Provider:    op.Operator,
		PhoneNumber: op.Phone,
		ExternalId:  op.ClientRef,
		Description: "Versement DIARRA",
	})
	if err != nil {
		return PayoutInitOutcome{}, err
	}
	return PayoutInitOutcome{Accepted: true, ProviderRef: resp.ID}, nil
}

func (a kpayAdapter) GetPayoutStatus(ctx context.Context, id string) (PayoutOutcome, error) {
	resp, err := a.KPayClient.GetPayoutStatus(ctx, id)
	if err != nil {
		return PayoutOutcome{}, err
	}
	return PayoutOutcome{Status: normalizeKPayStatus(resp.Status), FailureReason: resp.FailureReason}, nil
}

func (a kpayAdapter) InitiateRefund(ctx context.Context, paymentID, refundExternalID string) (RefundInitOutcome, error) {
	resp, err := a.KPayClient.InitiateRefund(ctx, paymentID, RefundInitRequest{ExternalId: refundExternalID})
	if err != nil {
		return RefundInitOutcome{}, err
	}
	return RefundInitOutcome{Accepted: true, ProviderRef: resp.ID}, nil
}
