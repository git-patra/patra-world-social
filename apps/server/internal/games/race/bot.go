package race

import "patra-world-social/server/internal/games/model"

// Prepare is a bounded, deterministic Fisher-Yates permutation. Only the lobby
// preparation seam calls it; the engine never reshuffles its persisted roster.
func Prepare(m model.MatchStart) ([]model.Actor, error) {
	a := append([]model.Actor{}, m.Actors...)
	state := uint32(m.Seed)
	if state == 0 {
		state = 1
	}
	for i := len(a) - 1; i > 0; i-- {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		j := int(state % uint32(i+1))
		a[i], a[j] = a[j], a[i]
	}
	return a, nil
}
func (e *Engine) bot(k *kart) Controls { // Project onto the racing line, then steer toward a lookahead point.
	line := e.track.Line
	best := int64(1 << 62)
	segment, along := 0, 0
	for i, a := range line {
		b := line[(i+1)%len(line)]
		dx, dy := b.X-a.X, b.Y-a.Y
		length := abs(dx) + abs(dy)
		offset := 0
		if dx != 0 {
			offset = (k.X - a.X) * sign(dx)
		} else {
			offset = (k.Y - a.Y) * sign(dy)
		}
		offset = clamp(offset, 0, length)
		x, y := a.X+sign(dx)*offset, a.Y+sign(dy)*offset
		dist := int64(k.X-x)*int64(k.X-x) + int64(k.Y-y)*int64(k.Y-y)
		if dist < best {
			best = dist
			segment = i
			along = offset
		}
	}
	left := along + e.rules.BotLookahead
	var target Point
	for n := 0; n <= len(line); n++ {
		a, b := line[segment], line[(segment+1)%len(line)]
		dx, dy := b.X-a.X, b.Y-a.Y
		length := abs(dx) + abs(dy)
		if left <= length {
			target = Point{a.X + sign(dx)*left, a.Y + sign(dy)*left}
			break
		}
		left -= length
		segment = (segment + 1) % len(line)
	}
	dx, dy := target.X-k.X, target.Y-k.Y
	heading, bestDot := 0, int64(-1<<62)
	for i, v := range e.track.Directions {
		dot := int64(dx)*int64(v.X) + int64(dy)*int64(v.Y)
		if dot > bestDot {
			heading = i
			bestDot = dot
		}
	}
	delta := (heading-k.Heading+540)%360 - 180
	steer := 0
	if delta > 2 {
		steer = 1
	} else if delta < -2 {
		steer = -1
	}
	speed := e.rules.BotSpeed - k.StartSlot*3 // modest consistency variation, never higher kart limits
	if abs(delta) > 45 {
		speed = 95
	}
	return Controls{Throttle: boolInt(k.Speed < speed), Brake: boolInt(k.Speed > speed+8), Steer: steer}
}
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func sign(v int) int {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
