-- Conversations email avec les vendeurs depuis atekossibrunel@diarra.app
-- (demandé le 2026-10-05). Objectif : garder une boucle de discussion par
-- vendeur pour comprendre leurs besoins et améliorer DIARRA. AUCUN envoi
-- n'est automatique : chaque message sortant est un brouillon ('draft')
-- jusqu'à validation explicite par l'admin dans /admin/vendor-mail, qui le
-- fait passer à 'approved' puis 'sent' une fois réellement envoyé via SMTP
-- (Postfix, déjà en place sur le VPS). Les messages entrants (réponses des
-- vendeurs, lus par IMAP sur la même boîte) sont enregistrés directement en
-- 'received', ils ne transitent jamais par la file de validation.

CREATE TABLE IF NOT EXISTS vendor_mail_threads (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vendor_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  subject TEXT NOT NULL,
  -- dernier Message-ID connu dans ce fil, pour que les réponses IMAP soient
  -- rattachées au bon fil via In-Reply-To/References plutôt que par sujet
  -- (le sujet peut être modifié par le client mail du vendeur).
  last_message_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (vendor_id)
);

CREATE TABLE IF NOT EXISTS vendor_mail_messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  thread_id UUID NOT NULL REFERENCES vendor_mail_threads(id) ON DELETE CASCADE,
  -- 'outbound' : rédigé pour être envoyé au vendeur (passe par draft/approved/sent).
  -- 'inbound'  : reçu du vendeur via IMAP (toujours 'received', jamais modéré).
  direction TEXT NOT NULL CHECK (direction IN ('outbound', 'inbound')),
  status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'approved', 'sent', 'received', 'rejected')),
  subject TEXT NOT NULL,
  body TEXT NOT NULL,
  -- Message-ID RFC 5322 de cet email (sortant : généré à l'envoi ; entrant :
  -- repris du header Message-ID source) — sert à éviter les doublons lors
  -- d'un nouveau passage IMAP et à chaîner In-Reply-To.
  message_id TEXT,
  in_reply_to TEXT,
  approved_by UUID REFERENCES users(id),
  approved_at TIMESTAMPTZ,
  sent_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_vendor_mail_messages_thread ON vendor_mail_messages(thread_id, created_at);
CREATE INDEX IF NOT EXISTS idx_vendor_mail_messages_status ON vendor_mail_messages(status) WHERE status = 'draft';
CREATE UNIQUE INDEX IF NOT EXISTS idx_vendor_mail_messages_message_id ON vendor_mail_messages(message_id) WHERE message_id IS NOT NULL;

-- Dernier UID IMAP traité sur la boîte atekossibrunel@diarra.app, pour que
-- le job de récupération ne retraite pas les mêmes emails à chaque passage
-- (IMAP UID est strictement croissant au sein d'un dossier donné).
CREATE TABLE IF NOT EXISTS vendor_mail_imap_state (
  mailbox TEXT PRIMARY KEY,
  last_uid BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
