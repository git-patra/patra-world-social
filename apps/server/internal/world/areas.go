package world

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"sort"
)

var ErrAreaClearing = errors.New("area clearing acknowledgement pending")
var ErrAreaBusy = errors.New("area already reserved")

type AreaLease struct {
	WorldID     string `json:"worldId"`
	Incarnation string `json:"incarnation"`
	AreaID      string `json:"areaId"`
	MatchID     string `json:"matchId"`
	Generation  uint64 `json:"generation"`
}
type AreaPayload struct {
	Ball     *Football   `json:"-"`
	Vehicles *[4]Vehicle `json:"-"`
	Phase    string      `json:"phase"`
	MatchID  string      `json:"matchId"`
	StartsAt int64       `json:"startsAt"`
	ClearAt  int64       `json:"clearAt"`
	ResetAt  int64       `json:"resetAt"`
}

func (p AreaPayload) MarshalJSON() ([]byte, error) {
	if p.Ball != nil {
		return json.Marshal(struct {
			Ball *Football `json:"ball"`
		}{p.Ball})
	}
	if p.Vehicles != nil {
		return json.Marshal(struct {
			Vehicles *[4]Vehicle `json:"vehicles"`
		}{p.Vehicles})
	}
	type plain AreaPayload
	return json.Marshal(plain(p))
}

type ActivitySnapshot struct {
	WorldID      string      `json:"worldId"`
	AreaID       string      `json:"areaId"`
	Incarnation  string      `json:"incarnation"`
	Generation   uint64      `json:"generation"`
	ServerTimeMs int64       `json:"serverTimeMs"`
	Tick         uint64      `json:"tick"`
	Kind         string      `json:"kind"`
	Payload      AreaPayload `json:"payload"`
}
type areaState struct {
	lease   AreaLease
	payload AreaPayload
}

func incarnation() string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}
func (r *Runtime) ReserveArea(ctx context.Context, worldID, areaID, matchID string) (AreaLease, error) {
	var out AreaLease
	var result error
	err := r.call(ctx, func() {
		a, ok := ActivityArea(areaID)
		if !ok || worldID == "" || matchID == "" {
			result = ErrMovement
			return
		}
		s := r.areas[areaID]
		if s != nil && s.payload.Phase != "OPEN" {
			if s.lease.WorldID == worldID && s.lease.MatchID == matchID {
				out = s.lease
			} else {
				result = ErrAreaBusy
			}
			return
		}
		r.areaGeneration++
		out = AreaLease{worldID, r.incarnation, areaID, matchID, r.areaGeneration}
		r.areas[areaID] = &areaState{out, AreaPayload{Phase: "CLEARING", MatchID: matchID, ClearAt: r.clock.Now().UnixMilli() + 3000}}
		if areaID == "penalty" {
			r.resetFootball(false)
		}
		if areaID == "race" {
			r.resetVehicles()
		}
		// Clear existing paths, held input and objects atomically with mask closure.
		for _, p := range r.players {
			p.dx = 0
			p.dy = 0
			p.path = nil
			if a.Bounds.Contains(Point{int(math.Round(p.X)), int(math.Round(p.Y))}) {
				p.X = float64(a.SafeReturn.X)
				p.Y = float64(a.SafeReturn.Y)
			}
		}
	})
	if err != nil {
		return out, err
	}
	return out, result
}
func (r *Runtime) mutateArea(ctx context.Context, l AreaLease, fn func(*areaState) error) error {
	var result error
	err := r.call(ctx, func() {
		s := r.areas[l.AreaID]
		if s == nil || s.lease != l {
			result = ErrActivitySuperseded
			return
		}
		result = fn(s)
	})
	if err != nil {
		return err
	}
	return result
}
func (r *Runtime) ClearArea(ctx context.Context, l AreaLease) error {
	return r.mutateArea(ctx, l, func(s *areaState) error {
		if s.payload.Phase == "RESERVED" || s.payload.Phase == "ACTIVE" {
			return nil
		}
		if s.payload.Phase != "CLEARING" {
			return ErrActivitySuperseded
		}
		if r.clock.Now().UnixMilli() < s.payload.ClearAt {
			return ErrAreaClearing
		}
		s.payload.Phase = "RESERVED"
		return nil
	})
}
func (r *Runtime) ActivateArea(ctx context.Context, l AreaLease, startsAt int64) error {
	return r.mutateArea(ctx, l, func(s *areaState) error {
		if s.payload.Phase == "ACTIVE" && s.payload.StartsAt == startsAt {
			return nil
		}
		if s.payload.Phase != "RESERVED" {
			return ErrActivitySuperseded
		}
		s.payload.Phase = "ACTIVE"
		s.payload.StartsAt = startsAt
		return nil
	})
}
func (r *Runtime) ReleaseArea(ctx context.Context, l AreaLease) error {
	return r.mutateArea(ctx, l, func(s *areaState) error {
		if s.payload.Phase == "RESETTING" || s.payload.Phase == "OPEN" {
			return nil
		}
		s.payload.Phase = "RESETTING"
		s.payload.ResetAt = r.clock.Now().UnixMilli() + 2000
		return nil
	})
}
func (r *Runtime) advanceAreas() {
	for id, s := range r.areas {
		if s.payload.Phase == "RESETTING" && r.clock.Now().UnixMilli() >= s.payload.ResetAt {
			s.payload.Phase = "OPEN"
			s.payload.MatchID = ""
			s.payload.StartsAt = 0
			if id == "penalty" {
				r.resetFootball(true)
			}
		}
	}
}
func (r *Runtime) blocked(p Point) bool {
	if r.m.Blocked(p) {
		return true
	}
	for id, s := range r.areas {
		if s.payload.Phase != "OPEN" {
			a, _ := ActivityArea(id)
			if a.Bounds.Contains(p) {
				return true
			}
		}
	}
	return false
}
func (r *Runtime) HasAreaLeases(ctx context.Context) (bool, error) {
	var yes bool
	err := r.call(ctx, func() {
		for _, s := range r.areas {
			if s.payload.Phase != "OPEN" {
				yes = true
			}
		}
	})
	return yes, err
}
func (r *Runtime) ActivitySnapshots(ctx context.Context, worldID string) ([]ActivitySnapshot, error) {
	out := []ActivitySnapshot{}
	err := r.call(ctx, func() {
		ball := r.football
		out = append(out, ActivitySnapshot{worldID, "penalty", r.incarnation, r.footballRevision, r.clock.Now().UnixMilli(), r.tick, "football", AreaPayload{Ball: &ball}})
		vehicles := r.vehicleSnapshot()
		out = append(out, ActivitySnapshot{worldID, "race", r.incarnation, r.vehicleRevision, r.clock.Now().UnixMilli(), r.tick, "vehicles", AreaPayload{Vehicles: &vehicles}})
		for _, a := range ActivityAreas() {
			s := r.areas[a.ID]
			p := AreaPayload{Phase: "OPEN"}
			g := uint64(0)
			if s != nil {
				p = s.payload
				g = s.lease.Generation
			}
			out = append(out, ActivitySnapshot{worldID, a.ID, r.incarnation, g, r.clock.Now().UnixMilli(), r.tick, "area", p})
		}
	})
	sort.Slice(out, func(i, j int) bool { return out[i].AreaID < out[j].AreaID })
	return out, err
}
