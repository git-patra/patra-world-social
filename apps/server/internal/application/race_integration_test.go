//go:build integration

package application

import (
	"context"
	"encoding/json"
	"fmt"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/games/race"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"reflect"
	"testing"
	"time"
)

func TestRaceApplicationGridPhysicsDurabilityAndTimedReturn(t *testing.T) {
	p := testutil.Database(t)
	users, tokens := testutil.Users(t, p, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	clock := &penaltyClock{at: time.Now()}
	app, err := newApplication(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true}, clock)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	w, err := app.Worlds.CreateWorld(ctx, users[0], "Synthetic race")
	if err != nil {
		t.Fatal(err)
	}
	p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, w.ID, users[1])
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
	counter := uint64(0)
	execute := func(j realtime.JoinContext, kind string, payload any) (any, error) {
		counter++
		if err := app.Hub.Validate(ctx, j); err != nil {
			return nil, err
		}
		raw, _ := json.Marshal(payload)
		return app.Hub.Execute(ctx, j, realtime.Command{V: 2, CommandID: fmt.Sprint(counter), Seq: counter, WorldID: j.WorldID, Type: kind, Payload: raw})
	}
	v, err := execute(seats[0], "lobby.join", map[string]any{"arenaId": "race"})
	if err != nil {
		t.Fatal(err)
	}
	l := v.(lobby.Lobby)
	execute(seats[0], "lobby.ready", map[string]any{"lobbyId": l.ID, "ready": true})
	app.Lobbies.SetStorageBlocked(false)
	m, err := startClearedMatch(t, ctx, app.Lobbies, app.Hub, clock, users[0], l.ID, "race-start")
	if err != nil {
		t.Fatal(err)
	}
	if _, e := app.Lobbies.StartMatch(ctx, users[0], l.ID, "duplicate-new-command"); e == nil {
		t.Fatal("race started twice")
	}
	repeat, e := startClearedMatch(t, ctx, app.Lobbies, app.Hub, clock, users[0], l.ID, "race-start")
	if e != nil || !reflect.DeepEqual(repeat, m) {
		t.Fatal("retry reshuffled frozen start", e)
	}
	var stored model.MatchStart
	var raw []byte
	if e = p.QueryRow(ctx, `SELECT start_payload FROM matches WHERE id=$1`, m.MatchID).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &stored)
	if !reflect.DeepEqual(stored.Actors, m.Actors) {
		t.Fatal("persisted grid differs")
	}
	for i, a := range m.Actors {
		var slot int
		if e = p.QueryRow(ctx, `SELECT roster_slot FROM match_participants WHERE match_id=$1 AND actor_id=$2`, m.MatchID, a.ActorID).Scan(&slot); e != nil || slot != i {
			t.Fatal("roster slot not prepared grid", e)
		}
	}
	snapshot := func() race.PublicSnapshot {
		t.Helper()
		var s race.PublicSnapshot
		if e := app.Lobbies.ReadEngine(ctx, w.ID, m.MatchID, func(engine lobby.Engine) error { s = engine.(*race.Adapter).Snapshot(); return nil }); e != nil {
			t.Fatal(e)
		}
		return s
	}
	first := snapshot()
	if first.Phase != "countdown" || len(first.Karts) != 5 {
		t.Fatal(first)
	}
	for i, k := range first.Karts {
		if k.Actor != m.Actors[i] || k.StartSlot != i {
			t.Fatal("runtime changed grid")
		}
	}
	if _, e := execute(seats[1], "race.snapshot", map[string]any{"matchId": m.MatchID}); e != nil {
		t.Fatal("spectator denied public view", e)
	}
	if _, e := execute(seats[1], "race.input", map[string]any{"matchId": m.MatchID, "throttle": 1, "brake": 0, "steer": 0}); e == nil {
		t.Fatal("spectator drove")
	}
	if e := app.Lobbies.ReadEngine(ctx, "other", m.MatchID, func(lobby.Engine) error { return nil }); e == nil {
		t.Fatal("cross-world snapshot")
	}

	clock.set(m.StartsAt - 1)
	app.Lobbies.Sync(ctx)
	pre := snapshot()
	for i, k := range pre.Karts {
		if k.X != first.Karts[i].X || k.Y != first.Karts[i].Y {
			t.Fatal("countdown movement")
		}
	}
	clock.set(m.StartsAt)
	app.Lobbies.Sync(ctx)
	if _, e := execute(seats[0], "race.input", map[string]any{"matchId": m.MatchID, "throttle": 1, "brake": 0, "steer": 0}); e != nil {
		t.Fatal(e)
	}
	for at := m.StartsAt + 50; at <= m.StartsAt+1000; at += 50 {
		clock.set(at)
		app.Lobbies.Sync(ctx)
	}
	s := snapshot()
	if s.Frame != 30 || s.DroppedSteps != 0 {
		t.Fatal("application scheduler did not step30Hz", s.Frame, s.DroppedSteps)
	}
	observedFirstFinisher := false
	for at := m.StartsAt + 1050; at <= m.StartsAt+300000; at += 50 {
		clock.set(at)
		app.Lobbies.Sync(ctx)
		s = snapshot()
		if !observedFirstFinisher && s.Phase != "finished" {
			for _, kart := range s.Karts {
				if kart.Finished {
					observedFirstFinisher = true
					areas, err := app.Hub.WorldActivities(ctx, w.ID)
					if err != nil {
						t.Fatal(err)
					}
					active := false
					for _, a := range areas {
						if a.Kind == "area" && a.AreaID == "race" {
							active = a.Payload.Phase == "ACTIVE" && a.Payload.MatchID == m.MatchID
						}
					}
					if !active {
						t.Fatal("first finisher reopened track while other racers remain")
					}
					break
				}
			}
		}
		if s.Phase == "finished" {
			break
		}
	}
	if !observedFirstFinisher {
		t.Fatal("race fixture did not exercise a partial finish")
	}
	if s.Phase != "finished" || s.Result == nil {
		t.Fatal("not naturally completed")
	}
	bots := 0
	for _, k := range s.Karts {
		if k.ActorType == "bot" && k.Finished {
			bots++
		}
	}
	if bots != 4 {
		t.Fatal("bot simulator did not finish", bots)
	}
	until := time.Now().Add(4 * time.Second)
	saved := false
	for time.Now().Before(until) {
		all, _ := app.Lobbies.List(ctx, w.ID)
		for _, l := range all {
			if l.MatchID == m.MatchID && l.State == "results" {
				saved = true
			}
		}
		if saved {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !saved {
		r, copyErr := app.Lobbies.CopyResult(ctx, m.MatchID)
		enqueueErr := app.History.Enqueue(ctx, r)
		_, finalizeErr := app.History.Finalize(ctx, m.MatchID)
		t.Fatal("durable result did not reach results", copyErr, enqueueErr, finalizeErr)
	}
	record, e := app.History.Detail(ctx, users[0], m.MatchID)
	if e != nil || record.Status != "completed" {
		t.Fatal("history missing", e)
	}
	var count int
	if e = p.QueryRow(ctx, `SELECT count(*) FROM race_results r JOIN match_participants p USING(match_id,actor_id) WHERE r.match_id=$1 AND r.start_slot=p.roster_slot`, m.MatchID).Scan(&count); e != nil || count != 5 {
		t.Fatal("projector grid mismatch", count, e)
	}
	if _, e = app.History.Finalize(ctx, m.MatchID); e != nil {
		t.Fatal(e)
	}
	p.QueryRow(ctx, `SELECT count(*) FROM race_results WHERE match_id=$1`, m.MatchID).Scan(&count)
	if count != 5 {
		t.Fatal("retry duplicated result")
	}
	end := clock.Now().UnixMilli()
	clock.set(end + 9999)
	app.Lobbies.Sync(ctx)
	if !app.Lobbies.Busy(users[0]) {
		t.Fatal("results released before10s")
	}
	clock.set(end + 10000)
	app.Lobbies.Sync(ctx)
	until = time.Now().Add(3 * time.Second)
	for app.Lobbies.Busy(users[0]) && time.Now().Before(until) {
		time.Sleep(10 * time.Millisecond)
	}
	if app.Lobbies.Busy(users[0]) {
		t.Fatal("results arena locked")
	}
	all, _ := app.Lobbies.List(ctx, w.ID)
	for _, l := range all {
		if l.Arena.ID == "race" && l.State != "idle" {
			t.Fatal("race did not return idle", l.State)
		}
	}
}

func TestRaceApplicationFiveHumanDeadlinePrecedesGraceExpiry(t *testing.T) {
	p := testutil.Database(t)
	users, _ := testutil.Users(t, p, 5)
	ctx := context.Background()
	clock := &penaltyClock{at: time.Now()}
	app, e := newApplication(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true}, clock)
	if e != nil {
		t.Fatal(e)
	}
	defer app.Close()
	w, e := app.Worlds.CreateWorld(ctx, users[0], "Five human limit")
	if e != nil {
		t.Fatal(e)
	}
	var l lobby.Lobby
	for _, user := range users[1:] {
		if _, err := p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, w.ID, user); err != nil {
			t.Fatal(err)
		}
	}
	for _, user := range users {
		l, e = app.Lobbies.JoinLobby(ctx, lobby.JoinContext{UserID: user, WorldID: w.ID, DisplayName: "Synthetic"}, "race")
		if e != nil {
			t.Fatal(e)
		}
		app.Lobbies.SetReady(ctx, user, l.ID, true)
	}
	app.Lobbies.SetStorageBlocked(false)
	m, e := startClearedMatch(t, ctx, app.Lobbies, app.Hub, clock, users[0], l.ID, "limit")
	if e != nil {
		t.Fatal(e)
	}
	clock.set(m.StartsAt + 285000)
	for _, user := range users {
		app.Lobbies.Notify("disconnect", w.ID, user)
	}
	app.Lobbies.Sync(ctx)
	clock.set(m.StartsAt + 300000)
	app.Lobbies.Sync(ctx)
	r, e := app.Lobbies.CopyResult(ctx, m.MatchID)
	if e != nil {
		t.Fatal("natural deadline lost to simultaneous grace expiry", e)
	}
	if r.FinishedAt != m.StartsAt+300000 || len(r.Participants) != 5 {
		t.Fatal(r)
	}
	for _, p := range r.Participants {
		if p.ActorType != "human" || !p.Race.DNF || !p.BotAssisted {
			t.Fatal("frozen actor / DNF / assistance missing", p)
		}
	}
}
