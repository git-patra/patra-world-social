package penalty_test

import (
	"context"
	"errors"
	"fmt"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/games/penalty"
	"patra-world-social/server/internal/lobby"
	"testing"
	"time"
)

type heldStartStore struct {
	store
	entered, release chan struct{}
	terminal         chan string
}

func (s heldStartStore) Persist(ctx context.Context, m model.MatchStart, host, command string) error {
	close(s.entered)
	select {
	case <-s.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s heldStartStore) Terminal(ctx context.Context, id, status string) error {
	select {
	case s.terminal <- status:
	default:
	}
	return nil
}

type activationResult struct {
	m   model.MatchStart
	err error
}

func heldActivation(t *testing.T) (*lobby.Service, *clock, heldStartStore, <-chan activationResult) {
	t.Helper()
	ctx := context.Background()
	c := &clock{at: time.UnixMilli(100000)}
	st := heldStartStore{entered: make(chan struct{}), release: make(chan struct{}), terminal: make(chan string, 1)}
	s := lobby.New(st, c, model.Version{})
	t.Cleanup(s.Close)
	s.Register("penalty", penalty.Factory(c))
	l, e := s.JoinLobby(ctx, lobby.JoinContext{UserID: "u", WorldID: "w"}, "penalty")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.JoinLobby(ctx, lobby.JoinContext{UserID: "v", WorldID: "w"}, "penalty"); e != nil {
		t.Fatal(e)
	}
	s.SetReady(ctx, "u", l.ID, true)
	s.SetReady(ctx, "v", l.ID, true)
	done := make(chan activationResult, 1)
	go func() { m, e := s.StartMatch(ctx, "u", l.ID, "held"); done <- activationResult{m, e} }()
	<-st.entered
	return s, c, st, done
}
func assertActivationForfeit(t *testing.T, s *lobby.Service, started activationResult, finished int64) {
	t.Helper()
	if started.err != nil {
		t.Fatal(started.err)
	}
	r, e := s.CopyResult(context.Background(), started.m.MatchID)
	if e != nil {
		t.Fatal("activation did not immediately expose finalizing result", e)
	}
	if r.StartsAt != started.m.StartsAt || r.FinishedAt != finished || len(r.Participants) != 2 || r.Participants[0].Outcome != "forfeit" || r.Participants[1].Outcome != "win" {
		t.Fatal("activation terminal result", r, started.m)
	}
	for i, p := range r.Participants {
		if p.Actor != started.m.Actors[i] {
			t.Fatal("frozen roster changed", r)
		}
	}
}
func TestActivationReplaysOneExpiredGraceBeforeFreshEngineCanContinue(t *testing.T) {
	for _, activationAt := range []int64{115000, 115100} {
		t.Run(fmt.Sprint(activationAt), func(t *testing.T) {
			s, c, st, done := heldActivation(t)
			ctx := context.Background()
			s.Notify("disconnect", "w", "u")
			s.Sync(ctx)
			c.set(time.UnixMilli(115000))
			s.Sync(ctx)
			c.set(time.UnixMilli(activationAt))
			close(st.release)
			started := <-done
			if started.m.StartsAt != activationAt {
				t.Fatal("official activation time", started.m)
			}
			assertActivationForfeit(t, s, started, activationAt)
			r, _ := s.CopyResult(ctx, started.m.MatchID)
			if len(r.PenaltyTurns) != 0 {
				t.Fatal("departed player continued timeout duel", r.PenaltyTurns)
			}
		})
	}
}
func TestActivationPreservesStillInGraceOriginalRemainingTime(t *testing.T) {
	s, c, st, done := heldActivation(t)
	ctx := context.Background()
	s.Notify("disconnect", "w", "u")
	s.Sync(ctx)
	c.set(time.UnixMilli(114999))
	close(st.release)
	started := <-done
	if started.err != nil || started.m.StartsAt != 114999 {
		t.Fatal(started)
	}
	if r, e := s.CopyResult(ctx, started.m.MatchID); e == nil {
		t.Fatal("still in grace ended early", r)
	}
	var snap penalty.PublicSnapshot
	if e := s.ReadEngine(ctx, "w", started.m.MatchID, func(engine lobby.Engine) error { snap = engine.(*penalty.Adapter).Snapshot(); return nil }); e != nil {
		t.Fatal(e)
	}
	if snap.ChoiceOpensAt != 114999 || snap.Result != nil {
		t.Fatal(snap)
	}
	c.set(time.UnixMilli(115000))
	s.Sync(ctx)
	assertActivationForfeit(t, s, started, 115000)
}
func TestActivationRetainsAllGoneWithoutFabricatingResult(t *testing.T) {
	s, c, st, done := heldActivation(t)
	ctx := context.Background()
	s.Notify("disconnect", "w", "u")
	s.Notify("disconnect", "w", "v")
	s.Sync(ctx)
	c.set(time.UnixMilli(115000))
	s.Sync(ctx)
	close(st.release)
	started := <-done
	if !errors.Is(started.err, lobby.ErrForbidden) {
		t.Fatal("all-gone activation accepted", started)
	}
	if r, e := s.CopyResult(ctx, started.m.MatchID); e == nil {
		t.Fatal("all-gone fabricated result", r)
	}
	select {
	case status := <-st.terminal:
		if status != "abandoned" {
			t.Fatal(status)
		}
	case <-time.After(time.Second):
		t.Fatal("all-gone start not released as abandoned")
	}
}
func TestActivationRetainsOriginalIntentionalLeaveAndClampsFinish(t *testing.T) {
	s, c, st, done := heldActivation(t)
	c.set(time.UnixMilli(100001))
	if e := s.Leave(context.Background(), "u", "w"); e != nil {
		t.Fatal(e)
	}
	c.set(time.UnixMilli(100100))
	close(st.release)
	started := <-done
	assertActivationForfeit(t, s, started, started.m.StartsAt)
	if started.m.StartsAt != 100100 {
		t.Fatal(started.m.StartsAt)
	}
}
