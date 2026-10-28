package leaderboard

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"patra-world-social/server/internal/games/model"
)

type ResultSummary struct {
	Entry
	Version  model.Version `json:"version"`
	Status   string        `json:"status"`
	Complete bool          `json:"complete"`
}
type Board struct {
	WorldID               string                 `json:"worldId"`
	Ruleset               string                 `json:"ruleset"`
	Top                   []Entry                `json:"top"`
	PersonalBest          *Entry                 `json:"personalBest"`
	LastResult            *ResultSummary         `json:"lastResult"`
	Changes               []ChampionChange       `json:"changes"`
	AdministrativeChanges []AdministrativeChange `json:"administrativeChanges"`
	Rulesets              []string               `json:"rulesets"`
}

func authorized(ctx context.Context, tx pgx.Tx, user, world string) error {
	var ok bool
	e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM world_members m JOIN worlds w ON w.id=m.world_id WHERE m.world_id=$1 AND m.user_id=$2 AND m.active AND w.archived_at IS NULL AND NOT EXISTS(SELECT 1 FROM world_bans b WHERE b.world_id=m.world_id AND b.user_id=m.user_id) AND NOT EXISTS(SELECT 1 FROM leaderboard_excluded_accounts x WHERE x.user_id=m.user_id))`, world, user).Scan(&ok)
	if e != nil {
		return e
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}
func (s *Service) readTx(ctx context.Context, user, world string) (pgx.Tx, error) {
	// One coherent committed snapshot for podium/PB/champion and authorization.
	tx, e := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return nil, e
	}
	if e = authorized(ctx, tx, user, world); e != nil {
		tx.Rollback(ctx)
		return nil, e
	}
	return tx, nil
}
func (s *Service) Board(ctx context.Context, user, world, rules string) (Board, error) {
	b := Board{WorldID: world, Ruleset: rules, Top: []Entry{}, Changes: []ChampionChange{}, AdministrativeChanges: []AdministrativeChange{}, Rulesets: []string{}}
	if len(rules) < 1 || len(rules) > 128 {
		return b, ErrInvalid
	}
	tx, e := s.readTx(ctx, user, world)
	if e != nil {
		return b, e
	}
	defer tx.Rollback(ctx)
	all, e := entries(ctx, tx, world, rules)
	if e != nil {
		return b, e
	}
	for _, v := range all {
		if v.UserID == user {
			v := v
			b.PersonalBest = &v
		}
	}
	b.Top = all[:min(3, len(all))]
	b.Changes, e = changes(ctx, tx, world, rules, 3)
	if e != nil {
		return b, e
	}
	rows, e := tx.Query(ctx, `SELECT ruleset FROM leaderboard_boards WHERE world_id=$1 AND game_type='duck' ORDER BY ruleset`, world)
	if e != nil {
		return b, e
	}
	for rows.Next() {
		var v string
		if e = rows.Scan(&v); e != nil {
			rows.Close()
			return b, e
		}
		b.Rulesets = append(b.Rulesets, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return b, e
	}
	rows, e = tx.Query(ctx, `SELECT id,reason,changed_at FROM leaderboard_admin_changes WHERE world_id=$1 AND ruleset=$2 AND game_type='duck' ORDER BY id DESC LIMIT 3`, world, rules)
	if e != nil {
		return b, e
	}
	for rows.Next() {
		var v AdministrativeChange
		if e = rows.Scan(&v.ID, &v.Reason, &v.ChangedAt); e != nil {
			rows.Close()
			return b, e
		}
		b.AdministrativeChanges = append(b.AdministrativeChanges, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return b, e
	}
	b.LastResult, e = summary(tx.QueryRow(ctx, summarySelect+` WHERE m.world_id=$1 AND p.user_id=$2 AND m.game_type='duck' AND (m.version->>'rulesId') || '@' || (m.version->>'rulesVersion')=$3 AND m.status IN ('completed','aborted_server','aborted_storage','abandoned') ORDER BY m.prepared_at DESC,m.id DESC LIMIT 1`, world, user, rules))
	if errors.Is(e, pgx.ErrNoRows) {
		e = nil
	}
	if e != nil {
		return b, e
	}
	return b, tx.Commit(ctx)
}
func (s *Service) Top(ctx context.Context, user, world, rules string, limit int) ([]Entry, error) {
	if limit < 1 || limit > 3 {
		return nil, ErrInvalid
	}
	b, e := s.Board(ctx, user, world, rules)
	if e != nil {
		return nil, e
	}
	return b.Top[:min(limit, len(b.Top))], nil
}
func changes(ctx context.Context, tx pgx.Tx, world, rules string, limit int) ([]ChampionChange, error) {
	rows, e := tx.Query(ctx, `SELECT c.id,c.ruleset,coalesce(c.user_id::text,''),coalesce(u.display_name,'Akun dihapus'),c.result_id::text,c.changed_at FROM champion_changes c LEFT JOIN users u ON u.id=c.user_id WHERE c.world_id=$1 AND c.game_type='duck' AND ($2='' OR c.ruleset=$2) ORDER BY c.id DESC LIMIT $3`, world, rules, limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []ChampionChange{}
	for rows.Next() {
		var v ChampionChange
		if e = rows.Scan(&v.ID, &v.Ruleset, &v.UserID, &v.DisplayName, &v.ResultID, &v.ChangedAt); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Service) RecentChampionChanges(ctx context.Context, user, world string, limit int) ([]ChampionChange, error) {
	if limit < 1 || limit > 3 {
		return nil, ErrInvalid
	}
	tx, e := s.readTx(ctx, user, world)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	out, e := changes(ctx, tx, world, "", limit)
	if e != nil {
		return nil, e
	}
	return out, tx.Commit(ctx)
}

const summarySelect = `SELECT coalesce(p.user_id::text,''),coalesce(u.display_name,'Akun dihapus'),m.id::text,coalesce(d.score,0),coalesce(d.hits,0),coalesce(d.shots,0),coalesce((extract(epoch FROM m.finished_at)*1000)::bigint,0),m.version,m.status,coalesce(d.complete,false) FROM matches m JOIN match_participants p ON p.match_id=m.id LEFT JOIN users u ON u.id=p.user_id LEFT JOIN duck_results d ON d.match_id=p.match_id AND d.actor_id=p.actor_id`

func summary(row pgx.Row) (*ResultSummary, error) {
	var v ResultSummary
	var raw []byte
	e := row.Scan(&v.UserID, &v.DisplayName, &v.ResultID, &v.Score, &v.Hits, &v.Shots, &v.AchievedAt, &raw, &v.Status, &v.Complete)
	if e != nil {
		return nil, e
	}
	e = json.Unmarshal(raw, &v.Version)
	return &v, e
}

// Result exposes only an eligible Duck summary from this world. It never invokes
// personal history or returns other matches, a roster, seed, or result-job payload.
func (s *Service) Result(ctx context.Context, user, world, id string) (*ResultSummary, error) {
	tx, e := s.readTx(ctx, user, world)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	v, e := summary(tx.QueryRow(ctx, summarySelect+` WHERE m.world_id=$1 AND m.id=$2 AND m.game_type='duck' AND m.status='completed' AND d.complete AND p.actor_type='human' AND NOT p.bot_assisted AND NOT EXISTS(SELECT 1 FROM leaderboard_excluded_accounts x WHERE x.user_id=p.user_id)`, world, id))
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if e != nil {
		return nil, e
	}
	return v, tx.Commit(ctx)
}

// InvalidationFor returns only public partition identity after confirmed worker
// completion, including recovered jobs. No personal outcome/history is broadcast.
func (s *Service) InvalidationFor(ctx context.Context, id string) (*Invalidation, error) {
	var v Invalidation
	var version model.Version
	var raw []byte
	e := s.Pool.QueryRow(ctx, `SELECT m.world_id::text,m.version FROM matches m WHERE m.id=$1 AND m.status='completed' AND m.game_type='duck' AND EXISTS(SELECT 1 FROM duck_results d WHERE d.match_id=m.id AND d.complete)`, id).Scan(&v.WorldID, &raw)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	if e = json.Unmarshal(raw, &version); e != nil {
		return nil, e
	}
	v.Ruleset = Ruleset(model.MatchResult{Version: version})
	v.ResultID = id
	return &v, nil
}

type Invalidation struct {
	WorldID  string `json:"worldId"`
	Ruleset  string `json:"ruleset"`
	ResultID string `json:"resultId"`
}
