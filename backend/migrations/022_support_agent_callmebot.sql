-- CallMeBot (callmebot.com) : chaque agent support peut enregistrer son
-- propre numéro WhatsApp + clé API personnelle CallMeBot, pour recevoir une
-- notification WhatsApp automatique (en plus de l'email) à chaque nouveau
-- contact. Les deux colonnes restent optionnelles : un agent sans clé
-- continue de recevoir uniquement l'email.
ALTER TABLE support_agents ADD COLUMN IF NOT EXISTS phone TEXT;
ALTER TABLE support_agents ADD COLUMN IF NOT EXISTS callmebot_apikey TEXT;
