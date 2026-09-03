package payment

import (
	"context"
	"fmt"
)

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
	_ PaymentProvider = paypalAdapter{}
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
	// UpdatedProviderRef — non vide seulement quand l'identifiant à utiliser
	// pour une action ultérieure (remboursement) a changé depuis l'appel
	// (PayPal uniquement : l'order ID sert au statut/à la capture, mais le
	// remboursement exige l'ID de CAPTURE, obtenu seulement une fois
	// capturée). L'appelant doit alors persister cette valeur via
	// SetProviderTransactionID — voir paypalAdapter.GetDepositStatus.
	UpdatedProviderRef string
}

type PayoutOp struct {
	Phone, Operator, Amount, Currency, ClientRef string
	// Email + AmountXOF : renseignés uniquement pour un versement PayPal
	// (recipient EMAIL, montant converti en USD par l'adaptateur). Ignorés par
	// les adaptateurs mobile money, qui utilisent Phone/Operator/Amount.
	Email     string
	AmountXOF int
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

// --- Adaptateur PayPal -------------------------------------------------------
//
// PayPal sert le dépôt carte/PayPal + son remboursement, ET depuis le
// 2026-09-03 les versements vendeur vers un email PayPal (Payouts API v1).
// InitiatePayout attend op.Email (destinataire) et op.AmountXOF (montant FCFA,
// converti en USD par SendPayout) ; op.ClientRef = ID du versement DIARRA.
// GetPayoutStatus attend le payout_batch_id renvoyé à l'initiation.

type paypalAdapter struct{ *PayPalClient }

// AsProvider expose ce client via l'interface PaymentProvider commune.
func (c *PayPalClient) AsProvider() PaymentProvider { return paypalAdapter{c} }

func (paypalAdapter) Name() string { return "paypal" }

func normalizePayPalStatus(s string) string {
	switch s {
	case "CREATED", "SAVED", "PAYER_ACTION_REQUIRED":
		return "pending"
	case "APPROVED":
		return "processing"
	case "COMPLETED":
		return "completed"
	case "VOIDED":
		return "failed"
	default:
		return "pending"
	}
}

// GetDepositStatus — si la commande est APPROVED (l'acheteur a validé côté
// PayPal mais l'argent n'a pas encore été encaissé), on la capture ici même :
// c'est ce même appel qui sert au polling depuis checkout/return (voir
// SaleHandler.CheckoutStatus), donc c'est le point naturel où déclencher la
// capture. CaptureOrder tolère un appel répété (statut déjà COMPLETED côté
// PayPal), donc pas de risque de double-capture au polling.
func (a paypalAdapter) GetDepositStatus(ctx context.Context, orderID string) (DepositOutcome, error) {
	status, err := a.PayPalClient.GetOrder(ctx, orderID)
	if err != nil {
		return DepositOutcome{}, err
	}
	if status.Status == "APPROVED" {
		captured, err := a.PayPalClient.CaptureOrder(ctx, orderID)
		if err != nil {
			return DepositOutcome{}, err
		}
		status = captured
	}
	out := DepositOutcome{Status: normalizePayPalStatus(status.Status)}
	if status.CaptureID != "" {
		out.UpdatedProviderRef = status.CaptureID
	}
	return out, nil
}

func normalizePayPalPayoutStatus(batchStatus, itemStatus string) string {
	// L'item prime quand il est concluant ; sinon on retombe sur le lot.
	switch itemStatus {
	case "SUCCESS":
		return "completed"
	case "FAILED", "RETURNED", "BLOCKED", "REFUNDED", "REVERSED":
		return "failed"
	case "UNCLAIMED", "ONHOLD", "PENDING":
		return "processing"
	}
	switch batchStatus {
	case "SUCCESS":
		return "completed"
	case "DENIED", "CANCELED":
		return "failed"
	case "PENDING", "PROCESSING", "NEW":
		return "processing"
	default:
		return "processing"
	}
}

func (a paypalAdapter) InitiatePayout(ctx context.Context, op PayoutOp) (PayoutInitOutcome, error) {
	if op.Email == "" {
		return PayoutInitOutcome{}, fmt.Errorf("email PayPal destinataire manquant")
	}
	resp, err := a.PayPalClient.SendPayout(ctx, PayPalPayoutRequest{
		SenderBatchID: "diarra-" + op.ClientRef,
		SenderItemID:  op.ClientRef,
		ReceiverEmail: op.Email,
		AmountXOF:     op.AmountXOF,
		Note:          "Versement DIARRA",
	})
	if err != nil {
		return PayoutInitOutcome{}, err
	}
	// PayPal accepte le lot de façon asynchrone (PENDING/PROCESSING) : on
	// considère "accepté" dès qu'un batch ID est revenu ; le statut réel est
	// confirmé ensuite via GetPayoutStatus (webhook + job de réconciliation).
	return PayoutInitOutcome{
		Accepted:    resp.BatchID != "" && resp.BatchStatus != "DENIED",
		ProviderRef: resp.BatchID,
	}, nil
}

func (a paypalAdapter) GetPayoutStatus(ctx context.Context, batchID string) (PayoutOutcome, error) {
	st, err := a.PayPalClient.GetPayoutBatch(ctx, batchID)
	if err != nil {
		return PayoutOutcome{}, err
	}
	return PayoutOutcome{
		Status:        normalizePayPalPayoutStatus(st.BatchStatus, st.ItemStatus),
		FailureReason: st.Reason,
	}, nil
}

// InitiateRefund attend l'ID de CAPTURE PayPal (pas l'ID de commande) en
// premier paramètre — voir sale.ProviderCaptureID, rempli lors de la capture
// dans GetDepositStatus ci-dessus.
func (a paypalAdapter) InitiateRefund(ctx context.Context, captureID, refundExternalID string) (RefundInitOutcome, error) {
	resp, err := a.PayPalClient.RefundCapture(ctx, captureID)
	if err != nil {
		return RefundInitOutcome{}, err
	}
	return RefundInitOutcome{Accepted: resp.Status == "COMPLETED", ProviderRef: resp.ID}, nil
}
