-- name: CreateUser :one
INSERT INTO users (username,display_name,password_hash,recovery_hash) VALUES ($1,$2,$3,$4) RETURNING id,username,display_name,avatar,created_at;

-- name: GetUserCredentials :one
SELECT id,username,display_name,avatar,password_hash,recovery_hash,created_at FROM users WHERE username=$1 AND deleted_at IS NULL;

-- name: CreateSession :exec
INSERT INTO sessions (token_hash,user_id,expires_at,idle_expires_at) VALUES ($1,$2,$3,$4);

-- name: RefreshSession :one
WITH refreshed AS (
  UPDATE sessions SET idle_expires_at=LEAST(expires_at,now()+interval '24 hours')
  WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>now() AND idle_expires_at>now()
  RETURNING user_id,expires_at,idle_expires_at
)
SELECT refreshed.user_id,u.username,u.display_name,u.avatar,refreshed.expires_at,refreshed.idle_expires_at
FROM refreshed JOIN users u ON u.id=refreshed.user_id AND u.deleted_at IS NULL;

-- name: RevokeSession :exec
UPDATE sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL;

-- name: RevokeUserSessions :exec
UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL;
