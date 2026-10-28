-- Partitions serialize finalizations, including the initially empty board.
CREATE TABLE leaderboard_boards (
 world_id uuid NOT NULL REFERENCES worlds(id),
 game_type text NOT NULL CHECK(game_type='duck'),
 ruleset text NOT NULL,
 champion_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
 PRIMARY KEY(world_id,game_type,ruleset)
);
CREATE TABLE leaderboard_entries (
 world_id uuid NOT NULL,
 game_type text NOT NULL DEFAULT 'duck',
 ruleset text NOT NULL,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 result_id uuid NOT NULL REFERENCES matches(id),
 score bigint NOT NULL CHECK(score>=0),
 hits integer NOT NULL CHECK(hits>=0),
 shots integer NOT NULL CHECK(shots>=hits AND shots<=240),
 achieved_at bigint NOT NULL,
 PRIMARY KEY(world_id,game_type,ruleset,user_id),
 FOREIGN KEY(world_id,game_type,ruleset) REFERENCES leaderboard_boards(world_id,game_type,ruleset)
);
CREATE TABLE champion_changes (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 world_id uuid NOT NULL,
 game_type text NOT NULL DEFAULT 'duck',
 ruleset text NOT NULL,
 user_id uuid REFERENCES users(id) ON DELETE SET NULL,
 result_id uuid NOT NULL REFERENCES matches(id),
 changed_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 FOREIGN KEY(world_id,game_type,ruleset) REFERENCES leaderboard_boards(world_id,game_type,ruleset)
);
CREATE INDEX champion_recent ON champion_changes(world_id,game_type,ruleset,id DESC);
-- Tombstones fence pending immutable jobs after T14 account erasure. No profile,
-- credential, or display data is retained here; intentionally no user FK.
CREATE TABLE leaderboard_excluded_accounts (user_id uuid PRIMARY KEY);
CREATE TABLE leaderboard_admin_changes (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 world_id uuid NOT NULL,
 game_type text NOT NULL DEFAULT 'duck',
 ruleset text NOT NULL,
 reason text NOT NULL CHECK(reason='account_deleted'),
 champion_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
 changed_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 FOREIGN KEY(world_id,game_type,ruleset) REFERENCES leaderboard_boards(world_id,game_type,ruleset)
);
-- Receipts make startup import and finalization share exactly one projection.
CREATE TABLE leaderboard_projections (
 match_id uuid PRIMARY KEY REFERENCES matches(id),
 projected_at timestamptz NOT NULL DEFAULT now()
);
