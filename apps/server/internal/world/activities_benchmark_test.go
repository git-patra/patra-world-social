package world

import (
	"fmt"
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
