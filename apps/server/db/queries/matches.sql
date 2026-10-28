-- Start persistence is a single transaction in lobby.PGStore. SQL below also
-- documents the nullable schedule projection consumed by T8 history recovery.
-- name: MatchExists :one
SELECT EXISTS(SELECT 1 FROM matches WHERE id=$1);

-- name: ScheduleMatch :execrows
UPDATE matches SET starts_at=to_timestamp($2::double precision/1000)
WHERE id=$1 AND status='running' AND (starts_at IS NULL OR starts_at=to_timestamp($2::double precision/1000));
