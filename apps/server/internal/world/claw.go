package world

import (
	"encoding/json"
	"math"
	"time"
)

// Cabinet coordinates are normalized; presets and timing are shared with the guest owner.
type ClawRules struct {
	LeaseMs         int64   `json:"leaseMs"`
	InputExpiryMs   int64   `json:"inputExpiryMs"`
	StepMs          int     `json:"stepMs"`
	MaxCatchupSteps int     `json:"maxCatchupSteps"`
	Speed           float64 `json:"speed"`
	GripTolerance   float64 `json:"gripTolerance"`
	PhaseMs         int64   `json:"phaseMs"`
	Presets         []struct {
		ID string  `json:"id"`
		X  float64 `json:"x"`
		Y  float64 `json:"y"`
	} `json:"presets"`
}

var ClawConfig = func() ClawRules {
	var c ActivityContract
	if e := json.Unmarshal(ActivitiesAsset, &c); e != nil {
		panic(e)
	}
	return c.ClawRules
}()

var clawEntrance = func() Point {
	var c ActivityContract
	json.Unmarshal(ActivitiesAsset, &c)
	return c.ClawMachines[0].Entrance
}()

type Claw struct {
	OperatorID   string  `json:"operatorId"`
	OperatorName string  `json:"operatorName"`
	LeaseUntil   int64   `json:"leaseUntil"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Phase        string  `json:"phase"`
	PhaseAt      int64   `json:"phaseAt"`
	DropID       uint64  `json:"dropId"`
	PrizeID      string  `json:"prizeId"`
	dx, dy       float64
	expires      time.Time
}

func (r *Runtime) resetClaw() {
	r.claw = Claw{X: .5, Y: .5, Phase: "idle", DropID: r.claw.DropID}
	r.clawRevision++
}
func (r *Runtime) releaseClaw(id string) {
	if id != "" && r.claw.OperatorID == id {
		r.resetClaw()
	}
}
func (r *Runtime) applyClaw(id string, epoch, seq uint64, i CasualIntent) error {
	p := r.players[id]
	b := &r.claw
	machine := i.MachineID
	if p == nil || p.epoch != epoch || p.Status == "playing" || r.riding(id) != nil || seq <= p.Seq || i.Incarnation != r.incarnation || i.AreaGeneration != 0 || i.Generation != r.clawRevision || machine != "claw-1" {
		return ErrActivitySuperseded
	}
	switch i.Action {
	case "claw.enter":
		if b.OperatorID != "" || math.Hypot(p.X-float64(clawEntrance.X), p.Y-float64(clawEntrance.Y)) > 2 {
			return ErrMovement
		}
		r.resetClaw()
		b = &r.claw
		b.OperatorID = id
		b.OperatorName = p.Name
		b.LeaseUntil = r.clock.Now().UnixMilli() + ClawConfig.LeaseMs
		b.Phase = "aiming"
		p.dx = 0
		p.dy = 0
		p.path = nil
	case "claw.leave":
		if b.OperatorID != id {
			return ErrActivitySuperseded
		}
		r.resetClaw()
	case "claw.move":
		if b.OperatorID != id || b.Phase != "aiming" || !valid(i.DX, -1, 1) || !valid(i.DY, -1, 1) {
			return ErrMovement
		}
		b.dx = i.DX
		b.dy = i.DY
		b.expires = r.clock.Now().Add(time.Duration(ClawConfig.InputExpiryMs) * time.Millisecond)
	case "claw.drop":
		if b.OperatorID != id || b.Phase != "aiming" {
			return ErrActivitySuperseded
		}
		b.dx = 0
		b.dy = 0
		b.Phase = "dropping"
		b.PhaseAt = r.clock.Now().UnixMilli()
		b.DropID++
		b.PrizeID = ""
		for _, preset := range ClawConfig.Presets {
			if math.Hypot(b.X-preset.X, b.Y-preset.Y) <= ClawConfig.GripTolerance {
				b.PrizeID = preset.ID
				break
			}
		}
		r.clawRevision++
	default:
		return ErrMovement
	}
	p.Seq = seq
	return nil
}
func (r *Runtime) advanceClaw() {
	b := &r.claw
	now := r.clock.Now().UnixMilli()
	if b.OperatorID == "" {
		return
	}
	if now >= b.LeaseUntil {
		r.resetClaw()
		return
	}
	if b.Phase == "aiming" {
		return
	}
	phase := "dropping"
	elapsed := now - b.PhaseAt
	if elapsed >= 3*ClawConfig.PhaseMs {
		phase = "aiming"
		b.PrizeID = ""
	} else if elapsed >= 2*ClawConfig.PhaseMs {
		phase = "releasing"
	} else if elapsed >= ClawConfig.PhaseMs {
		phase = "lifting"
	}
	if phase != b.Phase {
		b.Phase = phase
		r.clawRevision++
	}
}
func (r *Runtime) moveClaw(at time.Time) {
	b := &r.claw
	if b.OperatorID == "" || b.Phase != "aiming" || at.After(b.expires) {
		return
	}
	length := math.Max(1, math.Hypot(b.dx, b.dy))
	step := ClawConfig.Speed * float64(ClawConfig.StepMs) / 1000
	b.X = math.Max(0, math.Min(1, b.X+b.dx/length*step))
	b.Y = math.Max(0, math.Min(1, b.Y+b.dy/length*step))
}
