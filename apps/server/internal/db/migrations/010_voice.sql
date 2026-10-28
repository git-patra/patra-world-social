-- Revocation journal deliberately has no cascading user/world foreign keys.
-- Only opaque media identities/rooms survive a crash; never token/audio/chat.
CREATE TABLE voice_revocations (
 identity text PRIMARY KEY,
 room text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
