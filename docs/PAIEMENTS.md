# Paiements — architecture, filets de sécurité, erreurs des prestataires

Document technique sur l'intégration des prestataires de paiement dans DIARRA :
comment un paiement est confirmé, ce qui se passe si un prestataire ne
répond jamais, et comment lire les erreurs qu'ils renvoient. Mis à jour le
2026-09-03 (ajout PayPal, suspension de KPay).

## 1. Les trois prestataires

| Prestataire | Rôle | Statut |
|---|---|---|
| **PawaPay** | Mobile money (achat + versements vendeur) | Actif, seul prestataire mobile money |
| **KPay** | Mobile money 12 pays + carte/PayPal | **Suspendu** depuis le 2026-09-03 (intégration jamais finalisée). Code laissé en place (`kpay.go`, adaptateur, handlers, webhooks) mais dormant — le client n'est plus instancié dans `main.go` (`kpay` reste `nil`). |
| **PayPal** | Carte bancaire + compte PayPal (achat) | Actif, remplace KPay pour ce flux |

Tous les trois implémentent l'interface commune `payment.PaymentProvider`
(`backend/internal/payment/provider.go`) : `GetDepositStatus`,
`InitiatePayout`, `GetPayoutStatus`, `InitiateRefund`. Cette interface existe
pour que `webhook_handler.go`/`payout_handler.go`/`admin_handler.go` n'aient
pas à connaître le vocabulaire de statut propre à chaque prestataire — voir
§3 pour ce vocabulaire brut et sa normalisation.

Qui traite quoi, pour un NOUVEAU paiement (`SaleHandler.resolveCheckoutProvider`) :
- `payment_method` = `card` ou `paypal` → **toujours PayPal**, si activé
  (voir `model.SettingCardPaymentEnabled`, réglage admin dans
  `/admin/settings`, carte "Paiement carte bancaire / PayPal"). Si désactivé
  ou si PayPal n'est pas configuré côté serveur, replié silencieusement sur
  PawaPay (mobile money) — jamais d'échec dur.
- `payment_method` = `mobile_money` → PawaPay (réglage par pays,
  `model.CheckoutProviderSettingKey`, KPay refusé à l'écriture depuis sa
  suspension).

## 2. Comment un paiement est confirmé

Trois mécanismes, du plus rapide au plus lent — tous actifs en parallèle,
aucun ne dépend des autres pour fonctionner :

### a) Webhook (temps réel)
Le prestataire pousse une notification HTTP dès que le statut change.

- **PawaPay** : `POST /api/webhooks/pawapay`. ⚠️ **L'URL de callback ne peut
  PAS être envoyée dans la requête de création de paiement** — l'API Payment
  Page (`POST /v2/paymentpage`) la rejette (`UNSUPPORTED_PARAMETER`, incident
  du 2026-09-03). Elle doit être configurée **une fois, dans le tableau de
  bord PawaPay** (Merchant Settings → Callback URL →
  `https://diarra.app/api/webhooks/pawapay`). Sans ce réglage dashboard,
  seuls les mécanismes b) et c) ci-dessous confirment les ventes.
- **PayPal** : `POST /api/webhooks/paypal`. Signature vérifiée en renvoyant
  les en-têtes `Paypal-Transmission-*` + le corps brut à l'API PayPal
  elle-même (`VerifyWebhookSignature`) — PayPal ne permet pas de vérifier une
  signature localement comme un HMAC classique. Événements écoutés :
  `CHECKOUT.ORDER.APPROVED`, `PAYMENT.CAPTURE.COMPLETED`,
  `PAYMENT.CAPTURE.REFUNDED`. Le webhook ID (obtenu en créant le webhook sur
  developer.paypal.com) va dans `PAYPAL_WEBHOOK_ID`.
- **KPay** : webhooks toujours présents dans le code (`KPayPaymentWebhook`,
  `KPayPayoutWebhook`, `KPayRefundWebhook`) mais inertes tant que `kpay` vaut
  `nil` dans `main.go`.

### b) Vérification au retour de l'acheteur (polling)
Après paiement, l'acheteur revient sur `/checkout/return?token=...`, qui
appelle `GET /api/orders/status` (`SaleHandler.CheckoutStatus`). Si la vente
est encore `pending`, ce endpoint interroge directement le prestataire
(`GetDepositStatus`) avant de répondre. **Spécifique à PayPal** : une
commande approuvée (`APPROVED`) mais pas encore encaissée est **capturée à
cet instant précis** (`paypalAdapter.GetDepositStatus` appelle `CaptureOrder`
si nécessaire) — c'est le point normal de capture pour ce prestataire, pas
seulement un affichage.

### c) Réconciliation en tâche de fond (filet de sécurité)
Si ni le webhook ni le retour de l'acheteur ne se sont produits (onglet fermé
avant redirection, webhook jamais reçu) :
- `WebhookHandler.RunDepositReconcileLoop` : toutes les 10 minutes, revérifie
  chaque vente PawaPay `pending` de moins de 3 jours directement via l'API.
- Bouton admin **"Vérifier chez PawaPay"** (`/admin/pending-sales`,
  `POST /api/admin/sales/{id}/check-provider`,
  `AdminHandler.CheckSaleProvider`) : la même vérification, déclenchable à la
  demande sur une vente précise. Fonctionne aussi pour PayPal (capture +
  confirmation identique au polling ci-dessus) ; renvoie une erreur explicite
  pour KPay (`provider_check_unsupported`), qui n'a pas d'endpoint de statut
  fiable documenté.

Pour les **versements vendeur** (argent qui sort, pas qui rentre), le même
principe existe côté `PayoutHandler`/`AdminHandler` :
`RunPayoutReconcileLoop` (10 min, versements `processing` de moins de 3
jours) + bouton admin équivalent.

## 3. Lire les erreurs des prestataires

Chaque prestataire a son propre vocabulaire de statut et de code d'erreur.
Deux niveaux existent dans le code :

**Niveau brut** (spécifique à chaque prestataire, visible dans les logs
serveur et en base) :

| Prestataire | Champ | Exemples rencontrés |
|---|---|---|
| PawaPay | `failureReason.failureCode` / `.failureMessage` | `UNSUPPORTED_PARAMETER`, `INVALID_PARAMETER`, `DEPOSITS_NOT_ALLOWED` |
| KPay | `failureReason` (texte libre) | — |
| PayPal | statut HTTP + corps JSON (`ErrPaymentFailed` enveloppe le message brut) | erreurs OAuth, `ORDER_ALREADY_CAPTURED` |

**Niveau normalisé** (`payment.DepositOutcome.Status`, commun aux trois via
l'interface `PaymentProvider`) : `pending` | `processing` | `completed` |
`failed` | `cancelled`. C'est ce niveau que lisent `webhook_handler.go`,
`sale_handler.go` (`CheckoutStatus`) et `admin_handler.go` pour décider si
une vente passe à `paid`/`failed`/`refunded`.

**Ce qui n'existe PAS encore** : le code d'erreur brut n'est pas affiché à
l'acheteur sur la page de checkout — il voit un message générique ("paiement
échoué, réessayez"). Le code brut n'est visible que dans les logs serveur
(`log.Printf("payment_init_failed ...")`) et, pour une vente déjà créée, dans
la réponse de `CheckSaleProvider` (champ `provider_status`). Si un jour on
veut afficher une raison plus précise à l'acheteur, c'est par là qu'il faut
passer.

## 4. Où regarder en cas d'incident

1. Logs backend : `docker logs diarra-backend-1 --tail 100 | grep -i
   payment_init_failed` (VPS : `ssh diarra-vps`).
2. Statut réel d'une vente précise : bouton "Vérifier chez PawaPay"
   (`/admin/pending-sales`) — fonctionne aussi pour PayPal.
3. Rejouer un webhook manuellement (PawaPay) :
   `curl -X POST https://diarra.app/api/webhooks/pawapay -H 'Content-Type: application/json' -d '{"depositId":"<payment_reference>"}'`
4. Couper le paiement carte/PayPal sans toucher au code : `/admin/settings`
   → carte "Paiement carte bancaire / PayPal" → Désactivé. Le mobile money
   (PawaPay) continue de fonctionner normalement.

Voir aussi `JOURNAL-MODIFICATIONS.md` pour l'historique daté de chaque
changement sur ces flux.
