// Package leaderboard projects private Duck personal bests and audited ownership.
package leaderboard

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"patra-world-social/server/internal/games/model"
	"sort"
)

var ErrNotFound = errors.New("private board not found")
var ErrInvalid = errors.New("invalid leaderboard request")

const CurrentRuleset = "duck-v1@1"

type Service struct{ Pool *pgxpool.Pool }

func New(p *pgxpool.Pool) *Service       { return &Service{Pool: p} }
func Ruleset(r model.MatchResult) string { return fmt.Sprintf("%s@%d", r.RulesID, r.RulesVersion) }

type Entry struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	ResultID    string `json:"resultId"`
	Score       int64  `json:"score"`
	Hits        int64  `json:"hits"`
	Shots       int64  `json:"shots"`
	AchievedAt  int64  `json:"achievedAt"`
}

// Accuracy is compared by integer cross multiplication. Duck caps shots at 240,
// so products are exact and bounded. A zero-shot result has zero accuracy.
func better(a, b Entry) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	x, y := a.Hits*max(b.Shots, 1), b.Hits*max(a.Shots, 1)
	if x != y {
		return x > y
	}
	if a.AchievedAt != b.AchievedAt {
		return a.AchievedAt < b.AchievedAt
	}
	return a.ResultID < b.ResultID
}
func lockAccount(ctx context.Context, tx pgx.Tx, user string) error {
	_, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,12))`, user)
	return e
}
func lockBoard(ctx context.Context, tx pgx.Tx, world, rules string) error {
	if _, e := tx.Exec(ctx, `INSERT INTO leaderboard_boards(world_id,game_type,ruleset)VALUES($1,'duck',$2) ON CONFLICT DO NOTHING`, world, rules); e != nil {
		return e
	}
	_, e := tx.Exec(ctx, `SELECT 1 FROM leaderboard_boards WHERE world_id=$1 AND game_type='duck' AND ruleset=$2 FOR UPDATE`, world, rules)
	return e
}
func entries(ctx context.Context, tx pgx.Tx, world, rules string) ([]Entry, error) {
	rows, e := tx.Query(ctx, `SELECT e.user_id::text,u.display_name,e.result_id::text,e.score,e.hits,e.shots,e.achieved_at FROM leaderboard_entries e JOIN users u ON u.id=e.user_id WHERE e.world_id=$1 AND e.game_type='duck' AND e.ruleset=$2`, world, rules)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Entry{}
	for rows.Next() {
		var v Entry
		if e = rows.Scan(&v.UserID, &v.DisplayName, &v.ResultID, &v.Score, &v.Hits, &v.Shots, &v.AchievedAt); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return better(out[i], out[j]) })
	return out, rows.Err()
}

// ProjectResult must be composed after Duck detail validation in the existing
// finalizer transaction. Lock order: match (caller), account IDs, board partition.
// Account removal takes the same account lock before discovering any partitions.
func ProjectResult(ctx context.Context, tx pgx.Tx, r model.MatchResult) error {
	if r.GameType != "duck" {
		return nil
	}
	// Reentrant when called by T8. Backfill uses this same match fence before
	// account/partition locks, and receipts commit atomically with the projection.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,7))`, r.MatchID); err != nil {
		return err
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM leaderboard_projections WHERE match_id=$1)`, r.MatchID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	if err := projectResult(ctx, tx, r); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO leaderboard_projections(match_id)VALUES($1)`, r.MatchID)
	return err
}
func projectResult(ctx context.Context, tx pgx.Tx, r model.MatchResult) error {
	if r.GameType != "duck" {
		return nil
	}
	candidates := []Entry{}
	for _, p := range r.Participants {
		if p.ActorType != "human" || p.UserID == "" || p.BotAssisted || p.Outcome != "finished" || p.Duck == nil || p.Score == nil {
			continue
		}
		if p.Duck.Hits < 0 || p.Duck.Shots < p.Duck.Hits || p.Duck.Shots > 240 || *p.Score < 0 {
			return ErrInvalid
		}
		candidates = append(candidates, Entry{UserID: p.UserID, ResultID: r.MatchID, Score: *p.Score, Hits: int64(p.Duck.Hits), Shots: int64(p.Duck.Shots), AchievedAt: r.FinishedAt})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].UserID < candidates[j].UserID })
	eligible := []Entry{}
	for _, v := range candidates {
		if e := lockAccount(ctx, tx, v.UserID); e != nil {
			return e
		}
		var allowed bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1) AND NOT EXISTS(SELECT 1 FROM leaderboard_excluded_accounts WHERE user_id=$1)`, v.UserID).Scan(&allowed); e != nil {
			return e
		}
		if allowed {
			eligible = append(eligible, v)
		}
	}
	if len(eligible) == 0 {
		return nil
	}
	rules := Ruleset(r)
	if e := lockBoard(ctx, tx, r.WorldID, rules); e != nil {
		return e
	}
	current, e := entries(ctx, tx, r.WorldID, rules)
	if e != nil {
		return e
	}
	for _, v := range eligible {
		improves := true
		for _, old := range current {
			if old.UserID == v.UserID {
				improves = better(v, old)
				break
			}
		}
		if improves {
			if _, e = tx.Exec(ctx, `INSERT INTO leaderboard_entries(world_id,ruleset,user_id,result_id,score,hits,shots,achieved_at)VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(world_id,game_type,ruleset,user_id) DO UPDATE SET result_id=excluded.result_id,score=excluded.score,hits=excluded.hits,shots=excluded.shots,achieved_at=excluded.achieved_at`, r.WorldID, rules, v.UserID, v.ResultID, v.Score, v.Hits, v.Shots, v.AchievedAt); e != nil {
				return e
			}
		}
	}
	return updateChampion(ctx, tx, r.WorldID, rules, false)
}
