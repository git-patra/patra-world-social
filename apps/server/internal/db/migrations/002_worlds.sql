CREATE TABLE IF NOT EXISTS worlds (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 64),
 owner_id uuid NOT NULL REFERENCES users(id),
 archived_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS world_members (
 world_id uuid NOT NULL REFERENCES worlds(id),
 user_id uuid NOT NULL REFERENCES users(id),
 role text NOT NULL CHECK (role IN ('owner','moderator','member')),
 active boolean NOT NULL DEFAULT true,
 joined_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(world_id,user_id)
);
CREATE INDEX IF NOT EXISTS world_members_user_active_idx ON world_members(user_id,world_id) WHERE active;
CREATE UNIQUE INDEX IF NOT EXISTS world_one_active_owner_idx ON world_members(world_id) WHERE active AND role='owner';
CREATE TABLE IF NOT EXISTS world_bans (
 world_id uuid NOT NULL REFERENCES worlds(id),
 user_id uuid NOT NULL REFERENCES users(id),
 banned_by uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(world_id,user_id)
);
CREATE TABLE IF NOT EXISTS world_invites (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 world_id uuid NOT NULL REFERENCES worlds(id),
 issuer_id uuid NOT NULL REFERENCES users(id),
 target_user_id uuid REFERENCES users(id),
 token_hash bytea NOT NULL UNIQUE,
 expires_at timestamptz NOT NULL,
 revoked_at timestamptz,
 redeemed_by uuid REFERENCES users(id),
 redeemed_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS world_invites_world_idx ON world_invites(world_id);
