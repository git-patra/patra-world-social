package realtime

import (
	"context"
	"fmt"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	"patra-world-social/server/internal/world"
	"strings"
	"time"

	"encoding/json"
	"testing"
)

func TestRideWireIsExact(t *testing.T) {
	payload := map[string]any{"incarnation": "runtime", "areaGeneration": 0, "vehicleId": "bicycle-0", "generation": 1, "action": "mount"}
	makeCommand := func() []byte {
		b, _ := json.Marshal(map[string]any{"v": 2, "commandId": "ride", "worldId": "world", "seq": 1, "type": "activity.ride", "payload": payload})
		return b
	}
	if _, e := ValidateCommand(makeCommand()); e != nil {
		t.Fatal("valid mount rejected", e)
	}
	for _, field := range []string{"x", "score", "dx"} {
		payload[field] = 1
		if _, e := ValidateCommand(makeCommand()); e == nil {
			t.Fatal("accepted extra field", field)
		}
		delete(payload, field)
	}
	payload["action"] = "drive"
	payload["dx"] = 1
	payload["dy"] = 0
	if _, e := ValidateCommand(makeCommand()); e != nil {
		t.Fatal(e)
	}
	delete(payload, "dy")
	if _, e := ValidateCommand(makeCommand()); e == nil {
		t.Fatal("missing direction accepted")
	}
}

func TestRideSeatFencesLifecycleAndTwentyMountCycles(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	s, e := h.Join(ctx, JoinContext{Token: "u", WorldID: "w"})
	if e != nil {
		t.Fatal(e)
	}
	j := JoinContext{Token: "u", WorldID: "w", SessionID: s.SessionID, Epoch: s.Epoch}
	peer, e := h.Join(ctx, JoinContext{Token: "peer", WorldID: "w"})
	if e != nil {
		t.Fatal(e)
	}
	_ = peer
	// Drain ordered lifecycle delivery just as the transport would.
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-h.Events(j):
			}
		}
	}()
	seq := uint64(0)
	intent := func(action string) world.CasualIntent {
		states, e := h.ActivitySnapshots(ctx, j)
		if e != nil {
			t.Fatal(e)
		}
		var result world.CasualIntent
		for _, a := range states {
			if a.Kind == "area" && a.AreaID == "race" {
				result.AreaGeneration = a.Generation
			}
			if a.Kind == "vehicles" {
				result.Incarnation = a.Incarnation
				v := a.Payload.Vehicles[0]
				result.VehicleID = v.ID
				result.Generation = v.Generation
			}
		}
		result.Action = action
		return result
	}
	execute := func(i world.CasualIntent) error {
		seq++
		b, _ := json.Marshal(i)
		_, e := h.Execute(ctx, j, Command{V: 2, CommandID: fmt.Sprint(seq), Seq: seq, WorldID: "w", Type: "activity.ride", Payload: b})
		return e
	}
	if execute(intent("mount")) == nil {
		t.Fatal("remote mount accepted")
	}
	for cycle := 0; cycle < 20; cycle++ {
		at := world.Point{X: 39, Y: 46}
		h.SetActivity(ctx, "w", "u", false, &at)
		mount := intent("mount")
		if e := execute(mount); e != nil {
			t.Fatal(cycle, e)
		}
		if e := execute(mount); e == nil {
			t.Fatal("old generation mounted twice")
		}
		if e := execute(intent("dismount")); e != nil {
			t.Fatal(e)
		}
		p, _ := h.Snapshot(ctx, j)
		for _, a := range p.Avatars {
			if a.ID == "u" && (a.X != 45 || a.Y != 49) {
				t.Fatal("unsafe return", a)
			}
		}
	}
	at := world.Point{X: 39, Y: 46}
	h.SetActivity(ctx, "w", "u", false, &at)
	if e = execute(intent("mount")); e != nil {
		t.Fatal(e)
	}
	stale := intent("drive")
	stale.DX = 1
	h.Revoke("w", "u")
	if e = execute(stale); e == nil {
		t.Fatal("revoked rider input accepted")
	}
	states, _ := h.WorldActivities(ctx, "w")
	for _, s := range states {
		if s.Kind == "vehicles" {
			for _, v := range s.Payload.Vehicles {
				if v.RiderID != "" {
					t.Fatal("revocation left ghost rider")
				}
			}
		}
	}
}

func TestRideWebSocketContinuousDriveUsesMovementBudget(t *testing.T) {
	h := newTestHub(t)
	srv := httptest.NewServer(NewHandler(h, "http://village.test", true))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"?protocolVersion=2&worldId=w", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://village.test"}, "Cookie": {"patra_session=u"}}})
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
	var admission PublicSnapshot
	json.Unmarshal(ev.Payload, &admission)
	j := JoinContext{Token: "u", WorldID: "w", SessionID: admission.SessionID, Epoch: admission.Epoch}
	at := world.Point{X: 39, Y: 46}
	h.SetActivity(ctx, "w", "u", false, &at)
	states, _ := h.ActivitySnapshots(ctx, j)
	inc := ""
	for _, s := range states {
		if s.Kind == "vehicles" {
			inc = s.Incarnation
		}
	}
	send := func(seq int, payload map[string]any) {
		t.Helper()
		id := fmt.Sprint(seq)
		b, _ := json.Marshal(map[string]any{"v": 2, "commandId": id, "seq": seq, "worldId": "w", "type": "activity.ride", "payload": payload})
		if e := c.Write(ctx, websocket.MessageText, b); e != nil {
			t.Fatal(e)
		}
		for {
			_, raw, e := c.Read(ctx)
			if e != nil {
				t.Fatal(e)
			}
			event, e := ValidateEvent(raw)
			if e != nil {
				t.Fatal(string(raw), e)
			}
			var ack struct {
				CommandID string `json:"commandId"`
			}
			json.Unmarshal(event.Payload, &ack)
			if ack.CommandID == id {
				if event.Type != "command.ack" {
					t.Fatal(string(raw))
				}
				return
			}
		}
	}
	payload := map[string]any{"incarnation": inc, "areaGeneration": 0, "vehicleId": "bicycle-0", "generation": 1, "action": "mount"}
	send(1, payload)
	payload["generation"] = 2
	payload["action"] = "drive"
	payload["dx"] = 1.
	payload["dy"] = 0.
	// Twenty continuous intents in one control window must not hit the ten-control limit.
	for seq := 2; seq <= 21; seq++ {
		send(seq, payload)
	}
	time.Sleep(100 * time.Millisecond)
	states, _ = h.ActivitySnapshots(ctx, j)
	for _, s := range states {
		if s.Kind == "vehicles" && s.Payload.Vehicles[0].X <= 39 {
			t.Fatal("wire drive did not move owner")
		}
	}
	delete(payload, "dx")
	delete(payload, "dy")
	payload["action"] = "dismount"
	send(22, payload)
}

func TestNeutralRideIntentSerializesExactDriveUnion(t *testing.T) {
	b, e := json.Marshal(world.CasualIntent{Incarnation: "runtime", VehicleID: "bicycle-0", Generation: 2, Action: "drive"})
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(Command{V: 2, CommandID: "neutral", Seq: 2, WorldID: "w", Type: "activity.ride", Payload: b})
	if _, e = ValidateCommand(raw); e != nil {
		t.Fatal("neutral drive lost required axes", string(b), e)
	}
}
