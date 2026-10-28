CREATE TABLE penalty_turns (
 match_id uuid NOT NULL REFERENCES matches(id),
 turn integer NOT NULL CHECK(turn BETWEEN 1 AND 20),
 kicker_id uuid NOT NULL,
 keeper_id uuid NOT NULL CHECK(keeper_id<>kicker_id),
 kick_zone integer NOT NULL CHECK(kick_zone BETWEEN 0 AND 3),
 keep_zone integer NOT NULL CHECK(keep_zone BETWEEN 0 AND 3),
 kick_timeout boolean NOT NULL,
 keep_timeout boolean NOT NULL,
 goal boolean NOT NULL CHECK(goal=(kick_zone<>keep_zone)),
 PRIMARY KEY(match_id,turn)
);
