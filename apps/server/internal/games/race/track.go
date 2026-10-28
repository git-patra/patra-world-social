// Package race simulates authoritative kart controls using integer subunits.
package race

import (
	"encoding/json"
	"patra-world-social/server/internal/trackgeom"
	"patra-world-social/server/internal/world"
)

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}
type Rect = trackgeom.Rect
type Gate struct {
	Axis      string `json:"axis"`
	At        int    `json:"at"`
	Min       int    `json:"min"`
	Max       int    `json:"max"`
	Direction int    `json:"direction"`
}
type Track struct {
	V          int     `json:"v"`
	ID         string  `json:"id"`
	Checksum   string  `json:"checksum"`
	Outer      Rect    `json:"outer"`
	Inner      Rect    `json:"inner"`
	Grid       []Point `json:"grid"`
	Line       []Point `json:"line"`
	Gates      []Gate  `json:"gates"`
	Directions []Point `json:"directions"`
}
type Rules struct {
	TrackID         string `json:"trackId"`
	TrackChecksum   string `json:"trackChecksum"`
	Acceleration    int    `json:"acceleration"`
	Brake           int    `json:"brake"`
	Drag            int    `json:"drag"`
	MaxSpeed        int    `json:"maxSpeed"`
	SteerDegrees    int    `json:"steerDegrees"`
	Radius          int    `json:"radius"`
	BotSpeed        int    `json:"botSpeed"`
	BotLookahead    int    `json:"botLookahead"`
	IntentExpiryMs  int64  `json:"intentExpiryMs"`
	MaxCatchUpSteps int64  `json:"maxCatchUpSteps"`
	LimitMs         int64  `json:"limitMs"`
	FinishGraceMs   int64  `json:"finishGraceMs"`
	ResultsMs       int64  `json:"resultsMs"`
}

var trackAsset = trackgeom.Asset

func DefaultTrack() Track {
	var t Track
	if err := json.Unmarshal(trackAsset, &t); err != nil {
		panic(err)
	}
	return t
}
func DefaultRules() Rules {
	var r struct {
		Race Rules `json:"race"`
	}
	if err := json.Unmarshal(world.RulesAsset, &r); err != nil {
		panic(err)
	}
	return r.Race
}
func (t Track) Driveable(x, y, r int) bool {
	return trackgeom.Driveable(t.Outer, t.Inner, float64(x), float64(y), float64(r))
}
func crossed(g Gate, a, b Point) bool {
	old, next, lo, hi := a.X, b.X, a.Y, b.Y
	if g.Axis == "y" {
		old, next, lo, hi = a.Y, b.Y, a.X, b.X
	}
	if g.Direction*(old-g.At) >= 0 || g.Direction*(next-g.At) < 0 || next == old {
		return false
	} // exact integer intersection; no floating rounding near a gate
	n := int64(lo)*int64(next-old) + int64(hi-lo)*int64(g.At-old)
	d := int64(next - old)
	if d < 0 {
		n = -n
		d = -d
	}
	return n >= int64(g.Min)*d && n <= int64(g.Max)*d
}
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
