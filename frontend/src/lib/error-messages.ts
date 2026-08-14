// Traduit les codes d'erreur bruts renvoyés par l'API (ex: "email_required")
// en messages clairs pour l'utilisateur. Ne JAMAIS afficher err.message brut
// dans l'UI — toujours passer par friendlyError().
const ERROR_MESSAGES: Record<string, string> = {
  // Auth
  invalid_credentials: 'Email ou mot de passe incorrect.',
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

  // Versements
  invalid_amount: 'Montant invalide.',
  amount_below_minimum: 'Le montant minimum de versement est de 1 000 FCFA.',
  insufficient_balance: 'Solde disponible insuffisant pour ce montant.',
  payout_rejected: 'Le versement a été refusé par votre opérateur mobile money.',
  payout_method_required: 'Merci d\'enregistrer votre moyen de versement avant de demander un retrait.',

  // Générique
  invalid_request: 'Requête invalide, vérifiez les informations saisies.',
  unauthorized: 'Merci de vous connecter pour continuer.',
};

export function friendlyError(err: unknown): string {
  const raw = err instanceof Error ? err.message : String(err);
  return ERROR_MESSAGES[raw] || 'Une erreur est survenue. Merci de réessayer.';
}
