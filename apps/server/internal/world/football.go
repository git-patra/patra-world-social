package world

import (
	"math"
	"time"
)

const footballKickRadius = 1.5
const footballRadius = .15
const footballSpeed = 8.
const footballCooldown = 400 * time.Millisecond
const footballIdle = 15 * time.Second

var footballArea = func() AreaDefinition { a, _ := ActivityArea("penalty"); return a }()

type Football struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	VX       float64 `json:"vx"`
	VY       float64 `json:"vy"`
	Visible  bool    `json:"visible"`
	activeAt time.Time
}

func FootballInside(x, y float64) bool {
	b := footballArea.Bounds
	return valid(x, float64(b.X)+footballRadius, float64(b.X+b.Width)-footballRadius) && valid(y, float64(b.Y)+footballRadius, float64(b.Y+b.Height)-footballRadius)
}

func (r *Runtime) resetFootball(visible bool) {
	r.football = Football{X: 28, Y: 46, Visible: visible, activeAt: r.clock.Now()}
	r.footballRevision++
}

// Called only on the owner; rejected kicks never consume the movement sequence.
func (r *Runtime) kickFootball(id string, epoch, seq uint64, i CasualIntent) error {
	p := r.players[id]
	s := r.areas["penalty"]
	generation := uint64(0)
	if s != nil {
		generation = s.lease.Generation
	}
	if p == nil || p.epoch != epoch || p.Status == "playing" || r.claw.OperatorID == id || r.riding(id) != nil || seq <= p.Seq || i.Incarnation != r.incarnation || i.AreaGeneration != generation || i.Generation != r.footballRevision || !r.football.Visible || s != nil && s.payload.Phase != "OPEN" {
		return ErrActivitySuperseded
	}
	length := math.Hypot(i.DX, i.DY)
	if !valid(i.DX, -1, 1) || !valid(i.DY, -1, 1) || length == 0 || math.Hypot(p.X-r.football.X, p.Y-r.football.Y) > footballKickRadius || !p.lastFootballKick.IsZero() && r.clock.Now().Sub(p.lastFootballKick) < footballCooldown {
		return ErrMovement
	}
	r.football.VX = i.DX / length * footballSpeed
	r.football.VY = i.DY / length * footballSpeed
	r.football.activeAt = r.clock.Now()
	r.footballRevision++
	p.lastFootballKick = r.clock.Now()
	p.Seq = seq
	return nil
}

func (r *Runtime) moveFootball(at time.Time) {
	b := &r.football
	if !b.Visible {
		return
	}
	if !FootballInside(b.X, b.Y) || r.clock.Now().Sub(b.activeAt) >= footballIdle {
		r.resetFootball(true)
		return
	}
	if b.VX == 0 && b.VY == 0 {
		return
	}
	dx, dy := b.VX*.05, b.VY*.05
	steps := int(math.Ceil(math.Hypot(dx, dy) / .05))
	x, y := b.X, b.Y
	for j := 1; j <= steps; j++ {
		nx, ny := x+dx*float64(j)/float64(steps), y+dy*float64(j)/float64(steps)
		if !FootballInside(nx, ny) {
			b.VX = 0
			b.VY = 0
			return
		}
		b.X = nx
		b.Y = ny
	}
	b.VX *= .9
	b.VY *= .9
	if math.Hypot(b.VX, b.VY) < .05 {
		b.VX = 0
		b.VY = 0
	}
}
