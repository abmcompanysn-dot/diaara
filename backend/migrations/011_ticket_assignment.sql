-- Prise en charge d'un ticket support par un agent admin : assigned_admin_id
-- reste NULL tant que personne n'a réclamé le ticket ; le premier agent qui
-- clique "Prendre en charge" le fixe (via un UPDATE atomique côté repo), ce
-- qui empêche deux agents de traiter la même conversation en double.
ALTER TABLE support_tickets ADD COLUMN assigned_admin_id UUID REFERENCES users(id);
ALTER TABLE support_tickets ADD COLUMN claimed_at TIMESTAMPTZ;

-- Temps réel : notification à chaque nouveau ticket ou changement de statut/
-- prise en charge, pour que tous les agents admin connectés voient l'état à
-- jour sans recharger (même mécanisme que sales_channel/moderation_channel).
CREATE OR REPLACE FUNCTION notify_ticket_change() RETURNS TRIGGER AS $$
BEGIN
  PERFORM pg_notify('support_channel', row_to_json(NEW)::text);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ticket_notify
AFTER INSERT OR UPDATE ON support_tickets
FOR EACH ROW EXECUTE FUNCTION notify_ticket_change();

-- Notification aussi sur les nouveaux messages (pour rafraîchir une
-- conversation ouverte côté agent sans repoll).
CREATE OR REPLACE FUNCTION notify_ticket_message() RETURNS TRIGGER AS $$
BEGIN
  PERFORM pg_notify('support_channel', row_to_json(NEW)::text);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ticket_message_notify
AFTER INSERT ON ticket_messages
FOR EACH ROW EXECUTE FUNCTION notify_ticket_message();
