package world

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestVehiclePoolIsOwnedByWorld(t *testing.T) {
	r, _ := testRuntime(t)
	snapshots, err := r.ActivitySnapshots(context.Background(), "world")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range snapshots {
		if s.Kind == "vehicles" {
			found = true
		}
	}
	if !found {
		t.Fatal("world snapshots omit the four free vehicles")
	}
}
func TestSingleRiderAndContinuousBudget(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "U", 1)
	r.Join(ctx, "v", "V", 1)
	r.call(ctx, func() {
		for _, p := range r.players {
			p.X = 39
			p.Y = 46
		}
	})
	intent := CasualIntent{Incarnation: r.incarnation, VehicleID: "bicycle-0", Generation: 1, Action: "mount"}
	if err := r.ApplyCasualIntent(ctx, "u", 1, 1, intent); err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyCasualIntent(ctx, "v", 1, 1, intent); err == nil {
		t.Fatal("two riders acquired object")
	}
	intent.Generation = 2
	intent.Action = "drive"
	intent.DX = 1
	for seq := uint64(2); seq < 22; seq++ {
		if err := r.ApplyCasualIntent(ctx, "u", 1, seq, intent); err != nil {
			t.Fatal(err)
		}
		c.add(50 * time.Millisecond)
	}
	r.Snapshot(ctx)
	r.call(ctx, func() {
		v := r.vehicles[0]
		if math.Abs(v.X-45) > 1e-8 {
			t.Fatalf("one-second bicycle budget %+v", v)
		}
		if r.players["u"].X != v.X {
			t.Fatal("rider detached")
		}
	})
	c.add(time.Second)
	s, _ := r.Snapshot(ctx)
	x := s.Avatars[0].X
	c.add(time.Second)
	s, _ = r.Snapshot(ctx)
	if s.Avatars[0].X != x {
		t.Fatal("expired ride kept moving")
	}
}
func TestVehicleCannotLeaveTrackOrTunnel(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "U", 1)
	r.call(ctx, func() { r.players["u"].X = 43; r.players["u"].Y = 46 })
	i := CasualIntent{Incarnation: r.incarnation, VehicleID: "car-2", Generation: 1, Action: "mount"}
	if err := r.ApplyCasualIntent(ctx, "u", 1, 1, i); err != nil {
		t.Fatal(err)
	}
	i.Generation = 2
	i.Action = "drive"
	i.DY = -1
	for seq := uint64(2); seq < 202; seq++ {
		r.ApplyCasualIntent(ctx, "u", 1, seq, i)
		c.add(50 * time.Millisecond)
		r.call(ctx, func() {
			v := r.vehicles[2]
			if !RideDriveable(v.X, v.Y) || v.Y < 43 {
				t.Fatalf("vehicle crossed infield %+v", v)
			}
		})
	}
}
func TestRaceClearsRidersBeforeCountdownAndReconnectCannotMount(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "U", 1)
	r.call(ctx, func() { r.players["u"].X = 39; r.players["u"].Y = 46 })
	i := CasualIntent{Incarnation: r.incarnation, VehicleID: "bicycle-0", Generation: 1, Action: "mount"}
	if e := r.ApplyCasualIntent(ctx, "u", 1, 1, i); e != nil {
		t.Fatal(e)
	}
	lease, e := r.ReserveArea(ctx, "world", "race", "match")
	if e != nil {
		t.Fatal(e)
	}
	r.call(ctx, func() {
		if r.riding("u") != nil || r.players["u"].X != 45 || r.players["u"].Y != 49 {
			t.Fatal("rider not cleared before ACK")
		}
	})
	r.Join(ctx, "u", "U", 2)
	i.AreaGeneration = lease.Generation
	i.Generation = 4
	if r.ApplyCasualIntent(ctx, "u", 2, 1, i) == nil {
		t.Fatal("reconnect mounted closed track")
	}
	c.add(3 * time.Second)
	if e = r.ClearArea(ctx, lease); e != nil {
		t.Fatal(e)
	}
	r.ActivateArea(ctx, lease, c.Now().UnixMilli())
	r.Activity(ctx, "u", false, &Point{45, 49})
	states, _ := r.ActivitySnapshots(ctx, "world")
	for _, s := range states {
		if s.Kind == "area" && s.AreaID == "race" && s.Payload.Phase != "ACTIVE" {
			t.Fatal("first finisher reopened track")
		}
	}
	r.ReleaseArea(ctx, lease)
	c.add(2 * time.Second)
	r.Snapshot(ctx)
	r.call(ctx, func() { r.players["u"].X = 39; r.players["u"].Y = 46; i.Generation = r.vehicles[0].Generation })
	if e = r.ApplyCasualIntent(ctx, "u", 2, 2, i); e != nil {
		t.Fatal("release did not reopen", e)
	}
	r.Remove(ctx, "u")
	r.call(ctx, func() {
		if r.riding("u") != nil {
			t.Fatal("ghost rider")
		}
	})
}

func TestRideRejectsOldEpochIncarnationSequenceAndWalkingOverlap(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "U", 1)
	r.call(ctx, func() { r.players["u"].X = 43; r.players["u"].Y = 46 })
	intent := CasualIntent{Incarnation: r.incarnation, VehicleID: "car-2", Generation: 1, Action: "mount"}
	if e := r.ApplyCasualIntent(ctx, "u", 1, 1, intent); e != nil {
		t.Fatal(e)
	}
	intent.Generation = 2
	intent.Action = "drive"
	intent.DX = 1
	for _, bad := range []struct {
		epoch, seq uint64
		i          CasualIntent
	}{{2, 2, intent}, {1, 1, intent}, {1, 2, func() CasualIntent { i := intent; i.Incarnation = "old"; return i }()}, {1, 2, func() CasualIntent { i := intent; i.AreaGeneration = 99; return i }()}} {
		if r.ApplyCasualIntent(ctx, "u", bad.epoch, bad.seq, bad.i) == nil {
			t.Fatal("stale fence accepted", bad)
		}
	}
	for seq := uint64(2); seq <= 21; seq++ {
		if e := r.ApplyCasualIntent(ctx, "u", 1, seq, intent); e != nil {
			t.Fatal(e)
		}
		r.Input(ctx, "u", 1, seq+100, -1, 0)
		r.Destination(ctx, "u", 1, seq+200, 32, 32)
		c.add(50 * time.Millisecond)
	}
	r.call(ctx, func() {
		if math.Abs(r.vehicles[2].X-51) > 1e-8 {
			t.Fatal("car speed/overlap budget", r.vehicles[2])
		}
	})
	r.Join(ctx, "u", "U", 2)
	if r.ApplyCasualIntent(ctx, "u", 1, 999, intent) == nil {
		t.Fatal("old connection regained rider")
	}
	r.call(ctx, func() {
		if r.riding("u") != nil {
			t.Fatal("reconnect retained stale rider")
		}
	})
}
func TestVehicleSlotsAreReachableOnActualMap(t *testing.T) {
	m, e := LoadMap(MapAsset)
	if e != nil {
		t.Fatal(e)
	}
	c, e := LoadActivities(ActivitiesAsset, m)
	if e != nil {
		t.Fatal(e)
	}
	for _, s := range c.VehicleSlots {
		if m.Blocked(s.Position) || !m.Reachable(m.Spawn, s.Position) || !RideDriveable(float64(s.Position.X), float64(s.Position.Y)) {
			t.Fatal("inaccessible vehicle slot", s)
		}
	}
}

func TestVehicleCatchUpIsCappedAtFourSteps(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "U", 1)
	r.call(ctx, func() { r.players["u"].X = 43; r.players["u"].Y = 46 })
	i := CasualIntent{Incarnation: r.incarnation, VehicleID: "car-2", Generation: 1, Action: "mount"}
	if e := r.ApplyCasualIntent(ctx, "u", 1, 1, i); e != nil {
		t.Fatal(e)
	}
	i.Generation = 2
	i.Action = "drive"
	i.DX = 1
	if e := r.ApplyCasualIntent(ctx, "u", 1, 2, i); e != nil {
		t.Fatal(e)
	}
	c.add(250 * time.Millisecond)
	r.call(ctx, func() {
		if r.vehicles[2].X > 44.600001 {
			t.Fatal("more than four casual steps in catch-up", r.vehicles[2].X)
		}
	})
}
