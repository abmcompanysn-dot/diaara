-- Journal d'activité admin : trace qui a fait quoi (modération produit,
-- remboursement, versement, bannissement, changement de réglages...) pour
-- la traçabilité — utile dès qu'il y a plusieurs admins. Backoffice 360°,
-- volet "activité récente" (voir AdminHandler.ActivityLog).
--
-- admin_id nullable : certaines actions sont déclenchées automatiquement
-- (cron de réconciliation, ex. reconcileDepositsPass) sans admin humain —
-- pas question de les rattacher à un faux admin_id pour satisfaire une
-- contrainte NOT NULL.
CREATE TABLE admin_activity_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  admin_id UUID REFERENCES users(id) ON DELETE SET NULL,
  action TEXT NOT NULL, -- ex: "product_approved", "sale_refunded", "user_banned"
  target_type TEXT NOT NULL, -- ex: "product", "sale", "user", "payout", "settings"
  target_id TEXT, -- id de l'entité concernée (texte : pas toujours un UUID, ex. une clé de réglage)
  description TEXT NOT NULL, -- résumé lisible, ex. "Produit « X » approuvé"
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_admin_activity_log_created ON admin_activity_log(created_at DESC);
CREATE INDEX idx_admin_activity_log_admin ON admin_activity_log(admin_id);
CREATE INDEX idx_admin_activity_log_target ON admin_activity_log(target_type, target_id);
