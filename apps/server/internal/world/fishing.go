package world

import (
	"encoding/json"
	"math"
	"math/rand/v2"
)

type FishingRules struct {
	WaitMinMs int64    `json:"waitMinMs"`
	WaitMaxMs int64    `json:"waitMaxMs"`
	BiteMs    int64    `json:"biteMs"`
	ResultMs  int64    `json:"resultMs"`
	Proximity float64  `json:"proximity"`
	Catches   []string `json:"catches"`
}

var FishingConfig = func() FishingRules {
	var c ActivityContract
	if e := json.Unmarshal(ActivitiesAsset, &c); e != nil {
		panic(e)
	}
	return c.FishingRules
}()
var fishingPositions = func() map[string]Point {
	var c ActivityContract
	json.Unmarshal(ActivitiesAsset, &c)
	out := map[string]Point{}
	for _, s := range c.FishingSpots {
		out[s.ID] = s.Position
	}
	return out
}()

func validFishingID(id string) bool {
	return id == "fishing-0" || id == "fishing-1" || id == "fishing-2" || id == "fishing-3"
}

// Four timer states only; catches never enter the durable completion pipeline.
type Fishing struct {
	ID         string `json:"-"`
	Generation uint64 `json:"-"`
	OwnerID    string `json:"ownerId"`
	OwnerName  string `json:"ownerName"`
	Phase      string `json:"phase"`
	CastID     uint64 `json:"castId"`
	BiteAt     int64  `json:"biteAt"`
	EndsAt     int64  `json:"endsAt"`
	CatchID    string `json:"catchId"`
}

func (r *Runtime) initFishing() {
	for n := range r.fishing {
		r.fishing[n] = Fishing{ID: []string{"fishing-0", "fishing-1", "fishing-2", "fishing-3"}[n], Generation: 1, Phase: "idle"}
	}
}
func (r *Runtime) resetFishing(f *Fishing) {
	*f = Fishing{ID: f.ID, Generation: f.Generation + 1, CastID: f.CastID, Phase: "idle"}
}
func (r *Runtime) releaseFishing(id string) {
	if id == "" {
		return
	}
	for n := range r.fishing {
		f := &r.fishing[n]
		if f.OwnerID == id {
			r.resetFishing(f)
		}
	}
}
func (r *Runtime) applyFishing(id string, epoch, seq uint64, i CasualIntent) error {
	p := r.players[id]
	var f *Fishing
	for n := range r.fishing {
		if r.fishing[n].ID == i.SpotID {
			f = &r.fishing[n]
		}
	}
	if p == nil || p.epoch != epoch || seq <= p.Seq || p.Status == "playing" || r.riding(id) != nil || r.claw.OperatorID == id || i.Incarnation != r.incarnation || i.AreaGeneration != 0 || f == nil || f.Generation != i.Generation {
		return ErrActivitySuperseded
	}
	now := r.clock.Now().UnixMilli()
	switch i.Action {
	case "fishing.cast":
		spot := fishingPositions[i.SpotID]
		if f.OwnerID != "" || math.Hypot(p.X-float64(spot.X), p.Y-float64(spot.Y)) > FishingConfig.Proximity {
			return ErrMovement
		}
		for _, other := range r.fishing {
			if other.OwnerID == id {
				return ErrActivitySuperseded
			}
		}
		f.OwnerID = id
		f.OwnerName = p.Name
		f.Phase = "waiting"
		f.CastID++
		f.BiteAt = now + FishingConfig.WaitMinMs + rand.Int64N(FishingConfig.WaitMaxMs-FishingConfig.WaitMinMs+1)
		f.EndsAt = f.BiteAt + FishingConfig.BiteMs
		f.Generation++
		p.dx = 0
		p.dy = 0
		p.path = nil
	case "fishing.reel":
		if f.OwnerID != id || (f.Phase != "waiting" && f.Phase != "bite") {
			return ErrActivitySuperseded
		}
		if f.Phase == "bite" {
			f.CatchID = FishingConfig.Catches[rand.IntN(len(FishingConfig.Catches))]
		}
		f.Phase = "result"
		f.EndsAt = now + FishingConfig.ResultMs
		f.Generation++
	case "fishing.leave":
		if f.OwnerID != id {
			return ErrActivitySuperseded
		}
		r.resetFishing(f)
	default:
		return ErrMovement
	}
	p.Seq = seq
	return nil
}
func (r *Runtime) advanceFishing() {
	now := r.clock.Now().UnixMilli()
	for n := range r.fishing {
		f := &r.fishing[n]
		if f.OwnerID == "" {
			continue
		}
		p := r.players[f.OwnerID]
		spot := fishingPositions[f.ID]
		if p == nil || p.Status == "playing" || math.Hypot(p.X-float64(spot.X), p.Y-float64(spot.Y)) > FishingConfig.Proximity || now >= f.EndsAt {
			r.resetFishing(f)
			continue
		}
		if f.Phase == "waiting" && now >= f.BiteAt {
			f.Phase = "bite"
			f.Generation++
		}
	}
}
