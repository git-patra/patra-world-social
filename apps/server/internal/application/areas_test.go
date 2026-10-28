package application

import (
	"context"
	"encoding/json"
	"patra-world-social/server/internal/auth"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/world"
	"patra-world-social/server/internal/worlds"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type authorityClock struct {
	mu sync.Mutex
	at time.Time
}

func (c *authorityClock) Now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.at }
func (c *authorityClock) add(d time.Duration) { c.mu.Lock(); c.at = c.at.Add(d); c.mu.Unlock() }

type authorityAuth struct{}

func (authorityAuth) RequireSession(context.Context, string) (auth.Session, error) {
	return auth.Session{User: auth.User{ID: "user", DisplayName: "User"}}, nil
}
func (authorityAuth) AuthorizeMember(context.Context, string, string) (worlds.Membership, error) {
	return worlds.Membership{}, nil
}

type authorityStore struct{ entered, release chan struct{} }

func (s *authorityStore) Persist(context.Context, model.MatchStart, string, string) error {
	close(s.entered)
	<-s.release
	return nil
}
func (s *authorityStore) Lookup(context.Context, string) (bool, error)      { return true, nil }
func (s *authorityStore) Schedule(context.Context, string, int64) error     { return nil }
func (s *authorityStore) Terminal(context.Context, string, string) error    { return nil }
func (s *authorityStore) ReleaseUser(context.Context, string, string) error { return nil }

type authorityEngine struct{ started atomic.Int64 }

func (e *authorityEngine) Start(at int64)          { e.started.Store(at) }
func (e *authorityEngine) Tick(time.Time) bool     { return false }
func (e *authorityEngine) Presence(string, string) {}
func (e *authorityEngine) Close()                  {}
func TestStartWaitsForClearAckAndDurabilityAcrossActualOwners(t *testing.T) {
	for _, game := range []string{"race", "penalty", "pingpong"} {
		t.Run(game, func(t *testing.T) { testStartWaitsForClearAckAndDurability(t, game) })
	}
}
func testStartWaitsForClearAckAndDurability(t *testing.T, game string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clock := &authorityClock{at: time.Unix(1000, 0)}
	m, err := world.LoadMap(world.MapAsset)
	if err != nil {
		t.Fatal(err)
	}
	h := realtime.NewHub(authorityAuth{}, authorityAuth{}, m, clock)
	defer h.Close()
	store := &authorityStore{make(chan struct{}), make(chan struct{})}
	g := lobby.New(store, clock, model.Version{})
	defer g.Close()
	g.SetAreaCoordinator(areaCoordinator{h})
	h.GameBusy = g.MovementBusy
	engine := &authorityEngine{}
	g.Register(game, func(model.MatchStart) (lobby.Engine, error) { return engine, nil })
	seat, err := h.Join(ctx, realtime.JoinContext{Token: "token", WorldID: "world"})
	if err != nil {
		t.Fatal(err)
	}
	l, err := g.JoinLobby(ctx, lobby.JoinContext{UserID: "user", WorldID: "world", DisplayName: "User"}, game)
	if err != nil {
		t.Fatal(err)
	}
	g.SetReady(ctx, "user", l.ID, true)
	if game == "penalty" || game == "pingpong" {
		if err = g.ChooseBot(ctx, "user", l.ID, true); err != nil {
			t.Fatal(err)
		}
	}
	done := make(chan error, 1)
	go func() { _, err := g.StartMatch(ctx, "user", l.ID, "start"); done <- err }()
	phase := func(want string) {
		t.Helper()
		for {
			states, _ := h.WorldActivities(ctx, "world")
			for _, s := range states {
				if s.Kind == "area" && s.AreaID == game && s.Payload.Phase == want {
					return
				}
			}
			select {
			case <-ctx.Done():
				t.Fatalf("missing phase %s", want)
			case <-time.After(time.Millisecond):
			}
		}
	}
	phase("CLEARING")
	select {
	case <-store.entered:
		t.Fatal("SQL before notice ACK")
	default:
	}
	if engine.started.Load() != 0 {
		t.Fatal("engine before clear")
	}
	clock.add(3 * time.Second)
	select {
	case <-store.entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	phase("RESERVED")
	if engine.started.Load() != 0 {
		t.Fatal("engine before confirmed durability")
	}
	// Removing the final transport cannot discard the reservation while SQL is in flight.
	h.Leave(realtime.JoinContext{Token: "token", WorldID: "world", SessionID: seat.SessionID, Epoch: seat.Epoch})
	phase("RESERVED")
	close(store.release)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	phase("ACTIVE")
	resumed, err := h.Join(ctx, realtime.JoinContext{Token: "token", WorldID: "world"})
	if err != nil {
		t.Fatal(err)
	}
	admitted := realtime.JoinContext{Token: "token", WorldID: "world", SessionID: resumed.SessionID, Epoch: resumed.Epoch}
	if engine.started.Load() == 0 {
		t.Fatal("engine not started after both ACKs")
	}
	clock.add(3 * time.Second)
	g.Sync(ctx)
	all, _ := g.List(ctx, "world")
	var matchID string
	for _, l := range all {
		if l.Arena.ID == game {
			matchID = l.MatchID
		}
	}
	if err = h.SetActivity(ctx, "world", "user", true, nil); err != nil {
		t.Fatal(err)
	}
	if err = g.Transition(ctx, matchID, "finalizing"); err != nil {
		t.Fatal(err)
	}
	g.Sync(ctx)
	phase("RESETTING")
	clock.add(2 * time.Second)
	phase("OPEN")
	if game == "penalty" {
		states, err := h.WorldActivities(ctx, "world")
		if err != nil {
			t.Fatal(err)
		}
		var i world.CasualIntent
		for _, s := range states {
			if s.Kind == "football" {
				if !s.Payload.Ball.Visible {
					t.Fatal("result storage delay kept ball hidden")
				}
				i.Incarnation = s.Incarnation
				i.Generation = s.Generation
				i.Action = "kick"
				i.DY = -1
			}
			if s.Kind == "area" && s.AreaID == "penalty" {
				i.AreaGeneration = s.Generation
			}
		}
		at := world.Point{X: 28, Y: 46}
		if err = h.SetActivity(ctx, "world", "user", true, &at); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(i)
		_, err = h.Execute(ctx, admitted, realtime.Command{V: 2, CommandID: "kick-after-simulation", Seq: 1, WorldID: "world", Type: "activity.kick", Payload: raw})
		if err != nil {
			t.Fatal("spatial release did not restore football while durable result pending", err)
		}
	}
	before, err := h.Snapshot(ctx, admitted)
	if err != nil {
		t.Fatal(err)
	}
	if err = h.Submit(ctx, admitted, realtime.Command{V: 2, CommandID: "walk-after-simulation", Seq: 2, WorldID: "world", Type: "player.input", Payload: []byte(`{"dx":1,"dy":0}`)}); err != nil {
		t.Fatal("durable result claim blocked walking", err)
	}
	clock.add(100 * time.Millisecond)
	after, err := h.Snapshot(ctx, admitted)
	if err != nil || after.Avatars[0].X <= before.Avatars[0].X {
		t.Fatal("physical release did not restore movement", after, err)
	}
	if !g.Busy("user") {
		t.Fatal("spatial release bypassed durable result claim")
	}
}
