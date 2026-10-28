package duck

import (
	"context"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/lobby"
	"sync/atomic"
	"testing"
	"time"
)

type testClock struct{ at atomic.Int64 }

func (c *testClock) Now() time.Time { return time.UnixMilli(c.at.Load()) }

type testStore struct{ persisted model.MatchStart }

func (s *testStore) Persist(_ context.Context, m model.MatchStart, _, _ string) error {
	s.persisted = m
	return nil
}
func (*testStore) Lookup(context.Context, string) (bool, error)      { return true, nil }
func (*testStore) Schedule(context.Context, string, int64) error     { return nil }
func (*testStore) Terminal(context.Context, string, string) error    { return nil }
func (*testStore) ReleaseUser(context.Context, string, string) error { return nil }
func TestRealOwnerDelayedPresenceAndDurableMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, kind string
		at, ingest int64
		outcome    string
	}{{"normal", "disconnect", 63000, 65000, "finished"}, {"equal", "disconnect", 49000, 65000, "finished"}, {"expired", "disconnect", 44000, 65000, "incomplete"}, {"leave", "leave", 63000, 65000, "incomplete"}, {"resume-natural", "resume", 65000, 65000, "finished"}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			clock := &testClock{}
			clock.at.Store(1000)
			store := &testStore{}
			g := lobby.New(store, clock, model.Version{MapID: "map", RulesID: "old"})
			defer g.Close()
			if err := g.RegisterVersioned("duck", Factory(clock), lobby.RulesMetadata{ID: "duck-v1", Version: 1, Checksum: RulesChecksum()}); err != nil {
				t.Fatal(err)
			}
			l, err := g.JoinLobby(ctx, lobby.JoinContext{UserID: "u", WorldID: "world"}, "duck-1")
			if err != nil {
				t.Fatal(err)
			}
			g.SetReady(ctx, "u", l.ID, true)
			m, err := g.StartMatch(ctx, "u", l.ID, "start")
			if err != nil {
				t.Fatal(err)
			}
			if m.StartsAt != 4000 || store.persisted.RulesID != "duck-v1" || store.persisted.MapID != "map" {
				t.Fatal(m, store.persisted)
			}
			if tc.kind == "resume" {
				clock.at.Store(63000)
				g.Notify("disconnect", "world", "u")
				g.Sync(ctx)
			}
			entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
			go func() {
				done <- g.ReadEngine(ctx, "world", m.MatchID, func(lobby.Engine) error { close(entered); <-release; return nil })
			}()
			<-entered
			clock.at.Store(tc.at)
			g.Notify(tc.kind, "world", "u")
			clock.at.Store(tc.ingest)
			close(release)
			if err = <-done; err != nil {
				t.Fatal(err)
			}
			g.Sync(ctx)
			err = g.WithEngine(ctx, m.MatchID, "", func(engine lobby.Engine) error {
				a := engine.(*Adapter)
				r, e := a.Result()
				if e != nil || r.Participants[0].Outcome != tc.outcome || r.Participants[0].BotAssisted {
					t.Fatal(r, e)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			changes, err := g.DrainChanges(ctx)
			if err != nil || len(changes.Events) != 1 || changes.Events[0].Type != "duck.finished" {
				t.Fatal("terminal event missing", changes, err)
			}
			again, _ := g.DrainChanges(ctx)
			if len(again.Events) != 0 {
				t.Fatal("terminal repeated", again)
			}
			ls, _ := g.List(ctx, "world")
			for _, v := range ls {
				if v.MatchID == m.MatchID && (v.State != "finalizing" || v.Players[0].BotAssisted) {
					t.Fatal(v)
				}
			}
		})
	}
}

func TestAcceptedShotEventsSurviveSnapshotTickAndRejectsEmitNothing(t *testing.T) {
	clock := &testClock{}
	engine, err := Factory(clock)(model.MatchStart{MatchID: "m", WorldID: "w", Seed: 1, Actors: []model.Actor{{ActorID: "a", UserID: "u", ActorType: "human"}}})
	if err != nil {
		t.Fatal(err)
	}
	a := engine.(*Adapter)
	a.Start(4000)
	a.engine.targets = []target{{ID: 100, Kind: "normal", Born: 4500, X: 500, Y: 500, VX: 0}}
	hit, err := a.Shoot("u", 1, 1, Aim{.5, .5}, 1, time.UnixMilli(5000))
	if err != nil || hit.Points != 100 {
		t.Fatal(hit, err)
	}
	if _, err = a.Shoot("u", 1, 2, Aim{}, 2, time.UnixMilli(5100)); err == nil {
		t.Fatal("cooldown accepted")
	}
	miss, err := a.Shoot("u", 1, 3, Aim{0, .95}, 3, time.UnixMilli(5250))
	if err != nil || miss.Points != 0 {
		t.Fatal(miss, err)
	}
	a.Tick(time.UnixMilli(5300))
	events := a.DrainEvents()
	resolved := 0
	for _, event := range events {
		if event.Type == "duck.shot_resolved" {
			resolved++
		}
	}
	if resolved != 2 {
		t.Fatalf("accepted hit/miss each need one event; got %d in %+v", resolved, events)
	}
	if len(a.DrainEvents()) != 0 {
		t.Fatal("events replayed")
	}
	if _, err = a.Shoot("other", 1, 4, Aim{}, 4, time.UnixMilli(5600)); err == nil {
		t.Fatal("spectator accepted")
	}
	if a.HasEvents() {
		t.Fatal("rejection published sound")
	}
}
