-- name: ListWorldChatPage :many
SELECT c.id::text AS id,c.world_id::text AS world_id,c.user_id::text AS user_id,u.username,c.client_message_id,
CASE WHEN c.deleted_at IS NULL THEN c.text ELSE '' END AS text,c.created_at,(c.deleted_at IS NOT NULL)::boolean AS deleted
FROM chat_messages c JOIN users u ON u.id=c.user_id
WHERE c.world_id=sqlc.arg(world_id)::text::uuid AND c.id<sqlc.arg(before_id) AND c.created_at>now()-interval '30 days'
AND EXISTS(SELECT 1 FROM world_members m JOIN worlds w ON w.id=m.world_id WHERE m.world_id=c.world_id AND m.user_id=sqlc.arg(user_id)::text::uuid AND m.active AND w.archived_at IS NULL AND NOT EXISTS(SELECT 1 FROM world_bans b WHERE b.world_id=m.world_id AND b.user_id=m.user_id))
ORDER BY c.id DESC LIMIT 50;

-- name: CleanupWorldChat :exec
DELETE FROM chat_messages WHERE created_at<=now()-interval '30 days';
