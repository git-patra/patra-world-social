//go:build integration

package history

import (
	"context"
	"errors"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/lobby"
	"sync"
	"testing"
	"time"
)

type fixtureClock struct {
	mu sync.Mutex
	at time.Time
}

func (c *fixtureClock) Now() time.Time          { c.mu.Lock(); defer c.mu.Unlock(); return c.at }
func (c *fixtureClock) advance(d time.Duration) { c.mu.Lock(); c.at = c.at.Add(d); c.mu.Unlock() }

type fixtureEngine struct{ m model.MatchStart }

func (e *fixtureEngine) Start(at int64)          { e.m.StartsAt = at }
func (e *fixtureEngine) Tick(now time.Time) bool { return now.UnixMilli() >= e.m.StartsAt+1000 }
func (e *fixtureEngine) Presence(string, string) {}
func (e *fixtureEngine) Close()                  {}
func (e *fixtureEngine) Result() (model.MatchResult, error) {
	r := model.MatchResult{Version: e.m.Version, MatchID: e.m.MatchID, WorldID: e.m.WorldID, ArenaID: e.m.ArenaID, GameType: e.m.GameType, PreparedAt: e.m.PreparedAt, StartsAt: e.m.StartsAt, FinishedAt: e.m.StartsAt + 1000}
	for _, a := range e.m.Actors {
		r.Participants = append(r.Participants, model.ParticipantResult{Actor: a, Outcome: "finished"})
	}
	return r, nil
}

type missingSchedule struct{ *lobby.PGStore }

func (s missingSchedule) Schedule(context.Context, string, int64) error {
	return errors.New("synthetic unavailable schedule projection")
}
func TestPostgresLifecycleUsesActualRuntimeStartAndKeepsResultScreen(t *testing.T) {
	s, previous, users := fixture(t)
	ctx := context.Background()
	if e := s.RecoverPending(ctx); e != nil {
		t.Fatal(e)
	}
	c := &fixtureClock{at: time.UnixMilli(100000)}
	g := lobby.New(missingSchedule{&lobby.PGStore{Pool: s.Pool}}, c, previous.Version)
	defer g.Close()
	if e := g.Register("duck", func(m model.MatchStart) (lobby.Engine, error) { return &fixtureEngine{m: m}, nil }); e != nil {
		t.Fatal(e)
	}
	l, e := g.JoinLobby(ctx, lobby.JoinContext{UserID: users[0], WorldID: previous.WorldID, DisplayName: "synthetic"}, "duck-1")
	if e != nil {
		t.Fatal(e)
	}
	if e = g.SetReady(ctx, users[0], l.ID, true); e != nil {
		t.Fatal(e)
	}
	m, e := g.StartMatch(ctx, users[0], l.ID, "lifecycle")
	if e != nil {
		t.Fatal(e)
	}
	c.advance(time.UnixMilli(m.StartsAt + 1000).Sub(c.Now()))
	if e = g.Sync(ctx); e != nil {
		t.Fatal(e)
	}
	w := NewWorker(s, g)
	saved := false
	w.Notify = func(_ string, _ []string, event StatusEvent) {
		if event.Status == "completed" {
			var status string
			s.Pool.QueryRow(ctx, `SELECT status FROM matches WHERE id=$1`, m.MatchID).Scan(&status)
			if status != "completed" {
				t.Error("saved before commit")
			}
			saved = true
		}
	}
	w.Step(ctx, c.Now())
	all, e := g.List(ctx, previous.WorldID)
	if e != nil {
		t.Fatal(e)
	}
	var state string
	for _, l := range all {
		if l.MatchID == m.MatchID {
			state = l.State
		}
	}
	if state != "results" || !saved || !g.Busy(users[0]) {
		t.Fatal("result screen prematurely erased", state, saved)
	}
	d, e := s.Detail(ctx, users[0], m.MatchID)
	if e != nil || d.StartsAt == nil || d.StartsAt.UnixMilli() != m.StartsAt {
		t.Fatal(d, e)
	}
	if e = g.ReleaseMatch(m.MatchID, "completed"); e != nil {
		t.Fatal(e)
	}
	if g.Busy(users[0]) {
		t.Fatal("result return kept claim")
	}
}
