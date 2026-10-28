package history

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"patra-world-social/server/internal/games/model"
	"time"
)

func (s *Service) Enqueue(ctx context.Context, r model.MatchResult) error {
	payload, hash, e := encodeResult(r)
	if e != nil {
		return e
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if e = lockMatch(ctx, tx, r.MatchID); e != nil {
		return e
	}
	var raw []byte
	var status string
	var starts *time.Time
	if e = tx.QueryRow(ctx, `SELECT start_payload,status,starts_at FROM matches WHERE id=$1 FOR UPDATE`, r.MatchID).Scan(&raw, &status, &starts); e != nil {
		return e
	}
	var m model.MatchStart
	if e = json.Unmarshal(raw, &m); e != nil {
		return e
	}
	var at *int64
	if starts != nil {
		v := starts.UnixMilli()
		at = &v
	}
	if !bindResult(r, m, at) {
		return ErrInvalid
	}
	var prior []byte
	e = tx.QueryRow(ctx, `SELECT result_hash FROM result_jobs WHERE match_id=$1`, r.MatchID).Scan(&prior)
	if e == nil {
		if !bytes.Equal(prior, hash[:]) {
			return ErrConflict
		}
		return nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	if status != "running" && status != "finalizing" {
		return ErrInvalid
	}
	if _, e = tx.Exec(ctx, `INSERT INTO result_jobs(match_id,result_hash,payload)VALUES($1,$2,$3)`, r.MatchID, hash[:], payload); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `UPDATE matches SET status='finalizing' WHERE id=$1`, r.MatchID); e != nil {
		return e
	}
	return tx.Commit(ctx) // An error may mean ack loss. Reconcile under the same lock.
}
func (s *Service) Finalize(ctx context.Context, id string) (model.MatchResult, error) {
	var r model.MatchResult
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return r, e
	}
	defer tx.Rollback(ctx)
	if e = lockMatch(ctx, tx, id); e != nil {
		return r, e
	}
	var payload, hash []byte
	var completed *time.Time
	if e = tx.QueryRow(ctx, `SELECT payload,result_hash,completed_at FROM result_jobs WHERE match_id=$1 FOR UPDATE`, id).Scan(&payload, &hash, &completed); e != nil {
		return r, e
	}
	if e = json.Unmarshal(payload, &r); e != nil {
		return r, e
	}
	_, computed, e := encodeResult(r)
	if e != nil {
		return r, e
	}
	if !bytes.Equal(hash, computed[:]) {
		return r, ErrConflict
	}
	if completed != nil {
		return r, nil
	}
	if projector := s.Projectors[r.GameType]; projector != nil {
		if e = projector(ctx, tx, r); e != nil {
			return r, e
		}
	}
	for _, p := range r.Participants {
		b, e := json.Marshal(p)
		if e != nil {
			return r, e
		}
		tag, e := tx.Exec(ctx, `UPDATE match_participants SET result_payload=$3,bot_assisted=$4 WHERE match_id=$1 AND actor_id=$2`, id, p.ActorID, b, p.BotAssisted)
		if e != nil {
			return r, e
		}
		if tag.RowsAffected() != 1 {
			return r, ErrInvalid
		}
	}
	tag, e := tx.Exec(ctx, `UPDATE matches SET status='completed',starts_at=$2,finished_at=$3,result_payload=$4 WHERE id=$1 AND status='finalizing'`, id, time.UnixMilli(r.StartsAt), time.UnixMilli(r.FinishedAt), payload)
	if e != nil {
		return r, e
	}
	if tag.RowsAffected() != 1 {
		return r, ErrInvalid
	}
	if _, e = tx.Exec(ctx, `UPDATE result_jobs SET completed_at=now() WHERE match_id=$1`, id); e != nil {
		return r, e
	}
	// Durable claims remain until explicit result-screen return or recovery. Games
	// own their result display duration; completion alone must not erase the lobby.
	e = tx.Commit(ctx)
	return r, e
}

// Reconcile serializes with enqueue/finalize, including a canceled COMMIT whose
// outcome was unknown. Absence is never inferred from a read outside this lock.
func (s *Service) Reconcile(ctx context.Context, id string) (string, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(ctx)
	if e = lockMatch(ctx, tx, id); e != nil {
		return "", e
	}
	var status string
	var durable bool
	if e = tx.QueryRow(ctx, `SELECT status,EXISTS(SELECT 1 FROM result_jobs WHERE match_id=matches.id) FROM matches WHERE id=$1 FOR UPDATE`, id).Scan(&status, &durable); e != nil {
		return "", e
	}
	if durable {
		if e = tx.Commit(ctx); e != nil {
			return "", e
		}
		if _, e = s.Finalize(ctx, id); e != nil {
			return "", e
		}
		status = "completed"
	} else {
		if status == "running" || status == "finalizing" {
			status = "aborted_storage"
			if _, e = tx.Exec(ctx, `UPDATE matches SET status=$2,finished_at=now() WHERE id=$1`, id, status); e != nil {
				return "", e
			}
		}
		if _, e = tx.Exec(ctx, `DELETE FROM active_match_users WHERE match_id=$1`, id); e != nil {
			return "", e
		}
		return status, tx.Commit(ctx)
	}
	_, e = s.Pool.Exec(ctx, `DELETE FROM active_match_users WHERE match_id=$1`, id)
	return status, e
}

// RecoverPending runs once before admitting gameplay on this process. Runtime
// reconciliation must use Reconcile instead: it has different absent-job policy.
func (s *Service) RecoverPending(ctx context.Context) error {
	tx, e := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	// Persist takes the shared schema-scoped fence before its match/account
	// locks. Take it exclusively before enumeration to include invisible initial
	// starts, and retain it through recovery so no new starts can slip between.
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(current_schema() || '.matches.startup',21))`); e != nil {
		return e
	}
	// Enumeration selects candidates only. No absent-job or terminal-claim decision
	// may use this snapshot: a pre-restart enqueue may still be committing.
	rows, e := tx.Query(ctx, `SELECT id::text FROM matches m WHERE status IN ('running','finalizing') OR EXISTS(SELECT 1 FROM active_match_users a WHERE a.match_id=m.id) ORDER BY id`)
	if e != nil {
		return e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, id := range ids {
		if e = s.recoverStartupMatch(ctx, tx, id); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}
func (s *Service) recoverStartupMatch(ctx context.Context, tx pgx.Tx, id string) error {
	if e := lockMatch(ctx, tx, id); e != nil {
		return e
	}
	// A separate READ COMMITTED statement after acquiring the match fence sees
	// the settling enqueue's committed job, even if enumeration saw it absent.
	var status string
	var durable bool
	if e := tx.QueryRow(ctx, `SELECT status,EXISTS(SELECT 1 FROM result_jobs j WHERE j.match_id=m.id) FROM matches m WHERE id=$1 FOR UPDATE`, id).Scan(&status, &durable); e != nil {
		return e
	}
	if (status == "running" || status == "finalizing") && !durable {
		status = "aborted_server"
		if _, e := tx.Exec(ctx, `UPDATE matches SET status='aborted_server',finished_at=now() WHERE id=$1`, id); e != nil {
			return e
		}
	}
	// No old runtime survives startup, but durable pending jobs keep their claims
	// until the worker has actually finalized and reconciled them.
	if status != "running" && status != "finalizing" {
		if _, e := tx.Exec(ctx, `DELETE FROM active_match_users WHERE match_id=$1`, id); e != nil {
			return e
		}
	}
	return nil
}
