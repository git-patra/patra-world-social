package realtime

import (
	"context"
	"encoding/json"
	"patra-world-social/server/internal/world"
	"testing"
)

func TestClawWireAndPublicStateAreExact(t *testing.T) {
	i := world.CasualIntent{Incarnation: "runtime", Generation: 1, MachineID: "claw-1", Action: "claw.enter"}
	payload, _ := json.Marshal(i)
	command := Command{V: 2, CommandID: "claw", WorldID: "world", Seq: 1, Type: "activity.claw", Payload: payload}
	raw, _ := json.Marshal(command)
	if _, e := ValidateCommand(raw); e != nil {
		t.Fatal("valid claw entry rejected", e)
	}
	var p map[string]any
	json.Unmarshal(payload, &p)
	for _, key := range []string{"x", "prizeId", "score", "vehicleId"} {
		p[key] = 1
		command.Payload, _ = json.Marshal(p)
		raw, _ = json.Marshal(command)
		if _, e := ValidateCommand(raw); e == nil {
			t.Fatal("client outcome/extra admitted", key)
		}
		delete(p, key)
	}
	h := newTestHub(t)
	s, e := h.Join(context.Background(), JoinContext{Token: "u", WorldID: "w"})
	if e != nil {
		t.Fatal(e)
	}
	j := JoinContext{Token: "u", WorldID: "w", SessionID: s.SessionID, Epoch: s.Epoch}
	states, _ := h.ActivitySnapshots(context.Background(), j)
	found := false
	for _, s := range states {
		if s.Kind == "claw" {
			found = true
			b, _ := json.Marshal(s)
			var value map[string]any
			json.Unmarshal(b, &value)
			if e := validateActivityPayload(value); e != nil {
				t.Fatal("owner claw snapshot rejected", e)
			}
		}
	}
	if !found {
		t.Fatal("hub omitted cabinet")
	}
}
func TestClawActualHubBusyAndTakeoverCleanup(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	s, e := h.Join(ctx, JoinContext{Token: "u", WorldID: "w"})
	if e != nil {
		t.Fatal(e)
	}
	j := JoinContext{Token: "u", WorldID: "w", SessionID: s.SessionID, Epoch: s.Epoch}
	at := world.Point{X: 41, Y: 24}
	h.SetActivity(ctx, "w", "u", false, &at)
	states, _ := h.ActivitySnapshots(ctx, j)
	var i world.CasualIntent
	for _, s := range states {
		if s.Kind == "claw" {
			i = world.CasualIntent{Incarnation: s.Incarnation, Generation: s.Generation, MachineID: "claw-1", Action: "claw.enter"}
		}
	}
	payload, _ := json.Marshal(i)
	c := Command{V: 2, CommandID: "claw", WorldID: "w", Seq: 1, Type: "activity.claw", Payload: payload}
	h.GameBusy = func(string) bool { return true }
	if _, e = h.Execute(ctx, j, c); e == nil {
		t.Fatal("match owner entered claw")
	}
	h.GameBusy = func(string) bool { return false }
	c.Seq++
	if _, e = h.Execute(ctx, j, c); e != nil {
		t.Fatal(e)
	}
	s, e = h.Join(ctx, JoinContext{Token: "u", WorldID: "w", Takeover: true})
	if e != nil {
		t.Fatal(e)
	}
	j.SessionID = s.SessionID
	j.Epoch = s.Epoch
	states, _ = h.ActivitySnapshots(ctx, j)
	for _, s := range states {
		if s.Kind == "claw" && s.Payload.Claw.OperatorID != "" {
			t.Fatal("takeover retained operator")
		}
	}
}
