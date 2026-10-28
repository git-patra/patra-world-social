//go:build integration

package application

import (
	"context"
	"patra-world-social/server/internal/auth"
	"patra-world-social/server/internal/games/race"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"patra-world-social/server/internal/world"
	"patra-world-social/server/internal/worlds"
	"testing"
	"time"
)

func TestRaceWiredSchedulerPublishes15HzAndHostDepartureDoesNotStopPeer(t *testing.T) {
	p := testutil.Database(t)
	users, _ := testutil.Users(t, p, 3)
	ctx := context.Background()
	clock := &penaltyClock{at: time.Now()}
	m, err := world.LoadMap(world.MapAsset)
	if err != nil {
		t.Fatal(err)
	}
	ws := worlds.New(p)
	w, err := ws.CreateWorld(ctx, users[0], "Race clock")
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range users[1:] {
		p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, w.ID, u)
	}
	h := realtime.NewHub(auth.New(p), ws, m, clock)
	defer h.Close()
	g := wireLobbiesClock(p, h, m, clock)
	defer g.Close()
	var l lobby.Lobby
	for _, u := range users[:2] {
		l, err = g.JoinLobby(ctx, lobby.JoinContext{UserID: u, WorldID: w.ID, DisplayName: "Synthetic"}, "race")
		if err != nil {
			t.Fatal(err)
		}
		if err = g.SetReady(ctx, u, l.ID, true); err != nil {
			t.Fatal(err)
		}
	}
	start, err := g.StartMatch(ctx, users[0], l.ID, "clock")
	if err != nil {
		t.Fatal(err)
	}
	clock.set(start.StartsAt)
	g.Sync(ctx)
	g.DrainChanges(ctx)
	published := 0
	for at := start.StartsAt + 50; at <= start.StartsAt+1000; at += 50 {
		clock.set(at)
		g.Sync(ctx)
		c, e := g.DrainChanges(ctx)
		if e != nil {
			t.Fatal(e)
		}
		for _, event := range c.Events {
			if event.Type == "race.snapshot" {
				published++
			}
		}
	}
	var s race.PublicSnapshot
	read := func() {
		if e := g.ReadEngine(ctx, w.ID, start.MatchID, func(e lobby.Engine) error { s = e.(*race.Adapter).Snapshot(); return nil }); e != nil {
			t.Fatal(e)
		}
	}
	read()
	if s.Frame != 30 || published != 15 {
		t.Fatalf("actual wired50ms owner produced%d frames/%d events", s.Frame, published)
	}
	if err = g.Leave(ctx, users[0], w.ID); err != nil {
		t.Fatal(err)
	}
	read()
	if s.Phase != "racing" {
		t.Fatal("host departure stopped peer", s.Phase)
	}
	for _, k := range s.Karts {
		if k.UserID == users[0] && !k.DNF {
			t.Fatal("departed host not DNF")
		}
		if k.UserID == users[1] && k.DNF {
			t.Fatal("remaining peer DNF")
		}
	}
	late, err := g.JoinLobby(ctx, lobby.JoinContext{UserID: users[2], WorldID: w.ID, DisplayName: "Late"}, "race")
	if err != nil || len(late.Spectators) != 1 {
		t.Fatal("late arrival did not spectate", err)
	}
	read()
	for _, k := range s.Karts {
		if k.UserID == users[2] {
			t.Fatal("late spectator replaced bot")
		}
	}
	if err = g.Leave(ctx, users[1], w.ID); err != nil {
		t.Fatal(err)
	}
	until := time.Now().Add(2 * time.Second)
	for time.Now().Before(until) {
		g.Sync(ctx)
		var status string
		p.QueryRow(ctx, `SELECT status FROM matches WHERE id=$1`, start.MatchID).Scan(&status)
		if status == "abandoned" {
			var count int
			p.QueryRow(ctx, `SELECT count(*) FROM result_jobs WHERE match_id=$1`, start.MatchID).Scan(&count)
			if count != 0 {
				t.Fatal("abandoned published official result")
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("all-human-left race did not abandon/release")
}
