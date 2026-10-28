package penalty_test

import (
	"context"
	"fmt"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/games/penalty"
	"patra-world-social/server/internal/lobby"
	"sync"
	"testing"
	"time"
)

type clock struct {
	mu sync.Mutex
	at time.Time
}

func (c *clock) Now() time.Time   { c.mu.Lock(); defer c.mu.Unlock(); return c.at }
func (c *clock) set(at time.Time) { c.mu.Lock(); c.at = at; c.mu.Unlock() }

type store struct{}

func (store) Persist(context.Context, model.MatchStart, string, string) error { return nil }
func (store) Lookup(context.Context, string) (bool, error)                    { return true, nil }
func (store) Schedule(context.Context, string, int64) error                   { return nil }
func (store) Terminal(context.Context, string, string) error                  { return nil }
func (store) ReleaseUser(context.Context, string, string) error               { return nil }
func setup(t *testing.T) (*lobby.Service, *clock, string) {
	t.Helper()
	c := &clock{at: time.UnixMilli(100000)}
	s := lobby.New(store{}, c, model.Version{})
	t.Cleanup(s.Close)
	if err := s.Register("penalty", penalty.Factory(c)); err != nil {
		t.Fatal(err)
	}
	l, err := s.JoinLobby(context.Background(), lobby.JoinContext{UserID: "u", WorldID: "w"}, "penalty")
	if err != nil {
		t.Fatal(err)
	}
	s.ChooseBot(context.Background(), "u", l.ID, true)
	s.SetReady(context.Background(), "u", l.ID, true)
	m, err := s.StartMatch(context.Background(), "u", l.ID, "start")
	if err != nil {
		t.Fatal(err)
	}
	return s, c, m.MatchID
}
func TestOwnerReceiptQueueBoundaryCannotReviseReveal(t *testing.T) {
	for _, delay := range []int64{2999, 3000} {
		t.Run(time.Duration(delay).String(), func(t *testing.T) {
			s, c, m := setup(t)
			var snap penalty.PublicSnapshot
			s.ReadEngine(context.Background(), "w", m, func(e lobby.Engine) error { snap = e.(*penalty.Adapter).Snapshot(); return nil })
			c.set(time.UnixMilli(snap.ChoiceOpensAt + delay))
			err := s.InputEngine(context.Background(), "w", m, "u", func(e lobby.Engine, at time.Time) error {
				return e.(*penalty.Adapter).Choose("u", "choice", snap.TurnID, 2, at)
			})
			if (err == nil) != (delay == 2999) {
				t.Fatal(delay, err)
			}
		})
	}
	s, c, m := setup(t)
	var turnID string
	s.ReadEngine(context.Background(), "w", m, func(e lobby.Engine) error { turnID = e.(*penalty.Adapter).Snapshot().TurnID; return nil })
	entered, resume := make(chan struct{}), make(chan struct{})
	hold := make(chan error, 1)
	go func() {
		hold <- s.ReadEngine(context.Background(), "w", m, func(lobby.Engine) error { close(entered); <-resume; return nil })
	}()
	<-entered
	c.set(time.UnixMilli(102999))
	done := make(chan error, 1)
	go func() {
		done <- s.InputEngine(context.Background(), "w", m, "u", func(e lobby.Engine, at time.Time) error {
			return e.(*penalty.Adapter).Choose("u", "queued", turnID, 0, at)
		})
	}()
	c.set(time.UnixMilli(103000))
	close(resume)
	<-hold
	if err := <-done; err == nil {
		t.Fatal("late owner receipt revised reveal")
	}
	var snap penalty.PublicSnapshot
	s.ReadEngine(context.Background(), "w", m, func(e lobby.Engine) error { snap = e.(*penalty.Adapter).Snapshot(); return nil })
	if len(snap.Turns) != 1 || !snap.Turns[0].KickTimeout && !snap.Turns[0].KeepTimeout {
		t.Fatal(snap)
	}
	if s.ReadEngine(context.Background(), "other", m, func(lobby.Engine) error { t.Fatal("cross-world read"); return nil }) == nil {
		t.Fatal("cross-world allowed")
	}
}
func TestAdapterDuplicateIDCannotMutateLaterTurnAndFactoryInert(t *testing.T) {
	c := &clock{at: time.UnixMilli(10000)}
	engine, err := penalty.Factory(c)(model.MatchStart{MatchID: "m", GameType: "penalty", Actors: []model.Actor{{ActorID: "a", ActorType: "human", UserID: "u"}, {ActorID: "b", ActorType: "bot"}}})
	if err != nil {
		t.Fatal(err)
	}
	a := engine.(*penalty.Adapter)
	if a.Tick(c.Now()) {
		t.Fatal("factory active")
	}
	a.Start(10000)
	s := a.Snapshot()
	if a.Choose("u", "id", s.TurnID, 0, c.Now()) != nil {
		t.Fatal("choice rejected")
	}
	a.Tick(time.UnixMilli(14500))
	s = a.Snapshot()
	if a.Choose("u", "id", s.TurnID, 1, time.UnixMilli(14501)) == nil {
		t.Fatal("duplicate command changed new turn")
	}
}
func TestLobbyNaturalResultBeforeBothPermanentDepartures(t *testing.T) {
	c := &clock{at: time.UnixMilli(100000)}
	s := lobby.New(store{}, c, model.Version{})
	t.Cleanup(s.Close)
	s.Register("penalty", penalty.Factory(c))
	ctx := context.Background()
	l, err := s.JoinLobby(ctx, lobby.JoinContext{UserID: "u", WorldID: "w"}, "penalty")
	if err != nil {
		t.Fatal(err)
	}
	s.JoinLobby(ctx, lobby.JoinContext{UserID: "v", WorldID: "w"}, "penalty")
	s.SetReady(ctx, "u", l.ID, true)
	s.SetReady(ctx, "v", l.ID, true)
	m, err := s.StartMatch(ctx, "u", l.ID, "start")
	if err != nil {
		t.Fatal(err)
	}
	var first string
	for n := 0; n < 6; n++ {
		var snap penalty.PublicSnapshot
		s.ReadEngine(ctx, "w", m.MatchID, func(e lobby.Engine) error { snap = e.(*penalty.Adapter).Snapshot(); return nil })
		if n == 0 {
			first = snap.KickerID
		}
		for _, a := range snap.Actors {
			zone := 0
			if a.ActorID == snap.KeeperID && snap.KickerID == first {
				zone = 1
			}
			if err := s.InputEngine(ctx, "w", m.MatchID, a.UserID, func(e lobby.Engine, at time.Time) error {
				return e.(*penalty.Adapter).Choose(a.UserID, fmt.Sprint(n, a.UserID), snap.TurnID, zone, at)
			}); err != nil {
				t.Fatal(err)
			}
		}
		if n == 5 {
			s.Notify("disconnect", "w", "u")
			s.Notify("disconnect", "w", "v")
			s.Sync(ctx)
			c.set(time.UnixMilli(snap.ChoiceOpensAt + 20000))
		} else {
			c.set(time.UnixMilli(snap.AnimationEndsAt))
		}
		s.Sync(ctx)
	}
	r, err := s.CopyResult(ctx, m.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range r.Participants {
		if p.ActorID == first && p.Outcome != "win" || p.ActorID != first && p.Outcome != "loss" {
			t.Fatal("natural result overwritten by permanent departure", r)
		}
	}
}
