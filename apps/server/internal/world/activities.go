package world

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
)

//go:embed assets/activities.v1.json
var ActivitiesAsset []byte

type Bounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (b Bounds) Contains(p Point) bool {
	return p.X >= b.X && p.X < b.X+b.Width && p.Y >= b.Y && p.Y < b.Y+b.Height
}

type SceneTransform struct {
	Scale   float64 `json:"scale"`
	OffsetX float64 `json:"offsetX"`
	OffsetY float64 `json:"offsetY"`
}
type AreaDefinition struct {
	ID              string         `json:"id"`
	Kind            string         `json:"kind"`
	Bounds          Bounds         `json:"bounds"`
	Entrance        Point          `json:"entrance"`
	SafeReturn      Point          `json:"safeReturn"`
	SpectatorStrips []Bounds       `json:"spectatorStrips"`
	SceneTransform  SceneTransform `json:"sceneTransform"`
	SceneBounds     *Bounds        `json:"sceneBounds,omitempty"`
}
type ActivityContract struct {
	V            int              `json:"v"`
	MapID        string           `json:"mapId"`
	ClearMs      int64            `json:"clearMs"`
	ResetMs      int64            `json:"resetMs"`
	Areas        []AreaDefinition `json:"areas"`
	VehicleSlots []struct {
		ID       string `json:"id"`
		Kind     string `json:"kind"`
		Position Point  `json:"position"`
	} `json:"vehicleSlots"`
	FishingSpots []struct {
		ID       string `json:"id"`
		Position Point  `json:"position"`
	} `json:"fishingSpots"`
	ClawMachines []struct {
		ID       string `json:"id"`
		Position Point  `json:"position"`
		Entrance Point  `json:"entrance"`
		Capacity int    `json:"capacity"`
	} `json:"clawMachines"`
	ClawRules ClawRules          `json:"clawRules"`
	Caps      map[string]float64 `json:"caps"`
}

func LoadActivities(raw []byte, m MapDefinition) (ActivityContract, error) {
	var c ActivityContract
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(&c); e != nil {
		return c, e
	}
	bad := errors.New("invalid activity contract")
	if c.V != 1 || c.MapID != m.ID || c.ClearMs != 3000 || c.ResetMs < 0 || c.ResetMs > 2000 || len(c.Areas) != 4 {
		return c, bad
	}
	rules := c.ClawRules
	if rules.LeaseMs != 20000 || rules.InputExpiryMs != 250 || rules.StepMs != 50 || rules.MaxCatchupSteps != 4 || !valid(rules.Speed, .1, 2) || !valid(rules.GripTolerance, .01, .15) || rules.PhaseMs != 500 || len(rules.Presets) != 3 || len(c.ClawMachines) != 1 {
		return c, bad
	}
	machine := c.ClawMachines[0]
	if machine.ID != "claw-1" || machine.Capacity != 1 || m.Blocked(machine.Entrance) || !m.Reachable(m.Spawn, machine.Entrance) {
		return c, bad
	}
	seenPresets := map[string]bool{}
	for _, p := range rules.Presets {
		if (p.ID != "bear" && p.ID != "bunny" && p.ID != "fox") || seenPresets[p.ID] || !valid(p.X, 0, 1) || !valid(p.Y, 0, 1) {
			return c, bad
		}
		seenPresets[p.ID] = true
	}
	ids := map[string]bool{}
	for _, a := range c.Areas {
		if a.ID == "" || ids[a.ID] || a.Bounds.Width < 1 || a.Bounds.Height < 1 || a.Bounds.X < 0 || a.Bounds.Y < 0 || a.Bounds.X+a.Bounds.Width > 64 || a.Bounds.Y+a.Bounds.Height > 64 {
			return c, bad
		}
		ids[a.ID] = true
		for _, p := range []Point{a.Entrance, a.SafeReturn} {
			if m.Blocked(p) || !m.Reachable(m.Spawn, p) {
				return c, bad
			}
			for _, b := range c.Areas {
				if b.Bounds.Contains(p) {
					return c, bad
				}
			}
		}
	}
	return c, nil
}
func ActivityAreas() []AreaDefinition {
	var c ActivityContract
	if err := json.Unmarshal(ActivitiesAsset, &c); err != nil {
		panic(err)
	}
	return c.Areas
}
func ActivityArea(id string) (AreaDefinition, bool) {
	for _, a := range ActivityAreas() {
		if a.ID == id {
			return a, true
		}
	}
	return AreaDefinition{}, false
}
