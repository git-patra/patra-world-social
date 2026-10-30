package world

import (
	"context"
	"math"
	"time"
)

func valid(n, min, max float64) bool {
	return !math.IsNaN(n) && !math.IsInf(n, 0) && n >= min && n <= max
}
func (r *Runtime) Input(ctx context.Context, id string, epoch, seq uint64, dx, dy float64) error {
	if !valid(dx, -1, 1) || !valid(dy, -1, 1) {
		return ErrMovement
	}
	return r.call(ctx, func() {
		p := r.players[id]
		if p == nil || p.Status == "playing" || r.claw.OperatorID == id || r.riding(id) != nil || p.epoch != epoch || seq <= p.Seq {
			return
		}
		if dx != 0 || dy != 0 {
			r.releaseFishing(id)
		}
		p.Seq = seq
		p.dx = dx
		p.dy = dy
		p.path = nil
		p.expires = r.clock.Now().Add(250 * time.Millisecond)
	})
}
func (r *Runtime) Destination(ctx context.Context, id string, epoch, seq uint64, x, y float64) error {
	if !valid(x, 0, 63) || !valid(y, 0, 63) || x != math.Trunc(x) || y != math.Trunc(y) || r.m.Blocked(Point{int(x), int(y)}) {
		return ErrMovement
	}
	var invalid bool
	err := r.call(ctx, func() {
		p := r.players[id]
		if p == nil || p.Status == "playing" || r.claw.OperatorID == id || r.riding(id) != nil || p.epoch != epoch || seq <= p.Seq {
			return
		}
		start := Point{int(math.Round(p.X)), int(math.Round(p.Y))}
		goal := Point{int(x), int(y)}
		path := findPath(start, goal, r.blocked)
		if path == nil {
			invalid = true
			return
		}
		r.releaseFishing(id)
		p.Seq = seq
		p.dx = 0
		p.dy = 0
		p.path = path
	})
	if invalid {
		return ErrMovement
	}
	return err
}
func (m MapDefinition) path(start, goal Point) []Point { return findPath(start, goal, m.Blocked) }
func findPath(start, goal Point, blocked func(Point) bool) []Point {
	if blocked(goal) {
		return nil
	}
	q := []Point{start}
	prev := map[Point]Point{start: start}
	for cursor := 0; cursor < len(q); cursor++ {
		p := q[cursor]
		if p == goal {
			out := []Point{goal}
			for p != start {
				p = prev[p]
				out = append(out, p)
			}
			for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
				out[i], out[j] = out[j], out[i]
			}
			return out
		}
		for _, d := range []Point{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
			n := Point{p.X + d.X, p.Y + d.Y}
			if _, seen := prev[n]; !seen && !blocked(n) {
				prev[n] = p
				q = append(q, n)
			}
		}
	}
	return nil
}
func (r *Runtime) move(p *player, at time.Time) {
	if r.riding(p.ID) != nil || r.claw.OperatorID == p.ID {
		return
	}
	dx, dy := p.dx, p.dy
	step := .225
	if len(p.path) > 0 {
		target := p.path[0]
		if r.blocked(target) {
			p.path = nil
			p.dx = 0
			p.dy = 0
			return
		}
		dx = float64(target.X) - p.X
		dy = float64(target.Y) - p.Y
		dist := math.Hypot(dx, dy)
		if dist <= step {
			step = dist
			p.path = p.path[1:]
		}
	} else if at.After(p.expires) {
		return
	}
	length := math.Hypot(dx, dy)
	if length == 0 {
		return
	}
	if length > 1 || len(p.path) > 0 {
		dx /= length
		dy /= length
	} else if step != .225 {
		dx /= length
		dy /= length
	}
	nx, ny := p.X+dx*step, p.Y+dy*step
	// Rounded tile cells match the client map. Axis checks prevent diagonal corner cutting.
	blocked := func(x, y float64) bool {
		return !valid(x, 0, 63) || !valid(y, 0, 63) || r.blocked(Point{int(math.Round(x)), int(math.Round(y))})
	}
	if !blocked(nx, ny) && !blocked(nx, p.Y) && !blocked(p.X, ny) {
		p.X = nx
		p.Y = ny
	}
}
