package world

import (
	"context"
	"encoding/json"
	"math"
	"patra-world-social/server/internal/trackgeom"
	"time"
)

type Vehicle struct {
	ID         string  `json:"id"`
	Kind       string  `json:"kind"`
	Generation uint64  `json:"generation"`
	RiderID    string  `json:"riderId"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Heading    float64 `json:"heading"`
	dx, dy     float64
	expires    time.Time
}

// CasualIntent carries controls and fences only; never a client position.
type CasualIntent struct {
	Incarnation    string  `json:"incarnation"`
	AreaGeneration uint64  `json:"areaGeneration"`
	VehicleID      string  `json:"vehicleId"`
	Generation     uint64  `json:"generation"`
	Action         string  `json:"action"`
	DX             float64 `json:"dx,omitempty"`
	DY             float64 `json:"dy,omitempty"`
}

// MarshalJSON preserves the exact union, including explicit zero axes on neutral drive.
func (i CasualIntent) MarshalJSON() ([]byte, error) {
	payload := map[string]any{"incarnation": i.Incarnation, "areaGeneration": i.AreaGeneration, "vehicleId": i.VehicleID, "generation": i.Generation, "action": i.Action}
	if i.Action == "kick" {
		delete(payload, "vehicleId")
	}
	if i.Action == "drive" || i.Action == "kick" {
		payload["dx"] = i.DX
		payload["dy"] = i.DY
	}
	return json.Marshal(payload)
}

var rideTrack = func() struct{ Outer, Inner trackgeom.Rect } {
	var t struct{ Outer, Inner trackgeom.Rect }
	if err := json.Unmarshal(trackgeom.Asset, &t); err != nil {
		panic(err)
	}
	return t
}()
var rideArea = func() AreaDefinition { a, _ := ActivityArea("race"); return a }()

func RideDriveable(x, y float64) bool {
	t := rideArea.SceneTransform
	return trackgeom.Driveable(rideTrack.Outer, rideTrack.Inner, ((x+.5)*40-t.OffsetX)/t.Scale, ((y+.5)*40-320-t.OffsetY)/t.Scale, 512)
}
func (r *Runtime) initVehicles() {
	var c ActivityContract
	json.Unmarshal(ActivitiesAsset, &c)
	for i, s := range c.VehicleSlots {
		r.vehicles[i] = Vehicle{ID: s.ID, Kind: s.Kind, Generation: 1, X: float64(s.Position.X), Y: float64(s.Position.Y)}
	}
	r.vehicleRevision++
}
func (r *Runtime) riding(id string) *Vehicle {
	for i := range r.vehicles {
		if r.vehicles[i].RiderID == id && id != "" {
			return &r.vehicles[i]
		}
	}
	return nil
}
func (r *Runtime) vehicleSnapshot() [4]Vehicle { return r.vehicles }
func (r *Runtime) dismount(id string) {
	v := r.riding(id)
	if v == nil {
		return
	}
	if p := r.players[id]; p != nil {
		p.X = float64(rideArea.SafeReturn.X)
		p.Y = float64(rideArea.SafeReturn.Y)
		p.dx = 0
		p.dy = 0
		p.path = nil
	}
	v.RiderID = ""
	v.dx = 0
	v.dy = 0
	v.Generation++
	r.vehicleRevision++
}
func (r *Runtime) resetVehicles() {
	for i := range r.vehicles {
		r.dismount(r.vehicles[i].RiderID)
	}
	old := r.vehicles
	r.initVehicles()
	for i := range r.vehicles {
		r.vehicles[i].Generation = old[i].Generation + 1
	}
}
func (r *Runtime) ApplyCasualIntent(ctx context.Context, id string, epoch, seq uint64, intent CasualIntent) error {
	var result error
	err := r.call(ctx, func() {
		if ctx.Err() != nil {
			result = ctx.Err()
			return
		}
		if intent.Action == "kick" {
			result = r.kickFootball(id, epoch, seq, intent)
			return
		}
		p := r.players[id]
		areaGeneration := uint64(0)
		s := r.areas["race"]
		if s != nil {
			areaGeneration = s.lease.Generation
		}
		if p == nil || p.epoch != epoch || p.Status == "playing" || seq <= p.Seq || intent.Incarnation != r.incarnation || intent.AreaGeneration != areaGeneration || s != nil && s.payload.Phase != "OPEN" {
			result = ErrActivitySuperseded
			return
		}
		var v *Vehicle
		for i := range r.vehicles {
			if r.vehicles[i].ID == intent.VehicleID {
				v = &r.vehicles[i]
			}
		}
		if v == nil || v.Generation != intent.Generation {
			result = ErrActivitySuperseded
			return
		}
		switch intent.Action {
		case "mount":
			if v.RiderID != "" || r.riding(id) != nil || math.Hypot(p.X-v.X, p.Y-v.Y) > 2 {
				result = ErrMovement
				return
			}
			v.RiderID = id
			v.Generation++
			r.vehicleRevision++
			p.dx = 0
			p.dy = 0
			p.path = nil
			p.X = v.X
			p.Y = v.Y
		case "dismount":
			if v.RiderID != id {
				result = ErrActivitySuperseded
				return
			}
			r.dismount(id)
		case "drive":
			if v.RiderID != id || !valid(intent.DX, -1, 1) || !valid(intent.DY, -1, 1) {
				result = ErrMovement
				return
			}
			v.dx = intent.DX
			v.dy = intent.DY
			v.expires = r.clock.Now().Add(250 * time.Millisecond)
		default:
			result = ErrMovement
			return
		}
		p.Seq = seq
	})
	if err != nil {
		return err
	}
	return result
}
func (r *Runtime) moveVehicles(at time.Time) {
	for i := range r.vehicles {
		v := &r.vehicles[i]
		if v.RiderID == "" || at.After(v.expires) {
			continue
		}
		length := math.Hypot(v.dx, v.dy)
		if length == 0 {
			continue
		}
		speed := 6.
		if v.Kind == "car" {
			speed = 8
		}
		dx := v.dx / math.Max(1, length) * speed * .05
		dy := v.dy / math.Max(1, length) * speed * .05
		// Sweep at <= 0.05 tile intervals; no high-speed tunnel across the infield or wall.
		steps := int(math.Ceil(math.Hypot(dx, dy) / .05))
		x, y := v.X, v.Y
		for j := 1; j <= steps; j++ {
			nx, ny := v.X+dx*float64(j)/float64(steps), v.Y+dy*float64(j)/float64(steps)
			if !RideDriveable(nx, ny) {
				break
			}
			x, y = nx, ny
		}
		v.X = x
		v.Y = y
		v.Heading = math.Atan2(dy, dx) * 180 / math.Pi
		if p := r.players[v.RiderID]; p != nil {
			p.X = x
			p.Y = y
		}
	}
}
