package blaster

import (
	"math"
	"patra-world-social/server/internal/games/model"
	"testing"
	"time"
)

func fixture(n int) *Engine {
	a := []model.Actor{}
	for i := 0; i < n; i++ {
		id := string(rune('a' + i))
		a = append(a, model.Actor{ActorID: id, UserID: id, ActorType: "human"})
	}
	return New(model.MatchStart{MatchID: "m", StartsAt: 1000, Actors: a})
}
func TestAuthorityBoundsAndExpiry(t *testing.T) {
	e := fixture(4)
	if len(e.Snapshot().Players) != 4 || len(e.Snapshot().Bots) != 4 {
		t.Fatal("roster/wave")
	}
	if e.Input("spectator", 1, Controls{Fire: true}, time.UnixMilli(1000)) == nil {
		t.Fatal("spectator accepted")
	}
	if e.Input("a", 1, Controls{DX: 1, DY: 1, Aim: 0, Fire: true}, time.UnixMilli(1000)) != nil {
		t.Fatal("input")
	}
	if e.Input("a", 1, Controls{}, time.UnixMilli(1000)) == nil {
		t.Fatal("replay")
	}
	p := e.players[0]
	for at := int64(1050); at <= 1500; at += 50 {
		e.Tick(time.UnixMilli(at))
	}
	if d := math.Hypot(e.players[0].X-p.X, e.players[0].Y-p.Y); math.Abs(d-.16) > 1e-9 {
		t.Fatal("normalized speed/expiry", d)
	}
	if len(e.projectiles) > 2 {
		t.Fatal("fire expiry/cooldown")
	}
}
func TestWavesReviveAllDownDeadline(t *testing.T) {
	e := fixture(2)
	e.players[0].HP = 0
	e.bots = nil
	e.Tick(time.UnixMilli(1050))
	if e.wave != 2 || e.players[0].HP != 3 || len(e.bots) != 6 {
		t.Fatal("revive/wave")
	}
	e.bots = nil
	e.Tick(time.UnixMilli(1100))
	e.bots = nil
	e.Tick(time.UnixMilli(1150))
	if e.result == nil || e.result.Blaster.Reason != "cleared" {
		t.Fatal("win")
	}
	e = fixture(1)
	e.players[0].HP = 0
	e.Tick(time.UnixMilli(1050))
	if e.result.Blaster.Reason != "all_down" {
		t.Fatal("all down")
	}
	e = fixture(1)
	e.Tick(time.UnixMilli(181000))
	if e.result.Blaster.Reason != "timeout" || e.frame > 3601 {
		t.Fatal("deadline")
	}
}
func TestNoFriendlyDamageTTLAndLimits(t *testing.T) {
	e := fixture(4)
	e.bots = nil
	for i := 0; i < 48; i++ {
		e.projectiles = append(e.projectiles, Projectile{ID: i, X: .5, Y: .5, Owner: 0, TTL: 50})
	}
	e.players[1].X = .5
	e.players[1].Y = .5
	e.step()
	if e.players[1].HP != 3 {
		t.Fatal("friendly damage")
	}
	if len(e.projectiles) != 0 {
		t.Fatal("ttl")
	}
	for _, p := range e.players {
		for _, b := range e.bots {
			if math.Hypot(p.X-b.X, p.Y-b.Y) < .2 {
				t.Fatal("spawn overlap")
			}
		}
	}
}
func TestPresenceNeutralAndRoster(t *testing.T) {
	e := fixture(2)
	e.Input("a", 1, Controls{Fire: true}, time.UnixMilli(1000))
	e.Presence("a", "disconnect", time.UnixMilli(1000))
	if e.controls[0].Fire {
		t.Fatal("disconnect fire")
	}
	e.Presence("a", "resume", time.UnixMilli(1100))
	if e.controls[0].Fire {
		t.Fatal("resume fire")
	}
	e.Presence("a", "leave", time.UnixMilli(1100))
	if e.players[0].HP != 0 || e.result != nil {
		t.Fatal("team continues")
	}
	e.Presence("b", "leave", time.UnixMilli(1100))
	if e.result == nil {
		t.Fatal("all left")
	}
	for _, n := range []int{0, 5} {
		if _, err := Factory(nil)(model.MatchStart{Actors: make([]model.Actor, n)}); err == nil {
			t.Fatal("roster", n)
		}
	}
}

// Each measured iteration includes reset/copy overhead and one full 4/8/48 tick.
func BenchmarkWorstEntities(b *testing.B) {
	base := fixture(4)
	base.wave = 3
	base.spawn()
	for i := 0; i < 48; i++ {
		base.projectiles = append(base.projectiles, Projectile{ID: i, X: .5, Y: .5, VX: .01, Owner: 0, TTL: 1500})
	}
	players := make([]Player, 4)
	bots := make([]Bot, 8)
	shots := make([]Projectile, 48)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e := *base
		copy(players, base.players)
		copy(bots, base.bots)
		copy(shots, base.projectiles)
		e.players = players
		e.bots = bots
		e.projectiles = shots
		e.at += 50
		e.step()
	}
}

func TestProjectileCollisionBoundaryAndCooldown(t *testing.T) {
	e := fixture(1)
	e.bots = []Bot{{ID: 1, X: .5, Y: .5, NextFire: 99999}}
	e.projectiles = []Projectile{{X: .4, Y: .5, VX: 4, Owner: 0, TTL: 1500}}
	e.Tick(time.UnixMilli(1050))
	if e.kills != 1 || e.players[0].Kills != 1 {
		t.Fatal("swept collision missed", e.kills)
	}
	e = fixture(1)
	e.bots = []Bot{{ID: 1, X: 1.02, Y: .5, NextFire: 99999}}
	e.projectiles = []Projectile{{X: .99, Y: .5, VX: 1.2, Owner: 0, TTL: 1500}}
	e.Tick(time.UnixMilli(1050))
	if e.kills != 0 || len(e.projectiles) != 0 {
		t.Fatal("outside arena damage")
	}
	e = fixture(1)
	e.Input("a", 1, Controls{Aim: math.Pi / 2, Fire: true}, time.UnixMilli(1000))
	e.Tick(time.UnixMilli(1050))
	if len(e.projectiles) != 1 {
		t.Fatal("first shot")
	}
	e.Input("a", 2, Controls{Aim: math.Pi / 2, Fire: true}, time.UnixMilli(1200))
	e.Tick(time.UnixMilli(1250))
	if len(e.projectiles) != 1 {
		t.Fatal("premature shot")
	}
	e.Tick(time.UnixMilli(1300))
	if len(e.projectiles) != 2 {
		t.Fatal("cooldown shot")
	}
}
func TestExactWorstEntityBounds(t *testing.T) {
	e := fixture(4)
	e.wave = 3
	e.spawn()
	if len(e.bots) != 8 {
		t.Fatal("wave 3 bots")
	}
	for _, p := range e.players {
		for _, b := range e.bots {
			if math.Hypot(p.X-b.X, p.Y-b.Y) < .2 {
				t.Fatal("spawn overlap")
			}
		}
	}
	for i := 0; i < 60; i++ {
		e.shoot(.5, .5, 0, 0)
	}
	if len(e.projectiles) != 48 {
		t.Fatal("projectile cap", len(e.projectiles))
	}
	e.Tick(time.UnixMilli(1050))
	s := e.Snapshot()
	if len(s.Players) > 4 || len(s.Bots) > 8 || len(s.Projectiles) > 48 {
		t.Fatal("entity cap")
	}
}

func TestFriendlyShotAndFixedTick(t *testing.T) {
	e := fixture(4)
	p := e.players[1]
	e.projectiles = []Projectile{{X: p.X, Y: p.Y, Owner: 0, TTL: 1500}}
	e.Tick(time.UnixMilli(1049))
	if e.frame != 0 || e.projectiles[0].TTL != 1500 {
		t.Fatal("advanced before fixed tick")
	}
	e.Tick(time.UnixMilli(1050))
	if e.frame != 1 || e.players[1].HP != 3 || len(e.projectiles) != 1 || e.projectiles[0].TTL != 1450 {
		t.Fatal("friendly shot must pass through teammate on fixed tick")
	}
}
