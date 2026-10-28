-- Live usernames keep the existing public namespace. Operational identities use
-- a full UUID namespace that no live signup (past or future) can have claimed.
ALTER TABLE users DROP CONSTRAINT users_username_check;
UPDATE users SET username='~deleted:'||id::text WHERE deleted_at IS NOT NULL;
ALTER TABLE users ADD CONSTRAINT users_username_check CHECK (
 (deleted_at IS NULL AND username ~ '^[a-z0-9_]{3,24}$') OR
 (deleted_at IS NOT NULL AND username='~deleted:'||id::text)
);
