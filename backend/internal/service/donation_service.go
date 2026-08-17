package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"

	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/repository"
)

func uuidString() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

const (
	// Clés dans model (model.SettingDonationSharePct etc.) ; valeurs par
	// défaut si jamais réglées depuis l'admin.
	DefaultDonationSharePct     = 80.0
	DefaultDonationThresholdCFA = 100000.0
)

// DonationService pilote le programme de reversement automatique
// ("Fidélisation") : une part de la commission plateforme s'accumule dans
// une cagnotte partagée (voir DonationRepo.Accumulate, atomique en base —
// jamais tenue en mémoire, le backend tourne en plusieurs replicas) puis,
// au-delà d'un seuil configurable, est répartie à parts égales entre les
// destinataires actifs via un versement mobile money PawaPay.
type DonationService struct {
	repo             *repository.DonationRepo
	settingsRepo     *repository.SettingsRepo
	notificationRepo *repository.NotificationRepo
	userRepo         *repository.UserRepo
	pawapay          *payment.PawaPayClient
}

func NewDonationService(
	repo *repository.DonationRepo,
	settingsRepo *repository.SettingsRepo,
	notificationRepo *repository.NotificationRepo,
	userRepo *repository.UserRepo,
	pawapay *payment.PawaPayClient,
) *DonationService {
	return &DonationService{
		repo:             repo,
		settingsRepo:     settingsRepo,
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		pawapay:          pawapay,
	}
}

// Accumulate ajoute une part de la commission perçue sur une vente à la
// cagnotte, et déclenche une distribution si le seuil est franchi. Appelée
// en tâche de fond depuis WebhookHandler.PawaPayWebhook après confirmation
// d'un paiement — ne doit jamais faire échouer le webhook appelant, donc
// logge plutôt que de propager ses erreurs.
func (s *DonationService) Accumulate(ctx context.Context, platformFeeCFA int) {
	if platformFeeCFA <= 0 {
		return
	}
	sharePct := s.settingsRepo.GetFloat(ctx, model.SettingDonationSharePct, DefaultDonationSharePct)
	donationCFA := int(float64(platformFeeCFA) * sharePct / 100.0)
	if donationCFA <= 0 {
		return
	}

	newBalance, err := s.repo.Accumulate(ctx, donationCFA)
	if err != nil {
		log.Printf("donation: échec accumulation cagnotte: %v", err)
		return
	}

	if !s.settingsRepo.GetBool(ctx, model.SettingDonationEnabled, true) {
		return
	}
	threshold := int(s.settingsRepo.GetFloat(ctx, model.SettingDonationThresholdCFA, DefaultDonationThresholdCFA))
	if newBalance >= threshold && threshold > 0 {
		s.Distribute(ctx)
	}
}

// Distribute répartit le solde courant de la cagnotte à parts égales entre
// les destinataires actifs et déclenche un versement PawaPay pour chacun.
// Un échec d'envoi individuel ne bloque pas les autres destinataires ; le
// versement échoué reste visible dans l'historique admin avec un bouton
// "Réessayer" (voir AdminHandler-équivalent DonationHandler.RetryPayout).
func (s *DonationService) Distribute(ctx context.Context) {
	recipients, err := s.repo.ListActiveRecipients(ctx)
	if err != nil {
		log.Printf("donation: échec lecture destinataires: %v", err)
		return
	}
	if len(recipients) == 0 {
		// Pas de destinataire configuré : la cagnotte continue de grossir
		// jusqu'à ce qu'un admin en ajoute un.
		return
	}

	pool, err := s.repo.Pool(ctx)
	if err != nil || pool.BalanceCFA <= 0 {
		return
	}

	share := pool.BalanceCFA / len(recipients)
	if share <= 0 {
		return
	}
	// Le reliquat de la division entière reste dans la cagnotte plutôt que
	// d'être perdu ou réparti arbitrairement.
	totalDistributed := share * len(recipients)

	shares := make(map[string]int, len(recipients))
	for _, rec := range recipients {
		shares[rec.ID] = share
	}

	payouts, err := s.repo.DrainAndDistribute(ctx, totalDistributed, shares)
	if err != nil {
		log.Printf("donation: échec création des versements: %v", err)
		return
	}

	recipientByID := make(map[string]*model.DonationRecipient, len(recipients))
	for _, rec := range recipients {
		recipientByID[rec.ID] = rec
	}

	for _, payout := range payouts {
		rec := recipientByID[payout.RecipientID]
		if rec == nil {
			continue
		}
		s.sendPayout(ctx, payout, rec)
	}
}

// sendPayout appelle PawaPay pour un versement de don déjà créé en base
// (statut "requested") — même construction de requête que
// PayoutHandler.Create/AdminHandler.RetryPayout pour les versements vendeur.
func (s *DonationService) sendPayout(ctx context.Context, payout *model.DonationPayout, rec *model.DonationRecipient) {
	if s.pawapay == nil {
		reason := "payment_not_configured"
		s.repo.UpdatePayoutStatus(ctx, payout.ID, "failed", &reason)
		s.notifyAdmins(ctx, payout, rec, reason)
		return
	}

	pawapayID := uuidString()
	resp, err := s.pawapay.InitiatePayout(ctx, payment.PayoutRequest{
		PayoutId: pawapayID,
		Recipient: payment.Payer{
			Type: "MMO",
			AccountDetails: payment.AccountDetails{
				PhoneNumber: rec.PhoneNumber,
				Provider:    rec.Operator,
			},
		},
		Amount:            fmt.Sprintf("%d", payout.AmountCFA),
		Currency:          "XOF",
		ClientReferenceId: payout.ID,
		CustomerMessage:   "DIARRA FIDELISATION",
	})
	if err != nil || resp.Status != "ACCEPTED" {
		reason := "donation_payout_init_failed"
		if err == nil && resp.FailureReason != nil {
			reason = resp.FailureReason.FailureCode
		}
		s.repo.UpdatePayoutStatus(ctx, payout.ID, "failed", &reason)
		s.notifyAdmins(ctx, payout, rec, reason)
		return
	}
	if err := s.repo.SetPawaPayReference(ctx, payout.ID, pawapayID); err != nil {
		log.Printf("donation: échec enregistrement référence PawaPay: %v", err)
	}
}

// RetryPayout retente l'envoi d'un versement de don resté "échoué" — action
// admin explicite (bouton "Réessayer"), même mécanique que sendPayout.
func (s *DonationService) RetryPayout(ctx context.Context, payoutID string) error {
	if s.pawapay == nil {
		return fmt.Errorf("payment_not_configured")
	}
	payout, err := s.repo.FindPayoutByID(ctx, payoutID)
	if err != nil {
		return err
	}
	if payout.Status != "failed" {
		return fmt.Errorf("donation_payout_not_retryable")
	}
	rec, err := s.repo.FindRecipientByID(ctx, payout.RecipientID)
	if err != nil {
		return err
	}
	s.sendPayout(ctx, payout, rec)
	return nil
}

// notifyAdmins crée une notification in-app pour chaque administrateur
// quand un versement de don échoue — même mécanisme que les autres
// notifications (NotificationRepo), pas de canal dédié.
func (s *DonationService) notifyAdmins(ctx context.Context, payout *model.DonationPayout, rec *model.DonationRecipient, reason string) {
	if s.notificationRepo == nil || s.userRepo == nil {
		return
	}
	adminIDs, err := s.userRepo.ListAdminIDs(ctx)
	if err != nil {
		log.Printf("donation: échec notification admins: %v", err)
		return
	}
	title := "Versement de don échoué"
	body := fmt.Sprintf("Le versement de %d FCFA vers %s (%s) a échoué : %s.", payout.AmountCFA, rec.Name, rec.PhoneNumber, reason)
	for _, adminID := range adminIDs {
		s.notificationRepo.Create(ctx, adminID, "donation_payout_failed", title, body, "/admin/donations")
	}
}
