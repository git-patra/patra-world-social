-- name: CreatePrivateWorld :one
INSERT INTO worlds(name,owner_id) VALUES($1,$2) RETURNING *;

-- name: ListMemberWorlds :many
SELECT w.* FROM worlds w JOIN world_members m ON m.world_id=w.id
WHERE m.user_id=$1 AND m.active AND w.archived_at IS NULL ORDER BY w.created_at DESC;

-- name: GetActiveMembership :one
SELECT m.* FROM world_members m JOIN worlds w ON w.id=m.world_id
WHERE m.user_id=$1 AND m.world_id=$2 AND m.active AND w.archived_at IS NULL
AND NOT EXISTS(SELECT 1 FROM world_bans b WHERE b.world_id=m.world_id AND b.user_id=m.user_id);

-- name: ListWorldMembers :many
SELECT m.*,u.username FROM world_members m JOIN users u ON u.id=m.user_id
WHERE m.world_id=$1 AND m.active ORDER BY u.username;

-- name: LockWorld :one
SELECT * FROM worlds WHERE id=$1 FOR UPDATE;

-- name: GetInviteWorld :one
SELECT world_id FROM world_invites WHERE token_hash=$1;

-- name: LockInvite :one
SELECT * FROM world_invites WHERE token_hash=$1 FOR UPDATE;
