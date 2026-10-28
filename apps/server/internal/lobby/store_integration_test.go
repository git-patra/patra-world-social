//go:build integration

package lobby

import (
	"context"
	"errors"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/testutil"
	"sync"
	"testing"
	"time"
)

func TestPostgresConcurrentStartAndActiveClaims(t *testing.T) {
	p := testutil.Database(t)
	users, _ := testutil.Users(t, p, 3)
	ctx := context.Background()
	var world string
	if e := p.QueryRow(ctx, `INSERT INTO worlds(name,owner_id) VALUES('synthetic',$1) RETURNING id`, users[0]).Scan(&world); e != nil {
		t.Fatal(e)
	}
	for _, u := range users {
		if _, e := p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role) VALUES($1,$2,'member')`, world, u); e != nil {
			t.Fatal(e)
		}
	}
	store := &PGStore{Pool: p}
	s := New(store, &testClock{}, model.Version{MapID: "test", MapVersion: 1, RulesID: "test", RulesVersion: 1, ProtocolVersion: 1})
	defer s.Close()
	register(t, s, "race")
	l, e := s.JoinLobby(ctx, JoinContext{WorldID: world, UserID: users[0]}, "race")
	if e != nil {
		t.Fatal(e)
	}
	ready(t, s, users[0], l)
	var wg sync.WaitGroup
	out := make(chan model.MatchStart, 12)
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, e := s.StartMatch(ctx, users[0], l.ID, "same-command")
			out <- m
			errs <- e
		}()
	}
	wg.Wait()
	close(out)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	var id string
	for m := range out {
		if id != "" && m.MatchID != id {
			t.Fatal("duplicate match")
		}
		id = m.MatchID
	}
	var matches, actors, claims int
	p.QueryRow(ctx, `SELECT count(*) FROM matches`).Scan(&matches)
	p.QueryRow(ctx, `SELECT count(*) FROM match_participants`).Scan(&actors)
	p.QueryRow(ctx, `SELECT count(*) FROM active_match_users`).Scan(&claims)
	if matches != 1 || actors != 5 || claims != 1 {
		t.Fatal(matches, actors, claims)
	}
	other := model.MatchStart{MatchID: idForTest(), WorldID: world, ArenaID: "penalty", GameType: "penalty", Actors: []model.Actor{{ActorID: idForTest(), ActorType: "human", UserID: users[0], DisplayName: "test"}}, Version: model.Version{}}
	if e = store.Persist(ctx, other, users[0], "cross-game"); e == nil {
		t.Fatal("double busy claim accepted")
	}
	other.MatchID = idForTest()
	other.ArenaID = "race"
	other.Actors[0].UserID = users[1]
	if e = store.Persist(ctx, other, users[1], "same-arena"); e == nil {
		t.Fatal("double arena accepted")
	}
	if e = s.ReleaseMatch(id, "abandoned"); e != nil {
		t.Fatal(e)
	}
	p.QueryRow(ctx, `SELECT count(*) FROM active_match_users`).Scan(&claims)
	if claims != 0 {
		t.Fatal(claims)
	}
}
func idForTest() string { return id() }

type failStore struct {
	Store
	afterCommit       bool
	unavailableLookup bool
	precommit         bool
}

func (s *failStore) Persist(ctx context.Context, m model.MatchStart, h, c string) error {
	if s.precommit {
		return errors.New("injected precommit failure")
	}
	if e := s.Store.Persist(ctx, m, h, c); e != nil {
		return e
	}
	if s.afterCommit {
		return ErrCommitUnknown
	}
	return nil
}
func (s *failStore) Lookup(ctx context.Context, id string) (bool, error) {
	if s.unavailableLookup {
		return false, errors.New("injected lookup outage")
	}
	return s.Store.Lookup(ctx, id)
}
func TestPostgresStartFailuresAndLostCommitAcknowledgement(t *testing.T) {
	p := testutil.Database(t)
	users, _ := testutil.Users(t, p, 1)
	ctx := context.Background()
	var world string
	p.QueryRow(ctx, `INSERT INTO worlds(name,owner_id) VALUES('synthetic',$1) RETURNING id`, users[0]).Scan(&world)
	p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'owner')`, world, users[0])
	store := &failStore{Store: &PGStore{Pool: p}, precommit: true}
	s := New(store, &testClock{}, model.Version{})
	defer s.Close()
	register(t, s, "race")
	l, e := s.JoinLobby(ctx, JoinContext{WorldID: world, UserID: users[0]}, "race")
	if e != nil {
		t.Fatal(e)
	}
	ready(t, s, users[0], l)
	if _, e = s.StartMatch(ctx, users[0], l.ID, "precommit"); e == nil {
		t.Fatal("countdown on failure")
	}
	var n int
	p.QueryRow(ctx, `SELECT count(*) FROM matches`).Scan(&n)
	if n != 0 {
		t.Fatal(n)
	}
	store.precommit = false
	store.afterCommit = true
	store.unavailableLookup = true
	if _, e = s.StartMatch(ctx, users[0], l.ID, "lost-ack"); !errors.Is(e, ErrUnconfirmed) {
		t.Fatal(e)
	}
	list, _ := s.List(ctx, world)
	if list[0].State != "starting" || !s.Busy(users[0]) {
		t.Fatal("freed uncertain claim", list)
	}
	store.unavailableLookup = false
	m, e := s.StartMatch(ctx, users[0], l.ID, "lost-ack")
	if e != nil {
		t.Fatal(e)
	}
	retry, e := s.StartMatch(ctx, users[0], l.ID, "lost-ack")
	if e != nil || retry.MatchID != m.MatchID || retry.StartsAt != m.StartsAt {
		t.Fatal(e, retry, m)
	}
	p.QueryRow(ctx, `SELECT count(*) FROM matches`).Scan(&n)
	if n != 1 {
		t.Fatal(n)
	}
}

func TestPostgresParticipantWriteFailureRollsBackRunningAndClaims(t *testing.T) {
	p := testutil.Database(t)
	users, _ := testutil.Users(t, p, 1)
	ctx := context.Background()
	var world string
	p.QueryRow(ctx, `INSERT INTO worlds(name,owner_id)VALUES('synthetic',$1)RETURNING id`, users[0]).Scan(&world)
	p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'owner')`, world, users[0])
	if _, e := p.Exec(ctx, `CREATE FUNCTION reject_participant() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic write failure'; END $$; CREATE TRIGGER reject_participant BEFORE INSERT ON match_participants FOR EACH ROW EXECUTE FUNCTION reject_participant()`); e != nil {
		t.Fatal(e)
	}
	s := New(&PGStore{Pool: p}, &testClock{}, model.Version{})
	defer s.Close()
	register(t, s, "race")
	l, e := s.JoinLobby(ctx, JoinContext{WorldID: world, UserID: users[0]}, "race")
	if e != nil {
		t.Fatal(e)
	}
	ready(t, s, users[0], l)
	if _, e = s.StartMatch(ctx, users[0], l.ID, "transaction-failure"); e == nil {
		t.Fatal("started after participant write failed")
	}
	var matches, claims int
	p.QueryRow(ctx, `SELECT count(*) FROM matches`).Scan(&matches)
	p.QueryRow(ctx, `SELECT count(*) FROM active_match_users`).Scan(&claims)
	all, _ := s.List(ctx, world)
	if matches != 0 || claims != 0 || all[0].State != "lobby" || !all[0].Players[0].Ready {
		t.Fatal(matches, claims, all)
	}
	p.Exec(ctx, `DROP TRIGGER reject_participant ON match_participants`)
	m, e := s.StartMatch(ctx, users[0], l.ID, "transaction-failure")
	if e != nil {
		t.Fatal(e)
	}
	if e = (&PGStore{Pool: p}).Terminal(ctx, m.MatchID, "completed"); !errors.Is(e, ErrForbidden) {
		t.Fatal("release fabricated completed result", e)
	}
}
func TestPostgresRevokeDuringUnconfirmedStartReconcilesWithoutEngineActivation(t *testing.T) {
	p := testutil.Database(t)
	users, _ := testutil.Users(t, p, 1)
	ctx := context.Background()
	var world string
	p.QueryRow(ctx, `INSERT INTO worlds(name,owner_id)VALUES('synthetic',$1)RETURNING id`, users[0]).Scan(&world)
	p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'owner')`, world, users[0])
	clock := &testClock{}
	store := &failStore{Store: &PGStore{Pool: p}, afterCommit: true, unavailableLookup: true}
	s := New(store, clock, model.Version{})
	defer s.Close()
	engine := &testEngine{}
	s.Register("race", func(model.MatchStart) (Engine, error) { return engine, nil })
	l, e := s.JoinLobby(ctx, JoinContext{WorldID: world, UserID: users[0]}, "race")
	if e != nil {
		t.Fatal(e)
	}
	ready(t, s, users[0], l)
	if _, e = s.StartMatch(ctx, users[0], l.ID, "lost-ack"); !errors.Is(e, ErrUnconfirmed) {
		t.Fatal(e)
	}
	p.Exec(ctx, `UPDATE world_members SET active=false WHERE user_id=$1`, users[0])
	s.Notify("leave", world, users[0])
	s.Sync(ctx)
	if !s.Busy(users[0]) {
		t.Fatal("uncertain claim freed before reconciliation")
	}
	store.unavailableLookup = false
	clock.advance(time.Second)
	s.Sync(ctx)
	deadline := time.After(2 * time.Second)
	for s.Busy(users[0]) {
		select {
		case <-s.Changed:
			s.Sync(ctx)
		case <-deadline:
			t.Fatal("revoked uncertain start failed to release")
		}
	}
	for {
		all, _ := s.List(ctx, world)
		if all[0].State == "idle" {
			break
		}
		select {
		case <-s.Changed:
			s.Sync(ctx)
		case <-deadline:
			t.Fatal("abandoned arena failed to release")
		}
	}
	var state string
	var claims int
	p.QueryRow(ctx, `SELECT status FROM matches`).Scan(&state)
	p.QueryRow(ctx, `SELECT count(*) FROM active_match_users`).Scan(&claims)
	if state != "abandoned" || claims != 0 || engine.started != 0 {
		t.Fatal(state, claims, engine.started)
	}
}

func TestPostgresLookupCannotReportAbsentBeforeStartTransactionResolves(t *testing.T) {
	p := testutil.Database(t)
	ctx := context.Background()
	tx, e := p.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	match := id()
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,7))`, match); e != nil {
		t.Fatal(e)
	}
	short, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if found, e := (&PGStore{Pool: p}).Lookup(short, match); e == nil {
		t.Fatal("unresolved transaction reported absence", found)
	}
	tx.Rollback(ctx)
	if found, e := (&PGStore{Pool: p}).Lookup(ctx, match); e != nil || found {
		t.Fatal(found, e)
	}
}
