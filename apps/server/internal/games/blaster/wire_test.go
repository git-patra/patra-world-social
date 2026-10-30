package blaster

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/realtime"
	"strings"
	"testing"
)

func cappedWireSnapshot() Snapshot {
	s := Snapshot{MatchID: "00000000-0000-4000-8000-000000000001", Phase: "playing", Frame: 3599, ServerTimeMs: 1793340000000, StartsAt: 1793339820000, DeadlineAt: 1793340000000, Wave: 3}
	for i := 0; i < 4; i++ {
		id := fmt.Sprintf("00000000-0000-4000-8000-%012d", i)
		s.Actors = append(s.Actors, model.Actor{ActorID: id, UserID: id, ActorType: "human", DisplayName: strings.Repeat("<", 32), AvatarID: "pumpkin"})
		s.Players = append(s.Players, Player{ActorID: id, X: math.Sin(float64(i)), Y: .51314151617, Aim: math.Pi, HP: 3, Kills: 8, HitAt: 1793340000000})
	}
	for i := 0; i < 8; i++ {
		s.Bots = append(s.Bots, Bot{ID: i + 100, X: .41314151617, Y: .51314151617, NextFire: 1793340000000})
	}
	for i := 0; i < 48; i++ {
		s.Projectiles = append(s.Projectiles, Projectile{ID: 1000 + i, X: .41314151617, Y: .51314151617, VX: math.Sin(float64(i)), VY: math.Cos(float64(i)), Owner: i%5 - 1, TTL: 1450})
	}
	return s
}
func TestCappedWireSnapshotBudget(t *testing.T) {
	s := cappedWireSnapshot()
	raw, e := json.Marshal(s)
	if e != nil {
		t.Fatal(e)
	}
	// Same public event envelope fields as realtime.writeEvent, with 256B reserved
	// for envelope metadata and reliable lifecycle traffic amortization.
	event, e := json.Marshal(realtime.Event{V: 2, EventID: strings.Repeat("a", 24), ServerTimeMs: 1793340000000, Type: "blaster.snapshot", Payload: raw})
	if e != nil {
		t.Fatal(e)
	}
	if _, err := realtime.ValidateEvent(event); err != nil {
		t.Fatal(err)
	}
	fixture := "../../../../../contracts/blaster-wire.golden.json"
	if os.Getenv("UPDATE_ACTIVITY_WIRE_FIXTURE") == "1" {
		if err := os.WriteFile(fixture, append(event, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	saved, err := os.ReadFile(fixture)
	if err != nil || !bytes.Equal(bytes.TrimSpace(saved), event) {
		t.Fatal("Go encoded public event fixture drift", err)
	}
	t.Logf("CAPPED_BLASTER_EVENT bytes=%d exactKiBpsAt10Hz=%.3f", len(event), float64(len(event))*10/1024)
	kib := float64(len(raw)+256) * 10 / 1024
	t.Logf("CAPPED_BLASTER_WIRE payloadBytes=%d envelopeAllowance=256 participantsHz=10 KiBps=%.3f", len(raw), kib)
	if kib > 40 {
		t.Errorf("capped replaceable snapshot exceeds 40KiB/s participant budget: %.3f", kib)
	}
}

func TestWireProjectionDoesNotMutatePhysicsAndRoundTripsVisuals(t *testing.T) {
	before := cappedWireSnapshot()
	original := before.Projectiles[0]
	raw, e := json.Marshal(before)
	if e != nil {
		t.Fatal(e)
	}
	var decoded Snapshot
	if e = json.Unmarshal(raw, &decoded); e != nil {
		t.Fatal(e)
	}
	if len(decoded.Projectiles) != 48 || len(decoded.Bots) != 8 || len(decoded.Actors) != 4 {
		t.Fatal("wire entity loss")
	}
	if before.Projectiles[0] != original || math.Abs(decoded.Projectiles[0].X-original.X) > .0005 || decoded.Projectiles[0].ID != original.ID || decoded.Projectiles[0].Owner != original.Owner {
		t.Fatal("projection mutated physics or lost visual identity")
	}
	if strings.Contains(string(raw), "nextFire") || strings.Contains(string(raw), "ttl") || strings.Contains(string(raw), "vx") {
		t.Fatal("private simulation fields retransmitted")
	}
}
