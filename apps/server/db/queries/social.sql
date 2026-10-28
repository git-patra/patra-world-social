-- name: ExactSocialUsername :one
SELECT u.id::text AS id,u.username FROM users u WHERE u.deleted_at IS NULL AND u.username=sqlc.arg(username) AND u.id<>sqlc.arg(actor_id)::text::uuid
AND NOT EXISTS(SELECT 1 FROM user_blocks b WHERE (b.actor_id=sqlc.arg(actor_id)::text::uuid AND b.target_id=u.id) OR(b.target_id=sqlc.arg(actor_id)::text::uuid AND b.actor_id=u.id));

-- name: ListFriendRelations :many
SELECT u.id::text AS id,u.username,f.id::text AS request_id,f.target_id=sqlc.arg(actor_id)::text::uuid AS incoming,f.accepted
FROM friend_requests f JOIN users u ON u.id=CASE WHEN f.actor_id=sqlc.arg(actor_id)::text::uuid THEN f.target_id ELSE f.actor_id END
WHERE f.actor_id=sqlc.arg(actor_id)::text::uuid OR f.target_id=sqlc.arg(actor_id)::text::uuid ORDER BY f.created_at DESC;
