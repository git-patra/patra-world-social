package lobby

import (
	"context"
	"errors"
	"patra-world-social/server/internal/games/model"
	"sync"
	"testing"
	"time"
)

type rematchStore struct {
	memoryStore
	entered, release chan struct{}
	muResult         sync.Mutex
	failure          bool
	completed        bool
	terminalCalls    int
}

func (s *rematchStore) Terminal(ctx context.Context, match, status string) error {
	s.muResult.Lock()
	s.terminalCalls++
	s.muResult.Unlock()
	if status != "completed" || !s.completed {
		return ErrForbidden
	}
	if s.entered != nil {
		select {
		case s.entered <- struct{}{}:
		default:
		}
	}
	if s.release != nil {
		select {
		case <-s.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	s.muResult.Lock()
	defer s.muResult.Unlock()
	if s.failure {
		return errors.New("release failed")
	}
	return nil
}
func rematchFixture(t *testing.T) (*Service, *rematchStore, Lobby, model.MatchStart) {
	t.Helper()
	c := &testClock{now: time.UnixMilli(100000)}
	store := &rematchStore{memoryStore: memoryStore{rows: map[string]model.MatchStart{}}, completed: true}
	s := New(store, c, model.Version{})
	t.Cleanup(s.Close)
	register(t, s, "penalty")
	l := join(t, s, "u", "penalty")
	join(t, s, "v", "penalty")
	ready(t, s, "u", l)
	ready(t, s, "v", l)
	m, e := s.StartMatch(context.Background(), "u", l.ID, "start")
	if e != nil {
		t.Fatal(e)
	}
	s.Transition(context.Background(), m.MatchID, "finalizing")
	s.Transition(context.Background(), m.MatchID, "results")
	return s, store, l, m
}
func TestRematchPreservesAtomicPairClaimsAcrossGuardedDurableRelease(t *testing.T) {
	s, store, l, m := rematchFixture(t)
	ctx := context.Background()
	old, _ := s.List(ctx, "world")
	var prior Lobby
	for _, v := range old {
		if v.ID == l.ID {
			prior = v
		}
	}
	var stale []Return
	s.control(func() {
		for _, p := range prior.Players {
			stale = append(stale, s.arenaReturn("world", p.UserID, 50, 32))
		}
	})
	first, e := s.ResultAction(ctx, "world", m.MatchID, "u", true)
	if e != nil || first.Ready {
		t.Fatal(first, e)
	}
	last, e := s.ResultAction(ctx, "world", m.MatchID, "v", true)
	if e != nil || !last.Ready {
		t.Fatal(last, e)
	}
	store.entered = make(chan struct{}, 1)
	store.release = make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- s.CompleteResultAction(m.MatchID) }()
	<-store.entered
	if _, e = s.JoinLobby(ctx, JoinContext{UserID: "third", WorldID: "world"}, "penalty"); e != nil {
		t.Fatal("spectator admission", e)
	}
	if _, e = s.JoinLobby(ctx, JoinContext{UserID: "u", WorldID: "world"}, "race"); !errors.Is(e, ErrBusy) {
		t.Fatal("consented participant claim became free", e)
	}
	during, _ := s.List(ctx, "world")
	for _, v := range during {
		if v.Arena.ID == "penalty" && (v.ID != l.ID || len(v.Players) != 2 || v.Players[0].UserID != "u" || v.Players[1].UserID != "v") {
			t.Fatal("partial pair during release", v)
		}
	}
	if e = s.ReleaseMatch(m.MatchID, "completed"); !errors.Is(e, ErrBusy) {
		t.Fatal("unguarded detach bypassed reservation", e)
	}
	close(store.release)
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	all, _ := s.List(ctx, "world")
	for _, v := range all {
		if v.Arena.ID != "penalty" {
			continue
		}
		if v.ID == l.ID || v.MatchID != "" || v.State != "lobby" || len(v.Players) != 2 || v.HostID != "u" {
			t.Fatal("replacement", v)
		}
		for i, p := range v.Players {
			if p.UserID != prior.Players[i].UserID || p.ActorID == prior.Players[i].ActorID || p.Ready || !p.Connected {
				t.Fatal("pair identity/ready", v)
			}
		}
		if len(v.Spectators) != 0 {
			t.Fatal("old spectators promoted/retained", v)
		}
	}
	for _, r := range stale {
		if s.WithReturn(r, func() { t.Fatal("stale return moved rematched user") }) {
			t.Fatal("stale generation remained eligible")
		}
	}
	if _, e = s.JoinLobby(ctx, JoinContext{UserID: "third", WorldID: "world"}, "penalty"); !errors.Is(e, ErrFull) {
		t.Fatal("third joined exact rematch pair", e)
	}
	if _, e = s.JoinLobby(ctx, JoinContext{UserID: "v", WorldID: "world"}, "race"); !errors.Is(e, ErrBusy) {
		t.Fatal("new pair claim not retained", e)
	}
	if e = s.Leave(ctx, "u", "world"); e != nil {
		t.Fatal(e)
	}
	for _, r := range stale {
		if r.UserID == "u" && s.WithReturn(r, func() { t.Fatal("old generation moved released rematch participant") }) {
			t.Fatal("generation fence did not advance")
		}
	}

}
func TestRematchGuardedReleaseFailureRetainsOldResultsAndClaims(t *testing.T) {
	s, store, l, m := rematchFixture(t)
	ctx := context.Background()
	s.ResultAction(ctx, "world", m.MatchID, "u", true)
	s.ResultAction(ctx, "world", m.MatchID, "v", true)
	store.failure = true
	if e := s.CompleteResultAction(m.MatchID); e == nil {
		t.Fatal("failed release accepted")
	}
	all, _ := s.List(ctx, "world")
	for _, v := range all {
		if v.Arena.ID == "penalty" && (v.ID != l.ID || v.MatchID != m.MatchID || v.State != "results" || len(v.Players) != 2) {
			t.Fatal("old result lost", v)
		}
	}
	if !s.Busy("u") || !s.Busy("v") {
		t.Fatal("failed release dropped claims")
	}
	store.failure = false
	plan, e := s.ResultAction(ctx, "world", m.MatchID, "v", true)
	if e != nil || !plan.Ready {
		t.Fatal(plan, e)
	}
	if e = s.CompleteResultAction(m.MatchID); e != nil {
		t.Fatal(e)
	}
}
func TestRematchDoesNotRestoreDepartedOrRevokedConsentDuringRelease(t *testing.T) {
	for _, kind := range []string{"leave", "expired-disconnect"} {
		t.Run(kind, func(t *testing.T) {
			s, store, _, m := rematchFixture(t)
			ctx := context.Background()
			s.ResultAction(ctx, "world", m.MatchID, "u", true)
			s.ResultAction(ctx, "world", m.MatchID, "v", true)
			store.entered = make(chan struct{}, 1)
			store.release = make(chan struct{})
			done := make(chan error, 1)
			go func() { done <- s.CompleteResultAction(m.MatchID) }()
			<-store.entered
			if kind == "leave" {
				s.Notify("leave", "world", "u")
			} else {
				s.Notify("disconnect", "world", "u")
				s.Sync(ctx)
				s.clock.(*testClock).advance(15 * time.Second)
			}
			s.Sync(ctx)
			if !s.Busy("u") || !s.Busy("v") {
				t.Fatal("reservation freed before durable release")
			}
			close(store.release)
			if e := <-done; e == nil {
				t.Fatal("departed consent rebuilt")
			}
			all, _ := s.List(ctx, "world")
			for _, l := range all {
				if l.Arena.ID == "penalty" && (l.State != "idle" || len(l.Players) != 0) {
					t.Fatal("restored departed or partial pair", l)
				}
			}
			if s.Busy("u") || s.Busy("v") {
				t.Fatal("cancelled pair claims retained")
			}
		})
	}
}

func TestRematchCannotManufactureDurableCompletion(t *testing.T) {
	s, store, l, m := rematchFixture(t)
	ctx := context.Background()
	store.completed = false
	s.ResultAction(ctx, "world", m.MatchID, "u", true)
	s.ResultAction(ctx, "world", m.MatchID, "v", true)
	if err := s.CompleteResultAction(m.MatchID); !errors.Is(err, ErrForbidden) {
		t.Fatal("guarded release accepted undurable result", err)
	}
	all, _ := s.List(ctx, "world")
	for _, v := range all {
		if v.Arena.ID == "penalty" && (v.ID != l.ID || v.MatchID != m.MatchID || v.State != "results") {
			t.Fatal("failed guard destroyed old result", v)
		}
	}
	if !s.Busy("u") || !s.Busy("v") {
		t.Fatal("failed guard freed claims")
	}
}
