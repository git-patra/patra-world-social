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
	"patra-world-social/server/internal/games/blaster"
	"patra-world-social/server/internal/history"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"strings"
	"testing"
	"time"
)

func TestBlasterActualTransportHistoryAndSpectator(t *testing.T) {
	for _, mode := range []string{"timeout", "all_down", "host_leave_reconnect"} {
		t.Run(mode, func(t *testing.T) { testBlasterActual(t, mode) })
	}
}
func testBlasterActual(t *testing.T, mode string) {
	pool := testutil.Database(t)
	users, tokens := testutil.Users(t, pool, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	clock := &penaltyClock{at: time.Now()}
	app, err := newApplication(pool, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true}, clock)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	world, err := app.Worlds.CreateWorld(ctx, users[0], "Blaster test")
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range users[1:] {
		if _, err = pool.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, world.ID, user); err != nil {
			t.Fatal(err)
		}
	}
	server := httptest.NewServer(app.Handler)
	defer server.Close()
	seats := make([]struct {
		SessionID string `json:"sessionId"`
		Epoch     uint64 `json:"epoch"`
	}, 5)
	dial := func(i int) *websocket.Conn {
		t.Helper()
		c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/realtime?protocolVersion=2&worldId="+world.ID+fmt.Sprintf("&sessionId=%s&epoch=%d", seats[i].SessionID, seats[i].Epoch), &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://127.0.0.1:4176"}, "Cookie": {"patra_session=" + tokens[i]}}})
		if err != nil {
			t.Fatal(err)
		}
		for {
			_, raw, e := c.Read(ctx)
			if e != nil {
				t.Fatal(e)
			}
			var ev realtime.Event
			json.Unmarshal(raw, &ev)
			if ev.Type == "world.snapshot" {
				if e = json.Unmarshal(ev.Payload, &seats[i]); e != nil {
					t.Fatal(e)
				}
				break
			}
			if ev.Type == "command.error" {
				t.Fatal(string(ev.Payload))
			}
		}
		t.Cleanup(func() { c.CloseNow() })
		return c
	}
	clients := []*websocket.Conn{dial(0), dial(1), dial(2), dial(3)}
	host, spectator := clients[0], dial(4)
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
	kind, raw := send(host, "lobby.join", map[string]any{"arenaId": "blaster"})
	if kind != "command.ack" {
		t.Fatal("join", kind)
	}
	var l lobby.Lobby
	json.Unmarshal(raw, &l)
	send(host, "lobby.ready", map[string]any{"lobbyId": l.ID, "ready": true})
	for _, peer := range clients[1:] {
		if kind, _ = send(peer, "lobby.join", map[string]any{"arenaId": "blaster"}); kind != "command.ack" {
			t.Fatal("peer join")
		}
		if kind, _ = send(peer, "lobby.ready", map[string]any{"lobbyId": l.ID, "ready": true}); kind != "command.ack" {
			t.Fatal("peer ready")
		}
	}
	if kind, _ = send(spectator, "lobby.join", map[string]any{"arenaId": "blaster"}); kind != "command.error" {
		t.Fatal("fifth participant admitted")
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
	snapshot := func(c *websocket.Conn) blaster.Snapshot {
		t.Helper()
		kind, raw := send(c, "blaster.snapshot", map[string]any{"matchId": mid})
		if kind != "command.ack" {
			t.Fatal("snapshot", kind)
		}
		var s blaster.Snapshot
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	s := snapshot(host)
	if len(s.Actors) != 4 {
		t.Fatal("four humans", s.Actors)
	}
	for _, a := range s.Actors {
		if a.ActorType != "human" {
			t.Fatal("friendly bot")
		}
	}
	intent := map[string]any{"matchId": mid, "dx": 0, "dy": 0, "aim": 0, "fire": true}
	if kind, _ = send(spectator, "blaster.input", intent); kind != "command.error" {
		t.Fatal("spectator damage")
	}
	for _, c := range clients {
		if kind, _ = send(c, "blaster.input", intent); kind != "command.ack" {
			t.Fatal("human input")
		}
	}
	if mode == "host_leave_reconnect" {
		clients[1].CloseNow()
		time.Sleep(40 * time.Millisecond)
		clients[1] = dial(1)
		if kind, _ = send(clients[1], "blaster.input", intent); kind != "command.ack" {
			t.Fatal("reconnect input")
		}
		if kind, _ = send(host, "lobby.leave", map[string]any{}); kind != "command.ack" {
			t.Fatal("host leave")
		}
		if snapshot(clients[1]).Result != nil {
			t.Fatal("host departure ended team")
		}
	}
	if kind, _ = send(clients[1], "match.return", map[string]any{"matchId": mid}); kind != "command.error" {
		t.Fatal("premature return")
	}
	if mode == "timeout" {
		clock.set(s.DeadlineAt)
		app.Lobbies.Sync(ctx)
	} else {
		for at := s.StartsAt + 50; at <= s.StartsAt+30000; at += 50 {
			clock.set(at)
			app.Lobbies.Sync(ctx)
		}
	}
	for _, c := range clients {
		snapshot(c)
	}
	s = snapshot(spectator)
	if s.Result == nil || s.Result.Blaster == nil {
		t.Fatal("result", s)
	}
	wantReason := "all_down"
	if mode == "timeout" {
		wantReason = "timeout"
	}
	if s.Result.Blaster.Reason != wantReason {
		t.Fatal("reason", s.Result.Blaster)
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
	if err != nil || record.Status != "completed" || record.Result == nil || record.Result.RulesID != "blaster-v1" {
		t.Fatal("durable", record, err)
	}
	if _, err = app.History.Detail(ctx, users[4], mid); err == nil {
		t.Fatal("spectator history")
	}
	page, err := app.History.ListHistory(ctx, users[0], history.Filter{Game: "blaster"}, "")
	if err != nil || len(page.Items) != 1 {
		t.Fatal("filter", page, err)
	}
	var count int
	pool.QueryRow(ctx, `SELECT count(*) FROM match_participants WHERE match_id=$1 AND user_id IS NOT NULL`, mid).Scan(&count)
	want := 4
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
	if mode == "timeout" {
		for _, c := range clients {
			if kind, _ = send(c, "match.rematch", map[string]any{"matchId": mid}); kind != "command.ack" {
				t.Fatal("mutual rematch")
			}
		}
		all, _ = app.Lobbies.List(ctx, world.ID)
		for _, v := range all {
			if v.Arena.ID == "blaster" && (v.ID == l.ID || v.State != "lobby" || v.Players[0].Ready) {
				t.Fatal("fresh rematch", v)
			}
		}
	}
}
