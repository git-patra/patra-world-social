package world

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	c "patra-world-social/server/internal/contracts"
)

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}
type Building struct {
	ID       string `json:"id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Entrance Point  `json:"entrance"`
}
type District struct {
	ID     string `json:"id"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}
type MapDefinition struct {
	V         int        `json:"v"`
	ID        string     `json:"id"`
	Checksum  string     `json:"checksum"`
	Width     int        `json:"width"`
	Height    int        `json:"height"`
	Spawn     Point      `json:"spawn"`
	Obstacles []Point    `json:"obstacles"`
	Buildings []Building `json:"buildings"`
	Districts []District `json:"districts"`
}

func LoadMap(raw []byte) (MapDefinition, error) {
	var m MapDefinition
	v, e := c.Object(raw)
	bad := errors.New("invalid map")
	if e != nil || !validChecksum(v) || !c.Keys(v, "v", "id", "checksum", "width", "height", "spawn", "buildings", "districts", "obstacles") || v["v"] != float64(1) || v["width"] != float64(64) || v["height"] != float64(64) || !c.Text(v["id"], 128) || !c.Text(v["checksum"], 128) || !c.Point(v["spawn"]) {
		return m, bad
	}
	obstacles, ok := v["obstacles"].([]any)
	if !ok {
		return m, bad
	}
	for _, o := range obstacles {
		if !c.Point(o) {
			return m, bad
		}
		p := o.(map[string]any)
		if !c.Integer(p["x"], 0, 63) || !c.Integer(p["y"], 0, 63) {
			return m, bad
		}
	}
	for _, key := range []string{"buildings", "districts"} {
		items, ok := v[key].([]any)
		if !ok {
			return m, bad
		}
		ids := map[string]bool{}
		for _, item := range items {
			b, ok := item.(map[string]any)
			if !ok || !c.Keys(b, "id", "x", "y", "width", "height", "entrance") || !c.Text(b["id"], 128) || ids[b["id"].(string)] || !c.Number(b["x"], 0, 63) || !c.Number(b["y"], 0, 63) || !c.Integer(b["x"], 0, 63) || !c.Integer(b["y"], 0, 63) || !c.Integer(b["width"], 1, 64) || !c.Integer(b["height"], 1, 64) || b["x"].(float64)+b["width"].(float64) > 64 || b["y"].(float64)+b["height"].(float64) > 64 {
				return m, bad
			}
			ids[b["id"].(string)] = true
			if key == "districts" && !c.Keys(b, "id", "x", "y", "width", "height") {
				return m, bad
			}
			if key == "buildings" {
				if !c.Point(b["entrance"]) {
					return m, bad
				}
				p := b["entrance"].(map[string]any)
				if !c.Integer(p["x"], 0, 63) || !c.Integer(p["y"], 0, 63) {
					return m, bad
				}
			}
		}
	}
	if e = json.Unmarshal(raw, &m); e != nil {
		return m, bad
	}
	if m.Blocked(m.Spawn) {
		return m, bad
	}
	for _, b := range m.Buildings {
		if m.Blocked(b.Entrance) || !m.Reachable(m.Spawn, b.Entrance) {
			return m, bad
		}
	}
	for i, a := range m.Districts {
		for _, b := range m.Districts[i+1:] {
			if a.X < b.X+b.Width && b.X < a.X+a.Width && a.Y < b.Y+b.Height && b.Y < a.Y+a.Height {
				return m, bad
			}
		}
	}
	return m, nil
}
func (m MapDefinition) Blocked(p Point) bool {
	for _, o := range m.Obstacles {
		if p == o {
			return true
		}
	}
	if p.X < 0 || p.Y < 0 || p.X >= 64 || p.Y >= 64 {
		return true
	}
	for _, b := range m.Buildings {
		if p.X >= b.X && p.X < b.X+b.Width && p.Y >= b.Y && p.Y < b.Y+b.Height {
			return true
		}
	}
	return false
}
func (m MapDefinition) Reachable(start, goal Point) bool {
	q := []Point{start}
	seen := map[Point]bool{start: true}
	for len(q) > 0 {
		p := q[0]
		q = q[1:]
		if p == goal {
			return true
		}
		for _, d := range []Point{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
			n := Point{p.X + d.X, p.Y + d.Y}
			if !m.Blocked(n) && !seen[n] {
				seen[n] = true
				q = append(q, n)
			}
		}
	}
	return false
}

type Rules struct {
	V                         int             `json:"v"`
	ID                        string          `json:"id"`
	Checksum                  string          `json:"checksum"`
	SubunitsPerTile           int             `json:"subunitsPerTile"`
	HubTickHz                 int             `json:"hubTickHz"`
	HubSpeedSubunitsPerSecond int             `json:"hubSpeedSubunitsPerSecond"`
	MovementIntentExpiryMs    int             `json:"movementIntentExpiryMs"`
	RaceTickHz                int             `json:"raceTickHz"`
	RaceLaps                  int             `json:"raceLaps"`
	RaceSlots                 int             `json:"raceSlots"`
	PenaltyZones              int             `json:"penaltyZones"`
	PenaltyDeadlineMs         int             `json:"penaltyDeadlineMs"`
	Race                      json.RawMessage `json:"race"`
}

func LoadRules(raw []byte) (Rules, error) {
	var r Rules
	v, e := c.Object(raw)
	bad := errors.New("invalid rules")
	if e != nil || !validChecksum(v) || !c.Keys(v, "v", "id", "checksum", "subunitsPerTile", "hubTickHz", "hubSpeedSubunitsPerSecond", "movementIntentExpiryMs", "raceTickHz", "raceLaps", "raceSlots", "penaltyZones", "penaltyDeadlineMs", "race") || !c.Text(v["id"], 128) || !c.Text(v["checksum"], 128) {
		return r, bad
	}
	for k, n := range map[string]float64{"v": 1, "subunitsPerTile": 1024, "hubTickHz": 20, "raceTickHz": 30, "raceLaps": 5, "raceSlots": 5, "penaltyZones": 4, "penaltyDeadlineMs": 3000} {
		if v[k] != n {
			return r, bad
		}
	}
	if !c.Integer(v["hubSpeedSubunitsPerSecond"], 1, 100000) || !c.Integer(v["movementIntentExpiryMs"], 1, 1000) {
		return r, bad
	}
	race, ok := v["race"].(map[string]any)
	if !ok || !c.Keys(race, "trackId", "trackChecksum", "acceleration", "brake", "drag", "maxSpeed", "steerDegrees", "radius", "botSpeed", "botLookahead", "intentExpiryMs", "maxCatchUpSteps", "limitMs", "finishGraceMs", "resultsMs") || !c.Text(race["trackId"], 128) || !c.Text(race["trackChecksum"], 128) {
		return r, bad
	}
	for _, k := range []string{"acceleration", "brake", "drag", "maxSpeed", "steerDegrees", "radius", "botSpeed", "botLookahead", "intentExpiryMs", "maxCatchUpSteps", "limitMs", "finishGraceMs", "resultsMs"} {
		if !c.Integer(race[k], 1, 300000) {
			return r, bad
		}
	}
	e = json.Unmarshal(raw, &r)
	return r, e
}

// Checksums are SHA-256 over canonical recursively sorted JSON without checksum.
func validChecksum(v map[string]any) bool {
	expected, ok := v["checksum"].(string)
	if !ok {
		return false
	}
	copy := make(map[string]any, len(v))
	for k, x := range v {
		if k != "checksum" {
			copy[k] = x
		}
	}
	raw, e := json.Marshal(copy)
	return e == nil && fmt.Sprintf("%x", sha256.Sum256(raw)) == expected
}
