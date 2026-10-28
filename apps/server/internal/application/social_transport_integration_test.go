//go:build integration

package application

import (
	"context"
	"encoding/json"
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

func TestSocialHTTPAndDurableChatTransport(t *testing.T) {
	p := testutil.Database(t)
	users, tokens := testutil.Users(t, p, 3)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	app, e := New(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true})
	if e != nil {
		t.Fatal(e)
	}
	defer app.Close()
	w, e := app.Worlds.CreateWorld(ctx, users[0], "Pumpkin")
	if e != nil {
		t.Fatal(e)
	}
	invite, e := app.Worlds.IssueInvite(ctx, users[0], w.ID, users[1])
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
	request := func(method, path, body string, csrf *http.Cookie, origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: "patra_session", Value: tokens[0]})
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", origin)
		if csrf != nil {
			r.AddCookie(csrf)
			r.Header.Set("X-CSRF-Token", csrf.Value)
		}
		out := httptest.NewRecorder()
		app.Handler.ServeHTTP(out, r)
		return out
	}
	if out := request("POST", "/api/v1/friends/request", `{"username":"synthetic_1"}`, nil, "http://127.0.0.1:4176"); out.Code != 403 {
		t.Fatal("CSRF missing allowed", out.Code)
	}
	csrfResponse := request("GET", "/api/v1/auth/csrf", "", nil, "")
	csrf := csrfResponse.Result().Cookies()[0]
	if out := request("POST", "/api/v1/friends/request", `{"username":"synthetic_1"}`, csrf, "https://evil.invalid"); out.Code != 403 {
		t.Fatal("bad origin allowed")
	}
	if out := request("POST", "/api/v1/friends/request", `{"username":"synthetic_1","unexpected":true}`, csrf, "http://127.0.0.1:4176"); out.Code != 400 {
		t.Fatal("unknown JSON allowed", out.Code)
	}
	if out := request("POST", "/api/v1/friends/request", `{"username":"synthetic_1"}`, csrf, "http://127.0.0.1:4176"); out.Code != 204 {
		t.Fatal("friend API failed", out.Code, out.Body.String())
	}
	srv := httptest.NewServer(app.Handler)
	defer srv.Close()
	dial := func(i int, world, extra string) (*websocket.Conn, realtime.PublicSnapshot) {
		t.Helper()
		c, _, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/api/v1/realtime?worldId="+world+extra, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://127.0.0.1:4176"}, "Cookie": {"patra_session=" + tokens[i]}}})
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { c.CloseNow() })
		_, raw, e := c.Read(ctx)
		if e != nil {
			t.Fatal(e)
		}
		var ev realtime.Event
		if e = json.Unmarshal(raw, &ev); e != nil {
			t.Fatal(e)
		}
		if ev.Type != "world.snapshot" {
			t.Fatal("snapshot", string(raw))
		}
		var s realtime.PublicSnapshot
		if e = json.Unmarshal(ev.Payload, &s); e != nil {
			t.Fatal(e)
		}
		return c, s
	}
	owner, _ := dial(0, w.ID, "")
	friend, first := dial(1, w.ID, "")
	outsider, _ := dial(2, other.ID, "")
	readType := func(c *websocket.Conn, kind string) realtime.Event {
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
			if ev.Type == kind {
				return ev
			}
		}
	}
	send := func(command string) {
		payload := map[string]any{"v": 1, "commandId": command, "seq": 1, "worldId": w.ID, "type": "chat.send", "payload": map[string]string{"channelId": "world", "clientMessageId": "persistent-retry", "text": "<script>world only</script>"}}
		raw, _ := json.Marshal(payload)
		if e = owner.Write(ctx, websocket.MessageText, raw); e != nil {
			t.Fatal(e)
		}
	}
	send("control-1")
	ack := readType(owner, "command.ack")
	var result struct {
		CommandID string `json:"commandId"`
		Result    struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"result"`
	}
	if e = json.Unmarshal(ack.Payload, &result); e != nil || result.CommandID != "control-1" || result.Result.ID == "" {
		t.Fatal("durable result ack", e)
	}
	message := readType(friend, "chat.message")
	if !strings.Contains(string(message.Payload), result.Result.ID) {
		t.Fatal("live broadcast not persisted ID")
	}
	send("control-2")
	second := readType(owner, "command.ack")
	var retry struct {
		Result struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	json.Unmarshal(second.Payload, &retry)
	if retry.Result.ID != result.Result.ID {
		t.Fatal("reconnect dedup identity")
	}
	// Other world's next frame remains its own replaceable snapshot.
	if ev := readType(outsider, "world.delta"); strings.Contains(string(ev.Payload), users[0]) {
		t.Fatal("other-world chat/presence leak")
	}
	friend.CloseNow()
	extra := "&sessionId=" + first.SessionID + "&epoch=1"
	resumed, snapshot := dial(1, w.ID, extra)
	if snapshot.Epoch <= first.Epoch {
		t.Fatal("epoch did not rotate")
	}
	history, e := app.Chat.History(ctx, users[1], w.ID, "")
	if e != nil || len(history) != 1 || history[0].ID != result.Result.ID {
		t.Fatal("resync history dedup", e, len(history))
	}
	if e = app.Worlds.KickMember(ctx, users[0], w.ID, users[1]); e != nil {
		t.Fatal(e)
	}
	for {
		_, _, e = resumed.Read(ctx)
		if e != nil {
			break
		}
	}
	if websocket.CloseStatus(e) != websocket.StatusPolicyViolation {
		t.Fatal("kick transport not revoked", e)
	}
}
