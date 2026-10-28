// Package pingpong owns the fixed-step table, paddles, scoring and deadlines.
package pingpong

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"math"
	"patra-world-social/server/internal/games/model"
	"time"
)

//go:embed rules.v1.json
var rules []byte

func RulesChecksum() string { h := sha256.Sum256(rules); return hex.EncodeToString(h[:]) }

var ErrInput = errors.New("pingpong input closed, stale or unauthorized")

type Ball struct {
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	VX float64 `json:"vx"`
	VY float64 `json:"vy"`
}
type Snapshot struct {
	MatchID      string             `json:"matchId"`
	Phase        string             `json:"phase"`
	Frame        int64              `json:"frame"`
	ServerTimeMs int64              `json:"serverTimeMs"`
	StartsAt     int64              `json:"startsAt"`
	DeadlineAt   int64              `json:"deadlineAt"`
	RallyID      int                `json:"rallyId"`
	Server       int                `json:"server"`
	Actors       []model.Actor      `json:"actors"`
	Paddles      [2]float64         `json:"paddles"`
	Scores       [2]int64           `json:"scores"`
	Ball         Ball               `json:"ball"`
	Result       *model.MatchResult `json:"result,omitempty"`
}
type Engine struct {
	start        model.MatchStart
	at           int64
	frame        int64
	rally        int
	server       int
	paddles      [2]float64
	scores       [2]int64
	ball         Ball
	playing      bool
	axis         [2]float64
	expires      [2]int64
	seq          [2]uint64
	disconnected [2]int64
	result       *model.MatchResult
}

func New(m model.MatchStart) *Engine {
	m.Actors = append([]model.Actor{}, m.Actors...)
	return &Engine{start: m, at: m.StartsAt, paddles: [2]float64{.5, .5}, rally: 1, ball: Ball{X: .5, Y: .5}}
}
func (e *Engine) actor(user string) int {
	for i, a := range e.start.Actors {
		if a.ActorType == "human" && a.UserID == user {
			return i
		}
	}
	return -1
}
func (e *Engine) Input(user string, seq uint64, axis float64, at time.Time) error {
	e.Tick(at)
	i := e.actor(user)
	if i < 0 || e.result != nil || e.disconnected[i] != 0 || seq <= e.seq[i] || math.IsNaN(axis) || math.IsInf(axis, 0) || axis < -1 || axis > 1 {
		return ErrInput
	}
	e.seq[i] = seq
	e.axis[i] = axis
	e.expires[i] = at.UnixMilli() + 250
	return nil
}
func (e *Engine) Serve(user string, at time.Time) error {
	e.Tick(at)
	i := e.actor(user)
	if i != e.server || i < 0 || e.result != nil || e.playing || e.disconnected[i] != 0 || at.UnixMilli() < e.start.StartsAt {
		return ErrInput
	}
	e.serve()
	return nil
}
func (e *Engine) serve() {
	e.playing = true
	direction := 1.
	if e.server == 1 {
		direction = -1
	}
	e.ball = Ball{X: .5, Y: .5, VX: .6 * direction, VY: .3}
	if e.rally%2 == 0 {
		e.ball.VY = -.3
	}
}
func won(a, b int) bool               { return a >= 11 || (a >= 7 && a-b >= 2) }
func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
func (e *Engine) step() {
	for i, a := range e.start.Actors {
		axis := e.axis[i]
		speed := 1.
		if e.expires[i] < e.at || e.disconnected[i] != 0 {
			axis = 0
		}
		if a.ActorType == "bot" {
			axis = botAxis(e.ball.Y, e.paddles[i])
			speed = .65
		}
		e.paddles[i] = clamp(e.paddles[i]+axis*speed*.05, .12, .88)
	}
	if !e.playing {
		if e.start.Actors[e.server].ActorType == "bot" {
			e.serve()
		}
		return
	}
	old := e.ball
	e.ball.X += e.ball.VX * .05
	e.ball.Y += e.ball.VY * .05
	if e.ball.Y < .02 {
		e.ball.Y = .04 - e.ball.Y
		e.ball.VY = math.Abs(e.ball.VY)
	}
	if e.ball.Y > .98 {
		e.ball.Y = 1.96 - e.ball.Y
		e.ball.VY = -math.Abs(e.ball.VY)
	}
	for i, x := range []float64{.06, .94} {
		cross := i == 0 && old.X >= x && e.ball.X <= x || i == 1 && old.X <= x && e.ball.X >= x
		if cross && math.Abs(e.ball.Y-e.paddles[i]) <= .14 {
			e.ball.X = 2*x - e.ball.X
			e.ball.VX = -e.ball.VX
			e.ball.VY = clamp((e.ball.Y-e.paddles[i])*3, -.45, .45)
		}
	}
	scorer := -1
	if e.ball.X < 0 {
		scorer = 1
	}
	if e.ball.X > 1 {
		scorer = 0
	}
	if scorer >= 0 {
		e.scores[scorer]++
		if won(int(e.scores[scorer]), int(e.scores[1-scorer])) {
			e.finish(e.at, -1, "score")
			return
		}
		e.server = 1 - scorer
		e.rally++
		e.playing = false
		e.ball = Ball{X: .5, Y: .5}
	}
}
func (e *Engine) Tick(now time.Time) {
	if e.result != nil {
		return
	}
	limit := now.UnixMilli()
	deadline := e.start.StartsAt + 120000
	cutoff := deadline
	loser := -1
	for i, at := range e.disconnected {
		if at > 0 && at+15000 < cutoff {
			cutoff = at + 15000
			loser = i
		}
	}
	if limit > cutoff {
		limit = cutoff
	}
	steps := 0
	for e.at+50 <= limit && steps < 4 && e.result == nil {
		e.at += 50
		e.frame++
		e.step()
		steps++
	}
	if e.at+50 <= limit {
		e.at = limit - (limit-e.start.StartsAt)%50
		e.frame = (e.at - e.start.StartsAt) / 50
	}
	if e.result == nil && now.UnixMilli() >= cutoff {
		reason := "deadline"
		if loser >= 0 {
			reason = "forfeit"
		}
		e.finish(cutoff, loser, reason)
	}
}
func (e *Engine) Presence(user, kind string, at time.Time) {
	i := e.actor(user)
	if i < 0 {
		return
	}
	if kind == "disconnect" {
		if e.disconnected[i] == 0 {
			e.disconnected[i] = at.UnixMilli()
		}
		e.axis[i] = 0
		return
	}
	e.Tick(at)
	if e.result != nil {
		return
	}
	if kind == "resume" {
		e.disconnected[i] = 0
		e.axis[i] = 0
		e.seq[i] = 0
	}
	if kind == "leave" {
		e.finish(at.UnixMilli(), i, "forfeit")
	}
}
func (e *Engine) finish(at int64, loser int, reason string) {
	if e.result != nil {
		return
	}
	r := model.MatchResult{Version: e.start.Version, MatchID: e.start.MatchID, WorldID: e.start.WorldID, ArenaID: e.start.ArenaID, GameType: "pingpong", PreparedAt: e.start.PreparedAt, StartsAt: e.start.StartsAt, FinishedAt: at, Pingpong: &model.PingpongResult{Scores: e.scores, Rallies: e.rally, Reason: reason}}
	for i, a := range e.start.Actors {
		score := e.scores[i]
		position := 1
		outcome := "draw"
		if loser >= 0 {
			if loser == i {
				outcome = "forfeit"
				position = 2
			} else {
				outcome = "win"
			}
		} else if score > e.scores[1-i] {
			outcome = "win"
		} else if score < e.scores[1-i] {
			outcome = "loss"
			position = 2
		}
		r.Participants = append(r.Participants, model.ParticipantResult{Actor: a, Outcome: outcome, Score: &score, Position: &position})
	}
	e.result = &r
	e.ball.VX = 0
	e.ball.VY = 0
	e.frame++
}
func (e *Engine) Snapshot() Snapshot {
	phase := "serve"
	if e.playing {
		phase = "rally"
	}
	if e.result != nil {
		phase = "results"
	}
	s := Snapshot{MatchID: e.start.MatchID, Phase: phase, Frame: e.frame, ServerTimeMs: e.at, StartsAt: e.start.StartsAt, DeadlineAt: e.start.StartsAt + 120000, RallyID: e.rally, Server: e.server, Actors: append([]model.Actor{}, e.start.Actors...), Paddles: e.paddles, Scores: e.scores, Ball: e.ball}
	if e.result != nil {
		r := *e.result
		meta := *r.Pingpong
		r.Pingpong = &meta
		r.Participants = append([]model.ParticipantResult{}, r.Participants...)
		for i := range r.Participants {
			score, pos := *r.Participants[i].Score, *r.Participants[i].Position
			r.Participants[i].Score = &score
			r.Participants[i].Position = &pos
		}
		s.Result = &r
	}
	return s
}
