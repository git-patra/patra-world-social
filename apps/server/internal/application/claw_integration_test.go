//go:build integration

package application

import (
	"context"
	"encoding/json"
	"fmt"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"patra-world-social/server/internal/world"
	"testing"
	"time"
)

func TestClawActualDatabaseHasNoMatchResultOrHistoryWrites(t *testing.T) {
	pool := testutil.Database(t)
	users, tokens := testutil.Users(t, pool, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	clock := &penaltyClock{at: time.Now()}
	app, err := newApplication(pool, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true}, clock)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	w, err := app.Worlds.CreateWorld(ctx, users[0], "Claw test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, w.ID, users[1]); err != nil {
		t.Fatal(err)
	}
	seats := []realtime.JoinContext{}
	for n := range users {
		s, err := app.Hub.Join(ctx, realtime.JoinContext{Token: tokens[n], WorldID: w.ID})
		if err != nil {
			t.Fatal(err)
		}
		j := realtime.JoinContext{Token: tokens[n], WorldID: w.ID, SessionID: s.SessionID, Epoch: s.Epoch}
		seats = append(seats, j)
		events := app.Hub.Events(j)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-events:
				}
			}
		}()
	}
	at := world.Point{X: 41, Y: 24}
	if err = app.Hub.SetActivity(ctx, w.ID, users[0], false, &at); err != nil {
		t.Fatal(err)
	}
	state := func() world.ActivitySnapshot {
		t.Helper()
		ss, e := app.Hub.ActivitySnapshots(ctx, seats[0])
		if e != nil {
			t.Fatal(e)
		}
		for _, s := range ss {
			if s.Kind == "claw" {
				return s
			}
		}
		t.Fatal("missing claw")
		return world.ActivitySnapshot{}
	}
	seq := uint64(0)
	execute := func(action string, dx, dy float64) error {
		seq++
		s := state()
		i := world.CasualIntent{Incarnation: s.Incarnation, Generation: s.Generation, MachineID: "claw-1", Action: action, DX: dx, DY: dy}
		raw, _ := json.Marshal(i)
		_, e := app.Hub.Execute(ctx, seats[0], realtime.Command{V: 2, WorldID: w.ID, CommandID: fmt.Sprint(seq), Seq: seq, Type: "activity.claw", Payload: raw})
		return e
	}
	for cycle := 0; cycle < 20; cycle++ {
		if err = execute("claw.enter", 0, 0); err != nil {
			t.Fatal("entry", cycle, err)
		}
		for n := 0; n < 6; n++ {
			if err = execute("claw.move", -1, -1); err != nil {
				t.Fatal(err)
			}
			clock.set(clock.Now().Add(50 * time.Millisecond).UnixMilli())
		}
		if err = execute("claw.drop", 0, 0); err != nil {
			t.Fatal("drop", err)
		}
		s := state()
		if s.Payload.Claw.PrizeID != "bear" {
			t.Fatal("owner did not capture preset", s.Payload.Claw)
		}
		if err = execute("claw.drop", 0, 0); err == nil {
			t.Fatal("duplicate drop produced another cosmetic result")
		}
		peer, err := app.Hub.ActivitySnapshots(ctx, seats[1])
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range peer {
			if s.Kind == "claw" && (s.Payload.Claw.OperatorID != users[0] || s.Payload.Claw.PrizeID != "bear") {
				t.Fatal("spectator sees different outcome")
			}
		}
		clock.set(clock.Now().Add(1500 * time.Millisecond).UnixMilli())
		if s = state(); s.Payload.Claw.PrizeID != "" || s.Payload.Claw.Phase != "aiming" {
			t.Fatal("prize escaped transient animation", s.Payload.Claw)
		}
		if err = execute("claw.leave", 0, 0); err != nil {
			t.Fatal(err)
		}
	}
	if err = execute("claw.enter", 0, 0); err != nil {
		t.Fatal(err)
	}
	seq++
	raw, _ := json.Marshal(map[string]string{"arenaId": "pingpong"})
	if _, err = app.Hub.Execute(ctx, seats[0], realtime.Command{V: 2, WorldID: w.ID, CommandID: "join-match", Seq: seq, Type: "lobby.join", Payload: raw}); err != nil {
		t.Fatal("actual match lobby join", err)
	}
	if s := state(); s.Payload.Claw.OperatorID != "" {
		t.Fatal("actual match join retained claw operator")
	}
	if err = execute("claw.drop", 0, 0); err == nil {
		t.Fatal("lobby owner retained claw input")
	}
	for _, table := range []string{"matches", "match_participants", "result_jobs", "active_match_users", "leaderboard_entries", "champion_changes"} {
		var count int
		if err = pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("casual claw wrote %s rows: %d", table, count)
		}
	}
	t.Log("20 actual authenticated claw cycles; SQL matches, participants/history, result_jobs, active users, leaderboard and champion changes all zero")
}
