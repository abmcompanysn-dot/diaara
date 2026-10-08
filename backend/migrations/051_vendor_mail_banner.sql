-- Bandeau visuel optionnel pour une campagne vendor-mail ponctuelle (ex:
-- Octobre Rose, 2026-10-08) — remplace le bandeau texte "DIARRA" par défaut
-- dans l'email HTML envoyé (voir vendormail.Sender.Send/renderHTML).
-- NULL = bandeau texte par défaut, comportement inchangé pour tous les
-- messages vendor-mail déjà existants.
ALTER TABLE vendor_mail_messages
  ADD COLUMN banner_url TEXT;
