package world

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"
)

func clawState(t *testing.T, r *Runtime) (ActivitySnapshot, map[string]any) {
	t.Helper()
	ss, e := r.ActivitySnapshots(context.Background(), "world")
	if e != nil {
		t.Fatal(e)
	}
	for _, s := range ss {
		if s.Kind == "claw" {
			b, _ := json.Marshal(s.Payload)
			var p map[string]map[string]any
			json.Unmarshal(b, &p)
			return s, p["claw"]
		}
	}
	t.Fatal("world snapshots omit claw cabinet")
	return ActivitySnapshot{}, nil
}
func clawUser(t *testing.T, r *Runtime, id string, epoch uint64) {
	t.Helper()
	r.Join(context.Background(), id, id, epoch)
	r.call(context.Background(), func() { r.players[id].X = 41; r.players[id].Y = 24 })
}
func clawIntent(s ActivitySnapshot, action string) CasualIntent {
	return CasualIntent{Incarnation: s.Incarnation, Generation: s.Generation, MachineID: "claw-1", Action: action}
}
func TestClawSingleOperatorTwentySecondLease(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	clawUser(t, r, "u", 1)
	clawUser(t, r, "v", 1)
	s, _ := clawState(t, r)
	if e := r.ApplyCasualIntent(ctx, "u", 1, 1, clawIntent(s, "claw.enter")); e != nil {
		t.Fatal(e)
	}
	s, p := clawState(t, r)
	if p["operatorId"] != "u" || p["leaseUntil"] != float64(c.Now().Add(20*time.Second).UnixMilli()) {
		t.Fatal(p)
	}
	if r.ApplyCasualIntent(ctx, "v", 1, 1, clawIntent(s, "claw.enter")) == nil {
		t.Fatal("second operator admitted")
	}
	c.add(19999 * time.Millisecond)
	s, _ = clawState(t, r)
	if r.ApplyCasualIntent(ctx, "u", 1, 2, clawIntent(s, "claw.drop")) != nil {
		t.Fatal("last millisecond drop rejected")
	}
	c.add(time.Millisecond)
	s, p = clawState(t, r)
	if p["operatorId"] != "" || p["phase"] != "idle" {
		t.Fatal("lease expiry not authoritative", p)
	}
	if r.ApplyCasualIntent(ctx, "u", 1, 3, clawIntent(s, "claw.drop")) == nil {
		t.Fatal("drop at expiry admitted")
	}
	if e := r.ApplyCasualIntent(ctx, "v", 1, 1, clawIntent(s, "claw.enter")); e != nil {
		t.Fatal(e)
	}
}
func TestClawDropOutcomeUsesServerGripPosition(t *testing.T) {
	for _, hit := range []bool{false, true} {
		t.Run(map[bool]string{false: "miss", true: "preset"}[hit], func(t *testing.T) {
			r, c := testRuntime(t)
			ctx := context.Background()
			clawUser(t, r, "u", 1)
			s, _ := clawState(t, r)
			r.ApplyCasualIntent(ctx, "u", 1, 1, clawIntent(s, "claw.enter"))
			seq := uint64(2)
			if hit {
				for n := 0; n < 6; n++ {
					s, _ = clawState(t, r)
					i := clawIntent(s, "claw.move")
					i.DX = -1
					i.DY = -1
					if e := r.ApplyCasualIntent(ctx, "u", 1, seq, i); e != nil {
						t.Fatal(e)
					}
					seq++
					c.add(50 * time.Millisecond)
				}
			}
			s, _ = clawState(t, r)
			if e := r.ApplyCasualIntent(ctx, "u", 1, seq, clawIntent(s, "claw.drop")); e != nil {
				t.Fatal(e)
			}
			_, p := clawState(t, r)
			if p["phase"] != "dropping" || (p["prizeId"] != "") != hit {
				t.Fatal("outcome not determined by owner grip", p)
			}
			c.add(500 * time.Millisecond)
			_, p = clawState(t, r)
			if p["phase"] != "lifting" {
				t.Fatal(p)
			}
			c.add(500 * time.Millisecond)
			_, p = clawState(t, r)
			if p["phase"] != "releasing" {
				t.Fatal(p)
			}
			c.add(500 * time.Millisecond)
			_, p = clawState(t, r)
			if p["phase"] != "aiming" || p["prizeId"] != "" {
				t.Fatal("cosmetic prize persisted", p)
			}
		})
	}
}
func TestClawDuplicateDropDoesNotDuplicateReward(t *testing.T) {
	r, _ := testRuntime(t)
	ctx := context.Background()
	clawUser(t, r, "u", 1)
	s, _ := clawState(t, r)
	r.ApplyCasualIntent(ctx, "u", 1, 1, clawIntent(s, "claw.enter"))
	s, _ = clawState(t, r)
	drop := clawIntent(s, "claw.drop")
	if e := r.ApplyCasualIntent(ctx, "u", 1, 2, drop); e != nil {
		t.Fatal(e)
	}
	for _, seq := range []uint64{2, 3} {
		if r.ApplyCasualIntent(ctx, "u", 1, seq, drop) == nil {
			t.Fatal("duplicate drop admitted")
		}
	}
	_, p := clawState(t, r)
	if p["dropId"] != float64(1) {
		t.Fatal("duplicate cosmetic result", p)
	}
}
func TestClawDisconnectOrMatchJoinResetsOperator(t *testing.T) {
	for _, event := range []string{"idle", "remove", "takeover", "match"} {
		t.Run(event, func(t *testing.T) {
			r, _ := testRuntime(t)
			ctx := context.Background()
			clawUser(t, r, "u", 1)
			s, _ := clawState(t, r)
			r.ApplyCasualIntent(ctx, "u", 1, 1, clawIntent(s, "claw.enter"))
			s, _ = clawState(t, r)
			switch event {
			case "idle":
				r.Idle(ctx, "u")
			case "remove":
				r.Remove(ctx, "u")
			case "takeover":
				r.Join(ctx, "u", "U", 2)
			case "match":
				r.Activity(ctx, "u", true, nil)
			}
			next, p := clawState(t, r)
			if p["operatorId"] != "" || next.Generation <= s.Generation {
				t.Fatal("operator not reset", p)
			}
			if r.ApplyCasualIntent(ctx, "u", 1, 2, clawIntent(s, "claw.drop")) == nil {
				t.Fatal("old lease survived cleanup")
			}
		})
	}
}
func TestClawControlsBoundsExpiryAndFences(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	clawUser(t, r, "u", 1)
	s, _ := clawState(t, r)
	r.ApplyCasualIntent(ctx, "u", 1, 1, clawIntent(s, "claw.enter"))
	s, _ = clawState(t, r)
	good := clawIntent(s, "claw.move")
	good.DX = 1
	for _, bad := range []CasualIntent{func() CasualIntent { i := good; i.DX = math.NaN(); return i }(), func() CasualIntent { i := good; i.Incarnation = "old"; return i }(), func() CasualIntent { i := good; i.AreaGeneration = 1; return i }(), func() CasualIntent { i := good; i.MachineID = "car-2"; return i }()} {
		if r.ApplyCasualIntent(ctx, "u", 1, 2, bad) == nil {
			t.Fatal("invalid control accepted")
		}
	}
	if r.ApplyCasualIntent(ctx, "u", 2, 2, good) == nil {
		t.Fatal("old epoch admitted")
	}
	if e := r.ApplyCasualIntent(ctx, "u", 1, 2, good); e != nil {
		t.Fatal(e)
	}
	c.add(250 * time.Millisecond)
	_, p := clawState(t, r)
	x := p["x"].(float64)
	if x > .700001 {
		t.Fatal("catchup exceeded four steps", p)
	}
	c.add(time.Second)
	_, p = clawState(t, r)
	if p["x"] != x {
		t.Fatal("expired input moved grip", p)
	}
	if r.ApplyCasualIntent(ctx, "u", 1, 3, CasualIntent{Incarnation: s.Incarnation, Generation: 1, Action: "mount", VehicleID: "bicycle-0"}) == nil {
		t.Fatal("operator mounted overlapping ride")
	}
}

func TestClawSharedContractRejectsUnboundedRules(t *testing.T) {
	m, e := LoadMap(MapAsset)
	if e != nil {
		t.Fatal(e)
	}
	var base map[string]any
	json.Unmarshal(ActivitiesAsset, &base)
	for _, field := range []string{"leaseMs", "inputExpiryMs", "maxCatchupSteps", "gripTolerance", "presets"} {
		t.Run(field, func(t *testing.T) {
			var c map[string]any
			raw, _ := json.Marshal(base)
			json.Unmarshal(raw, &c)
			rules := c["clawRules"].(map[string]any)
			rules[field] = nil
			raw, _ = json.Marshal(c)
			if _, e := LoadActivities(raw, m); e == nil {
				t.Fatal("invalid claw rules accepted", field)
			}
		})
	}
}
func TestClawWireUsesMachineIdentifier(t *testing.T) {
	i := CasualIntent{Incarnation: "runtime", Generation: 1, MachineID: "claw-1", Action: "claw.enter"}
	raw, _ := json.Marshal(i)
	var p map[string]any
	json.Unmarshal(raw, &p)
	if p["machineId"] != "claw-1" {
		t.Fatal("machine identifier lost during serialization", string(raw))
	}
}
