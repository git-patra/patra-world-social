// Package blaster owns a bounded cooperative arena. Only intent enters this model.
package blaster

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"patra-world-social/server/internal/games/model"
	"time"
)

//go:embed rules.v1.json
var rulesJSON []byte
var rules struct {
	TickMs, MaxCatchupSteps, DeadlineMs, InputExpiryMs, DisconnectGraceMs, MaxPlayers, MaxBots, MaxProjectiles, HP int
	Waves                                                                                                          []int
	PlayerSpeed, BotSpeed, ProjectileSpeed                                                                         float64
	FireCooldownMs, BotCooldownMs, ProjectileTTL                                                                   int
	PlayerRadius, BotRadius, ProjectileRadius                                                                      float64
}

func init() {
	if err := json.Unmarshal(rulesJSON, &rules); err != nil {
		panic(err)
	}
}
func RulesChecksum() string { h := sha256.Sum256(rulesJSON); return hex.EncodeToString(h[:]) }

var ErrInput = errors.New("blaster input closed, stale or unauthorized")

type Controls struct {
	DX   float64 `json:"dx"`
	DY   float64 `json:"dy"`
	Aim  float64 `json:"aim"`
	Fire bool    `json:"fire"`
}
type Player struct {
	ActorID string  `json:"actorId"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Aim     float64 `json:"aim"`
	HP      int     `json:"hp"`
	Kills   int     `json:"kills"`
	HitAt   int64   `json:"hitAt"`
}
type Bot struct {
	ID       int     `json:"id"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	NextFire int64   `json:"nextFire"`
}
type Projectile struct {
	ID    int     `json:"id"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	VX    float64 `json:"vx"`
	VY    float64 `json:"vy"`
	Owner int     `json:"owner"`
	TTL   int     `json:"ttl"`
}
type Snapshot struct {
	MatchID      string             `json:"matchId"`
	Phase        string             `json:"phase"`
	Frame        int64              `json:"frame"`
	ServerTimeMs int64              `json:"serverTimeMs"`
	StartsAt     int64              `json:"startsAt"`
	DeadlineAt   int64              `json:"deadlineAt"`
	Wave         int                `json:"wave"`
	Actors       []model.Actor      `json:"actors"`
	Players      []Player           `json:"players"`
	Bots         []Bot              `json:"bots"`
	Projectiles  []Projectile       `json:"projectiles"`
	Result       *model.MatchResult `json:"result,omitempty"`
}
type Engine struct {
	start                           model.MatchStart
	at, frame                       int64
	wave, nextID, kills             int
	players                         []Player
	bots                            []Bot
	projectiles                     []Projectile
	controls                        [4]Controls
	seq                             [4]uint64
	expires, nextFire, disconnected [4]int64
	left                            [4]bool
	result                          *model.MatchResult
}

func New(m model.MatchStart) *Engine {
	m.Actors = append([]model.Actor{}, m.Actors...)
	e := &Engine{start: m, at: m.StartsAt, wave: 1, players: []Player{}, bots: []Bot{}, projectiles: []Projectile{}}
	for i, a := range m.Actors {
		e.players = append(e.players, Player{ActorID: a.ActorID, X: .35 + float64(i)*.1, Y: .5, HP: 3})
	}
	e.spawn()
	return e
}
func (e *Engine) actor(user string) int {
	for i, a := range e.start.Actors {
		if a.ActorType == "human" && a.UserID == user {
			return i
		}
	}
	return -1
}
func (e *Engine) Input(user string, seq uint64, c Controls, at time.Time) error {
	e.Tick(at)
	i := e.actor(user)
	if i < 0 || i >= 4 || e.result != nil || e.left[i] || e.disconnected[i] != 0 || e.players[i].HP == 0 || seq <= e.seq[i] || !finite(c.DX) || !finite(c.DY) || !finite(c.Aim) || math.Abs(c.DX) > 1 || math.Abs(c.DY) > 1 || math.Abs(c.Aim) > math.Pi {
		return ErrInput
	}
	e.seq[i] = seq
	e.controls[i] = c
	e.expires[i] = at.UnixMilli() + int64(rules.InputExpiryMs)
	return nil
}
func finite(v float64) bool           { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
func (e *Engine) shoot(x, y, aim float64, owner int) {
	if len(e.projectiles) >= rules.MaxProjectiles {
		return
	}
	e.nextID++
	e.projectiles = append(e.projectiles, Projectile{ID: e.nextID, X: x, Y: y, VX: math.Cos(aim) * rules.ProjectileSpeed, VY: math.Sin(aim) * rules.ProjectileSpeed, Owner: owner, TTL: rules.ProjectileTTL})
}
func (e *Engine) step() {
	dt := float64(rules.TickMs) / 1000
	alive := false
	for i := range e.players {
		p := &e.players[i]
		if e.left[i] || e.disconnected[i] > 0 && e.at >= e.disconnected[i]+int64(rules.DisconnectGraceMs) {
			p.HP = 0
		}
		if p.HP == 0 {
			continue
		}
		alive = true
		c := e.controls[i]
		if e.expires[i] <= e.at || e.disconnected[i] != 0 {
			c = Controls{Aim: p.Aim}
		}
		n := math.Max(1, math.Hypot(c.DX, c.DY))
		p.X = clamp(p.X+c.DX/n*rules.PlayerSpeed*dt, rules.PlayerRadius, 1-rules.PlayerRadius)
		p.Y = clamp(p.Y+c.DY/n*rules.PlayerSpeed*dt, rules.PlayerRadius, 1-rules.PlayerRadius)
		p.Aim = c.Aim
		if c.Fire && e.at >= e.nextFire[i] {
			e.shoot(p.X, p.Y, c.Aim, i)
			e.nextFire[i] = e.at + int64(rules.FireCooldownMs)
		}
	}
	if !alive {
		e.finish(e.at, "all_down")
		return
	}
	e.moveBots(dt)
	out := e.projectiles[:0]
	for _, p := range e.projectiles {
		oldX, oldY := p.X, p.Y
		p.X += p.VX * dt
		p.Y += p.VY * dt
		p.TTL -= rules.TickMs
		if p.TTL <= 0 || p.X < 0 || p.X > 1 || p.Y < 0 || p.Y > 1 {
			continue
		}
		hit := false
		if p.Owner >= 0 {
			for j, b := range e.bots {
				if segmentDistance(b.X, b.Y, oldX, oldY, p.X, p.Y) <= rules.BotRadius+rules.ProjectileRadius {
					e.bots = append(e.bots[:j], e.bots[j+1:]...)
					e.kills++
					e.players[p.Owner].Kills++
					hit = true
					break
				}
			}
		} else {
			for j := range e.players {
				v := &e.players[j]
				if v.HP > 0 && segmentDistance(v.X, v.Y, oldX, oldY, p.X, p.Y) <= rules.PlayerRadius+rules.ProjectileRadius {
					v.HP--
					v.HitAt = e.at
					hit = true
					break
				}
			}
		}
		if !hit {
			out = append(out, p)
		}
	}
	e.projectiles = out
	alive = false
	for _, p := range e.players {
		alive = alive || p.HP > 0
	}
	if !alive {
		e.finish(e.at, "all_down")
		return
	}
	if len(e.bots) == 0 {
		if e.wave == len(rules.Waves) {
			e.finish(e.at, "cleared")
		} else {
			e.wave++
			e.spawn()
		}
	}
}
func segmentDistance(x, y, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	n := dx*dx + dy*dy
	t := 0.
	if n > 0 {
		t = clamp(((x-ax)*dx+(y-ay)*dy)/n, 0, 1)
	}
	return math.Hypot(x-ax-t*dx, y-ay-t*dy)
}
func (e *Engine) Tick(now time.Time) {
	if e.result != nil {
		return
	}
	deadline := e.start.StartsAt + int64(rules.DeadlineMs)
	limit := min(now.UnixMilli(), deadline)
	steps := 0
	for e.at+int64(rules.TickMs) <= limit && steps < rules.MaxCatchupSteps && e.result == nil {
		e.at += int64(rules.TickMs)
		e.frame++
		e.step()
		steps++
	}
	if e.at+int64(rules.TickMs) <= limit {
		e.at = limit - (limit-e.start.StartsAt)%int64(rules.TickMs)
		e.frame = (e.at - e.start.StartsAt) / int64(rules.TickMs)
	}
	if e.result == nil && now.UnixMilli() >= deadline {
		e.finish(deadline, "timeout")
	}
}
func (e *Engine) Presence(user, kind string, at time.Time) {
	i := e.actor(user)
	if i < 0 || e.result != nil {
		return
	}
	if kind == "disconnect" {
		if e.disconnected[i] == 0 {
			e.disconnected[i] = at.UnixMilli()
		}
		e.controls[i] = Controls{}
		return
	}
	e.Tick(at)
	if e.result != nil {
		return
	}
	e.controls[i] = Controls{}
	if kind == "resume" {
		e.disconnected[i] = 0
		e.seq[i] = 0
	}
	if kind == "leave" || kind == "expired" {
		e.left[i] = true
		e.players[i].HP = 0
		alive := false
		for _, p := range e.players {
			alive = alive || p.HP > 0
		}
		if !alive {
			e.finish(at.UnixMilli(), "all_down")
		}
	}
}
func (e *Engine) finish(at int64, reason string) {
	if e.result != nil {
		return
	}
	waves := e.wave - 1
	if reason == "cleared" {
		waves = 3
	}
	r := model.MatchResult{Version: e.start.Version, MatchID: e.start.MatchID, WorldID: e.start.WorldID, ArenaID: e.start.ArenaID, GameType: "blaster", PreparedAt: e.start.PreparedAt, StartsAt: e.start.StartsAt, FinishedAt: at, Blaster: &model.BlasterResult{Waves: waves, Kills: e.kills, Reason: reason}}
	for i, a := range e.start.Actors {
		outcome := "loss"
		if reason == "cleared" {
			outcome = "win"
		}
		if e.left[i] {
			outcome = "forfeit"
		}
		score := int64(e.players[i].Kills)
		r.Participants = append(r.Participants, model.ParticipantResult{Actor: a, Outcome: outcome, Score: &score})
	}
	e.result = &r
	e.projectiles = []Projectile{}
	e.frame++
}
func (e *Engine) Snapshot() Snapshot {
	phase := "playing"
	if e.result != nil {
		phase = "results"
	}
	s := Snapshot{MatchID: e.start.MatchID, Phase: phase, Frame: e.frame, ServerTimeMs: e.at, StartsAt: e.start.StartsAt, DeadlineAt: e.start.StartsAt + int64(rules.DeadlineMs), Wave: e.wave, Actors: append([]model.Actor{}, e.start.Actors...), Players: append([]Player{}, e.players...), Bots: append([]Bot{}, e.bots...), Projectiles: append([]Projectile{}, e.projectiles...)}
	if e.result != nil {
		raw, _ := json.Marshal(e.result)
		json.Unmarshal(raw, &s.Result)
	}
	return s
}
