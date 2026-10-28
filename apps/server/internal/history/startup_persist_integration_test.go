//go:build integration

package history

import (
	"context"
	"errors"
	"fmt"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/lobby"
	"strings"
	"testing"
	"time"
)

// Removing the startup fence lets recovery finish while the initial Persist is
// still invisible, leaving its late-committing running row and busy claim behind.
func TestPostgresStartupWaitsForInitialPersist(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		t.Run(map[bool]string{false: "persist_before_ack_error", true: "rollback"}[rollback], func(t *testing.T) {
			s, old, users := fixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			if _, e := s.Pool.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role) VALUES($1,$2,'member')`, old.WorldID, users[1]); e != nil {
				t.Fatal(e)
			}
			m := model.MatchStart{MatchID: "33333333-3333-4333-8333-333333333333", WorldID: old.WorldID, ArenaID: "duck-2", GameType: "duck", Version: old.Version, PreparedAt: 1000, Actors: []model.Actor{{ActorID: "44444444-4444-4444-8444-444444444444", ActorType: "human", UserID: users[1], DisplayName: "synthetic"}}}
			failure := ""
			if rollback {
				failure = "RAISE EXCEPTION 'synthetic initial persist rollback';"
			}
			// Hold the real write after both the match and its claim were inserted,
			// while neither is visible to startup's READ COMMITTED enumeration.
			if _, e := s.Pool.Exec(ctx, fmt.Sprintf(`CREATE FUNCTION hold_initial_persist() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(hashtextextended(current_schema() || '.test.initial-persist',0)); %s RETURN NEW; END $$; CREATE TRIGGER hold_initial_persist AFTER INSERT ON active_match_users FOR EACH ROW EXECUTE FUNCTION hold_initial_persist()`, failure)); e != nil {
				t.Fatal(e)
			}
			gate, e := s.Pool.Begin(ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer gate.Rollback(context.Background())
			var gatePID int
			if e = gate.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&gatePID); e != nil {
				t.Fatal(e)
			}
			if _, e = gate.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(current_schema() || '.test.initial-persist',0))`); e != nil {
				t.Fatal(e)
			}
			persisted := make(chan error, 1)
			go func() {
				e := (&lobby.PGStore{Pool: s.Pool}).Persist(ctx, m, users[1], "initial")
				if e == nil {
					// Lose only the acknowledgement, after the actual PGStore COMMIT.
					e = lobby.ErrCommitUnknown
				}
				persisted <- e
			}()
			var persistPID int
			deadline := time.Now().Add(3 * time.Second)
			for persistPID == 0 {
				if e = s.Pool.QueryRow(ctx, `SELECT coalesce((SELECT pid FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid)) LIMIT 1),0)`, gatePID).Scan(&persistPID); e != nil {
					t.Fatal(e)
				}
				if time.Now().After(deadline) {
					t.Fatal("initial Persist did not reach the real transaction gate")
				}
				if persistPID == 0 {
					time.Sleep(5 * time.Millisecond)
				}
			}
			recovered := make(chan error, 1)
			go func() { recovered <- s.RecoverPending(ctx) }()
			early := false
			var recoveryErr error
			deadline = time.Now().Add(3 * time.Second)
			for {
				select {
				case recoveryErr = <-recovered:
					early = true
				default:
				}
				var waiting bool
				if e = s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid)))`, persistPID).Scan(&waiting); e != nil {
					t.Fatal(e)
				}
				if waiting || early {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("startup neither completed nor waited on the initial Persist")
				}
				time.Sleep(5 * time.Millisecond)
			}
			var beforeStatus string
			var beforeClaims, invisible int
			if e = s.Pool.QueryRow(ctx, `SELECT status,(SELECT count(*) FROM active_match_users),(SELECT count(*) FROM matches WHERE id=$2) FROM matches WHERE id=$1`, old.MatchID, m.MatchID).Scan(&beforeStatus, &beforeClaims, &invisible); e != nil {
				t.Fatal(e)
			}
			if e = gate.Commit(ctx); e != nil {
				t.Fatal(e)
			}
			persistErr := <-persisted
			if !early {
				recoveryErr = <-recovered
			}
			if early || beforeStatus != "running" || beforeClaims != 1 || invisible != 0 {
				t.Fatalf("startup made an orphan decision before invisible initial Persist settled: returned=%v old_status=%s claims=%d visible_new=%d recovery_err=%v", early, beforeStatus, beforeClaims, invisible, recoveryErr)
			}
			if recoveryErr != nil {
				t.Fatal(recoveryErr)
			}
			if persistErr == nil || (!rollback && !errors.Is(persistErr, lobby.ErrCommitUnknown)) {
				t.Fatal("expected failed acknowledgement or real rollback", persistErr)
			}
			if rollback && !strings.Contains(persistErr.Error(), "synthetic initial persist rollback") {
				t.Fatal("Persist failed outside the gated rollback", persistErr)
			}
			var running, claims, matches int
			if e = s.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM matches WHERE status!='aborted_server'),(SELECT count(*) FROM active_match_users),(SELECT count(*) FROM matches)`).Scan(&running, &claims, &matches); e != nil {
				t.Fatal(e)
			}
			wantMatches := 2
			if rollback {
				wantMatches = 1
			}
			if running != 0 || claims != 0 || matches != wantMatches {
				t.Fatalf("startup left an orphan after initial Persist resolved: non_aborted=%d claims=%d matches=%d want=%d", running, claims, matches, wantMatches)
			}
			if e = s.RecoverPending(ctx); e != nil {
				t.Fatal("repeated startup recovery", e)
			}
		})
	}
}

func TestPostgresInitialPersistFencesBeforeMatchLock(t *testing.T) {
	s, old, users := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	m := model.MatchStart{MatchID: "33333333-3333-4333-8333-333333333333", WorldID: old.WorldID, ArenaID: "duck-2", GameType: "duck", Version: old.Version, PreparedAt: 1000, Actors: []model.Actor{{ActorID: "44444444-4444-4444-8444-444444444444", ActorType: "human", UserID: users[1], DisplayName: "synthetic"}}}
	if _, e := s.Pool.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role) VALUES($1,$2,'member')`, old.WorldID, users[1]); e != nil {
		t.Fatal(e)
	}
	gate, e := s.Pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer gate.Rollback(context.Background())
	var gatePID int
	if e = gate.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&gatePID); e != nil {
		t.Fatal(e)
	}
	if e = lockMatch(ctx, gate, m.MatchID); e != nil {
		t.Fatal(e)
	}
	persisted := make(chan error, 1)
	go func() { persisted <- (&lobby.PGStore{Pool: s.Pool}).Persist(ctx, m, users[1], "initial") }()
	var persistPID int
	deadline := time.Now().Add(3 * time.Second)
	for persistPID == 0 {
		if e = s.Pool.QueryRow(ctx, `SELECT coalesce((SELECT pid FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid)) LIMIT 1),0)`, gatePID).Scan(&persistPID); e != nil {
			t.Fatal(e)
		}
		if time.Now().After(deadline) {
			t.Fatal("Persist did not wait for its match lock")
		}
		time.Sleep(5 * time.Millisecond)
	}
	// A Persist already waiting on its match lock must hold the shared startup
	// fence. Otherwise startup can hold the exclusive fence while Persist holds
	// a match lock, reversing recovery's startup -> match lock order.
	short, stop := context.WithTimeout(ctx, 150*time.Millisecond)
	e = s.RecoverPending(short)
	stop()
	if e == nil {
		t.Fatal("startup passed Persist before it acquired its match lock")
	}
	if !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal("startup failed before waiting for Persist", e)
	}
	if e = gate.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	if e = <-persisted; e != nil {
		t.Fatal(e)
	}
	if e = s.RecoverPending(ctx); e != nil {
		t.Fatal(e)
	}
	var status string
	var claims int
	if e = s.Pool.QueryRow(ctx, `SELECT status,(SELECT count(*) FROM active_match_users) FROM matches WHERE id=$1`, m.MatchID).Scan(&status, &claims); e != nil {
		t.Fatal(e)
	}
	if status != "aborted_server" || claims != 0 {
		t.Fatal("canceled startup leaked its fence or missed the settled match", status, claims)
	}
}
