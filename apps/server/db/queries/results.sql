-- name: PendingResultJobs :many
SELECT match_id FROM result_jobs WHERE completed_at IS NULL AND next_attempt_at <= now() ORDER BY next_attempt_at, match_id LIMIT 50;

-- name: LockResultJob :one
SELECT * FROM result_jobs WHERE match_id=$1 FOR UPDATE;

-- name: HasPendingResults :one
SELECT EXISTS(SELECT 1 FROM result_jobs WHERE completed_at IS NULL);
