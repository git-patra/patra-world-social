package blaster

import "math"

// Fixed perimeter candidates bound spawn work and guarantee .2 separation.
func (e *Engine) spawn() {
	e.bots = []Bot{}
	e.projectiles = []Projectile{}
	for i := range e.players {
		if !e.left[i] && e.disconnected[i] == 0 {
			e.players[i].HP = rules.HP
		}
		e.controls[i] = Controls{Aim: e.players[i].Aim}
	}
	for i := 0; i < rules.Waves[e.wave-1]; i++ {
		bestX, bestY, best := .05, .05, -1.
		for j := 0; j < 32; j++ {
			side, t := j/8, float64(j%8)/7*.9+.05
			x, y := t, .05
			switch side {
			case 1:
				x, y = .95, t
			case 2:
				x, y = t, .95
			case 3:
				x, y = .05, t
			}
			distance := 2.
			for _, p := range e.players {
				distance = math.Min(distance, math.Hypot(x-p.X, y-p.Y))
			}
			for _, b := range e.bots {
				distance = math.Min(distance, math.Hypot(x-b.X, y-b.Y))
			}
			if distance > best {
				bestX, bestY, best = x, y, distance
			}
		}
		e.bots = append(e.bots, Bot{ID: (e.wave-1)*8 + i, X: bestX, Y: bestY, NextFire: e.at + int64(rules.BotCooldownMs+i*100)})
	}
}
func (e *Engine) moveBots(dt float64) {
	for i := range e.bots {
		b := &e.bots[i]
		target := -1
		distance := 2.
		for j, p := range e.players {
			d := math.Hypot(p.X-b.X, p.Y-b.Y)
			if p.HP > 0 && d < distance {
				target, distance = j, d
			}
		}
		if target < 0 {
			continue
		}
		p := e.players[target]
		aim := math.Atan2(p.Y-b.Y, p.X-b.X)
		if distance > .2 {
			b.X = clamp(b.X+math.Cos(aim)*rules.BotSpeed*dt, .025, .975)
			b.Y = clamp(b.Y+math.Sin(aim)*rules.BotSpeed*dt, .025, .975)
		}
		if e.at >= b.NextFire {
			e.shoot(b.X, b.Y, aim, -1)
			b.NextFire = e.at + int64(rules.BotCooldownMs)
		}
	}
}
