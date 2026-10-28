//go:build integration

package leaderboard

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"patra-world-social/server/internal/games/duck"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/history"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/testutil"
	"sync"
	"testing"
	"time"
)

type fixture struct {
	s     *Service
	h     *history.Service
	users []string
	world string
	n     int
}

func setup(t *testing.T) *fixture {
	t.Helper()
	p := testutil.Database(t)
	users, _ := testutil.Users(t, p, 5)
	f := &fixture{s: New(p), h: history.New(p), users: users}
	ctx := context.Background()
	must(t, p.QueryRow(ctx, `INSERT INTO worlds(name,owner_id)VALUES('synthetic',$1) RETURNING id`, users[0]).Scan(&f.world))
	for i, u := range users {
		role := "member"
		if i == 0 {
			role = "owner"
		}
		_, e := p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,$3)`, f.world, u, role)
		must(t, e)
	}
	f.h.Projectors["duck"] = func(ctx context.Context, tx pgx.Tx, r model.MatchResult) error {
		if e := duck.Project(ctx, tx, r); e != nil {
			return e
		}
		return ProjectResult(ctx, tx, r)
	}
	return f
}
func must(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func (f *fixture) result(t *testing.T, user int, score int64, hits, shots int, outcome string, version ...int) model.MatchResult {
	t.Helper()
	f.n++
	id := fmt.Sprintf("00000000-0000-4000-8000-%012d", f.n)
	m := model.MatchStart{MatchID: id, WorldID: f.world, ArenaID: fmt.Sprint("duck-", f.n), GameType: "duck", Version: model.Version{MapID: "fixture", MapVersion: 1, RulesID: "duck-v1", RulesVersion: 1, RulesChecksum: duck.RulesChecksum(), ProtocolVersion: 1}, PreparedAt: int64(f.n * 100000), Actors: []model.Actor{{ActorID: id, ActorType: "human", UserID: f.users[user], DisplayName: "Synthetic"}}}
	if len(version) > 0 {
		m.RulesVersion = version[0]
	}
	// Release the prior completed fixture's durable claim as the return flow does.
	_, e := f.s.Pool.Exec(context.Background(), `DELETE FROM active_match_users WHERE user_id=$1`, f.users[user])
	must(t, e)
	must(t, (&lobby.PGStore{Pool: f.s.Pool}).Persist(context.Background(), m, f.users[user], id))
	return model.MatchResult{Version: m.Version, MatchID: id, WorldID: f.world, ArenaID: m.ArenaID, GameType: "duck", PreparedAt: m.PreparedAt, StartsAt: m.PreparedAt + 3000, FinishedAt: m.PreparedAt + 63000, Participants: []model.ParticipantResult{{Actor: m.Actors[0], Outcome: outcome, Score: &score, Duck: &model.DuckResult{Hits: hits, Shots: shots}}}}
}
func (f *fixture) save(t *testing.T, r model.MatchResult) {
	t.Helper()
	must(t, f.h.Enqueue(context.Background(), r))
	_, e := f.h.Finalize(context.Background(), r.MatchID)
	must(t, e)
}
func TestProjectionUniquePBChampionAndPrivateReads(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	b, e := f.s.Board(ctx, f.users[0], f.world, "duck-v1@1")
	must(t, e)
	if len(b.Top) != 0 || b.PersonalBest != nil || b.LastResult != nil || len(b.Changes) != 0 {
		t.Fatal("invented empty board", b)
	}
	r := f.result(t, 0, 100, 1, 3, "finished")
	f.save(t, r)
	f.save(t, r)
	f.save(t, f.result(t, 0, 200, 2, 6, "finished"))
	f.save(t, f.result(t, 0, 100, 1, 1, "finished"))
	f.save(t, f.result(t, 1, 200, 2, 5, "finished"))
	f.save(t, f.result(t, 2, 300, 3, 10, "finished"))
	f.save(t, f.result(t, 3, 400, 4, 10, "incomplete"))
	b, e = f.s.Board(ctx, f.users[0], f.world, "duck-v1@1")
	must(t, e)
	if len(b.Top) != 3 || b.Top[0].UserID != f.users[2] || b.Top[1].UserID != f.users[1] || b.Top[2].UserID != f.users[0] || b.PersonalBest.Score != 200 || b.LastResult.Score != 100 || len(b.Changes) != 3 {
		t.Fatalf("board %+v", b)
	}
	if b.Changes[2].UserID != f.users[0] || b.Changes[1].UserID != f.users[1] || b.Changes[0].UserID != f.users[2] {
		t.Fatal(b.Changes)
	}
	if _, e = f.s.Result(ctx, f.users[1], f.world, r.MatchID); e != nil {
		t.Fatal("member summary", e)
	}
	if _, e = f.h.Detail(ctx, f.users[1], r.MatchID); !errors.Is(e, history.ErrNotFound) {
		t.Fatal("private history leaked", e)
	}
	_, e = f.s.Pool.Exec(ctx, `UPDATE world_members SET active=false WHERE user_id=$1`, f.users[0])
	must(t, e)
	if _, e = f.s.Board(ctx, f.users[0], f.world, "duck-v1@1"); !errors.Is(e, ErrNotFound) {
		t.Fatal("former member board", e)
	}
	if _, e = f.s.Result(ctx, f.users[0], f.world, r.MatchID); !errors.Is(e, ErrNotFound) {
		t.Fatal("former member summary", e)
	}
	if _, e = f.h.Detail(ctx, f.users[0], r.MatchID); e != nil {
		t.Fatal("own history lost", e)
	}
}
func TestFinalizationRowLockAtomicRollbackAndConcurrentResults(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	a := f.result(t, 0, 100, 1, 3, "finished")
	b := f.result(t, 1, 200, 2, 4, "finished")
	must(t, f.h.Enqueue(ctx, a))
	must(t, f.h.Enqueue(ctx, b))
	// An explicit partition row lock must fence both different match finalizers.
	tx, e := f.s.Pool.Begin(ctx)
	must(t, e)
	defer tx.Rollback(ctx)
	must(t, lockBoard(ctx, tx, f.world, "duck-v1@1"))
	short, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if _, e = f.h.Finalize(short, a.MatchID); e == nil {
		t.Fatal("finalizer ignored board lock")
	}
	var n int
	must(t, f.s.Pool.QueryRow(ctx, `SELECT count(*) FROM duck_results`).Scan(&n))
	if n != 0 {
		t.Fatal("partial detail escaped failed transaction")
	}
	must(t, tx.Commit(ctx))
	_, e = f.s.Pool.Exec(ctx, `CREATE FUNCTION reject_completion() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.status='completed' THEN RAISE EXCEPTION 'synthetic'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_completion BEFORE UPDATE ON matches FOR EACH ROW EXECUTE FUNCTION reject_completion()`)
	must(t, e)
	if _, e = f.h.Finalize(ctx, a.MatchID); e == nil {
		t.Fatal("fault not injected")
	}
	must(t, f.s.Pool.QueryRow(ctx, `SELECT count(*) FROM leaderboard_entries`).Scan(&n))
	if n != 0 {
		t.Fatal("PB escaped rollback")
	}
	_, e = f.s.Pool.Exec(ctx, `DROP TRIGGER reject_completion ON matches`)
	must(t, e)
	var wg sync.WaitGroup
	for _, r := range []model.MatchResult{a, b, a, b} {
		wg.Add(1)
		go func(r model.MatchResult) {
			defer wg.Done()
			if _, err := f.h.Finalize(ctx, r.MatchID); err != nil {
				t.Error(err)
			}
		}(r)
	}
	wg.Wait()
	board, e := f.s.Board(ctx, f.users[0], f.world, "duck-v1@1")
	must(t, e)
	if len(board.Top) != 2 || board.Top[0].UserID != f.users[1] || len(board.Changes) < 1 || len(board.Changes) > 2 || board.Changes[0].UserID != f.users[1] {
		t.Fatal(board)
	}
	must(t, f.s.Pool.QueryRow(ctx, `SELECT count(*) FROM duck_results`).Scan(&n))
	if n != 2 {
		t.Fatal("detail lost or duplicated", n)
	}
}
func TestExcludedAccountsAndRulesetIsolation(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	r := f.result(t, 0, 100, 1, 1, "finished")
	f.save(t, r)
	// Future rulesets use the same generic projection contract; current duck engine
	// still rejects unknown rules. No future rules are enabled by this fixture.
	tx, e := f.s.Pool.Begin(ctx)
	must(t, e)
	defer tx.Rollback(ctx)
	r = f.result(t, 0, 100, 1, 1, "finished", 2)
	must(t, ProjectResult(ctx, tx, r))
	must(t, tx.Commit(ctx))
	b, e := f.s.Board(ctx, f.users[0], f.world, "duck-v1@2")
	must(t, e)
	if len(b.Top) != 1 || len(b.Rulesets) != 2 {
		t.Fatal(b)
	}
	for _, kind := range []string{"bot", "guest", "incomplete", "assisted"} {
		tx, e := f.s.Pool.Begin(ctx)
		must(t, e)
		bad := f.result(t, 0, 100, 1, 1, "finished", 3)
		bad.Participants = append([]model.ParticipantResult{}, r.Participants...)
		switch kind {
		case "bot":
			bad.Participants[0].ActorType = "bot"
			bad.Participants[0].UserID = ""
		case "guest":
			bad.Participants[0].UserID = ""
		case "incomplete":
			bad.Participants[0].Outcome = "incomplete"
		case "assisted":
			bad.Participants[0].BotAssisted = true
		}
		must(t, ProjectResult(ctx, tx, bad))
		must(t, tx.Commit(ctx))
	}
	b, e = f.s.Board(ctx, f.users[0], f.world, "duck-v1@3")
	must(t, e)
	if len(b.Top) != 0 {
		t.Fatal("ineligible podium", b)
	}
}
func TestAccountRemovalSerializesBothOrdersAndReplay(t *testing.T) {
	for _, removeFirst := range []bool{true, false} {
		t.Run(fmt.Sprint(removeFirst), func(t *testing.T) {
			f := setup(t)
			ctx := context.Background()
			f.save(t, f.result(t, 1, 100, 1, 1, "finished"))
			r := f.result(t, 0, 200, 2, 2, "finished")
			must(t, f.h.Enqueue(ctx, r))
			tx, e := f.s.Pool.Begin(ctx)
			must(t, e)
			defer tx.Rollback(ctx)
			if removeFirst {
				must(t, RemoveAccount(ctx, tx, f.users[0]))
			} else {
				must(t, ProjectResult(ctx, tx, r))
			}
			short, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			if removeFirst {
				if _, e = f.h.Finalize(short, r.MatchID); e == nil {
					t.Fatal("projection bypassed account lock")
				}
			} else {
				other, e := f.s.Pool.Begin(ctx)
				must(t, e)
				if e = RemoveAccount(short, other, f.users[0]); e == nil {
					t.Fatal("removal bypassed account lock")
				}
				other.Rollback(ctx)
			}
			must(t, tx.Rollback(ctx)) // A rolled-back exclusion/projection must not persist.
			f.save(t, r)
			tx, e = f.s.Pool.Begin(ctx)
			must(t, e)
			must(t, RemoveAccount(ctx, tx, f.users[0]))
			must(t, tx.Commit(ctx))
			tx, e = f.s.Pool.Begin(ctx)
			must(t, e)
			r = f.result(t, 0, 200, 2, 2, "finished", 99)
			must(t, ProjectResult(ctx, tx, r))
			must(t, RemoveAccount(ctx, tx, f.users[0]))
			must(t, tx.Commit(ctx))
			b, e := f.s.Board(ctx, f.users[1], f.world, "duck-v1@1")
			must(t, e)
			if len(b.Top) != 1 || b.Top[0].UserID != f.users[1] || len(b.Changes) != 2 || b.Changes[0].UserID != "" || b.Changes[0].DisplayName != "Akun dihapus" || len(b.AdministrativeChanges) != 1 {
				t.Fatal(b)
			}
			b, e = f.s.Board(ctx, f.users[1], f.world, "duck-v1@99")
			must(t, e)
			if len(b.Top) != 0 {
				t.Fatal("removed account repopulated unseen partition", b)
			}
		})
	}
}

func TestCommittedRemovalBeforeUnseenPartitionFinalization(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	r := f.result(t, 0, 100, 1, 1, "finished")
	must(t, f.h.Enqueue(ctx, r))
	tx, e := f.s.Pool.Begin(ctx)
	must(t, e)
	must(t, RemoveAccount(ctx, tx, f.users[0]))
	must(t, tx.Commit(ctx))
	_, e = f.h.Finalize(ctx, r.MatchID)
	must(t, e)
	var count int
	must(t, f.s.Pool.QueryRow(ctx, `SELECT count(*) FROM leaderboard_entries`).Scan(&count))
	if count != 0 {
		t.Fatal("late job recreated removed account", count)
	}
	must(t, f.s.Pool.QueryRow(ctx, `SELECT count(*) FROM duck_results`).Scan(&count))
	if count != 1 {
		t.Fatal("immutable history not finalized", count)
	}
	_, e = f.h.Finalize(ctx, r.MatchID)
	must(t, e)
	b, e := f.s.Board(ctx, f.users[1], f.world, "duck-v1@1")
	must(t, e)
	if len(b.Top) != 0 || len(b.Changes) != 0 {
		t.Fatal(b)
	}
}
func TestBannedAndArchivedWorldReadBoundaries(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	r := f.result(t, 0, 100, 1, 1, "finished")
	f.save(t, r)
	_, e := f.s.Pool.Exec(ctx, `INSERT INTO world_bans(world_id,user_id,banned_by)VALUES($1,$2,$3)`, f.world, f.users[1], f.users[0])
	must(t, e)
	if _, e = f.s.Top(ctx, f.users[1], f.world, "duck-v1@1", 3); !errors.Is(e, ErrNotFound) {
		t.Fatal("banned board", e)
	}
	if _, e = f.s.RecentChampionChanges(ctx, f.users[1], f.world, 3); !errors.Is(e, ErrNotFound) {
		t.Fatal("banned champion history", e)
	}
	_, e = f.s.Pool.Exec(ctx, `UPDATE worlds SET archived_at=now() WHERE id=$1`, f.world)
	must(t, e)
	if _, e = f.s.Result(ctx, f.users[0], f.world, r.MatchID); !errors.Is(e, ErrNotFound) {
		t.Fatal("archived summary", e)
	}
	if _, e = f.s.Board(ctx, f.users[0], f.world, "duck-v1@1"); !errors.Is(e, ErrNotFound) {
		t.Fatal("archived board", e)
	}
	if _, e = f.h.Detail(ctx, f.users[0], r.MatchID); e != nil {
		t.Fatal("personal history lost after archive", e)
	}
}

func TestHistoricalBackfillOrderedRestartAndRollback(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	// Simulate accepted pre009 completion: detail/history exist, no board projection.
	f.h.Projectors["duck"] = duck.Project
	a := f.result(t, 0, 100, 1, 1, "finished")
	f.save(t, a)
	b := f.result(t, 1, 200, 2, 3, "finished")
	f.save(t, b)
	f.save(t, f.result(t, 2, 500, 1, 1, "incomplete"))
	delete(f.h.Projectors, "duck")
	f.save(t, f.result(t, 3, 500, 1, 1, "finished")) // Generic pre-Duck history has no validated detail.
	tx, e := f.s.Pool.Begin(ctx)
	must(t, e)
	must(t, RemoveAccount(ctx, tx, f.users[1]))
	must(t, tx.Commit(ctx))
	_, e = f.s.Pool.Exec(ctx, `CREATE FUNCTION reject_receipt() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic backfill receipt'; END $$; CREATE TRIGGER reject_receipt BEFORE INSERT ON leaderboard_projections FOR EACH ROW EXECUTE FUNCTION reject_receipt()`)
	must(t, e)
	if e = f.s.Backfill(ctx); e == nil {
		t.Fatal("receipt rollback fault missing")
	}
	var n int
	must(t, f.s.Pool.QueryRow(ctx, `SELECT count(*) FROM leaderboard_entries`).Scan(&n))
	if n != 0 {
		t.Fatal("backfill escaped receipt rollback")
	}
	_, e = f.s.Pool.Exec(ctx, `DROP TRIGGER reject_receipt ON leaderboard_projections`)
	must(t, e)
	must(t, f.s.Backfill(ctx))
	must(t, f.s.Backfill(ctx))
	board, e := f.s.Board(ctx, f.users[0], f.world, "duck-v1@1")
	must(t, e)
	if len(board.Top) != 1 || board.Top[0].UserID != f.users[0] || len(board.Changes) != 1 || board.Changes[0].ResultID != a.MatchID {
		t.Fatal(board)
	}
	must(t, f.s.Pool.QueryRow(ctx, `SELECT count(*) FROM duck_results`).Scan(&n))
	if n != 3 {
		t.Fatal("backfill duplicated/lost Duck detail")
	}
	must(t, f.s.Pool.QueryRow(ctx, `SELECT count(*) FROM leaderboard_projections`).Scan(&n))
	if n != 4 {
		t.Fatal("missing completion receipts", n)
	}
	d, e := f.h.Detail(ctx, f.users[0], a.MatchID)
	must(t, e)
	if *d.Result.Participants[0].Score != 100 {
		t.Fatal("history rewritten")
	}
}

func TestBackfillUsesFinishedTimeOrderAndCancellation(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	f.h.Projectors["duck"] = duck.Project
	a := f.result(t, 0, 100, 1, 1, "finished")
	b := f.result(t, 1, 200, 2, 3, "finished")
	c := f.result(t, 2, 300, 3, 4, "finished")
	f.save(t, c)
	f.save(t, b)
	f.save(t, a) // Completion insertion order differs from result time.
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if e := f.s.Backfill(canceled); e == nil {
		t.Fatal("ignored startup cancellation")
	}
	must(t, f.s.Backfill(ctx))
	board, e := f.s.Board(ctx, f.users[0], f.world, "duck-v1@1")
	must(t, e)
	if len(board.Changes) != 3 || board.Changes[0].ResultID != c.MatchID || board.Changes[1].ResultID != b.MatchID || board.Changes[2].ResultID != a.MatchID {
		t.Fatal("historical import order", board.Changes)
	}
	must(t, f.s.Backfill(ctx))
	board, e = f.s.Board(ctx, f.users[0], f.world, "duck-v1@1")
	must(t, e)
	if len(board.Changes) != 3 {
		t.Fatal("backfill duplicated victories")
	}
}
