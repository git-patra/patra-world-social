package realtime

import (
	"context"
	"encoding/json"
	"patra-world-social/server/internal/world"
	"testing"
)

func TestFishingExactWireAndFourPublicSpots(t *testing.T) {
	for _, action := range []string{"fishing.cast", "fishing.reel", "fishing.leave"} {
		i := world.CasualIntent{Incarnation: "runtime", Generation: 1, SpotID: "fishing-0", Action: action}
		payload, _ := json.Marshal(i)
		cmd := Command{V: 2, CommandID: action, WorldID: "world", Seq: 1, Type: "activity.fishing", Payload: payload}
		raw, _ := json.Marshal(cmd)
		if _, e := ValidateCommand(raw); e != nil {
			t.Fatal(e)
		}
		var p map[string]any
		json.Unmarshal(payload, &p)
		for _, k := range []string{"x", "dx", "catchId", "score", "machineId", "vehicleId"} {
			p[k] = 1
			cmd.Payload, _ = json.Marshal(p)
			raw, _ = json.Marshal(cmd)
			if _, e := ValidateCommand(raw); e == nil {
				t.Fatal("extra admitted", k)
			}
			delete(p, k)
		}
	}
	h := newTestHub(t)
	ctx := context.Background()
	s, e := h.Join(ctx, JoinContext{Token: "u", WorldID: "w"})
	if e != nil {
		t.Fatal(e)
	}
	j := JoinContext{Token: "u", WorldID: "w", SessionID: s.SessionID, Epoch: s.Epoch}
	states, _ := h.ActivitySnapshots(ctx, j)
	count := 0
	for _, s := range states {
		if s.Kind == "fishing" {
			count++
			raw, _ := json.Marshal(s)
			var p map[string]any
			json.Unmarshal(raw, &p)
			if e := validateActivityPayload(p); e != nil {
				t.Fatal(e)
			}
		}
	}
	if count != 4 {
		t.Fatal(count)
	}
	at := world.Point{X: 53, Y: 30}
	h.SetActivity(ctx, "w", "u", false, &at)
	states, _ = h.ActivitySnapshots(ctx, j)
	var i world.CasualIntent
	for _, s := range states {
		if s.Kind == "fishing" && s.AreaID == "fishing-0" {
			i = world.CasualIntent{Incarnation: s.Incarnation, Generation: s.Generation, SpotID: s.AreaID, Action: "fishing.cast"}
		}
	}
	payload, _ := json.Marshal(i)
	cmd := Command{V: 2, CommandID: "cast", WorldID: "w", Seq: 1, Type: "activity.fishing", Payload: payload}
	h.GameBusy = func(string) bool { return true }
	if _, e = h.Execute(ctx, j, cmd); e == nil {
		t.Fatal("busy owner cast")
	}
	h.GameBusy = func(string) bool { return false }
	cmd.Seq++
	if _, e = h.Execute(ctx, j, cmd); e != nil {
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
		if s.Kind == "fishing" && s.Payload.Fishing.OwnerID != "" {
			t.Fatal("takeover retained fishing")
		}
	}
}
