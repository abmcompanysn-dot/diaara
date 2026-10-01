// Traduit les codes d'erreur bruts renvoyés par l'API (ex: "email_required")
// en messages clairs pour l'utilisateur. Ne JAMAIS afficher err.message brut
// dans l'UI — toujours passer par friendlyError().
const ERROR_MESSAGES: Record<string, string> = {
  // Auth
  invalid_credentials: 'Email ou mot de passe incorrect.',
  // Webinaires YES Business (voir backend handler/webinar_handler.go).
  yes_not_configured: 'Yes.abmcy n’est pas configuré sur le serveur (clé API manquante).',
  yes_auth_failed: 'Yes.abmcy a refusé la clé API DIARRA (signature invalide). Vérifiez la configuration serveur.',
  yes_unavailable: 'Yes.abmcy est momentanément indisponible. Réessayez dans quelques minutes.',
  webinar_invalid: 'Yes.abmcy a refusé certains champs du webinaire. Vérifiez le formulaire.',
  webinar_not_found: 'Webinaire introuvable chez Yes.abmcy.',
  registration_not_found: 'Cette inscription est introuvable chez Yes.abmcy (supprimée ?).',
  webinar_state_conflict: 'Action impossible dans l’état actuel du webinaire (déjà démarré ou terminé ?).',
  title_required: 'Le titre est obligatoire.',
  invalid_start_date: 'Date de début invalide.',
  user_already_exists: 'Un compte existe déjà avec cet email ou ce numéro.',
  email_already_exists: 'Un compte existe déjà avec cet email.',
  invalid_role: 'Rôle invalide.',
  invalid_or_expired_token: 'Ce lien a expiré, merci de recommencer.',
  account_locked: 'Compte temporairement verrouillé suite à plusieurs tentatives. Réessayez plus tard.',
  email_not_verified: "Merci de vérifier votre email avant de continuer.",
  phone_not_verified: 'Merci de vérifier votre numéro de téléphone avant de continuer.',
  email_required: "L'email est requis.",
  password_too_short: 'Le mot de passe doit contenir au moins 8 caractères.',

  // Vérification par code (OTP)
  channel_and_code_required: 'Merci de saisir le code reçu.',
  channel_required: 'Canal de vérification manquant.',
  invalid_channel: 'Canal de vérification invalide.',
  invalid_or_expired_code: 'Ce code est invalide ou a expiré, réessayez ou demandez-en un nouveau.',
  too_many_attempts: 'Trop de tentatives, merci de demander un nouveau code.',
  verification_failed: 'La vérification a échoué, réessayez.',
  resend_too_soon: "Merci de patienter avant de redemander un code.",
  send_failed: "Impossible d'envoyer le code, réessayez.",

  // Checkout / commandes
  name_required: 'Merci d\'indiquer votre nom complet.',
  country_required: 'Merci de choisir votre pays.',
  email_required_for_guest_checkout: 'Merci d\'indiquer votre email pour recevoir votre commande.',
  payment_details_required: 'Merci de renseigner votre numéro et votre opérateur mobile money.',
  unsupported_operator: "Cet opérateur n'est pas disponible pour ce pays.",
  invalid_phone_number: 'Numéro de téléphone invalide.',
  invalid_phone: "Numéro de téléphone invalide (indicatif pays requis, ex : +221 77 123 45 67).",
  phone_already_used: 'Ce numéro est déjà associé à un autre compte DIARRA.',
  product_not_found: 'Produit introuvable.',
  product_not_available: "Ce produit n'est plus disponible à la vente.",
  referral_link_not_found: "Lien d'affiliation introuvable.",
  referral_link_mismatch: "Ce lien d'affiliation ne correspond pas à ce produit.",
  self_referral_forbidden: 'Vous ne pouvez pas utiliser votre propre lien.',
  order_creation_failed: 'Impossible de créer la commande, réessayez.',
  payment_init_failed: "Le paiement n'a pas pu être initié, réessayez.",
  payment_rejected: 'Le paiement a été refusé. Vérifiez votre solde et réessayez.',
  guest_account_creation_failed: 'Impossible de créer votre commande, réessayez.',

  // Produits
  title_price_category_file_required: 'Titre, prix, catégorie et fichier sont obligatoires.',
  invalid_affiliate_config: "Configuration d'affiliation invalide.",
  creation_failed: 'Échec de la création, réessayez.',
  update_failed: 'Échec de la mise à jour, réessayez.',
  forbidden: "Vous n'avez pas accès à cette action.",
  not_found: 'Introuvable.',
  product_in_use:
    'Impossible de supprimer ce produit : il a déjà des ventes, des liens d\'affiliation ou fait partie d\'un pack. Refusez la demande de suppression à la place.',

  // Versements
  invalid_amount: 'Montant invalide.',
  amount_below_minimum: 'Le montant minimum de versement est de 1 000 FCFA.',
  insufficient_balance: 'Solde disponible insuffisant pour ce montant.',
  payout_rejected: 'Le versement a été refusé par votre opérateur mobile money.',
  payout_method_required: 'Merci d\'enregistrer votre moyen de versement avant de demander un retrait.',

  // Remboursements (admin)
  sale_not_refundable: 'Cette vente ne peut pas être remboursée (déjà remboursée ou non payée).',
  manual_refund_required:
    "PayDunya n'a pas d'API de remboursement automatisé : effectuez ce remboursement manuellement depuis le dashboard PayDunya, puis marquez la vente comme remboursée ici si besoin.",
  missing_provider_transaction_id: 'Référence de transaction manquante, remboursement impossible automatiquement.',
  refund_rejected: 'Le remboursement a été refusé par le prestataire de paiement.',

  // Sponsorisation Facebook/Instagram (voir backend handler/ad_handler.go).
  ads_unavailable: 'La sponsorisation n’est pas encore disponible. Revenez bientôt !',
  product_not_approved: 'Ce produit doit être validé par DIARRA avant de pouvoir être sponsorisé.',
  invalid_duration: 'Durée invalide.',
  countries_required: 'Choisissez au moins un pays de diffusion.',
  message_too_long: 'Le texte de la pub est trop long (500 caractères maximum).',
  ad_amount_below_minimum: 'Montant trop faible pour cette durée.',
  ad_amount_above_maximum: 'Montant trop élevé.',
  ad_creation_failed: 'La sponsorisation n’a pas pu être enregistrée. Réessayez.',
  ad_launch_failed: 'Meta n’a pas pu créer la publicité. Le montant a été recrédité sur votre solde.',
  ad_not_running: 'Cette pub n’est pas en cours de diffusion.',
  ad_stop_failed: 'Impossible d’arrêter la pub chez Meta pour le moment.',
  invalid_ads_enabled: 'Valeur invalide pour l’activation des pubs.',
  invalid_ads_commission: 'La commission doit être comprise entre 0 et 90 %.',
  invalid_ads_min_daily: 'Le budget journalier minimum doit être d’au moins 100 FCFA.',

  // Générique
  invalid_request: 'Requête invalide, vérifiez les informations saisies.',
  unauthorized: 'Merci de vous connecter pour continuer.',
};

export function friendlyError(err: unknown): string {
  const raw = err instanceof Error ? err.message : String(err);
  return ERROR_MESSAGES[raw] || 'Une erreur est survenue. Merci de réessayer.';
}
