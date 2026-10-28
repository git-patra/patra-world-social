CREATE TABLE friend_requests (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 actor_id uuid NOT NULL REFERENCES users(id),
 target_id uuid NOT NULL REFERENCES users(id),
 accepted boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT now(),
 CHECK(actor_id<>target_id)
);
CREATE UNIQUE INDEX friend_pair_idx ON friend_requests(least(actor_id,target_id),greatest(actor_id,target_id));
CREATE TABLE user_blocks (
 actor_id uuid NOT NULL REFERENCES users(id), target_id uuid NOT NULL REFERENCES users(id),
 PRIMARY KEY(actor_id,target_id), CHECK(actor_id<>target_id)
);
CREATE TABLE chat_messages (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 world_id uuid NOT NULL REFERENCES worlds(id), user_id uuid NOT NULL REFERENCES users(id),
 client_message_id text NOT NULL CHECK(char_length(client_message_id) BETWEEN 1 AND 128),
 text text NOT NULL CHECK(char_length(text) BETWEEN 1 AND 500),
 created_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz,
 UNIQUE(world_id,user_id,client_message_id)
);
CREATE INDEX chat_world_history_idx ON chat_messages(world_id,id DESC);
CREATE INDEX chat_retention_idx ON chat_messages(created_at);
CREATE TABLE chat_mutes (
 world_id uuid NOT NULL REFERENCES worlds(id), user_id uuid NOT NULL REFERENCES users(id),
 expires_at timestamptz NOT NULL, PRIMARY KEY(world_id,user_id)
);
CREATE TABLE moderation_audit (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 world_id uuid NOT NULL REFERENCES worlds(id), actor_id uuid NOT NULL REFERENCES users(id),
 target_id uuid REFERENCES users(id), action text NOT NULL,
 message_id bigint, expires_at timestamptz, created_at timestamptz NOT NULL DEFAULT now()
);
-- Per-account token bucket survives reconnect and is serialized by the user row.
CREATE TABLE chat_limits (
 user_id uuid PRIMARY KEY REFERENCES users(id), tokens double precision NOT NULL,
 updated_at timestamptz NOT NULL
);
