package voice

import (
	"bytes"
	"os"
	"patra-world-social/server/internal/world"
	"testing"
)

func TestZonesMatchSharedContractAndReachableMap(t *testing.T) {
	m, e := world.LoadMap(world.MapAsset)
	if e != nil {
		t.Fatal(e)
	}
	if e = ValidateZones(m); e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile("../../../../contracts/voice-tables.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	embedded, _ := zonesFS.ReadFile("tables.v1.json")
	if !bytes.Equal(raw, embedded) {
		t.Fatal("zone contract drift")
	}
	seen := map[world.Point]bool{m.Spawn: true}
	q := []world.Point{m.Spawn}
	for len(q) > 0 {
		p := q[0]
		q = q[1:]
		for _, d := range []world.Point{{X: 1, Y: 0}, {X: -1, Y: 0}, {X: 0, Y: 1}, {X: 0, Y: -1}} {
			n := world.Point{X: p.X + d.X, Y: p.Y + d.Y}
			if !seen[n] && !m.Blocked(n) {
				seen[n] = true
				q = append(q, n)
			}
		}
	}
	for _, z := range TableZones.Tables {
		if !seen[world.Point{X: int(z.X), Y: int(z.Y)}] {
			t.Fatal("unreachable table")
		}
	}
}
