package lobby

import (
	"context"
	"errors"
	"fmt"
	"patra-world-social/server/internal/games/model"
	"sync"
	"testing"
	"time"
)

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time          { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *testClock) advance(d time.Duration) { c.mu.Lock(); c.now = c.now.Add(d); c.mu.Unlock() }

type memoryStore struct {
	mu      sync.Mutex
	rows    map[string]model.MatchStart
	fail    bool
	hold    chan struct{}
	entered chan struct{}
}

func (s *memoryStore) Persist(_ context.Context, m model.MatchStart, host, command string) error {
	if s.entered != nil {
		select {
		case s.entered <- struct{}{}:
		default:
		}
	}
	if s.hold != nil {
		<-s.hold
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail {
		return errors.New("storage unavailable")
	}
	s.rows[m.MatchID] = m
	return nil
}
func (s *memoryStore) Lookup(_ context.Context, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.rows[id]
	return ok, nil
}
func (s *memoryStore) Schedule(context.Context, string, int64) error     { return nil }
func (s *memoryStore) Terminal(context.Context, string, string) error    { return nil }
func (s *memoryStore) ReleaseUser(context.Context, string, string) error { return nil }

type testEngine struct {
	started int64
	events  []string
}

func (e *testEngine) Start(at int64)             { e.started = at }
func (e *testEngine) Tick(time.Time) bool        { return false }
func (e *testEngine) Presence(user, kind string) { e.events = append(e.events, user+":"+kind) }
func (e *testEngine) Close()                     {}
func setup(t *testing.T) (*Service, *memoryStore, *testClock) {
	t.Helper()
	c := &testClock{now: time.Unix(100, 0)}
	store := &memoryStore{rows: map[string]model.MatchStart{}}
	s := New(store, c, model.Version{})
	t.Cleanup(s.Close)
	return s, store, c
}
func join(t *testing.T, s *Service, user, arena string) Lobby {
	t.Helper()
	l, e := s.JoinLobby(context.Background(), JoinContext{UserID: user, WorldID: "world", DisplayName: user}, arena)
	if e != nil {
		t.Fatal(e)
	}
	return l
}
func ready(t *testing.T, s *Service, user string, l Lobby) {
	t.Helper()
	if e := s.SetReady(context.Background(), user, l.ID, true); e != nil {
		t.Fatal(e)
	}
}
func register(t *testing.T, s *Service, game string) {
	t.Helper()
	if e := s.Register(game, func(model.MatchStart) (Engine, error) { return &testEngine{}, nil }); e != nil {
		t.Fatal(e)
	}
}

func TestLobbyOwnershipSlotsAndUnavailableEngine(t *testing.T) {
	s, _, _ := setup(t)
	l := join(t, s, "a", "race")
	join(t, s, "b", "race")
	if _, e := s.JoinLobby(context.Background(), JoinContext{UserID: "a", WorldID: "world"}, "penalty"); !errors.Is(e, ErrBusy) {
		t.Fatal(e)
	}
	for _, u := range []string{"c", "d", "e"} {
		join(t, s, u, "race")
	}
	if _, e := s.JoinLobby(context.Background(), JoinContext{UserID: "f", WorldID: "world"}, "race"); !errors.Is(e, ErrFull) {
		t.Fatal(e)
	}
	if e := s.SetReady(context.Background(), "spoof", l.ID, true); !errors.Is(e, ErrForbidden) {
		t.Fatal(e)
	}
	if _, e := s.StartMatch(context.Background(), "a", l.ID, "start"); !errors.Is(e, ErrUnavailable) {
		t.Fatal(e)
	}
	if e := s.Leave(context.Background(), "a", "world"); e != nil {
		t.Fatal(e)
	}
	all, _ := s.List(context.Background(), "world")
	if all[0].HostID != "b" {
		t.Fatal(all)
	}
	for _, u := range []string{"b", "c", "d", "e"} {
		s.Leave(context.Background(), u, "world")
	}
	all, _ = s.List(context.Background(), "world")
	if all[0].State != "idle" {
		t.Fatal(all)
	}
}
func TestConcurrentStartFreezesRosterAndCountsFullCountdownAfterStorage(t *testing.T) {
	s, store, c := setup(t)
	register(t, s, "race")
	l := join(t, s, "a", "race")
	join(t, s, "b", "race")
	ready(t, s, "a", l)
	if _, e := s.StartMatch(context.Background(), "a", l.ID, "first"); !errors.Is(e, ErrNotReady) {
		t.Fatal(e)
	}
	ready(t, s, "b", l)
	store.hold = make(chan struct{})
	store.entered = make(chan struct{}, 1)
	type result struct {
		m model.MatchStart
		e error
	}
	out := make(chan result, 2)
	go func() { m, e := s.StartMatch(context.Background(), "a", l.ID, "first"); out <- result{m, e} }()
	<-store.entered
	go func() { m, e := s.StartMatch(context.Background(), "a", l.ID, "first"); out <- result{m, e} }()
	c.advance(5 * time.Second)
	if _, e := s.StartMatch(context.Background(), "a", l.ID, "other"); !errors.Is(e, ErrBusy) {
		t.Fatal(e)
	}
	late := join(t, s, "late", "race")
	if len(late.Spectators) != 1 {
		t.Fatal(late)
	}
	close(store.hold)
	a, b := <-out, <-out
	if a.e != nil || b.e != nil || a.m.MatchID != b.m.MatchID {
		t.Fatal(a, b)
	}
	if a.m.StartsAt != c.Now().Add(3*time.Second).UnixMilli() || len(a.m.Actors) != 5 {
		t.Fatal(a.m)
	}
	store.mu.Lock()
	n := len(store.rows)
	store.mu.Unlock()
	if n != 1 {
		t.Fatal(n)
	}
	if _, e := s.StartMatch(context.Background(), "b", l.ID, "first"); !errors.Is(e, ErrForbidden) {
		t.Fatal(e)
	}
}
func TestStartFailureRestoresLobbyAndCanRetry(t *testing.T) {
	s, store, _ := setup(t)
	register(t, s, "race")
	l := join(t, s, "a", "race")
	ready(t, s, "a", l)
	store.fail = true
	if _, e := s.StartMatch(context.Background(), "a", l.ID, "start"); e == nil {
		t.Fatal("started without storage")
	}
	all, _ := s.List(context.Background(), "world")
	if all[0].State != "lobby" || !all[0].Players[0].Ready {
		t.Fatal(all)
	}
	store.fail = false
	if _, e := s.StartMatch(context.Background(), "a", l.ID, "retry"); e != nil {
		t.Fatal(e)
	}
}
func TestThreeGameFixtureAndGrace(t *testing.T) {
	s, _, c := setup(t)
	for _, game := range []string{"race", "penalty", "duck"} {
		register(t, s, game)
	}
	for _, v := range [][2]string{{"a", "race"}, {"b", "penalty"}, {"c", "duck-1"}, {"d", "duck-2"}, {"e", "duck-3"}, {"f", "duck-4"}} {
		l := join(t, s, v[0], v[1])
		ready(t, s, v[0], l)
		if v[1] == "penalty" {
			if e := s.ChooseBot(context.Background(), v[0], l.ID, true); e != nil {
				t.Fatal(e)
			}
		}
		if _, e := s.StartMatch(context.Background(), v[0], l.ID, "start-"+v[0]); e != nil {
			t.Fatal(e)
		}
	}
	s.Notify("disconnect", "world", "a")
	s.Sync(context.Background())
	if !s.Busy("a") {
		t.Fatal("lost grace seat")
	}
	c.advance(14 * time.Second)
	s.Notify("resume", "world", "a")
	s.Sync(context.Background())
	c.advance(2 * time.Second)
	s.Sync(context.Background())
	if !s.Busy("a") {
		t.Fatal("resume lost seat")
	}
	s.Notify("disconnect", "world", "a")
	s.Sync(context.Background())
	c.advance(15 * time.Second)
	s.Sync(context.Background())
	deadline := time.After(time.Second)
	for s.Busy("a") {
		select {
		case <-s.Changed:
			s.Sync(context.Background())
		case <-deadline:
			t.Fatal("expired game seat retained")
		}
	}
	if !s.Busy("b") {
		t.Fatal("other game lost seat")
	}
}

func TestCoalescedDisconnectResumeReachesEngineAndMarksAssistance(t *testing.T) {
	s, _, c := setup(t)
	register(t, s, "race")
	l := join(t, s, "a", "race")
	ready(t, s, "a", l)
	m, e := s.StartMatch(context.Background(), "a", l.ID, "start")
	if e != nil {
		t.Fatal(e)
	}
	// Hold owner while both transport transitions arrive, forcing one mailbox item.
	entered, release := make(chan struct{}), make(chan struct{})
	go s.call(context.Background(), func() { close(entered); <-release })
	<-entered
	s.Notify("disconnect", "world", "a")
	c.advance(time.Second)
	s.Notify("resume", "world", "a")
	close(release)
	s.Sync(context.Background())
	s.WithEngine(context.Background(), m.MatchID, "a", func(engine Engine) error {
		events := engine.(*testEngine).events
		if len(events) != 2 || events[0] != "a:disconnect" || events[1] != "a:resume" {
			t.Errorf("lost or duplicated transitions: %v", events)
		}
		return nil
	})
	all, _ := s.List(context.Background(), "world")
	if !all[0].Players[0].BotAssisted || !all[0].Players[0].Connected {
		t.Fatal(all)
	}
}
func TestLeaveWhileStartPersistsDoesNotActivateAndFailureTransfersHost(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			s, store, _ := setup(t)
			engine := &testEngine{}
			s.Register("race", func(model.MatchStart) (Engine, error) { return engine, nil })
			l := join(t, s, "a", "race")
			if fail {
				join(t, s, "b", "race")
				ready(t, s, "b", l)
			}
			ready(t, s, "a", l)
			store.hold = make(chan struct{})
			store.entered = make(chan struct{}, 1)
			store.fail = fail
			out := make(chan error, 1)
			go func() { _, e := s.StartMatch(context.Background(), "a", l.ID, "start"); out <- e }()
			<-store.entered
			s.Notify("leave", "world", "a")
			s.Sync(context.Background())
			close(store.hold)
			<-out
			all, _ := s.List(context.Background(), "world")
			if fail {
				if all[0].HostID != "b" || len(all[0].Players) != 1 || all[0].State != "lobby" {
					t.Fatal(all)
				}
			} else {
				if engine.started != 0 {
					t.Fatal("engine activated with no human")
				}
				if all[0].State == "countdown" || all[0].State == "running" {
					t.Fatal("activated departed roster", all)
				}
			}
		})
	}
}

type slowScheduleStore struct {
	*memoryStore
	scheduled chan struct{}
	release   chan struct{}
}

func (s *slowScheduleStore) Schedule(ctx context.Context, id string, at int64) error {
	close(s.scheduled)
	select {
	case <-s.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func TestScheduleProjectionCannotDelayCountdownOrOwnerAndShutdownCancelsIt(t *testing.T) {
	c := &testClock{now: time.Unix(100, 0)}
	store := &slowScheduleStore{memoryStore: &memoryStore{rows: map[string]model.MatchStart{}}, scheduled: make(chan struct{}), release: make(chan struct{})}
	s := New(store, c, model.Version{})
	register(t, s, "race")
	l := join(t, s, "a", "race")
	ready(t, s, "a", l)
	m, e := s.StartMatch(context.Background(), "a", l.ID, "start")
	if e != nil {
		t.Fatal(e)
	}
	<-store.scheduled
	c.advance(2 * time.Second)
	all, e := s.List(context.Background(), "world")
	if e != nil || all[0].State != "countdown" || m.StartsAt != 103000 {
		t.Fatal(all, m, e)
	}
	c.advance(time.Second)
	all, _ = s.List(context.Background(), "world")
	if all[0].State != "running" {
		t.Fatal(all)
	}
	closed := make(chan struct{})
	go func() { s.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("schedule worker did not cancel on shutdown")
	}
}

type finishingEngine struct {
	testEngine
	finished bool
}

func (e *finishingEngine) Tick(time.Time) bool { return e.finished }
func TestEngineFinishAndFinalizerHooksStayOnOwner(t *testing.T) {
	s, _, c := setup(t)
	engine := &finishingEngine{}
	s.Register("race", func(model.MatchStart) (Engine, error) { return engine, nil })
	l := join(t, s, "a", "race")
	ready(t, s, "a", l)
	m, e := s.StartMatch(context.Background(), "a", l.ID, "start")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.WithEngine(context.Background(), m.MatchID, "spoof", func(Engine) error { t.Fatal("spoof reached engine"); return nil }); !errors.Is(e, ErrForbidden) {
		t.Fatal(e)
	}
	c.advance(3 * time.Second)
	s.WithEngine(context.Background(), m.MatchID, "a", func(e Engine) error { e.(*finishingEngine).finished = true; return nil })
	s.Sync(context.Background())
	all, _ := s.List(context.Background(), "world")
	if all[0].State != "finalizing" {
		t.Fatal(all)
	}
	if e = s.Transition(context.Background(), m.MatchID, "results"); e != nil {
		t.Fatal(e)
	}
	if e = s.ReleaseMatch(m.MatchID, "completed"); e != nil {
		t.Fatal(e)
	}
	if s.Busy("a") {
		t.Fatal("completed ownership retained")
	}
}

func TestDeparturesCannotOverwriteNaturallyFinalizingMatch(t *testing.T) {
	s, _, c := setup(t)
	engine := &finishingEngine{}
	s.Register("penalty", func(model.MatchStart) (Engine, error) { return engine, nil })
	l := join(t, s, "a", "penalty")
	s.ChooseBot(context.Background(), "a", l.ID, true)
	ready(t, s, "a", l)
	m, e := s.StartMatch(context.Background(), "a", l.ID, "start")
	if e != nil {
		t.Fatal(e)
	}
	c.advance(3 * time.Second)
	s.WithEngine(context.Background(), m.MatchID, "a", func(e Engine) error { e.(*finishingEngine).finished = true; return nil })
	s.Sync(context.Background())
	s.Leave(context.Background(), "a", "world")
	all, _ := s.List(context.Background(), "world")
	if all[1].State != "finalizing" {
		t.Fatal("natural result replaced by departure", all)
	}
}

func TestDurableStartCompletionWaitsForSpaceInBoundedOwnerQueue(t *testing.T) {
	s, store, _ := setup(t)
	register(t, s, "race")
	l := join(t, s, "a", "race")
	ready(t, s, "a", l)
	store.hold = make(chan struct{})
	store.entered = make(chan struct{}, 1)
	finished := make(chan error, 1)
	go func() { _, e := s.StartMatch(context.Background(), "a", l.ID, "start"); finished <- e }()
	<-store.entered
	entered, release := make(chan struct{}), make(chan struct{})
	go s.control(func() { close(entered); <-release })
	<-entered
	for i := 0; i < cap(s.requests); i++ {
		s.requests <- request{fn: func() {}, done: make(chan struct{})}
	}
	close(store.hold)
	close(release)
	select {
	case e := <-finished:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("durable start completion dropped on full queue")
	}
}

func TestRegistryStartDelayForRaceAndDuck(t *testing.T) {
	for _, game := range []string{"race", "penalty", "duck"} {
		t.Run(game, func(t *testing.T) {
			s, _, c := setup(t)
			register(t, s, game)
			arenaID := game
			if game == "duck" {
				arenaID = "duck-1"
			}
			l := join(t, s, "a", arenaID)
			ready(t, s, "a", l)
			if game == "penalty" {
				s.ChooseBot(context.Background(), "a", l.ID, true)
			}
			m, e := s.StartMatch(context.Background(), "a", l.ID, "start")
			if e != nil {
				t.Fatal(e)
			}
			delay := time.Duration(0)
			state := "running"
			if game == "race" || game == "duck" {
				delay = 3 * time.Second
				state = "countdown"
			}
			if m.StartsAt != c.Now().Add(delay).UnixMilli() {
				t.Fatalf("%s official start=%d, expected %d", game, m.StartsAt, c.Now().Add(delay).UnixMilli())
			}
			all, _ := s.List(context.Background(), "world")
			for _, v := range all {
				if v.ID == l.ID && v.State != state {
					t.Fatalf("%s state=%s, expected %s", game, v.State, state)
				}
			}
		})
	}
}

func TestReturnGenerationFencesRejoinAndAcknowledgesOnlyDeliveredGeneration(t *testing.T) {
	s, _, _ := setup(t)
	ctx := context.Background()
	l := join(t, s, "a", "race")
	join(t, s, "b", "race")
	s.Leave(ctx, "a", "world")
	changes, _ := s.DrainChanges(ctx)
	if len(changes.Returns) != 1 {
		t.Fatal(changes)
	}
	old := changes.Returns[0]
	rejoined := join(t, s, "a", "race")
	if rejoined.ID != l.ID {
		t.Fatal("fixture must rejoin same lobby")
	}
	if s.WithReturn(old, func() { t.Fatal("old generation applied while busy") }) {
		t.Fatal("stale return accepted")
	}
	s.Leave(ctx, "a", "world")
	s.AckReturn(old)
	changes, _ = s.DrainChanges(ctx)
	if len(changes.Returns) != 1 || changes.Returns[0].Generation == old.Generation {
		t.Fatal("old acknowledgement removed newer return", changes)
	}
	if s.WithReturn(old, func() { t.Fatal("old generation applied after new claim released") }) {
		t.Fatal("stale generation accepted after release")
	}
	current := changes.Returns[0]
	applied := false
	if !s.WithReturn(current, func() { applied = true }) || !applied {
		t.Fatal("current return rejected")
	}
	s.AckReturn(current)
	changes, _ = s.DrainChanges(ctx)
	if len(changes.Returns) != 0 {
		t.Fatal(changes)
	}
}
