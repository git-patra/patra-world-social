//go:build integration

package application

import (
	"context"
	"encoding/json"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	"patra-world-social/server/internal/auth"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"patra-world-social/server/internal/world"
	"patra-world-social/server/internal/worlds"
	"strings"
	"testing"
	"time"
)

func TestLobbyProtocolUsesAdmittedIdentityAndPublicResync(t *testing.T) {
	p := testutil.Database(t)
	users, tokens := testutil.Users(t, p, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	app, e := New(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true})
	if e != nil {
		t.Fatal(e)
	}
	defer app.Close()
	w, e := app.Worlds.CreateWorld(ctx, users[0], "Synthetic lobby")
	if e != nil {
		t.Fatal(e)
	}
	p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, w.ID, users[1])
	server := httptest.NewServer(app.Handler)
	defer server.Close()
	dial := func(i int) (*websocket.Conn, realtime.PublicSnapshot) {
		t.Helper()
		c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/realtime?protocolVersion=2&worldId="+w.ID, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://127.0.0.1:4176"}, "Cookie": {"patra_session=" + tokens[i]}}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { c.CloseNow() })
		_, raw, err := c.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var envelope realtime.Event
		json.Unmarshal(raw, &envelope)
		var snap realtime.PublicSnapshot
		json.Unmarshal(envelope.Payload, &snap)
		return c, snap
	}
	host, snap := dial(0)
	peer, _ := dial(1)
	execute := func(c *websocket.Conn, id, kind string, payload any) realtime.Event {
		t.Helper()
		raw, _ := json.Marshal(map[string]any{"v": 2, "commandId": id, "seq": 1, "worldId": w.ID, "type": kind, "payload": payload})
		if err := c.Write(ctx, websocket.MessageText, raw); err != nil {
			t.Fatal(err)
		}
		for {
			_, raw, err := c.Read(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var ev realtime.Event
			json.Unmarshal(raw, &ev)
			var ack struct {
				CommandID string `json:"commandId"`
			}
			json.Unmarshal(ev.Payload, &ack)
			if ack.CommandID == id {
				return ev
			}
		}
	}
	joined := execute(host, "join", "lobby.join", map[string]any{"arenaId": "duck-1"})
	if joined.Type != "command.ack" {
		t.Fatal(string(joined.Payload))
	}
	var ack struct {
		Result struct {
			ID              string `json:"id"`
			HostID          string `json:"hostId"`
			EngineAvailable bool   `json:"engineAvailable"`
		}
	}
	json.Unmarshal(joined.Payload, &ack)
	if ack.Result.HostID != users[0] || !ack.Result.EngineAvailable {
		t.Fatal(string(joined.Payload))
	}
	if ev := execute(peer, "spoof-host", "match.start", map[string]any{"lobbyId": ack.Result.ID}); ev.Type != "command.error" {
		t.Fatal("non-host started", string(ev.Payload))
	}
	if ev := execute(host, "not-ready", "match.start", map[string]any{"lobbyId": ack.Result.ID}); ev.Type != "command.error" {
		t.Fatal("unready production game started")
	}
	if ev := execute(host, "ready", "lobby.ready", map[string]any{"lobbyId": ack.Result.ID, "ready": true}); ev.Type != "command.ack" {
		t.Fatal(string(ev.Payload))
	}
	listed := execute(host, "list", "lobby.list", map[string]any{})
	if strings.Contains(string(listed.Payload), "seed") || !strings.Contains(string(listed.Payload), "duck-4") {
		t.Fatal(string(listed.Payload))
	}
	if app.Hub.Submit(ctx, realtime.JoinContext{Token: tokens[0], WorldID: w.ID, SessionID: snap.SessionID, Epoch: snap.Epoch}, realtime.Command{V: 2, CommandID: "move", WorldID: w.ID, Seq: 2, Type: "player.input", Payload: json.RawMessage(`{"dx":1,"dy":0}`)}) == nil {
		t.Fatal("busy user moved")
	}
	if ev := execute(host, "leave", "lobby.leave", map[string]any{}); ev.Type != "command.ack" {
		t.Fatal(string(ev.Payload))
	}
	// Current membership revocation must also release generic lobby ownership.
	execute(host, "join-again", "lobby.join", map[string]any{"arenaId": "penalty"})
	app.Hub.Revoke(w.ID, users[0])
	app.Lobbies.Sync(ctx)
	if app.Lobbies.Busy(users[0]) {
		t.Fatal("revoked lobby seat retained")
	}
}

type gatedArenaReturn struct {
	hub              *realtime.Hub
	observed, resume chan struct{}
}

func (g *gatedArenaReturn) ReturnActivity(ctx context.Context, w, u string, at world.Point, guard world.ActivityGuard) error {
	close(g.observed)
	select {
	case <-g.resume:
	case <-ctx.Done():
		return ctx.Err()
	}
	return g.hub.ReturnActivity(ctx, w, u, at, guard)
}

type rejectedArenaReturn struct{}

func (rejectedArenaReturn) ReturnActivity(context.Context, string, string, world.Point, world.ActivityGuard) error {
	return world.ErrBusy
}

func arenaReturnFixture(t *testing.T) (context.Context, *lobby.Service, *realtime.Hub, realtime.JoinContext, string, func(string) error) {
	t.Helper()
	p := testutil.Database(t)
	users, tokens := testutil.Users(t, p, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	a := auth.New(p)
	w := worlds.New(p)
	m, e := world.LoadMap(world.MapAsset)
	if e != nil {
		t.Fatal(e)
	}
	hub := realtime.NewHub(a, w, m, world.WallClock{})
	g := wireLobbies(p, hub, m)
	t.Cleanup(func() { hub.Close(); g.Close() })
	room, e := w.CreateWorld(ctx, users[0], "Synthetic return")
	if e != nil {
		t.Fatal(e)
	}
	j := realtime.JoinContext{Token: tokens[0], WorldID: room.ID}
	snap, e := hub.Join(ctx, j)
	if e != nil {
		t.Fatal(e)
	}
	j.SessionID = snap.SessionID
	j.Epoch = snap.Epoch
	join := func(arena string) error {
		payload, _ := json.Marshal(map[string]string{"arenaId": arena})
		_, e := hub.Execute(ctx, j, realtime.Command{V: 2, CommandID: "join-" + arena, WorldID: room.ID, Type: "lobby.join", Payload: payload})
		return e
	}
	if e = join("race"); e != nil {
		t.Fatal(e)
	}
	if e = g.Leave(ctx, users[0], room.ID); e != nil {
		t.Fatal(e)
	}
	return ctx, g, hub, j, users[0], join
}
func TestArenaReturnCannotOverwriteNewerLobbyClaim(t *testing.T) {
	ctx, g, hub, j, user, join := arenaReturnFixture(t)
	changes, e := g.DrainChanges(ctx)
	if e != nil || len(changes.Returns) != 1 {
		t.Fatal(changes, e)
	}
	gate := &gatedArenaReturn{hub: hub, observed: make(chan struct{}), resume: make(chan struct{})}
	done := make(chan bool, 1)
	go func() { done <- applyLobbyReturns(ctx, g, gate, changes.Returns) }()
	<-gate.observed
	if e = join("penalty"); e != nil {
		t.Fatal(e)
	}
	before, e := hub.Snapshot(ctx, j)
	if e != nil {
		t.Fatal(e)
	}
	close(gate.resume)
	<-done
	after, e := hub.Snapshot(ctx, j)
	if e != nil {
		t.Fatal(e)
	}
	if !g.Busy(user) || before.Avatars[0].X != after.Avatars[0].X || before.Avatars[0].Y != after.Avatars[0].Y || after.Avatars[0].Status != "playing" {
		t.Fatalf("stale arena return overwrote new claim: before=%+v after=%+v", before.Avatars, after.Avatars)
	}
}
func TestArenaReturnRemainsPendingAfterDeliveryError(t *testing.T) {
	ctx, g, hub, j, _, _ := arenaReturnFixture(t)
	changes, e := g.DrainChanges(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if !applyLobbyReturns(ctx, g, rejectedArenaReturn{}, changes.Returns) {
		t.Fatal("delivery error was treated as successful")
	}
	retry, e := g.DrainChanges(ctx)
	if e != nil || len(retry.Returns) != 1 {
		t.Fatal("failed return was lost", retry, e)
	}
	if applyLobbyReturns(ctx, g, hub, retry.Returns) {
		t.Fatal("successful return requested retry")
	}
	snap, e := hub.Snapshot(ctx, j)
	if e != nil {
		t.Fatal(e)
	}
	if snap.Avatars[0].X != 45 || snap.Avatars[0].Y != 49 || snap.Avatars[0].Status != "available" {
		t.Fatal(snap.Avatars)
	}
	done, e := g.DrainChanges(ctx)
	if e != nil || len(done.Returns) != 0 {
		t.Fatal("delivered return remained pending", done, e)
	}
}
