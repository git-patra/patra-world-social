//go:build integration

package application

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	"patra-world-social/server/internal/games/duck"
	"patra-world-social/server/internal/games/model"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/leaderboard"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"strings"
	"testing"
	"time"
)

func TestLeaderboardHTTPCommittedWorldTransportAndRecovery(t *testing.T) {
	p := testutil.Database(t)
	users, tokens := testutil.Users(t, p, 3)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	app, e := New(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true})
	if e != nil {
		t.Fatal(e)
	}
	defer app.Close()
	world, e := app.Worlds.CreateWorld(ctx, users[0], "Podium")
	if e != nil {
		t.Fatal(e)
	}
	invite, e := app.Worlds.IssueInvite(ctx, users[0], world.ID, users[1])
	if e != nil {
		t.Fatal(e)
	}
	if _, e = app.Worlds.AcceptTargetedInvite(ctx, users[1], invite.ID); e != nil {
		t.Fatal(e)
	}
	other, e := app.Worlds.CreateWorld(ctx, users[2], "Other")
	if e != nil {
		t.Fatal(e)
	}
	srv := httptest.NewServer(app.Handler)
	defer srv.Close()
	request := func(user int, path string, status int) []byte {
		t.Helper()
		req, e := http.NewRequestWithContext(ctx, "GET", srv.URL+path, nil)
		if e != nil {
			t.Fatal(e)
		}
		if user >= 0 {
			req.AddCookie(&http.Cookie{Name: "patra_session", Value: tokens[user]})
		}
		res, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		var raw json.RawMessage
		if e = json.NewDecoder(res.Body).Decode(&raw); e != nil && status != 401 {
			t.Fatal(e)
		}
		if res.StatusCode != status {
			t.Fatalf("%s status %d want %d: %s", path, res.StatusCode, status, raw)
		}
		if status == 200 && res.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("private response cacheable")
		}
		return raw
	}
	boardPath := "/api/v1/leaderboards/" + world.ID
	request(-1, boardPath, 401)
	request(2, boardPath, 404)
	request(0, "/api/v1/leaderboards/not-a-uuid", 400)
	var empty leaderboard.Board
	json.Unmarshal(request(0, boardPath, 200), &empty)
	if len(empty.Top) != 0 {
		t.Fatal("fake scores")
	}
	dial := func(i int, w string) *websocket.Conn {
		t.Helper()
		c, _, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/api/v1/realtime?worldId="+w, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://127.0.0.1:4176"}, "Cookie": {"patra_session=" + tokens[i]}}})
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { c.CloseNow() })
		return c
	}
	peers := []*websocket.Conn{dial(0, world.ID), dial(1, world.ID)}
	outsider := dial(2, other.ID)
	// Observe actual handshake before publishing; no preview accounts/session injection.
	readUntil := func(c *websocket.Conn, kind string, check func(realtime.Event)) realtime.Event {
		t.Helper()
		for {
			_, raw, e := c.Read(ctx)
			if e != nil {
				t.Fatal(e)
			}
			ev, e := realtime.ValidateEvent(raw)
			if e != nil {
				t.Fatal(e)
			}
			if check != nil {
				check(ev)
			}
			if ev.Type == kind {
				return ev
			}
		}
	}
	for _, c := range append(peers, outsider) {
		readUntil(c, "world.snapshot", nil)
	}
	id := "a1000000-0000-4000-8000-000000000001"
	m := model.MatchStart{MatchID: id, WorldID: world.ID, ArenaID: "duck-1", GameType: "duck", Version: model.Version{MapID: "fixture", MapVersion: 1, RulesID: "duck-v1", RulesVersion: 1, RulesChecksum: duck.RulesChecksum(), ProtocolVersion: 1}, PreparedAt: 1000, Actors: []model.Actor{{ActorID: id, ActorType: "human", UserID: users[0], DisplayName: "Synthetic"}}}
	if e = (&lobby.PGStore{Pool: p}).Persist(ctx, m, users[0], "synthetic-podium"); e != nil {
		t.Fatal(e)
	}
	score := int64(500)
	result := model.MatchResult{Version: m.Version, MatchID: id, WorldID: world.ID, ArenaID: m.ArenaID, GameType: "duck", PreparedAt: 1000, StartsAt: 4000, FinishedAt: 64000, Participants: []model.ParticipantResult{{Actor: m.Actors[0], Outcome: "finished", Score: &score, Duck: &model.DuckResult{Hits: 1, Shots: 2}}}}
	// The worker discovers this durable job as recovery work. Holding the partition
	// proves it cannot announce high score before the complete transaction commits.
	tx, e := p.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `INSERT INTO leaderboard_boards(world_id,game_type,ruleset)VALUES($1,'duck','duck-v1@1')`, world.ID); e != nil {
		t.Fatal(e)
	}
	if e = app.History.Enqueue(ctx, result); e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(request(1, boardPath, 200), &empty)
	if len(empty.Top) != 0 {
		t.Fatal("uncommitted score visible")
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	for i, c := range peers {
		ev := readUntil(c, "leaderboard.changed", func(ev realtime.Event) {
			if i == 1 && ev.Type == "match.result_status" {
				t.Fatal("spectator received private status")
			}
		})
		var notice leaderboard.Invalidation
		if e = json.Unmarshal(ev.Payload, &notice); e != nil {
			t.Fatal(e)
		}
		if notice.WorldID != world.ID || notice.ResultID != id || notice.Ruleset != "duck-v1@1" {
			t.Fatal(notice)
		}
		if strings.Contains(string(ev.Payload), "score") || strings.Contains(string(ev.Payload), users[0]) {
			t.Fatal("private payload widened")
		}
		var b leaderboard.Board
		json.Unmarshal(request(i, boardPath, 200), &b)
		if len(b.Top) != 1 || b.Top[0].Score != 500 || len(b.Changes) != 1 {
			t.Fatal(b)
		}
		if i == 1 && b.PersonalBest != nil {
			t.Fatal("other account PB")
		}
	}
	request(1, boardPath+"/results/"+id, 200)
	request(1, "/api/v1/history/"+id, 404)
	request(2, boardPath+"/results/"+id, 404)
	request(0, "/api/v1/leaderboards/"+other.ID+"/results/"+id, 404)
	if _, e = app.History.Finalize(ctx, id); e != nil {
		t.Fatal(e)
	} // Exact replay must not add another victory.
	for _, c := range peers {
		app.Hub.Publish(world.ID, "presence.changed", map[string]any{"barrier": true})
		readUntil(c, "presence.changed", func(ev realtime.Event) {
			if ev.Type == "leaderboard.changed" {
				t.Fatal("duplicate completion invalidation")
			}
		})
	}
	app.Hub.Publish(other.ID, "presence.changed", map[string]any{"barrier": true})
	readUntil(outsider, "presence.changed", func(ev realtime.Event) {
		if ev.Type == "leaderboard.changed" || ev.Type == "match.result_status" {
			t.Fatal("other-world result leak", fmt.Sprint(ev))
		}
	})
	// A load after a missed notification/reconnect still returns the durable board.
	peers[1].CloseNow()
	var b leaderboard.Board
	json.Unmarshal(request(1, boardPath, 200), &b)
	if len(b.Top) != 1 {
		t.Fatal(b)
	}
	if e = app.Worlds.KickMember(ctx, users[0], world.ID, users[1]); e != nil {
		t.Fatal(e)
	}
	request(1, boardPath, 404)
	request(1, boardPath+"/results/"+id, 404)
	request(0, "/api/v1/history/"+id, 200)
}

func TestLeaderboardStartupImportsPriorCompletedHistory(t *testing.T) {
	p := testutil.Database(t)
	users, _ := testutil.Users(t, p, 1)
	ctx := context.Background()
	options := httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true}
	app, e := New(p, options)
	if e != nil {
		t.Fatal(e)
	}
	w, e := app.Worlds.CreateWorld(ctx, users[0], "Prior Duck")
	if e != nil {
		t.Fatal(e)
	}
	app.Close()
	// Prior server composition: only Duck detail projection existed before T12.
	app.History.Projectors["duck"] = duck.Project
	id := "a2000000-0000-4000-8000-000000000001"
	m := model.MatchStart{MatchID: id, WorldID: w.ID, ArenaID: "duck-1", GameType: "duck", Version: model.Version{MapID: "fixture", MapVersion: 1, RulesID: "duck-v1", RulesVersion: 1, RulesChecksum: duck.RulesChecksum(), ProtocolVersion: 1}, PreparedAt: 1000, Actors: []model.Actor{{ActorID: id, ActorType: "human", UserID: users[0], DisplayName: "Synthetic"}}}
	if e = (&lobby.PGStore{Pool: p}).Persist(ctx, m, users[0], "prior-podium"); e != nil {
		t.Fatal(e)
	}
	score := int64(100)
	r := model.MatchResult{Version: m.Version, MatchID: id, WorldID: w.ID, ArenaID: m.ArenaID, GameType: "duck", PreparedAt: 1000, StartsAt: 4000, FinishedAt: 64000, Participants: []model.ParticipantResult{{Actor: m.Actors[0], Outcome: "finished", Score: &score, Duck: &model.DuckResult{Hits: 1, Shots: 1}}}}
	if e = app.History.Enqueue(ctx, r); e != nil {
		t.Fatal(e)
	}
	if _, e = app.History.Finalize(ctx, id); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		restarted, e := New(p, options)
		if e != nil {
			t.Fatal(e)
		}
		board, err := restarted.Leaderboard.Board(ctx, users[0], w.ID, "duck-v1@1")
		restarted.Close()
		if err != nil || len(board.Top) != 1 || len(board.Changes) != 1 || board.Top[0].Score != 100 {
			t.Fatal(board, err)
		}
	}
}
