//go:build integration

package application

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"strings"
	"testing"
	"time"
)

func TestRaceWebSocketContinuousInputUsesMovementBudget(t *testing.T) {
	p := testutil.Database(t)
	users, tokens := testutil.Users(t, p, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clock := &penaltyClock{at: time.Now()}
	app, err := newApplication(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true}, clock)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	w, err := app.Worlds.CreateWorld(ctx, users[0], "Wire race")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(app.Handler)
	defer server.Close()
	c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/realtime?worldId="+w.ID, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://127.0.0.1:4176"}, "Cookie": {"patra_session=" + tokens[0]}}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	if _, _, err = c.Read(ctx); err != nil {
		t.Fatal(err)
	}
	seq := uint64(0)
	send := func(kind string, payload any) json.RawMessage {
		t.Helper()
		seq++
		id := fmt.Sprint("race", seq)
		raw, _ := json.Marshal(map[string]any{"v": 1, "commandId": id, "seq": seq, "worldId": w.ID, "type": kind, "payload": payload})
		if err := c.Write(ctx, websocket.MessageText, raw); err != nil {
			t.Fatal(err)
		}
		for {
			_, raw, err := c.Read(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var event realtime.Event
			json.Unmarshal(raw, &event)
			var ack struct {
				CommandID string          `json:"commandId"`
				Result    json.RawMessage `json:"result"`
			}
			json.Unmarshal(event.Payload, &ack)
			if ack.CommandID == id {
				if event.Type != "command.ack" {
					t.Fatal(kind, string(event.Payload))
				}
				return ack.Result
			}
		}
	}
	raw := send("lobby.join", map[string]any{"arenaId": "race"})
	var l struct {
		ID       string `json:"id"`
		MatchID  string `json:"matchId"`
		StartsAt int64  `json:"startsAt"`
	}
	json.Unmarshal(raw, &l)
	send("lobby.ready", map[string]any{"lobbyId": l.ID, "ready": true})
	app.Lobbies.SetStorageBlocked(false)
	raw = send("match.start", map[string]any{"lobbyId": l.ID})
	json.Unmarshal(raw, &l)
	clock.set(l.StartsAt)
	app.Lobbies.Sync(ctx)
	for i := 0; i < 20; i++ {
		send("race.input", map[string]any{"matchId": l.MatchID, "throttle": 1, "brake": 0, "steer": 0})
	}
	clock.set(l.StartsAt + 200)
	app.Lobbies.Sync(ctx)
	s := send("race.snapshot", map[string]any{"matchId": l.MatchID})
	if strings.Contains(string(s), "seed") {
		t.Fatal("seed exposed")
	}
	var snap struct {
		Karts []struct {
			UserID string `json:"userId"`
			Speed  int    `json:"speed"`
		}
	}
	json.Unmarshal(s, &snap)
	moving := false
	for _, k := range snap.Karts {
		if k.UserID == users[0] && k.Speed > 0 {
			moving = true
		}
	}
	if !moving {
		t.Fatal("valid wire input never moved human")
	}
}
