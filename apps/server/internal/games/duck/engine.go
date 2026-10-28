package duck

import (
	"errors"
	"math"
	"patra-world-social/server/internal/games/model"
	"time"
)

var ErrInput = errors.New("invalid duck input or inactive seat")

type InputTiming struct {
	ReceivedAt   time.Time
	ClientTimeMs float64
	RewindMs     int64
}
type ShotResult struct {
	MatchID  string `json:"matchId"`
	Seq      uint64 `json:"seq"`
	TargetID int    `json:"targetId"`
	Points   int64  `json:"points"`
	Score    int64  `json:"score"`
	Hits     int    `json:"hits"`
	Shots    int    `json:"shots"`
	RewindMs int64  `json:"rewindMs"`
}
type PublicSnapshot struct {
	MatchID      string             `json:"matchId"`
	WorldID      string             `json:"worldId"`
	Actor        model.Actor        `json:"actor"`
	Phase        string             `json:"phase"`
	StartsAt     int64              `json:"startsAt"`
	DeadlineAt   int64              `json:"deadlineAt"`
	ServerTimeMs int64              `json:"serverTimeMs"`
	Frame        int64              `json:"frame"`
	DroppedSteps int64              `json:"droppedSteps"`
	Targets      []PublicTarget     `json:"targets"`
	Score        int64              `json:"score"`
	Hits         int                `json:"hits"`
	Shots        int                `json:"shots"`
	Result       *model.MatchResult `json:"result,omitempty"`
}
type PublicEvent struct {
	Type     string
	Snapshot PublicSnapshot
}
type historyFrame struct {
	at      int64
	targets []target
}
type Engine struct {
	start                                                                       model.MatchStart
	rules                                                                       Rules
	rng                                                                         model.RNG
	phase                                                                       string
	now, frame, dropped, nextSpawn, lastShot, disconnected, leaveAt, finishedAt int64
	spawnID                                                                     int
	seq                                                                         uint64
	score                                                                       int64
	hits, shots                                                                 int
	targets                                                                     []target
	history                                                                     []historyFrame
	result                                                                      *model.MatchResult
}

func New(start model.MatchStart, rules Rules, rng model.RNG) *Engine {
	return &Engine{start: start, rules: rules, rng: rng, phase: "countdown", now: start.StartsAt - rules.CountdownMs, nextSpawn: start.StartsAt, lastShot: -1}
}
func (e *Engine) deadline() int64 { return e.start.StartsAt + e.rules.DurationMs }
func (e *Engine) Tick(now time.Time) []PublicEvent {
	at := now.UnixMilli()
	if e.result != nil || at < e.now {
		return nil
	}
	e.now = at
	cutoff := e.deadline()
	outcome := "finished"
	if e.disconnected > 0 && e.disconnected+15000 < cutoff {
		cutoff = e.disconnected + 15000
		outcome = "incomplete"
	}
	if e.leaveAt > 0 && e.leaveAt <= cutoff {
		cutoff = e.leaveAt
		outcome = "incomplete"
	}
	if at < e.start.StartsAt {
		if at >= cutoff {
			e.finish(max(cutoff, e.start.StartsAt), outcome)
			return []PublicEvent{{"duck.finished", e.Snapshot()}}
		}
		return nil
	}
	e.phase = "playing"
	until := min(at, cutoff)
	desired := max(int64(0), (until-e.start.StartsAt)*30/1000)
	if desired-e.frame > e.rules.MaxCatchUpSteps {
		skip := desired - e.frame - e.rules.MaxCatchUpSteps
		e.frame += skip
		e.dropped += skip
	}
	publish := false
	for e.frame < desired {
		e.frame++
		stepAt := e.start.StartsAt + e.frame*1000/30
		e.step(stepAt)
		if e.frame%2 == 0 {
			publish = true
		}
	}
	if at >= cutoff {
		e.finish(max(cutoff, e.start.StartsAt), outcome)
		return []PublicEvent{{"duck.finished", e.Snapshot()}}
	}
	if publish {
		return []PublicEvent{{"duck.snapshot", e.Snapshot()}}
	}
	return nil
}
func (e *Engine) step(at int64) {
	for e.nextSpawn <= at && e.nextSpawn < e.deadline() {
		e.spawnID++
		t := spawn(e.rng, e.spawnID, e.nextSpawn, e.rules)
		if e.nextSpawn+e.rules.LifetimeMs+e.rules.MaxRewindMs >= at {
			e.targets = append(e.targets, t)
		}
		e.nextSpawn += e.rules.SpawnMs
	}
	kept := e.targets[:0]
	for _, t := range e.targets {
		if t.Born+e.rules.LifetimeMs+e.rules.MaxRewindMs >= at {
			kept = append(kept, t)
		}
	}
	e.targets = kept
	e.history = append(e.history, historyFrame{at, append([]target{}, e.targets...)})
	if len(e.history) > 8 {
		e.history = append([]historyFrame{}, e.history[len(e.history)-8:]...)
	}
}
func (e *Engine) visible(at int64) []PublicTarget {
	out := []PublicTarget{}
	for _, t := range e.targets {
		if !t.Dead && t.Born <= at && at < t.Born+e.rules.LifetimeMs {
			out = append(out, t.public(at, e.rules))
		}
	}
	return out
}
func (e *Engine) Shoot(actorID string, seq uint64, aim Aim, sample InputTiming) (ShotResult, error) {
	r := ShotResult{}
	at := sample.ReceivedAt.UnixMilli()
	if len(e.start.Actors) != 1 || e.start.Actors[0].ActorID != actorID || seq <= e.seq || e.result != nil || at < e.start.StartsAt || at >= e.deadline() || at < e.now || e.disconnected > 0 || e.leaveAt > 0 || math.IsNaN(aim.X) || math.IsNaN(aim.Y) || math.IsInf(aim.X, 0) || math.IsInf(aim.Y, 0) || aim.X < 0 || aim.X > 1 || aim.Y < 0 || aim.Y > 1 || sample.RewindMs < 0 || sample.RewindMs > e.rules.MaxRewindMs {
		return r, ErrInput
	}
	if e.lastShot >= 0 && at-e.lastShot < e.rules.CooldownMs {
		return r, ErrInput
	}
	e.Tick(sample.ReceivedAt)
	e.step(at)
	e.seq = seq
	e.lastShot = at
	e.shots++
	viewAt := max(e.start.StartsAt, at-sample.RewindMs)
	// Use the newest retained simulation snapshot preceding server-selected
	// display time. Analytic velocity advances that snapshot only to the bounded
	// display instant; dead IDs are checked against current authoritative state.
	historical := e.targets
	if sample.RewindMs > 0 {
		found := false
		for i := len(e.history) - 1; i >= 0; i-- {
			if e.history[i].at <= viewAt {
				historical = e.history[i].targets
				found = true
				break
			}
		}
		if !found {
			viewAt = at
			historical = e.targets
		}
	}
	best := -1
	for _, t := range historical {
		i := -1
		for n, live := range e.targets {
			if live.ID == t.ID && !live.Dead {
				i = n
				break
			}
		}
		if i < 0 {
			continue
		}
		if t.Dead || viewAt < t.Born || viewAt >= t.Born+e.rules.LifetimeMs {
			continue
		}
		p := t.public(viewAt, e.rules)
		if math.Hypot(p.X-aim.X, p.Y-aim.Y) <= p.Radius && (best < 0 || t.ID > e.targets[best].ID) {
			best = i
		}
	}
	r = ShotResult{MatchID: e.start.MatchID, Seq: seq, RewindMs: at - viewAt}
	if best >= 0 {
		e.targets[best].Dead = true
		r.TargetID = e.targets[best].ID
		r.Points = e.rules.Points[e.targets[best].Kind]
		e.score += r.Points
		e.hits++
	}
	r.Score = e.score
	r.Hits = e.hits
	r.Shots = e.shots
	return r, nil
}
func (e *Engine) Presence(user, kind string, at time.Time) {
	if len(e.start.Actors) != 1 || e.start.Actors[0].UserID != user || e.result != nil {
		return
	}
	ms := at.UnixMilli()
	switch kind {
	case "disconnect":
		if e.disconnected == 0 {
			e.disconnected = ms
		}
	case "resume":
		e.Tick(at)
		if e.result == nil {
			e.disconnected = 0
		}
	case "leave":
		e.leaveAt = ms
	case "expired": /* Tick evaluates original grace */
	}
}
func (e *Engine) finish(at int64, outcome string) {
	e.finishedAt = at
	e.phase = "finished"
	score := e.score
	e.result = &model.MatchResult{Version: e.start.Version, MatchID: e.start.MatchID, WorldID: e.start.WorldID, ArenaID: e.start.ArenaID, GameType: "duck", PreparedAt: e.start.PreparedAt, StartsAt: e.start.StartsAt, FinishedAt: at, Participants: []model.ParticipantResult{{Actor: e.start.Actors[0], Outcome: outcome, Score: &score, Duck: &model.DuckResult{Hits: e.hits, Shots: e.shots}}}}
}
func (e *Engine) Result() (model.MatchResult, bool) {
	if e.result == nil {
		return model.MatchResult{}, false
	}
	r := *e.result
	r.Participants = append([]model.ParticipantResult{}, r.Participants...)
	p := r.Participants[0]
	score := *p.Score
	detail := *p.Duck
	r.Participants[0].Score = &score
	r.Participants[0].Duck = &detail
	return r, true
}
func (e *Engine) Snapshot() PublicSnapshot {
	s := PublicSnapshot{MatchID: e.start.MatchID, WorldID: e.start.WorldID, Actor: e.start.Actors[0], Phase: e.phase, StartsAt: e.start.StartsAt, DeadlineAt: e.deadline(), ServerTimeMs: e.now, Frame: e.frame, DroppedSteps: e.dropped, Targets: e.visible(e.now), Score: e.score, Hits: e.hits, Shots: e.shots}
	if r, ok := e.Result(); ok {
		s.Result = &r
	}
	return s
}
