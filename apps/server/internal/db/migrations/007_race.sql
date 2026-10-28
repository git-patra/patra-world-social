CREATE TABLE race_results (
 match_id uuid NOT NULL,
 actor_id uuid NOT NULL,
 start_slot integer NOT NULL CHECK(start_slot BETWEEN 0 AND 4),
 position integer NOT NULL CHECK(position BETWEEN 1 AND 5),
 laps integer NOT NULL CHECK(laps BETWEEN 0 AND 5),
 lap_times_ms bigint[] NOT NULL,
 best_lap_ms bigint,
 elapsed_ms bigint,
 dnf boolean NOT NULL,
 bot_assisted boolean NOT NULL,
 PRIMARY KEY(match_id,actor_id),
 FOREIGN KEY(match_id,actor_id) REFERENCES match_participants(match_id,actor_id),
 UNIQUE(match_id,start_slot),
 UNIQUE(match_id,position),
 CHECK(cardinality(lap_times_ms)=laps),
 CHECK(dnf OR (laps=5 AND elapsed_ms IS NOT NULL)),
 CHECK(elapsed_ms IS NULL OR elapsed_ms BETWEEN 0 AND 300000)
);
