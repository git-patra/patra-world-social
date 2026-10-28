-- Additive: historical rows, rules metadata and result hashes stay untouched.
ALTER TABLE matches DROP CONSTRAINT matches_game_type_check;
ALTER TABLE matches ADD CONSTRAINT matches_game_type_check CHECK(game_type IN ('race','penalty','duck','pingpong','blaster'));
