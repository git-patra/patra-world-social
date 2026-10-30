package world

import (
	"fmt"
	"sort"
	"testing"
	"time"
)

// Deterministic simulation-only baseline; network/owner-queue latency is measured
// separately by the combined application load fixture.
func BenchmarkWorldTwentyAvatarTick(b *testing.B) {
	m, err := LoadMap(MapAsset)
	if err != nil {
		b.Fatal(err)
	}
	c := &fakeClock{now: time.Unix(1000, 0)}
	r := &Runtime{m: m, clock: c, last: c.Now(), players: map[string]*player{}}
	for i := 0; i < 20; i++ {
		id := fmt.Sprint(i)
		r.players[id] = &player{Avatar: Avatar{ID: id, X: float64(m.Spawn.X), Y: float64(m.Spawn.Y)}, dx: 1, expires: c.Now().Add(365 * 24 * time.Hour)}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%100 == 0 {
			for _, p := range r.players {
				p.X = float64(m.Spawn.X)
				p.Y = float64(m.Spawn.Y)
			}
		}
		c.add(50 * time.Millisecond)
		r.advance()
	}
}

// Pure world-owner advance cost, excluding queue wait, encoding, DB, game-owner
// engines and transport. Three active 20-avatar worlds with bounded casual pools.
func TestCombinedWorldTickDistribution(t *testing.T) {
	m, e := LoadMap(MapAsset)
	if e != nil {
		t.Fatal(e)
	}
	c := &fakeClock{now: time.Unix(1000, 0)}
	worlds := make([]*Runtime, 3)
	for w := range worlds {
		r := &Runtime{m: m, clock: c, last: c.Now(), players: map[string]*player{}, areas: map[string]*areaState{}}
		r.initVehicles()
		r.resetFootball(true)
		r.resetClaw()
		r.initFishing()
		for i := 0; i < 20; i++ {
			id := fmt.Sprint(i)
			r.players[id] = &player{Avatar: Avatar{ID: id, X: float64(m.Spawn.X), Y: float64(m.Spawn.Y)}, dx: 1, expires: c.Now().Add(time.Hour)}
		}
		worlds[w] = r
	}
	durations := make([]int64, 0, 15000)
	start := time.Now()
	for n := 0; n < 5000; n++ {
		c.add(50 * time.Millisecond)
		for _, r := range worlds {
			for i := range r.vehicles {
				v := &r.vehicles[i]
				v.RiderID = fmt.Sprint(i)
				v.dx = 1
				v.expires = c.Now().Add(time.Second)
			}
			r.football.VX = 1
			for i := range r.fishing {
				r.fishing[i].OwnerID = fmt.Sprint(14 + i)
				r.fishing[i].Phase = "waiting"
				r.fishing[i].BiteAt = c.Now().UnixMilli() + 1000
				r.fishing[i].EndsAt = c.Now().UnixMilli() + 3000
			}
			before := time.Now()
			r.advance()
			durations = append(durations, time.Since(before).Nanoseconds())
		}
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	t.Logf("WORLD_TICK_METRICS worlds=3 avatarsPerWorld=20 samples=%d elapsedMs=%.3f p50us=%.3f p95us=%.3f p99us=%.3f", len(durations), float64(time.Since(start).Nanoseconds())/1e6, float64(durations[len(durations)/2])/1000, float64(durations[len(durations)*95/100])/1000, float64(durations[len(durations)*99/100])/1000)
}
