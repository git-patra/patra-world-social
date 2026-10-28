package penalty_test

import (
	"context"
	"fmt"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/games/penalty"
	"patra-world-social/server/internal/lobby"
	"testing"
	"time"
)

func pair(t *testing.T) (*lobby.Service, *clock, model.MatchStart) {
	t.Helper()
	c := &clock{at: time.UnixMilli(100000)}
	s := lobby.New(store{}, c, model.Version{})
	t.Cleanup(s.Close)
	s.Register("penalty", func(m model.MatchStart) (lobby.Engine, error) { m.Seed = 0; return penalty.Factory(c)(m) })
	ctx := context.Background()
	l, e := s.JoinLobby(ctx, lobby.JoinContext{UserID: "u", WorldID: "w"}, "penalty")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.JoinLobby(ctx, lobby.JoinContext{UserID: "v", WorldID: "w"}, "penalty"); e != nil {
		t.Fatal(e)
	}
	s.SetReady(ctx, "u", l.ID, true)
	s.SetReady(ctx, "v", l.ID, true)
	m, e := s.StartMatch(ctx, "u", l.ID, "pair")
	if e != nil {
		t.Fatal(e)
	}
	return s, c, m
}
func holdOwner(t *testing.T, s *lobby.Service, m string) func() {
	t.Helper()
	entered, resume, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- s.ReadEngine(context.Background(), "w", m, func(lobby.Engine) error { close(entered); <-resume; return nil })
	}()
	<-entered
	return func() {
		close(resume)
		if e := <-done; e != nil {
			t.Fatal(e)
		}
		if e := s.Sync(context.Background()); e != nil {
			t.Fatal(e)
		}
	}
}
func TestDelayedDualDisconnectUsesOriginalGraceAndFinishedAt(t *testing.T) {
	s, c, m := pair(t)
	release := holdOwner(t, s, m.MatchID)
	s.Notify("disconnect", "w", "u")
	s.Notify("disconnect", "w", "v")
	c.set(time.UnixMilli(100100))
	release()
	c.set(time.UnixMilli(115000))
	s.Sync(context.Background())
	r, e := s.CopyResult(context.Background(), m.MatchID)
	if e != nil {
		t.Fatal(e)
	}
	if r.FinishedAt != 115000 || r.Participants[0].Outcome != "abandoned" || r.Participants[1].Outcome != "abandoned" {
		t.Fatalf("original grace115000: got %d %s/%s", r.FinishedAt, r.Participants[0].Outcome, r.Participants[1].Outcome)
	}
}
func TestCoalescedResumeAtOriginalGraceBoundary(t *testing.T) {
	for _, offset := range []int64{-1, 0, 1} {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			s, c, m := pair(t)
			release := holdOwner(t, s, m.MatchID)
			s.Notify("disconnect", "w", "u")
			c.set(time.UnixMilli(115000 + offset))
			s.Notify("resume", "w", "u")
			c.set(time.UnixMilli(115100))
			release()
			r, e := s.CopyResult(context.Background(), m.MatchID)
			if offset < 0 {
				if e == nil {
					t.Fatal("timely resume forfeited", r)
				}
				var snap penalty.PublicSnapshot
				s.ReadEngine(context.Background(), "w", m.MatchID, func(engine lobby.Engine) error { snap = engine.(*penalty.Adapter).Snapshot(); return nil })
				if err := s.InputEngine(context.Background(), "w", m.MatchID, "u", func(engine lobby.Engine, at time.Time) error {
					return engine.(*penalty.Adapter).Choose("u", "resumed", snap.TurnID, 1, at)
				}); err != nil {
					t.Fatal("timely resume failed", err)
				}
				return
			}
			if e != nil || r.FinishedAt != 115000 || r.Participants[0].Outcome != "forfeit" || r.Participants[1].Outcome != "win" {
				t.Fatal("expired original grace", r, e)
			}
		})
	}
}
func TestDelayedNaturalFinishBeforeAtAfterOriginalGraceCutoff(t *testing.T) {
	c := &clock{at: time.UnixMilli(100000)}
	raw, e := penalty.Factory(c)(model.MatchStart{MatchID: "baseline", GameType: "penalty", Seed: 0, Actors: []model.Actor{{ActorID: "a", ActorType: "human", UserID: "u"}, {ActorID: "b", ActorType: "human", UserID: "v"}}})
	if e != nil {
		t.Fatal(e)
	}
	raw.Start(100000)
	raw.Tick(time.UnixMilli(1000000))
	natural, e := raw.(*penalty.Adapter).Result()
	if e != nil {
		t.Fatal(e)
	}
	for _, offset := range []int64{-1, 0, 1} {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			s, c, m := pair(t)
			release := holdOwner(t, s, m.MatchID)
			cutoff := natural.FinishedAt + offset
			c.set(time.UnixMilli(cutoff - 15000))
			s.Notify("disconnect", "w", "u")
			s.Notify("disconnect", "w", "v")
			c.set(time.UnixMilli(cutoff - 14900))
			release()
			c.set(time.UnixMilli(natural.FinishedAt + 10000))
			s.Sync(context.Background())
			r, err := s.CopyResult(context.Background(), m.MatchID)
			if err != nil {
				t.Fatal(err)
			}
			if offset < 0 {
				if r.FinishedAt != cutoff || r.Participants[0].Outcome != "abandoned" || r.Participants[1].Outcome != "abandoned" {
					t.Fatal("natural after true cutoff won", r, cutoff)
				}
			} else {
				if r.FinishedAt != natural.FinishedAt {
					t.Fatal(r.FinishedAt, natural.FinishedAt)
				}
				for i, p := range r.Participants {
					if p.Outcome != natural.Participants[i].Outcome {
						t.Fatal(r, natural)
					}
				}
			}
		})
	}
}
func TestRejectedChoiceFloodPreservesBothActorsLaterEligibility(t *testing.T) {
	c := &clock{at: time.UnixMilli(100000)}
	raw, e := penalty.Factory(c)(model.MatchStart{MatchID: "m", GameType: "penalty", Actors: []model.Actor{{ActorID: "a", ActorType: "human", UserID: "u"}, {ActorID: "b", ActorType: "human", UserID: "v"}}})
	if e != nil {
		t.Fatal(e)
	}
	a := raw.(*penalty.Adapter)
	a.Start(100000)
	turn := a.Snapshot().TurnID
	if err := a.Choose("u", "accepted", turn, 0, c.Now()); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 512; i++ {
		stale := "wrong-turn"
		if i%2 == 0 {
			stale = turn
		}
		if err := a.Choose("u", fmt.Sprint("rejected", i), stale, 2, c.Now()); err == nil {
			t.Fatal("bad choice accepted")
		}
	}
	a.Tick(time.UnixMilli(104500))
	next := a.Snapshot().TurnID
	if err := a.Choose("u", "accepted", next, 3, time.UnixMilli(104501)); err == nil {
		t.Fatal("accepted command replay changed later turn")
	}
	for _, user := range []string{"v", "u"} {
		if err := a.Choose(user, "fresh-"+user, next, 1, time.UnixMilli(104501)); err != nil {
			t.Fatal("failed flood disabled valid actor", user, err)
		}
	}
}
func TestBothCoalescedResumesCannotReviveOriginalGrace(t *testing.T) {
	s, c, m := pair(t)
	release := holdOwner(t, s, m.MatchID)
	s.Notify("disconnect", "w", "u")
	s.Notify("disconnect", "w", "v")
	c.set(time.UnixMilli(115000))
	s.Notify("resume", "w", "u")
	s.Notify("resume", "w", "v")
	c.set(time.UnixMilli(115100))
	release()
	r, e := s.CopyResult(context.Background(), m.MatchID)
	if e != nil || r.FinishedAt != 115000 || r.Participants[0].Outcome != "abandoned" || r.Participants[1].Outcome != "abandoned" {
		t.Fatal(r, e)
	}
}
