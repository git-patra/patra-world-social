//go:build integration

package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	"patra-world-social/server/internal/games/pingpong"
	"patra-world-social/server/internal/history"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"strings"
	"testing"
	"time"
)

func TestPingpongActualTransportHistoryAndSpectator(t *testing.T) {
	for _, bot := range []bool{true, false} {
		t.Run(map[bool]string{true: "BOT", false: "two humans"}[bot], func(t *testing.T) { testPingpongActual(t, bot) })
	}
}
func testPingpongActual(t *testing.T, bot bool) {
	pool := testutil.Database(t)
	users, tokens := testutil.Users(t, pool, 3)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	clock := &penaltyClock{at: time.Now()}
	app, err := newApplication(pool, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true}, clock)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	world, err := app.Worlds.CreateWorld(ctx, users[0], "Pingpong test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, world.ID, users[1]); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, world.ID, users[2]); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(app.Handler)
	defer server.Close()
	dial := func(i int) *websocket.Conn {
		t.Helper()
		c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/realtime?protocolVersion=2&worldId="+world.ID, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://127.0.0.1:4176"}, "Cookie": {"patra_session=" + tokens[i]}}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { c.CloseNow() })
		return c
	}
	host, peer, spectator := dial(0), dial(1), dial(2)
	seq := uint64(0)
	lastWrite := map[*websocket.Conn]time.Time{}
	send := func(c *websocket.Conn, kind string, payload any) (string, json.RawMessage) {
		t.Helper()
		if wait := 125*time.Millisecond - time.Since(lastWrite[c]); wait > 0 {
			time.Sleep(wait)
		}
		lastWrite[c] = time.Now()
		seq++
		id := fmt.Sprint("ping-", seq)
		p, _ := json.Marshal(payload)
		raw, _ := json.Marshal(realtime.Command{V: 2, WorldID: world.ID, CommandID: id, Seq: seq, Type: kind, Payload: p})
		if err := c.Write(ctx, websocket.MessageText, raw); err != nil {
			t.Fatal(err)
		}
		for {
			_, raw, err := c.Read(ctx)
			if err != nil {
				t.Fatal(err)
			}
			ev, err := realtime.ValidateEvent(raw)
			if err != nil {
				t.Fatal(err)
			}
			var ack struct {
				CommandID string          `json:"commandId"`
				Result    json.RawMessage `json:"result"`
			}
			json.Unmarshal(ev.Payload, &ack)
			if ack.CommandID == id {
				return ev.Type, ack.Result
			}
		}
	}
	kind, raw := send(host, "lobby.join", map[string]any{"arenaId": "pingpong"})
	if kind != "command.ack" {
		t.Fatal("join", kind)
	}
	var l lobby.Lobby
	json.Unmarshal(raw, &l)
	send(host, "lobby.ready", map[string]any{"lobbyId": l.ID, "ready": true})
	if kind, _ = send(host, "match.start", map[string]any{"lobbyId": l.ID}); kind != "command.error" {
		t.Fatal("implicit BOT")
	}
	if bot {
		if kind, _ = send(host, "lobby.bot", map[string]any{"lobbyId": l.ID, "enabled": true}); kind != "command.ack" {
			t.Fatal("explicit BOT")
		}
	} else {
		if kind, _ = send(peer, "lobby.join", map[string]any{"arenaId": "pingpong"}); kind != "command.ack" {
			t.Fatal("peer join")
		}
		if kind, _ = send(peer, "lobby.ready", map[string]any{"lobbyId": l.ID, "ready": true}); kind != "command.ack" {
			t.Fatal("peer ready")
		}
	}
	for n := 0; n < 40; n++ {
		kind, _ = send(host, "match.start", map[string]any{"lobbyId": l.ID})
		if kind == "command.ack" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if kind != "command.ack" {
		t.Fatal("start")
	}
	awaitClearedStart(t, ctx, app.Lobbies, app.Hub, clock, world.ID, l.ID)
	all, _ := app.Lobbies.List(ctx, world.ID)
	for _, v := range all {
		if v.ID == l.ID {
			l = v
		}
	}
	mid := l.MatchID
	if mid == "" {
		t.Fatal("match missing")
	}
	snapshot := func(c *websocket.Conn) pingpong.Snapshot {
		t.Helper()
		kind, raw := send(c, "pingpong.snapshot", map[string]any{"matchId": mid})
		if kind != "command.ack" {
			t.Fatal("snapshot", kind)
		}
		var s pingpong.Snapshot
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	s := snapshot(host)
	if len(s.Actors) != 2 || bot && (s.Actors[1].ActorType != "bot" || s.Actors[1].UserID != "") || !bot && s.Actors[1].ActorType != "human" {
		t.Fatal("BOT roster", s.Actors)
	}
	if kind, _ = send(spectator, "pingpong.input", map[string]any{"matchId": mid, "axis": 1}); kind != "command.error" {
		t.Fatal("spectator paddle")
	}
	if kind, _ = send(spectator, "pingpong.serve", map[string]any{"matchId": mid, "rallyId": s.RallyID}); kind != "command.error" {
		t.Fatal("spectator serve")
	}
	if kind, _ = send(host, "pingpong.serve", map[string]any{"matchId": mid, "rallyId": s.RallyID}); kind != "command.ack" {
		t.Fatal("serve")
	}
	if kind, _ = send(host, "pingpong.input", map[string]any{"matchId": mid, "axis": 1}); kind != "command.ack" {
		t.Fatal("paddle")
	}
	if kind, _ = send(host, "match.return", map[string]any{"matchId": mid}); kind != "command.error" {
		t.Fatal("premature return")
	}
	// Drive the real deadline while consuming both actual WebSocket delivery queues.
	for at := s.StartsAt + 5000; at <= s.DeadlineAt; at += 5000 {
		clock.set(at)
		app.Lobbies.Sync(ctx)
		snapshot(host)
		snapshot(spectator)
		snapshot(peer)
	}
	s = snapshot(host)
	if s.Result == nil || s.Result.Pingpong == nil || s.Result.Pingpong.Reason != "deadline" {
		t.Fatal("deadline", s)
	}
	until := time.Now().Add(4 * time.Second)
	for time.Now().Before(until) {
		record, e := app.History.Detail(ctx, users[0], mid)
		if e == nil && record.Status == "completed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	record, err := app.History.Detail(ctx, users[0], mid)
	if err != nil || record.Status != "completed" || record.Result == nil || record.Result.RulesID != "pingpong-v1" {
		t.Fatal("durable", record, err)
	}
	if _, err = app.History.Detail(ctx, users[2], mid); err == nil {
		t.Fatal("spectator history")
	}
	page, err := app.History.ListHistory(ctx, users[0], history.Filter{Game: "pingpong"}, "")
	if err != nil || len(page.Items) != 1 {
		t.Fatal("filter", page, err)
	}
	var count int
	pool.QueryRow(ctx, `SELECT count(*) FROM match_participants WHERE match_id=$1 AND user_id IS NOT NULL`, mid).Scan(&count)
	want := 2
	if bot {
		want = 1
	}
	if count != want {
		t.Fatal("spectator row", count)
	}
	var before, after []byte
	pool.QueryRow(ctx, `SELECT result_hash FROM result_jobs WHERE match_id=$1`, mid).Scan(&before)
	if _, err = app.History.Finalize(ctx, mid); err != nil {
		t.Fatal("retry", err)
	}
	pool.QueryRow(ctx, `SELECT result_hash FROM result_jobs WHERE match_id=$1`, mid).Scan(&after)
	if !bytes.Equal(before, after) {
		t.Fatal("result changed")
	}
	if kind, _ = send(host, "match.rematch", map[string]any{"matchId": mid}); kind != "command.ack" {
		t.Fatal("rematch")
	}
	if !bot {
		if kind, _ = send(peer, "match.rematch", map[string]any{"matchId": mid}); kind != "command.ack" {
			t.Fatal("mutual rematch")
		}
	}
	all, _ = app.Lobbies.List(ctx, world.ID)
	for _, v := range all {
		if v.Arena.ID == "pingpong" && (v.ID == l.ID || v.State != "lobby" || v.Players[0].Ready) {
			t.Fatal("fresh rematch", v)
		}
	}
}
