CREATE TABLE duck_results (
 match_id uuid NOT NULL REFERENCES matches(id),
 actor_id uuid NOT NULL,
 score bigint NOT NULL CHECK(score>=0),
 hits integer NOT NULL CHECK(hits>=0),
 shots integer NOT NULL CHECK(shots>=hits AND shots<=240),
 complete boolean NOT NULL,
 PRIMARY KEY(match_id,actor_id),
 FOREIGN KEY(match_id,actor_id) REFERENCES match_participants(match_id,actor_id)
);
