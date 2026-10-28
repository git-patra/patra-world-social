//go:build integration

package application

import (
	"context"
	"encoding/json"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"patra-world-social/server/internal/world"
	"testing"
	"time"
)

func TestFreeRiderIsClearedByActualRaceStartAndMembershipRevocation(t *testing.T) {
	p := testutil.Database(t)
	users, tokens := testutil.Users(t, p, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	clock := &penaltyClock{at: time.Now()}
	app, e := newApplication(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true}, clock)
	if e != nil {
		t.Fatal(e)
	}
	defer app.Close()
	w, e := app.Worlds.CreateWorld(ctx, users[0], "Casual track")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role) VALUES($1,$2,'member')`, w.ID, users[1]); e != nil {
		t.Fatal(e)
	}
	seats := []realtime.JoinContext{}
	for i := range users {
		s, e := app.Hub.Join(ctx, realtime.JoinContext{Token: tokens[i], WorldID: w.ID})
		if e != nil {
			t.Fatal(e)
		}
		j := realtime.JoinContext{Token: tokens[i], WorldID: w.ID, SessionID: s.SessionID, Epoch: s.Epoch}
		seats = append(seats, j)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-app.Hub.Events(j):
				}
			}
		}()
	}
	at := world.Point{X: 39, Y: 46}
	if e = app.Hub.SetActivity(ctx, w.ID, users[1], false, &at); e != nil {
		t.Fatal(e)
	}
	states, _ := app.Hub.ActivitySnapshots(ctx, seats[1])
	inc := ""
	for _, s := range states {
		if s.Kind == "vehicles" {
			inc = s.Incarnation
		}
	}
	mount := world.CasualIntent{Incarnation: inc, VehicleID: "bicycle-0", Generation: 1, Action: "mount"}
	raw, _ := json.Marshal(mount)
	if _, e = app.Hub.Execute(ctx, seats[1], realtime.Command{V: 2, CommandID: "mount", Seq: 1, WorldID: w.ID, Type: "activity.ride", Payload: raw}); e != nil {
		t.Fatal(e)
	}
	l, e := app.Lobbies.JoinLobby(ctx, lobby.JoinContext{UserID: users[0], WorldID: w.ID, DisplayName: "Host", AvatarID: "pumpkin"}, "race")
	if e != nil {
		t.Fatal(e)
	}
	if e = app.Lobbies.SetReady(ctx, users[0], l.ID, true); e != nil {
		t.Fatal(e)
	}
	app.Lobbies.SetStorageBlocked(false)
	m, e := startClearedMatch(t, ctx, app.Lobbies, app.Hub, clock, users[0], l.ID, "clear-rider")
	if e != nil {
		t.Fatal(e)
	}
	states, e = app.Hub.WorldActivities(ctx, w.ID)
	if e != nil {
		t.Fatal(e)
	}
	for _, s := range states {
		if s.Kind == "vehicles" {
			for _, v := range s.Payload.Vehicles {
				if v.RiderID != "" {
					t.Fatal("race started with free rider", v)
				}
			}
		}
		if s.Kind == "area" && s.AreaID == "race" {
			if s.Payload.Phase != "ACTIVE" || s.Payload.MatchID != m.MatchID {
				t.Fatal("race missing active lease", s)
			}
		}
	}
	s, e := app.Hub.Snapshot(ctx, seats[1])
	if e != nil {
		t.Fatal(e)
	}
	for _, a := range s.Avatars {
		if a.ID == users[1] && (a.X != 45 || a.Y != 49) {
			t.Fatal("visitor not at safe perimeter", a)
		}
	}
	app.Hub.Revoke(w.ID, users[1])
	if _, e = app.Hub.Execute(ctx, seats[1], realtime.Command{V: 2, CommandID: "stale", Seq: 2, WorldID: w.ID, Type: "activity.ride", Payload: raw}); e == nil {
		t.Fatal("revoked membership admitted ride")
	}
}
