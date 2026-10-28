-- name: OwnMatch :one
SELECT m.* FROM matches m
WHERE m.id=$1 AND EXISTS(SELECT 1 FROM match_participants p WHERE p.match_id=m.id AND p.user_id=$2);

-- name: OwnHistory :many
SELECT m.id,m.world_id,m.game_type,m.status,m.version,m.prepared_at,m.starts_at,m.finished_at
FROM matches m WHERE EXISTS(SELECT 1 FROM match_participants p WHERE p.match_id=m.id AND p.user_id=$1)
ORDER BY m.prepared_at DESC,m.id DESC LIMIT $2;
