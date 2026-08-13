-- Migration 005: permissions admin fines
-- Un admin sans ligne dans cette table garde l'accès complet (compat. avec
-- le compte admin existant). Un admin avec au moins une ligne est restreint
-- aux scopes listés.

CREATE TABLE IF NOT EXISTS admin_permissions (
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  permission TEXT NOT NULL CHECK (permission IN ('moderation','users','finance','infra')),
  granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, permission)
);
