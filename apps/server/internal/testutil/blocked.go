//go:build integration

package testutil

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

// BlockedPID observes the actual database wait edge, not a scheduling sleep.
func BlockedPID(t *testing.T, p *pgxpool.Pool, blocker int) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for {
		var pid int
		e := p.QueryRow(ctx, `SELECT coalesce((SELECT pid FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid)) LIMIT 1),0)`, blocker).Scan(&pid)
		if e != nil {
			t.Fatal(e)
		}
		if pid != 0 {
			return pid
		}
		select {
		case <-ctx.Done():
			t.Fatal("writer never reached expected lock")
		case <-time.After(time.Millisecond):
		}
	}
}
