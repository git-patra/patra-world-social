package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	"patra-world-social/server/internal/world"
	"strings"
	"testing"
	"time"
)

func TestTransportAdmissionMovementAndRevocation(t *testing.T) {
	h := newTestHub(t)
	srv := httptest.NewServer(NewHandler(h, "http://village.test", true))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "?protocolVersion=2&worldId=a"
	for _, header := range []http.Header{{"Origin": {"http://evil.test"}, "Cookie": {"patra_session=u"}}, {"Origin": {"http://village.test"}}} {
		c, res, e := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: header})
		if e == nil {
			c.CloseNow()
			t.Fatal("unauthorized handshake accepted")
		}
		if res.StatusCode != 403 && res.StatusCode != 401 {
			t.Fatal(res.StatusCode)
		}
	}
	c, _, e := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://village.test"}, "Cookie": {"patra_session=u"}}})
	if e != nil {
		t.Fatal(e)
	}
	defer c.CloseNow()
	_, raw, e := c.Read(ctx)
	if e != nil {
		t.Fatal(e)
	}
	ev, e := ValidateEvent(raw)
	if e != nil || ev.Type != "world.snapshot" {
		t.Fatalf("snapshot %s %v", raw, e)
	}
	var first PublicSnapshot
	json.Unmarshal(ev.Payload, &first)
	cmd := []byte(`{"v":2,"commandId":"move","seq":1,"worldId":"a","type":"player.input","payload":{"dx":1,"dy":0}}`)
	if e = c.Write(ctx, websocket.MessageText, cmd); e != nil {
		t.Fatal(e)
	}
	moved := false
	for !moved {
		_, raw, e = c.Read(ctx)
		if e != nil {
			t.Fatal(e)
		}
		ev, e = ValidateEvent(raw)
		if e != nil {
			t.Fatal(e)
		}
		if ev.Type == "world.delta" {
			var s PublicSnapshot
			json.Unmarshal(ev.Payload, &s)
			moved = s.Avatars[0].X > first.Avatars[0].X
		}
	}
	h.Revoke("a", "u")
	for {
		_, _, e = c.Read(ctx)
		if e != nil {
			break
		}
	}
	if websocket.CloseStatus(e) != websocket.StatusPolicyViolation {
		t.Fatalf("revoke close %v", e)
	}
}
func TestMalformedFrameClosesWithPolicyReason(t *testing.T) {
	h := newTestHub(t)
	srv := httptest.NewServer(NewHandler(h, "http://village.test", true))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"?protocolVersion=2&worldId=a", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://village.test"}, "Cookie": {"patra_session=u"}}})
	if e != nil {
		t.Fatal(e)
	}
	defer c.CloseNow()
	if _, _, e = c.Read(ctx); e != nil {
		t.Fatal(e)
	}
	c.Write(ctx, websocket.MessageText, []byte(`{"bad":true}`))
	for e == nil {
		_, _, e = c.Read(ctx)
	}
	if websocket.CloseStatus(e) != websocket.StatusPolicyViolation {
		t.Fatalf("malformed frame needs policy close: %v", e)
	}
}

func TestTransportTakeoverAndFloodAreFenced(t *testing.T) {
	h := newTestHub(t)
	srv := httptest.NewServer(NewHandler(h, "http://village.test", true))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dial := func(extra string) (*websocket.Conn, Event) {
		t.Helper()
		c, _, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"?protocolVersion=2&worldId=a"+extra, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://village.test"}, "Cookie": {"patra_session=u"}}})
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { c.CloseNow() })
		_, raw, e := c.Read(ctx)
		if e != nil {
			t.Fatal(e)
		}
		ev, e := ValidateEvent(raw)
		if e != nil {
			t.Fatal(e)
		}
		return c, ev
	}
	old, first := dial("")
	if first.Type != "world.snapshot" {
		t.Fatal(first)
	}
	duplicate, ev := dial("")
	if ev.Type != "command.error" || !strings.Contains(string(ev.Payload), "takeover_required") {
		t.Fatal("implicit takeover")
	}
	duplicate.CloseNow()
	next, ev := dial("&takeover=true")
	if ev.Type != "world.snapshot" {
		t.Fatal(ev)
	}
	for {
		_, _, e := old.Read(ctx)
		if e != nil {
			if websocket.CloseStatus(e) != websocket.StatusPolicyViolation {
				t.Fatal(e)
			}
			break
		}
	}
	for i := 0; i < 100; i++ {
		raw, _ := json.Marshal(Command{V: 2, CommandID: "flood", Seq: uint64(i + 1), WorldID: "a", Type: "player.input", Payload: []byte(`{"dx":1,"dy":0}`)})
		if e := next.Write(ctx, websocket.MessageText, raw); e != nil {
			break
		}
	}
	for {
		_, _, e := next.Read(ctx)
		if e != nil {
			if websocket.CloseStatus(e) != websocket.StatusPolicyViolation {
				t.Fatal(e)
			}
			break
		}
	}
}
func TestHeartbeatDisconnectsUnreadClient(t *testing.T) {
	h := newTestHub(t)
	srv := httptest.NewServer(NewHandler(h, "http://village.test", true))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	c, _, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"?protocolVersion=2&worldId=a", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://village.test"}, "Cookie": {"patra_session=u"}}})
	if e != nil {
		t.Fatal(e)
	}
	defer c.CloseNow()
	_, raw, e := c.Read(ctx)
	if e != nil {
		t.Fatal(e)
	}
	var ev Event
	json.Unmarshal(raw, &ev)
	var s PublicSnapshot
	json.Unmarshal(ev.Payload, &s)
	select {
	case <-h.Done(JoinContext{Token: "u", WorldID: "a", SessionID: s.SessionID, Epoch: s.Epoch}):
	case <-ctx.Done():
		t.Fatal("non-reading peer retained live connection")
	}
}
func TestReceivedLeaveSurvivesImmediateTransportClose(t *testing.T) {
	h := newTestHub(t)
	srv := httptest.NewServer(NewHandler(h, "http://village.test", true))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"?protocolVersion=2&worldId=a", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://village.test"}, "Cookie": {"patra_session=u"}}})
	if e != nil {
		t.Fatal(e)
	}
	defer c.CloseNow()
	_, raw, e := c.Read(ctx)
	if e != nil {
		t.Fatal(e)
	}
	var ev Event
	json.Unmarshal(raw, &ev)
	var snapshot PublicSnapshot
	json.Unmarshal(ev.Payload, &snapshot)
	entered, release := make(chan struct{}), make(chan struct{})
	h.mu.Lock()
	h.members = blockingMembers{entered, release}
	h.mu.Unlock()
	c.Write(ctx, websocket.MessageText, []byte(`{"v":2,"commandId":"input","seq":1,"worldId":"a","type":"player.input","payload":{"dx":1,"dy":0}}`))
	<-entered
	c.Write(ctx, websocket.MessageText, []byte(`{"v":2,"commandId":"leave","seq":2,"worldId":"a","type":"world.leave","payload":{}}`))
	c.Close(websocket.StatusNormalClosure, "leaving")
	close(release)
	select {
	case <-h.Done(JoinContext{SessionID: snapshot.SessionID, Epoch: snapshot.Epoch}):
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	h.members = testMembers{}
	if _, e = h.Join(ctx, JoinContext{Token: "u", WorldID: "b"}); e != nil {
		t.Fatalf("explicit leave retained old seat: %v", e)
	}
}
func TestAcknowledgedLeaveAllowsAnotherWorld(t *testing.T) {
	h := newTestHub(t)
	srv := httptest.NewServer(NewHandler(h, "http://village.test", true))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dial := func(worldID string) *websocket.Conn {
		t.Helper()
		c, _, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"?protocolVersion=2&worldId="+worldID, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://village.test"}, "Cookie": {"patra_session=u"}}})
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { c.CloseNow() })
		_, raw, e := c.Read(ctx)
		if e != nil {
			t.Fatal(e)
		}
		ev, e := ValidateEvent(raw)
		if e != nil || ev.Type != "world.snapshot" {
			t.Fatalf("world join: %s %v", raw, e)
		}
		return c
	}
	first := dial("a")
	first.Write(ctx, websocket.MessageText, []byte(`{"v":2,"commandId":"leave","seq":1,"worldId":"a","type":"world.leave","payload":{}}`))
	for {
		_, raw, e := first.Read(ctx)
		if e != nil {
			t.Fatal(e)
		}
		ev, e := ValidateEvent(raw)
		if e != nil {
			t.Fatal(e)
		}
		if ev.Type == "command.ack" {
			var p struct {
				CommandID string `json:"commandId"`
			}
			json.Unmarshal(ev.Payload, &p)
			if p.CommandID == "leave" {
				break
			}
		}
	}
	first.CloseNow()
	dial("b")
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.seats) != 1 || h.rooms["a"] != nil {
		t.Fatal("acknowledged leave retained previous world")
	}
}

func TestProtocolOneHandshakeRequiresReloadBeforeAdmission(t *testing.T) {
	h := newTestHub(t)
	srv := httptest.NewServer(NewHandler(h, "http://village.test", true))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"?protocolVersion=1&worldId=a", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://village.test"}, "Cookie": {"patra_session=u"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	_, raw, err := c.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var e Event
	json.Unmarshal(raw, &e)
	if e.Type != "command.error" || !strings.Contains(string(e.Payload), "Reload") {
		t.Fatal(string(raw))
	}
	_, _, err = c.Read(ctx)
	if websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatal(err)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.seats) != 0 {
		t.Fatal("incompatible client admitted")
	}
}

// The production reader must admit a full aiming stream, including neutral input.
func TestClawTransportSustainsAimingAndKeepsDiscreteLimit(t *testing.T) {
	for _, action := range []string{"claw.move", "claw.enter", "claw.drop", "claw.leave"} {
		t.Run(action, func(t *testing.T) {
			h := newTestHub(t)
			srv := httptest.NewServer(NewHandler(h, "http://village.test", true))
			defer srv.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"?protocolVersion=2&worldId=a", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://village.test"}, "Cookie": {"patra_session=u"}}})
			if err != nil {
				t.Fatal(err)
			}
			defer c.CloseNow()
			_, raw, err := c.Read(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var ev Event
			json.Unmarshal(raw, &ev)
			var first PublicSnapshot
			json.Unmarshal(ev.Payload, &first)
			j := JoinContext{Token: "u", WorldID: "a", SessionID: first.SessionID, Epoch: first.Epoch}
			at := world.Point{X: 41, Y: 24}
			if err = h.SetActivity(ctx, "a", "u", false, &at); err != nil {
				t.Fatal(err)
			}
			states, err := h.ActivitySnapshots(ctx, j)
			if err != nil {
				t.Fatal(err)
			}
			var intent world.CasualIntent
			for _, state := range states {
				if state.Kind == "claw" {
					intent = world.CasualIntent{Incarnation: state.Incarnation, Generation: state.Generation, MachineID: "claw-1", Action: "claw.enter"}
				}
			}
			send := func(seq int, action string) error {
				intent.Action = action
				if action == "claw.move" && seq%2 == 0 {
					intent.DX = 1
				} else {
					intent.DX = 0
				}
				payload, _ := json.Marshal(intent)
				raw, _ := json.Marshal(Command{V: 2, CommandID: fmt.Sprint(seq), Seq: uint64(seq), WorldID: "a", Type: "activity.claw", Payload: payload})
				if err := c.Write(ctx, websocket.MessageText, raw); err != nil {
					return err
				}
				for {
					_, raw, err := c.Read(ctx)
					if err != nil {
						return err
					}
					var event Event
					json.Unmarshal(raw, &event)
					var result struct {
						CommandID string `json:"commandId"`
					}
					json.Unmarshal(event.Payload, &result)
					if result.CommandID == fmt.Sprint(seq) {
						if action == "claw.move" && event.Type != "command.ack" {
							t.Fatalf("move rejected: %s", raw)
						}
						return nil
					}
				}
			}
			if action == "claw.move" {
				if err := send(1, "claw.enter"); err != nil {
					t.Fatal(err)
				}
				states, _ = h.ActivitySnapshots(ctx, j)
				for _, state := range states {
					if state.Kind == "claw" {
						intent.Generation = state.Generation
					}
				}
				for seq := 2; seq <= 27; seq++ {
					time.Sleep(50 * time.Millisecond)
					if err := send(seq, action); err != nil {
						t.Fatalf("20Hz aiming closed at input %d: %v", seq-1, err)
					}
				}
			} else {
				for seq := 1; seq <= 11; seq++ {
					err = send(seq, action)
					if err != nil {
						break
					}
				}
				if websocket.CloseStatus(err) != websocket.StatusPolicyViolation || !strings.Contains(err.Error(), "control rate exceeded") {
					t.Fatalf("discrete action escaped control limit: %v", err)
				}
			}
		})
	}
}
