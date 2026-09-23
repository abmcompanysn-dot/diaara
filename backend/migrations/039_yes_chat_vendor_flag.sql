-- Migration 039 : achat conversationnel YES Business en bêta — réservé aux
-- vendeurs choisis par un admin (voir handler.YesHandler.OpenConversation,
-- qui refuse si le vendeur du produit n'a pas ce flag), pas ouvert à tous
-- les vendeurs dès le lancement.
ALTER TABLE users ADD COLUMN IF NOT EXISTS yes_chat_enabled BOOLEAN NOT NULL DEFAULT FALSE;
