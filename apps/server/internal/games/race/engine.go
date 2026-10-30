package race

import (
	"errors"
	"patra-world-social/server/internal/games/model"
	"sort"
	"time"
)

var ErrInput = errors.New("invalid race input or inactive seat")

type Controls struct {
	Throttle int `json:"throttle"`
	Brake    int `json:"brake"`
	Steer    int `json:"steer"`
}
type PublicKart struct {
	model.Actor
	StartSlot      int     `json:"startSlot"`
	X              int     `json:"x"`
	Y              int     `json:"y"`
	Heading        int     `json:"heading"`
	Speed          int     `json:"speed"`
	Laps           int     `json:"laps"`
	NextCheckpoint int     `json:"nextCheckpoint"`
	Position       int     `json:"position"`
	BotAssisted    bool    `json:"botAssisted"`
	DNF            bool    `json:"dnf"`
	Finished       bool    `json:"finished"`
	ElapsedMs      *int64  `json:"elapsedMs,omitempty"`
	LapTimesMs     []int64 `json:"lapTimesMs"`
	BestLapMs      *int64  `json:"bestLapMs,omitempty"`
}
type PublicSnapshot struct {
	MatchID       string             `json:"matchId"`
	WorldID       string             `json:"worldId"`
	TrackID       string             `json:"trackId"`
	TrackChecksum string             `json:"trackChecksum"`
	Phase         string             `json:"phase"`
	StartsAt      int64              `json:"startsAt"`
	ServerTimeMs  int64              `json:"serverTimeMs"`
	DeadlineAt    int64              `json:"deadlineAt"`
	FinishedAt    int64              `json:"finishedAt,omitempty"`
	Frame         int64              `json:"frame"`
	DroppedSteps  int64              `json:"droppedSteps"`
	Karts         []PublicKart       `json:"karts"`
	Result        *model.MatchResult `json:"result,omitempty"`
}
type PublicEvent struct {
	Type     string
	Snapshot PublicSnapshot
}
type kart struct {
	PublicKart
	controls                Controls
	seq                     uint64
	inputAt                 int64
	beforeDisconnect        Controls
	beforeDisconnectInputAt int64
	lapStarted              int64
	disconnectedAt          int64
	departed                bool
	progress                int
	remainderX              int
	remainderY              int
}
type Engine struct {
	start      model.MatchStart
	track      Track
	rules      Rules
	karts      []*kart
	frame      int64
	now        int64
	dropped    int64
	deadline   int64
	phase      string
	finishedAt int64
}

func New(start model.MatchStart, track Track, rules Rules) *Engine {
	e := &Engine{start: start, track: track, rules: rules, now: start.StartsAt - 3000, deadline: start.StartsAt + rules.LimitMs, phase: "countdown"}
	for i, a := range start.Actors {
		p := track.Grid[i]
		e.karts = append(e.karts, &kart{PublicKart: PublicKart{Actor: a, StartSlot: i, X: p.X, Y: p.Y, LapTimesMs: []int64{}}, lapStarted: start.StartsAt})
	}
	e.rank()
	return e
}
func (e *Engine) Input(actor string, seq uint64, input Controls) error {
	return e.InputAt(actor, seq, input, time.UnixMilli(e.now))
}
func (e *Engine) InputAt(actor string, seq uint64, input Controls, at time.Time) error {
	if input.Throttle < 0 || input.Throttle > 1 || input.Brake < 0 || input.Brake > 1 || input.Steer < -1 || input.Steer > 1 || e.phase == "finished" || e.phase == "abandoned" {
		return ErrInput
	}
	for _, k := range e.karts {
		if k.ActorID == actor && k.ActorType == "human" && !k.departed && k.disconnectedAt == 0 && !k.Finished {
			if seq <= k.seq {
				return ErrInput
			}
			k.seq = seq
			k.controls = input
			k.inputAt = at.UnixMilli()
			return nil
		}
	}
	return ErrInput
}
func (e *Engine) Tick(now time.Time) []PublicEvent {
	if e.phase == "finished" || e.phase == "abandoned" {
		return nil
	}
	at := now.UnixMilli()
	if at < e.now {
		return nil
	}
	e.now = at
	if at < e.start.StartsAt {
		e.recordAssistance(at)
		return nil
	}
	e.phase = "racing"
	target := (min64(at, e.deadline) - e.start.StartsAt) * 30 / 1000
	if target-e.frame > e.rules.MaxCatchUpSteps {
		skip := target - e.frame - e.rules.MaxCatchUpSteps
		e.frame += skip
		e.dropped += skip
	}
	publish := false
	for e.frame < target {
		e.frame++
		stepAt := e.start.StartsAt + e.frame*1000/30
		for _, k := range e.karts {
			if !k.departed && k.disconnectedAt > 0 && stepAt > k.disconnectedAt+15000 {
				e.depart(k)
			}
			if !k.Finished && !k.DNF {
				input, inputAt := k.controls, k.inputAt
				takeover := k.disconnectedAt > 0 && k.disconnectedAt < e.deadline && stepAt >= k.disconnectedAt
				if k.disconnectedAt > 0 && !takeover {
					// Metadata can arrive before catch-up. Retain the old intent
					// before takeover, including a final frame whose natural
					// cutoff wins over a simultaneous disconnect.
					input, inputAt = k.beforeDisconnect, k.beforeDisconnectInputAt
				}
				if k.ActorType == "bot" || takeover {
					input = e.bot(k)
				} else if stepAt-inputAt >= e.rules.IntentExpiryMs || stepAt < inputAt {
					input = Controls{}
				}
				e.move(k, input, stepAt)
			}
		}
		e.abandon(stepAt)
		if e.phase == "abandoned" {
			e.recordAssistance(stepAt)
			return nil
		}
		e.rank()
		if e.frame%2 == 0 {
			publish = true
		}
		all := true
		for _, k := range e.karts {
			if !k.Finished && !k.DNF {
				all = false
			}
		}
		if all {
			e.finish(stepAt)
			break
		}
		if stepAt >= e.deadline {
			e.finish(e.deadline)
			break
		}
	}
	// Absolute limits bound the arena even when the owner was suspended. Skipped
	// steps never invent movement, checkpoints or laps.
	if e.phase != "finished" && at >= e.deadline {
		e.finish(e.deadline)
	}
	if e.phase != "finished" {
		for _, k := range e.karts {
			if !k.departed && k.disconnectedAt > 0 && at >= k.disconnectedAt+15000 {
				e.depart(k)
			}
		}
		e.abandon(at)
		e.recordAssistance(at)
	}
	if e.phase == "finished" {
		return []PublicEvent{{"race.finished", e.Snapshot("")}}
	}
	if publish {
		return []PublicEvent{{"race.snapshot", e.Snapshot("")}}
	}
	return nil
}
func (e *Engine) move(k *kart, c Controls, at int64) {
	if c.Brake > 0 {
		k.Speed = max(0, k.Speed-e.rules.Brake)
	} else if c.Throttle > 0 {
		k.Speed = min(e.rules.MaxSpeed, k.Speed+e.rules.Acceleration)
	} else {
		k.Speed = max(0, k.Speed-e.rules.Drag)
	}
	if k.Speed > 0 {
		k.Heading = (k.Heading + c.Steer*e.rules.SteerDegrees + 360) % 360
	}
	d := e.track.Directions[k.Heading]
	old := Point{k.X, k.Y}
	nx, ny := k.Speed*d.X+k.remainderX, k.Speed*d.Y+k.remainderY
	x, y := k.X+nx/1024, k.Y+ny/1024
	k.remainderX, k.remainderY = nx%1024, ny%1024
	if e.track.Driveable(x, y, e.rules.Radius) {
		k.X = x
		k.Y = y
	} else { // axis sliding allows recovery from a wall with ordinary steering
		if e.track.Driveable(x, k.Y, e.rules.Radius) {
			k.X = x
		}
		if e.track.Driveable(k.X, y, e.rules.Radius) {
			k.Y = y
		}
		k.Speed /= 2
		k.remainderX, k.remainderY = 0, 0
	}
	e.cross(k, old, at)
	k.progress = e.progress(k)
}
func (e *Engine) cross(k *kart, old Point, at int64) {
	if k.Finished || k.Laps >= 5 {
		return
	}
	if crossed(e.track.Gates[k.NextCheckpoint], old, Point{k.X, k.Y}) {
		k.NextCheckpoint++
		if k.NextCheckpoint == len(e.track.Gates) {
			k.NextCheckpoint = 0
			k.Laps++
			lap := at - k.lapStarted
			k.lapStarted = at
			k.LapTimesMs = append(k.LapTimesMs, lap)
			if k.BestLapMs == nil || lap < *k.BestLapMs {
				v := lap
				k.BestLapMs = &v
			}
			if k.Laps == 5 {
				k.Finished = true
				k.Speed = 0
				elapsed := at - e.start.StartsAt
				k.ElapsedMs = &elapsed
				e.deadline = min64(e.deadline, at+e.rules.FinishGraceMs)
			}
		}
	}
}

// Assistance is history on the authoritative timeline, not arrival metadata.
// Call only after advancement determines whether a natural finish shortened it.
func (e *Engine) recordAssistance(through int64) {
	for _, k := range e.karts {
		if k.disconnectedAt > 0 && k.disconnectedAt <= through && (k.ElapsedMs == nil || k.disconnectedAt < e.start.StartsAt+*k.ElapsedMs) {
			k.BotAssisted = true
		}
	}
}
func (e *Engine) finish(at int64) {
	// Natural finish wins over a disconnect at the exact global boundary.
	// An earlier genuine takeover remains recorded even after a later resume.
	e.recordAssistance(at - 1)
	e.phase = "finished"
	e.finishedAt = at
	for _, k := range e.karts {
		if !k.Finished {
			k.DNF = true
		}
		k.Speed = 0
	}
	e.rank()
}
func (e *Engine) depart(k *kart) {
	k.departed = true
	k.controls = Controls{}
	if !k.Finished {
		k.DNF = true
		k.Speed = 0
	}
}
func (e *Engine) abandon(at int64) {
	for _, k := range e.karts {
		if k.ActorType == "human" && !k.departed {
			return
		}
	}
	e.phase = "abandoned"
	e.finishedAt = at
}
func (e *Engine) Presence(user, kind string, at time.Time) {
	if e.phase == "finished" || e.phase == "abandoned" {
		return
	}
	if kind == "disconnect" {
		// The owner installs the complete batch in unspecified map order before
		// advancing time. This branch must only install metadata, including
		// during activation replay; no physics or terminal decision belongs here.
		for _, k := range e.karts {
			if k.UserID == user && k.ActorType == "human" && !k.departed && k.disconnectedAt == 0 {
				k.beforeDisconnect, k.beforeDisconnectInputAt = k.controls, k.inputAt
				k.controls = Controls{}
				k.disconnectedAt = at.UnixMilli()
			}
		}
		return
	}
	// Ordered resume/departure follows complete metadata installation. Advance
	// first so a natural finish wins at the exact deadline.
	e.Tick(at)
	if e.phase == "finished" || e.phase == "abandoned" {
		return
	}
	for _, k := range e.karts {
		if k.UserID != user || k.ActorType != "human" || k.departed {
			continue
		}
		switch kind {
		case "resume":
			if k.disconnectedAt > 0 && at.UnixMilli() < k.disconnectedAt+15000 {
				k.disconnectedAt = 0
				k.beforeDisconnect = Controls{}
				k.beforeDisconnectInputAt = 0
				k.seq = 0
				k.controls = Controls{}
			}
		case "leave", "expired":
			e.depart(k)
		}
	}
	e.abandon(at.UnixMilli())
}
func (e *Engine) rank() {
	ordered := append([]*kart{}, e.karts...)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.Finished != b.Finished {
			return a.Finished
		}
		if a.Finished {
			if *a.ElapsedMs != *b.ElapsedMs {
				return *a.ElapsedMs < *b.ElapsedMs
			}
			return a.StartSlot < b.StartSlot
		}
		if a.Laps != b.Laps {
			return a.Laps > b.Laps
		}
		if a.NextCheckpoint != b.NextCheckpoint {
			return a.NextCheckpoint > b.NextCheckpoint
		}
		if a.progress != b.progress {
			return a.progress > b.progress
		}
		return a.StartSlot < b.StartSlot
	})
	for i, k := range ordered {
		k.Position = i + 1
	}
}
func (e *Engine) progress(k *kart) int {
	switch k.NextCheckpoint {
	case 0:
		return k.X + k.Y
	case 1:
		return k.Y - k.X
	case 2:
		return -k.X - k.Y
	default:
		return k.X - k.Y
	}
}
func (e *Engine) Snapshot(viewer string) PublicSnapshot {
	s := PublicSnapshot{MatchID: e.start.MatchID, WorldID: e.start.WorldID, TrackID: e.track.ID, TrackChecksum: e.track.Checksum, Phase: e.phase, FinishedAt: e.finishedAt, StartsAt: e.start.StartsAt, ServerTimeMs: e.now, DeadlineAt: e.deadline, Frame: e.frame, DroppedSteps: e.dropped, Karts: []PublicKart{}}
	for _, k := range e.karts {
		v := k.PublicKart
		v.LapTimesMs = append([]int64{}, k.LapTimesMs...)
		v.BestLapMs = copyInt(k.BestLapMs)
		v.ElapsedMs = copyInt(k.ElapsedMs)
		s.Karts = append(s.Karts, v)
	}
	if r, ok := e.Result(); ok {
		s.Result = &r
	}
	return s
}
func (e *Engine) Result() (model.MatchResult, bool) {
	if e.phase != "finished" {
		return model.MatchResult{}, false
	}
	r := model.MatchResult{Version: e.start.Version, MatchID: e.start.MatchID, WorldID: e.start.WorldID, ArenaID: e.start.ArenaID, GameType: "race", PreparedAt: e.start.PreparedAt, StartsAt: e.start.StartsAt, FinishedAt: e.finishedAt}
	for _, k := range e.karts {
		p := k.Position
		out := "finished"
		if k.DNF {
			out = "dnf"
		}
		r.Participants = append(r.Participants, model.ParticipantResult{Actor: k.Actor, BotAssisted: k.BotAssisted, Outcome: out, Position: &p, Race: &model.RaceResult{Laps: k.Laps, LapTimesMs: append([]int64{}, k.LapTimesMs...), BestLapMs: copyInt(k.BestLapMs), ElapsedMs: copyInt(k.ElapsedMs), DNF: k.DNF}})
	}
	return r, true
}
func copyInt(v *int64) *int64 {
	if v == nil {
		return nil
	}
	n := *v
	return &n
}
func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
