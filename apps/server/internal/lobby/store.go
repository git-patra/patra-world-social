package lobby

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"patra-world-social/server/internal/games/model"
	"sort"
	"time"
)

type PGStore struct{ Pool *pgxpool.Pool }

func (s *PGStore) Persist(ctx context.Context, m model.MatchStart, host, command string) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	// Startup takes this schema-scoped fence exclusively before enumeration, so
	// even an initial row that is not yet visible must settle before recovery.
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock_shared(hashtextextended(current_schema() || '.matches.startup',21))`); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,7))`, m.MatchID); e != nil {
		return e
	}
	// Startup fence -> match lock 7 -> sorted account locks 12. Deletion takes
	// account locks only, never a startup/match lock, so there is no reverse order.
	users := []string{}
	for _, a := range m.Actors {
		if a.ActorType == "human" {
			users = append(users, a.UserID)
		}
	}
	sort.Strings(users)
	for _, user := range users {
		if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,12))`, user); e != nil {
			return e
		}
		var active bool
		if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND deleted_at IS NULL)`, user).Scan(&active); e != nil {
			return e
		}
		if !active {
			return ErrForbidden
		}
	}
	version, e := json.Marshal(m.Version)
	payload, _ := json.Marshal(m)
	if e != nil {
		return e
	}
	var existing bool
	if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM matches WHERE id=$1 AND host_id=$2 AND command_id=$3 AND start_payload=$4::jsonb)`, m.MatchID, host, command, payload).Scan(&existing); e != nil {
		return e
	}
	if existing {
		return nil
	}
	// Membership is rechecked inside the transaction; authorization at transport
	// alone cannot grant a durable match claim after membership was revoked.
	for _, a := range m.Actors {
		if a.ActorType != "human" {
			continue
		}
		var active bool
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM world_members m JOIN worlds w ON w.id=m.world_id WHERE m.world_id=$1 AND m.user_id=$2 AND m.active AND w.archived_at IS NULL AND NOT EXISTS(SELECT 1 FROM world_bans b WHERE b.world_id=m.world_id AND b.user_id=m.user_id))`, m.WorldID, a.UserID).Scan(&active)
		if e != nil {
			return e
		}
		if !active {
			return ErrForbidden
		}
	}
	tag, e := tx.Exec(ctx, `INSERT INTO matches(id,world_id,arena_id,game_type,host_id,command_id,seed,version,prepared_at,start_payload) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(id) DO NOTHING`, m.MatchID, m.WorldID, m.ArenaID, m.GameType, host, command, m.Seed, version, time.UnixMilli(m.PreparedAt), payload)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		var same bool
		if e = tx.QueryRow(ctx, `SELECT host_id=$2 AND command_id=$3 AND start_payload=$4::jsonb FROM matches WHERE id=$1`, m.MatchID, host, command, payload).Scan(&same); e != nil {
			return e
		}
		if !same {
			return ErrForbidden
		}
		return nil
	}
	for slot, a := range m.Actors {
		var user any
		if a.ActorType == "human" {
			user = a.UserID
		}
		_, e = tx.Exec(ctx, `INSERT INTO match_participants(match_id,actor_id,actor_type,user_id,display_name,avatar_id,roster_slot) VALUES($1,$2,$3,$4,$5,$6,$7)`, m.MatchID, a.ActorID, a.ActorType, user, a.DisplayName, a.AvatarID, slot)
		if e != nil {
			return e
		}
		if user != nil {
			_, e = tx.Exec(ctx, `INSERT INTO active_match_users(user_id,match_id) VALUES($1,$2)`, user, m.MatchID)
			if e != nil {
				return e
			}
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return fmt.Errorf("%w: %v", ErrCommitUnknown, e)
	}
	return nil
}

// Lookup shares the start transaction's lock. A false result proves that the
// original transaction has resolved without this row, rather than racing a
// COMMIT whose acknowledgement was lost.
func (s *PGStore) Lookup(ctx context.Context, id string) (bool, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,7))`, id); e != nil {
		return false, e
	}
	var found bool
	e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM matches WHERE id=$1)`, id).Scan(&found)
	return found, e
}
func (s *PGStore) Schedule(ctx context.Context, id string, startsAt int64) error {
	tag, e := s.Pool.Exec(ctx, `UPDATE matches SET starts_at=$2 WHERE id=$1 AND status='running' AND (starts_at IS NULL OR starts_at=$2)`, id, time.UnixMilli(startsAt))
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return errors.New("schedule projection rejected")
	}
	return nil
}
func (s *PGStore) Terminal(ctx context.Context, id, status string) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	// A completed result belongs to the finalizer; release cannot manufacture one.
	if status == "completed" {
		var current string
		if e = tx.QueryRow(ctx, `SELECT status FROM matches WHERE id=$1 FOR UPDATE`, id).Scan(&current); e != nil {
			return e
		}
		if current != "completed" {
			return ErrForbidden
		}
	} else {
		tag, err := tx.Exec(ctx, `UPDATE matches SET status=$2,finished_at=coalesce(finished_at,now()) WHERE id=$1 AND status IN ('running','finalizing',$2)`, id, status)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrForbidden
		}
	}
	if _, e = tx.Exec(ctx, `DELETE FROM active_match_users WHERE match_id=$1`, id); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

// ReleaseUser frees only this match's claim after an intentional departure or
// expired game grace. The immutable participant remains for engine DNF/history.
func (s *PGStore) ReleaseUser(ctx context.Context, match, user string) error {
	_, e := s.Pool.Exec(ctx, `DELETE FROM active_match_users WHERE match_id=$1 AND user_id=$2`, match, user)
	return e
}
