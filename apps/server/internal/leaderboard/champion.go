package leaderboard

import (
	"context"
	"github.com/jackc/pgx/v5"
	"time"
)

type ChampionChange struct {
	ID          int64     `json:"id"`
	Ruleset     string    `json:"ruleset"`
	UserID      string    `json:"userId"`
	DisplayName string    `json:"displayName"`
	ResultID    string    `json:"resultId"`
	ChangedAt   time.Time `json:"changedAt"`
}
type AdministrativeChange struct {
	ID        int64     `json:"id"`
	Reason    string    `json:"reason"`
	ChangedAt time.Time `json:"changedAt"`
}

func updateChampion(ctx context.Context, tx pgx.Tx, world, rules string, administrative bool) error {
	all, e := entries(ctx, tx, world, rules)
	if e != nil {
		return e
	}
	var old string
	if e = tx.QueryRow(ctx, `SELECT coalesce(champion_user_id::text,'') FROM leaderboard_boards WHERE world_id=$1 AND game_type='duck' AND ruleset=$2`, world, rules).Scan(&old); e != nil {
		return e
	}
	var next, match string
	if len(all) > 0 {
		next, match = all[0].UserID, all[0].ResultID
	}
	if _, e = tx.Exec(ctx, `UPDATE leaderboard_boards SET champion_user_id=nullif($3,'')::uuid WHERE world_id=$1 AND game_type='duck' AND ruleset=$2`, world, rules, next); e != nil {
		return e
	}
	if administrative {
		_, e = tx.Exec(ctx, `INSERT INTO leaderboard_admin_changes(world_id,ruleset,reason,champion_user_id)VALUES($1,$2,'account_deleted',nullif($3,'')::uuid)`, world, rules, next)
	} else if old != next && next != "" {
		_, e = tx.Exec(ctx, `INSERT INTO champion_changes(world_id,ruleset,user_id,result_id)VALUES($1,$2,$3,$4)`, world, rules, next, match)
	}
	return e
}

// RemoveAccount is a T14 deletion-transaction seam, not an HTTP deletion API.
// Call before erasing identity, commit with the rest of deletion, then invalidate
// these worlds. Repeated calls are harmless. Never publish before that commit.
func RemoveAccount(ctx context.Context, tx pgx.Tx, user string) error {
	if e := lockAccount(ctx, tx, user); e != nil {
		return e
	}
	if _, e := tx.Exec(ctx, `INSERT INTO leaderboard_excluded_accounts(user_id)VALUES($1) ON CONFLICT DO NOTHING`, user); e != nil {
		return e
	}
	rows, e := tx.Query(ctx, `SELECT world_id::text,ruleset FROM leaderboard_entries WHERE user_id=$1 ORDER BY world_id,ruleset`, user)
	if e != nil {
		return e
	}
	type partition struct{ world, rules string }
	parts := []partition{}
	for rows.Next() {
		var p partition
		if e = rows.Scan(&p.world, &p.rules); e != nil {
			rows.Close()
			return e
		}
		parts = append(parts, p)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, p := range parts {
		if e = lockBoard(ctx, tx, p.world, p.rules); e != nil {
			return e
		}
	}
	if _, e = tx.Exec(ctx, `DELETE FROM leaderboard_entries WHERE user_id=$1`, user); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `UPDATE champion_changes SET user_id=NULL WHERE user_id=$1`, user); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `UPDATE leaderboard_admin_changes SET champion_user_id=NULL WHERE champion_user_id=$1`, user); e != nil {
		return e
	}
	for _, p := range parts {
		if e = updateChampion(ctx, tx, p.world, p.rules, true); e != nil {
			return e
		}
	}
	return nil
}
