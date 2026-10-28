CREATE TABLE matches (
 id uuid PRIMARY KEY,
 world_id uuid NOT NULL REFERENCES worlds(id),
 arena_id text NOT NULL,
 game_type text NOT NULL CHECK(game_type IN ('race','penalty','duck')),
 host_id uuid NOT NULL REFERENCES users(id),
 command_id text NOT NULL CHECK(char_length(command_id) BETWEEN 1 AND 128),
 status text NOT NULL DEFAULT 'running' CHECK(status IN ('running','finalizing','completed','aborted_server','aborted_storage','abandoned')),
 seed bigint NOT NULL,
 version jsonb NOT NULL,
 start_payload jsonb NOT NULL,
 prepared_at timestamptz NOT NULL,
 starts_at timestamptz,
 finished_at timestamptz,
 UNIQUE(host_id,command_id)
);
CREATE UNIQUE INDEX match_one_active_arena ON matches(world_id,arena_id) WHERE status IN ('running','finalizing');
CREATE TABLE match_participants (
 match_id uuid NOT NULL REFERENCES matches(id),
 actor_id uuid NOT NULL,
 actor_type text NOT NULL CHECK(actor_type IN ('human','bot')),
 user_id uuid REFERENCES users(id),
 display_name text NOT NULL,
 avatar_id text NOT NULL,
 roster_slot integer NOT NULL CHECK(roster_slot BETWEEN 0 AND 4),
 bot_assisted boolean NOT NULL DEFAULT false,
 PRIMARY KEY(match_id,actor_id),
 UNIQUE(match_id,roster_slot),
 CHECK((actor_type='human')=(user_id IS NOT NULL))
);
CREATE UNIQUE INDEX match_one_human ON match_participants(match_id,user_id) WHERE user_id IS NOT NULL;
CREATE TABLE active_match_users (
 user_id uuid PRIMARY KEY REFERENCES users(id),
 match_id uuid NOT NULL REFERENCES matches(id)
);
