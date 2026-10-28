// Package social implements explicit friendships without granting world access.
package social

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"patra-world-social/server/internal/db"
	"patra-world-social/server/internal/identity"
	"regexp"
	"strings"
)

var ErrInput = errors.New("invalid social action")
var ErrForbidden = errors.New("social action unavailable")
var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,24}$`)

type Person struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	RequestID string `json:"requestId,omitempty"`
	Incoming  bool   `json:"incoming"`
	Accepted  bool   `json:"accepted"`
	Online    bool   `json:"online"`
}
type Service struct {
	Pool   *pgxpool.Pool
	Online func(string) bool
}

func New(pool *pgxpool.Pool) *Service { return &Service{Pool: pool} }

func lockAccounts(ctx context.Context, tx pgx.Tx, ids ...string) error {
	e := identity.LockActiveAccounts(ctx, tx, ids...)
	if errors.Is(e, identity.ErrDeleted) {
		return ErrForbidden
	}
	return e
}

// Pair locks serialize block, request and direct invitation initiation.
func LockPair(ctx context.Context, tx pgx.Tx, a, b string) error {
	_, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(least($1::text,$2::text)||':'||greatest($1::text,$2::text),0))`, a, b)
	return e
}
func Blocked(ctx context.Context, tx pgx.Tx, a, b string) (bool, error) {
	var blocked bool
	e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_blocks WHERE (actor_id=$1 AND target_id=$2) OR(actor_id=$2 AND target_id=$1))`, a, b).Scan(&blocked)
	return blocked, e
}
func (s *Service) Search(ctx context.Context, actor, username string) (Person, error) {
	if !usernamePattern.MatchString(username) {
		return Person{}, ErrInput
	}
	row, e := db.New(s.Pool).ExactSocialUsername(ctx, db.ExactSocialUsernameParams{ActorID: actor, Username: strings.ToLower(username)})
	p := Person{ID: row.ID, Username: row.Username}
	if errors.Is(e, pgx.ErrNoRows) {
		return p, ErrForbidden
	}
	return p, e
}
func (s *Service) RequestFriend(ctx context.Context, actor, username string) error {
	target, e := s.Search(ctx, actor, username)
	if e != nil {
		return e
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if e = lockAccounts(ctx, tx, actor, target.ID); e != nil {
		return e
	}
	if e = LockPair(ctx, tx, actor, target.ID); e != nil {
		return e
	}
	blocked, e := Blocked(ctx, tx, actor, target.ID)
	if e != nil {
		return e
	}
	if blocked {
		return ErrForbidden
	}
	_, e = tx.Exec(ctx, `INSERT INTO friend_requests(actor_id,target_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, actor, target.ID)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Service) RespondFriend(ctx context.Context, actor, request string, accept bool) error {
	var other string
	e := s.Pool.QueryRow(ctx, `SELECT actor_id FROM friend_requests WHERE id=$1 AND target_id=$2`, request, actor).Scan(&other)
	if errors.Is(e, pgx.ErrNoRows) {
		return ErrForbidden
	}
	if e != nil {
		return e
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if e = lockAccounts(ctx, tx, actor, other); e != nil {
		return e
	}
	if e = LockPair(ctx, tx, actor, other); e != nil {
		return e
	}
	blocked, e := Blocked(ctx, tx, actor, other)
	if e != nil {
		return e
	}
	if blocked {
		return ErrForbidden
	}
	sql := `DELETE FROM friend_requests WHERE id=$1 AND target_id=$2 AND NOT accepted`
	if accept {
		sql = `UPDATE friend_requests SET accepted=true WHERE id=$1 AND target_id=$2 AND NOT accepted`
	}
	tag, e := tx.Exec(ctx, sql, request, actor)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return ErrForbidden
	}
	return tx.Commit(ctx)
}
func (s *Service) Remove(ctx context.Context, actor, target string, block bool) error {
	if actor == target {
		return ErrInput
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if e = lockAccounts(ctx, tx, actor, target); e != nil {
		return e
	}
	if e = LockPair(ctx, tx, actor, target); e != nil {
		return e
	}
	if block {
		if _, e = tx.Exec(ctx, `INSERT INTO user_blocks(actor_id,target_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, actor, target); e != nil {
			return e
		}
	}
	if _, e = tx.Exec(ctx, `DELETE FROM friend_requests WHERE(actor_id=$1 AND target_id=$2)OR(actor_id=$2 AND target_id=$1)`, actor, target); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Service) Unblock(ctx context.Context, actor, target string) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if e = lockAccounts(ctx, tx, actor, target); e != nil {
		return e
	}
	if e = LockPair(ctx, tx, actor, target); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `DELETE FROM user_blocks WHERE actor_id=$1 AND target_id=$2`, actor, target); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Service) Blocks(ctx context.Context, actor string) ([]Person, error) {
	rows, e := s.Pool.Query(ctx, `SELECT u.id,u.username FROM user_blocks b JOIN users u ON u.id=b.target_id WHERE b.actor_id=$1 ORDER BY u.username`, actor)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Person{}
	for rows.Next() {
		var p Person
		if e = rows.Scan(&p.ID, &p.Username); e != nil {
			return nil, e
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (s *Service) List(ctx context.Context, actor string) ([]Person, error) {
	rows, e := db.New(s.Pool).ListFriendRelations(ctx, actor)
	if e != nil {
		return nil, e
	}
	out := []Person{}
	for _, row := range rows {
		p := Person{ID: row.ID, Username: row.Username, RequestID: row.RequestID, Incoming: row.Incoming, Accepted: row.Accepted}
		if p.Accepted && s.Online != nil {
			p.Online = s.Online(p.ID)
		}
		out = append(out, p)
	}
	return out, nil
}
