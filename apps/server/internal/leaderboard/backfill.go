package leaderboard

import (
	"context"
	"encoding/json"
	"patra-world-social/server/internal/games/model"
)

// Backfill runs before serving or starting workers under the caller's startup
// deadline. It reads at most 100 IDs at once and commits each receipt with its
// PB/champion changes. Cancellation fails startup; restarting resumes remaining
// rows without wiping boards or rewriting immutable history/Duck detail.
// Historical order is FinishedAt then result ID, since old matches had no shared
// podium lock. This is reproducible import order, not claimed old commit order.
func (s *Service) Backfill(ctx context.Context) error {
	for {
		rows, e := s.Pool.Query(ctx, `SELECT m.id::text FROM matches m WHERE m.game_type='duck' AND m.status='completed' AND NOT EXISTS(SELECT 1 FROM leaderboard_projections p WHERE p.match_id=m.id) ORDER BY m.finished_at,m.id LIMIT 100`)
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
		if len(ids) == 0 {
			return nil
		}
		for _, id := range ids {
			if e = s.backfillResult(ctx, id); e != nil {
				return e
			}
		}
	}
}
func (s *Service) backfillResult(ctx context.Context, id string) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,7))`, id); e != nil {
		return e
	}
	var raw []byte
	var hasCompleteDetail bool
	if e = tx.QueryRow(ctx, `SELECT result_payload,EXISTS(SELECT 1 FROM duck_results d WHERE d.match_id=matches.id AND d.complete) FROM matches WHERE id=$1 AND status='completed' AND game_type='duck'`, id).Scan(&raw, &hasCompleteDetail); e != nil {
		return e
	}
	// Pre-Duck generic history may have a finished outcome without a validated
	// complete Duck detail. It remains history, never an official score.
	if !hasCompleteDetail {
		if _, e = tx.Exec(ctx, `INSERT INTO leaderboard_projections(match_id)VALUES($1) ON CONFLICT DO NOTHING`, id); e != nil {
			return e
		}
		return tx.Commit(ctx)
	}
	var result model.MatchResult
	if e = json.Unmarshal(raw, &result); e != nil {
		return e
	}
	if result.MatchID != id {
		return ErrInvalid
	}
	if e = ProjectResult(ctx, tx, result); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
