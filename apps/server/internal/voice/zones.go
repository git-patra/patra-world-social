package voice

import (
	"embed"
	"encoding/json"
	"errors"
	"math"
	"patra-world-social/server/internal/world"
)

//go:embed tables.v1.json
var zonesFS embed.FS

type Zone struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	X, Y, Radius float64
}
type Zones struct {
	V           int    `json:"v"`
	ID          string `json:"id"`
	MapID       string `json:"mapId"`
	MapVersion  int    `json:"mapVersion"`
	MapChecksum string `json:"mapChecksum"`
	Tables      []Zone `json:"tables"`
}

var TableZones = func() Zones {
	raw, _ := zonesFS.ReadFile("tables.v1.json")
	var z Zones
	if err := json.Unmarshal(raw, &z); err != nil {
		panic(err)
	}
	return z
}()

func InTable(id int, x, y float64) bool {
	for _, z := range TableZones.Tables {
		if z.ID == id {
			return math.Hypot(x-z.X, y-z.Y) <= z.Radius
		}
	}
	return false
}
func ValidateZones(m world.MapDefinition) error {
	z := TableZones
	if z.V != 1 || z.MapID != m.ID || z.MapVersion != m.V || z.MapChecksum != m.Checksum || len(z.Tables) != 2 {
		return errors.New("voice map mismatch")
	}
	for i, t := range z.Tables {
		p := world.Point{X: int(t.X), Y: int(t.Y)}
		if t.ID != i || t.Radius <= 0 || t.X-t.Radius < 0 || t.Y-t.Radius < 0 || t.X+t.Radius >= float64(m.Width) || t.Y+t.Radius >= float64(m.Height) || m.Blocked(p) {
			return errors.New("invalid voice zone")
		}
		for _, o := range z.Tables[:i] {
			if math.Hypot(t.X-o.X, t.Y-o.Y) <= t.Radius+o.Radius {
				return errors.New("overlapping tables")
			}
		}
	}
	return nil
}
