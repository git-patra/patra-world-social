package identity

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

var ErrDeleted = errors.New("account unavailable")

// LockActiveAccounts must precede world rows, social pair locks, invite rows and
// relationship writes. Deletion shares namespace 12 and never takes world/pair
// locks. Discover immutable counterpart IDs before calling; never add an account
// lock after taking a world/pair lock. Empty optional target IDs are ignored.
func LockActiveAccounts(ctx context.Context, tx pgx.Tx, ids ...string) error {
	// UUID spellings supplied by callers must hash to the same fence as DB IDs.
	rows, e := tx.Query(ctx, `SELECT DISTINCT value::uuid::text FROM unnest($1::text[]) AS value WHERE value<>'' ORDER BY 1`, ids)
	if e != nil {
		return e
	}
	var ordered []string
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		ordered = append(ordered, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	last := ""
	for _, id := range ordered {
		if id == "" || id == last {
			continue
		}
		last = id
		if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,12))`, id); e != nil {
			return e
		}
	}
	last = ""
	for _, id := range ordered {
		if id == "" || id == last {
			continue
		}
		last = id
		var alive bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND deleted_at IS NULL)`, id).Scan(&alive); e != nil {
			return e
		}
		if !alive {
			return ErrDeleted
		}
	}
	return nil
}
