-- Migration 041 : yes_session_id n'est plus UNIQUE côté DIARRA.
--
-- YES Business est idempotent sur session/initiate (voir doc : "Ouvre (ou
-- retrouve, si idempotent) une session de vente") — le même acheteur+vendeur
-- +produit peut recevoir le MÊME yes_session_id sur plusieurs appels. Chaque
-- micro-ticket DIARRA garde sa propre ligne conversational_sessions (son
-- propre paiement, sa propre traçabilité), mais peut légitimement pointer
-- vers le même yes_session_id/chat_url qu'une session précédente.
--
-- Découvert en prod : "duplicate key value violates unique constraint
-- conversational_sessions_yes_session_id_key" bloquait
-- OpenSessionAfterPayment dès qu'un acheteur retestait un micro-ticket sur
-- le même vendeur/produit — la session YES existante était bien renvoyée
-- (200, chat_url valide) mais jamais persistée côté DIARRA.
ALTER TABLE conversational_sessions DROP CONSTRAINT conversational_sessions_yes_session_id_key;
CREATE INDEX idx_conv_sessions_yes_session_id ON conversational_sessions (yes_session_id);
