//go:build integration

package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"patra-world-social/server/internal/games/duck"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/history"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"patra-world-social/server/internal/world"
	"testing"
	"time"
)

func TestFishingActualDatabaseNeverCallsCompletionOrPersistsCatch(t *testing.T) {
	pool := testutil.Database(t)
	users, tokens := testutil.Users(t, pool, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	clock := &penaltyClock{at: time.Now()}
	app, e := newApplication(pool, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true}, clock)
	if e != nil {
		t.Fatal(e)
	}
	defer app.Close()
	w, e := app.Worlds.CreateWorld(ctx, users[0], "Fishing test")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = pool.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role) VALUES($1,$2,'member')`, w.ID, users[1]); e != nil {
		t.Fatal(e)
	}
	seats := []realtime.JoinContext{}
	for n := range users {
		s, e := app.Hub.Join(ctx, realtime.JoinContext{Token: tokens[n], WorldID: w.ID})
		if e != nil {
			t.Fatal(e)
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
	at := world.Point{X: 53, Y: 30}
	if e = app.Hub.SetActivity(ctx, w.ID, users[0], false, &at); e != nil {
		t.Fatal(e)
	}
	// Seed via the same durable fixture path used by history/finalizer tests.
	m := model.MatchStart{MatchID: "11111111-1111-4111-8111-111111111111", WorldID: w.ID, ArenaID: "duck-1", GameType: "duck", Version: model.Version{MapID: "fixture", MapVersion: 1, RulesID: "duck-v1", RulesVersion: 1, RulesChecksum: duck.RulesChecksum(), ProtocolVersion: 1}, PreparedAt: 1000, Actors: []model.Actor{{ActorID: "22222222-2222-4222-8222-222222222222", ActorType: "human", UserID: users[0], DisplayName: "synthetic"}}}
	if e = (&lobby.PGStore{Pool: pool}).Persist(ctx, m, users[0], "existing-history"); e != nil {
		t.Fatal(e)
	}
	score := int64(300)
	result := model.MatchResult{Version: m.Version, MatchID: m.MatchID, WorldID: m.WorldID, ArenaID: m.ArenaID, GameType: m.GameType, PreparedAt: m.PreparedAt, StartsAt: 4000, FinishedAt: 64000, Participants: []model.ParticipantResult{{Actor: m.Actors[0], Outcome: "finished", Score: &score, Duck: &model.DuckResult{Hits: 1, Shots: 2}}}}
	if e = app.History.Enqueue(ctx, result); e != nil {
		t.Fatal(e)
	}
	if _, e = app.History.Finalize(ctx, m.MatchID); e != nil {
		t.Fatal(e)
	}
	if _, e = pool.Exec(ctx, `DELETE FROM active_match_users WHERE match_id=$1`, m.MatchID); e != nil {
		t.Fatal(e)
	}
	historyJSON := func() []byte {
		t.Helper()
		page, e := app.History.ListHistory(ctx, users[0], history.Filter{}, "")
		if e != nil || len(page.Items) != 1 || page.Items[0].Status != "completed" || page.Items[0].Result == nil {
			t.Fatal("existing account history missing", page, e)
		}
		b, e := json.Marshal(page)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	beforeHistory := historyJSON()
	fingerprint := func() map[string]string {
		t.Helper()
		out := map[string]string{}
		for _, table := range []string{"matches", "match_participants", "result_jobs", "active_match_users", "leaderboard_entries", "champion_changes", "duck_results"} {
			var n int
			var hash string
			if e := pool.QueryRow(ctx, `SELECT count(*),md5(coalesce(string_agg(row_to_json(t)::text,'' ORDER BY row_to_json(t)::text),'')) FROM `+table+` t`).Scan(&n, &hash); e != nil {
				t.Fatal(e)
			}
			out[table] = fmt.Sprintf("%d:%s", n, hash)
		}
		return out
	}
	before := fingerprint()
	start := clock.Now()
	state := func(j realtime.JoinContext) world.ActivitySnapshot {
		t.Helper()
		ss, e := app.Hub.ActivitySnapshots(ctx, j)
		if e != nil {
			t.Fatal(e)
		}
		for _, s := range ss {
			if s.Kind == "fishing" && s.AreaID == "fishing-0" {
				return s
			}
		}
		t.Fatal("missing fishing")
		return world.ActivitySnapshot{}
	}
	seq := uint64(0)
	execute := func(action string) error {
		seq++
		s := state(seats[0])
		i := world.CasualIntent{Incarnation: s.Incarnation, Generation: s.Generation, SpotID: s.AreaID, Action: action}
		raw, _ := json.Marshal(i)
		_, e := app.Hub.Execute(ctx, seats[0], realtime.Command{V: 2, WorldID: w.ID, CommandID: fmt.Sprint(seq), Seq: seq, Type: "activity.fishing", Payload: raw})
		return e
	}
	for n := 0; n < 40; n++ {
		if e = execute("fishing.cast"); e != nil {
			t.Fatal("cast", n, e)
		}
		s := state(seats[0])
		clock.set(s.Payload.Fishing.BiteAt)
		s = state(seats[0])
		if s.Payload.Fishing.Phase != "bite" {
			t.Fatal(s)
		}
		if e = execute("fishing.reel"); e != nil {
			t.Fatal("reel", e)
		}
		s = state(seats[0])
		if s.Payload.Fishing.CatchID == "" {
			t.Fatal("missing cosmetic catch")
		}
		peer := state(seats[1])
		if *peer.Payload.Fishing != *s.Payload.Fishing {
			t.Fatal("peer saw different catch")
		}
		if e = execute("fishing.reel"); e == nil {
			t.Fatal("duplicate reel")
		}
		clock.set(s.Payload.Fishing.EndsAt)
		if s = state(seats[0]); s.Payload.Fishing.CatchID != "" || s.Payload.Fishing.OwnerID != "" {
			t.Fatal("catch survived3s")
		}
	}
	if clock.Now().Sub(start) < 180*time.Second {
		t.Fatal("not several minutes")
	}
	if e = execute("fishing.cast"); e != nil {
		t.Fatal(e)
	}
	// Simulate a browser reload/disconnect, a fresh takeover, and explicit logout/leave.
	app.Hub.Disconnect(seats[0].SessionID, "reload")
	if state(seats[1]).Payload.Fishing.OwnerID != "" {
		t.Fatal("disconnect retained cast")
	}
	rejoin := func() {
		t.Helper()
		seat, e := app.Hub.Join(ctx, realtime.JoinContext{Token: tokens[0], WorldID: w.ID, Takeover: true})
		if e != nil {
			t.Fatal(e)
		}
		seats[0] = realtime.JoinContext{Token: tokens[0], WorldID: w.ID, SessionID: seat.SessionID, Epoch: seat.Epoch}
		seq = 0
		if e = app.Hub.SetActivity(ctx, w.ID, users[0], false, &at); e != nil {
			t.Fatal(e)
		}
	}
	rejoin()
	if e = execute("fishing.cast"); e != nil {
		t.Fatal(e)
	}
	app.Hub.Leave(seats[0])
	if state(seats[1]).Payload.Fishing.OwnerID != "" {
		t.Fatal("logout retained cast")
	}
	rejoin()
	if e = execute("fishing.cast"); e != nil {
		t.Fatal(e)
	}
	seq++
	raw, _ := json.Marshal(map[string]string{"arenaId": "pingpong"})
	if _, e = app.Hub.Execute(ctx, seats[0], realtime.Command{V: 2, WorldID: w.ID, CommandID: "join-match", Seq: seq, Type: "lobby.join", Payload: raw}); e != nil {
		t.Fatal(e)
	}
	if state(seats[0]).Payload.Fishing.OwnerID != "" {
		t.Fatal("match join retained cast")
	}
	if !bytes.Equal(beforeHistory, historyJSON()) {
		t.Fatal("existing personal account history changed")
	}
	after := fingerprint()
	for table, hash := range before {
		if after[table] != hash {
			t.Fatalf("fishing changed durable table %s: %s -> %s", table, hash, after[table])
		}
		t.Log(table, hash)
	}
	t.Logf("40 authenticated catch/repeat cycles across %s simulated; no durable count/hash change", clock.Now().Sub(start))
}
