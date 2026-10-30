package blaster

import (
	"encoding/json"
	"math"
)

// MarshalJSON projects replaceable visual state. Physics retains full precision
// and private motion/expiry/fire timers; the canvas only interpolates positions.
// One thousandth of the 280px arena is at most 0.14px rounding error.
func (s Snapshot) MarshalJSON() ([]byte, error) {
	type plain Snapshot
	type point struct {
		ID int     `json:"id"`
		X  float64 `json:"x"`
		Y  float64 `json:"y"`
	}
	type shot [4]float64
	players := append([]Player(nil), s.Players...)
	round := func(v float64) float64 { return math.Round(v*1000) / 1000 }
	for i := range players {
		players[i].X = round(players[i].X)
		players[i].Y = round(players[i].Y)
		players[i].Aim = round(players[i].Aim)
	}
	bots := make([]point, len(s.Bots))
	for i, b := range s.Bots {
		bots[i] = point{b.ID, round(b.X), round(b.Y)}
	}
	shots := make([]shot, len(s.Projectiles))
	for i, p := range s.Projectiles {
		shots[i] = shot{float64(p.ID), round(p.X), round(p.Y), float64(p.Owner)}
	}
	copy := plain(s)
	copy.Players = players
	return json.Marshal(struct {
		plain
		Bots        []point `json:"bots"`
		Projectiles []shot  `json:"projectiles"`
	}{copy, bots, shots})
}

// Accept the compact public tuple as well as older object snapshots in diagnostic
// clients. Simulation snapshots and stored authoritative results stay typed.
func (s *Snapshot) UnmarshalJSON(raw []byte) error {
	type plain Snapshot
	var wire struct {
		plain
		Projectiles []json.RawMessage `json:"projectiles"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	*s = Snapshot(wire.plain)
	s.Projectiles = make([]Projectile, len(wire.Projectiles))
	for i, p := range wire.Projectiles {
		if len(p) > 0 && p[0] == '[' {
			var v [4]float64
			if err := json.Unmarshal(p, &v); err != nil {
				return err
			}
			s.Projectiles[i] = Projectile{ID: int(v[0]), X: v[1], Y: v[2], Owner: int(v[3])}
		} else {
			if err := json.Unmarshal(p, &s.Projectiles[i]); err != nil {
				return err
			}
		}
	}
	return nil
}
