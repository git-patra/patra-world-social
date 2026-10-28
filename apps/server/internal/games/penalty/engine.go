// Package penalty owns the secret choices. Only PublicSnapshot leaves the owner.
package penalty

import (
	"errors"
	"fmt"
	"patra-world-social/server/internal/games/model"
	"time"
)

const ChoiceMs int64 = 3000
const AnimationMs int64 = 1500

var ErrChoice = errors.New("choice closed, locked, stale, or unauthorized")

type PublicSnapshot struct {
	MatchID          string                    `json:"matchId"`
	Phase            string                    `json:"phase"`
	TurnID           string                    `json:"turnId"`
	Turn             int                       `json:"turn"`
	KickerID         string                    `json:"kickerId"`
	KeeperID         string                    `json:"keeperId"`
	ChoiceOpensAt    int64                     `json:"choiceOpensAt"`
	ChoiceDeadlineAt int64                     `json:"choiceDeadlineAt"`
	AnimationEndsAt  int64                     `json:"animationEndsAt"`
	Actors           []model.Actor             `json:"actors"`
	Scores           [2]int64                  `json:"scores"`
	Chosen           [2]bool                   `json:"chosen"`
	Turns            []model.PenaltyTurnResult `json:"turns"`
	Result           *model.MatchResult        `json:"result,omitempty"`
}
type PublicEvent struct {
	Type     string
	Snapshot PublicSnapshot
}
type Engine struct {
	start        model.MatchStart
	clock        model.Clock
	rng          model.RNG
	first, turn  int
	opens        int64
	choices      [2]int
	chosen       [2]bool
	automatic    [2]bool
	scores       [2]int64
	kicks        [2]int
	turns        []model.PenaltyTurnResult
	disconnected [2]int64
	result       *model.MatchResult
	revealed     bool
	events       []PublicEvent
}

func New(start model.MatchStart, clock model.Clock, rng model.RNG) *Engine {
	start.Actors = append([]model.Actor(nil), start.Actors...)
	e := &Engine{start: start, clock: clock, rng: rng, first: rng.Intn(2), turn: 1, opens: start.StartsAt, turns: []model.PenaltyTurnResult{}}
	e.open()
	return e
}
func (e *Engine) turnID() string { return fmt.Sprintf("%s:%d", e.start.MatchID, e.turn) }
func (e *Engine) kicker() int    { return (e.first + e.turn - 1) % 2 }
func (e *Engine) open() {
	e.chosen = [2]bool{}
	e.automatic = [2]bool{}
	e.revealed = false
	for i, a := range e.start.Actors {
		if a.ActorType == "bot" {
			e.choices[i] = BotChoice(e.rng)
			e.chosen[i] = true
		}
	}
	e.emit("penalty.turn_opened")
}
func (e *Engine) Choose(actorID, turnID string, zone int, receivedAt time.Time) error {
	at := receivedAt.UnixMilli()
	if e.result != nil || e.revealed || turnID != e.turnID() || zone < 0 || zone > 3 || at < e.opens || at >= e.opens+ChoiceMs {
		return ErrChoice
	}
	for i, a := range e.start.Actors {
		if a.ActorID == actorID && a.ActorType == "human" && !e.chosen[i] && e.disconnected[i] == 0 {
			e.choices[i] = zone
			e.chosen[i] = true
			e.emit("penalty.choice_ack")
			return nil
		}
	}
	return ErrChoice
}
func (e *Engine) Tick(now time.Time) []PublicEvent {
	limit := now.UnixMilli()
	cutoff := int64(0)
	for _, at := range e.disconnected {
		if at > 0 && (cutoff == 0 || at+15000 < cutoff) {
			cutoff = at + 15000
		}
	}
	if cutoff > 0 && limit > cutoff {
		limit = cutoff
	}
	for e.result == nil {
		if !e.revealed {
			if limit < e.opens+ChoiceMs {
				break
			}
			for i := range e.choices {
				if !e.chosen[i] {
					e.choices[i] = e.rng.Intn(4)
					e.chosen[i] = true
					e.automatic[i] = true
				}
			}
			k := e.kicker()
			g := e.choices[k] != e.choices[1-k]
			if g {
				e.scores[k]++
			}
			e.kicks[k]++
			e.turns = append(e.turns, model.PenaltyTurnResult{Turn: e.turn, KickerID: e.start.Actors[k].ActorID, KeeperID: e.start.Actors[1-k].ActorID, KickZone: e.choices[k], KeepZone: e.choices[1-k], KickTimeout: e.automatic[k], KeepTimeout: e.automatic[1-k], Goal: g})
			e.revealed = true
			e.emit("penalty.revealed")
		}
		if limit < e.opens+ChoiceMs+AnimationMs {
			break
		}
		end := e.opens + ChoiceMs + AnimationMs
		if e.naturalEnd() {
			e.finish(end, -1, false)
			break
		}
		e.turn++
		e.opens = end
		e.open()
	}
	if e.result == nil && cutoff > 0 && now.UnixMilli() >= cutoff {
		loser := -1
		both := e.disconnected[0] > 0 && e.disconnected[0] <= cutoff && e.disconnected[1] > 0 && e.disconnected[1] <= cutoff
		for i, at := range e.disconnected {
			if at > 0 && at+15000 <= cutoff {
				loser = i
				break
			}
		}
		e.finish(cutoff, loser, both)
	}
	out := e.events
	e.events = nil
	return out
}
func (e *Engine) naturalEnd() bool {
	if e.kicks[0] <= 5 && e.kicks[1] <= 5 {
		if e.scores[0] > e.scores[1]+int64(5-e.kicks[1]) || e.scores[1] > e.scores[0]+int64(5-e.kicks[0]) {
			return true
		}
	}
	return e.kicks[0] >= 5 && e.kicks[0] == e.kicks[1] && (e.scores[0] != e.scores[1] || e.kicks[0] >= 10)
}
func (e *Engine) finish(at int64, loser int, abandoned bool) {
	if e.result != nil {
		return
	}
	if at < e.start.StartsAt {
		at = e.start.StartsAt
	}
	r := model.MatchResult{Version: e.start.Version, MatchID: e.start.MatchID, WorldID: e.start.WorldID, ArenaID: e.start.ArenaID, GameType: e.start.GameType, PreparedAt: e.start.PreparedAt, StartsAt: e.start.StartsAt, FinishedAt: at, PenaltyTurns: append([]model.PenaltyTurnResult{}, e.turns...)}
	for i, a := range e.start.Actors {
		score := e.scores[i]
		outcome := "draw"
		position := 1
		if abandoned {
			outcome = "abandoned"
		} else if loser >= 0 {
			if i == loser {
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
	e.emit("penalty.snapshot")
}

// Presence is supplied by the owner; disconnect never pauses the choice clock.
func (e *Engine) Presence(user, kind string, at time.Time) {
	// Disconnect metadata must be installed before any delayed catch-up.
	if kind == "disconnect" {
		for i, a := range e.start.Actors {
			if a.UserID == user && a.ActorType == "human" && (e.disconnected[i] == 0 || at.UnixMilli() < e.disconnected[i]) {
				e.disconnected[i] = at.UnixMilli()
			}
		}
		return
	}
	events := e.Tick(at)
	e.events = append(events, e.events...)
	if e.result != nil {
		return
	}
	for i, a := range e.start.Actors {
		if a.UserID == user && a.ActorType == "human" {
			switch kind {
			case "resume":
				e.disconnected[i] = 0
			case "leave":
				e.finish(at.UnixMilli(), i, false)
			}
		}
	}
}
func (e *Engine) Snapshot(viewerID string) PublicSnapshot {
	k := e.kicker()
	phase := "choosing"
	if e.revealed {
		phase = "animation"
	}
	if e.result != nil {
		phase = "results"
	}
	s := PublicSnapshot{MatchID: e.start.MatchID, Phase: phase, TurnID: e.turnID(), Turn: e.turn, KickerID: e.start.Actors[k].ActorID, KeeperID: e.start.Actors[1-k].ActorID, ChoiceOpensAt: e.opens, ChoiceDeadlineAt: e.opens + ChoiceMs, AnimationEndsAt: e.opens + ChoiceMs + AnimationMs, Actors: append([]model.Actor{}, e.start.Actors...), Scores: e.scores, Chosen: e.chosen, Turns: append([]model.PenaltyTurnResult{}, e.turns...)}
	if r, ok := e.Result(); ok {
		s.Result = &r
	}
	return s
}
func (e *Engine) Result() (model.MatchResult, bool) {
	if e.result == nil {
		return model.MatchResult{}, false
	}
	r := *e.result
	r.Participants = append([]model.ParticipantResult{}, r.Participants...)
	for i := range r.Participants {
		score, position := *r.Participants[i].Score, *r.Participants[i].Position
		r.Participants[i].Score = &score
		r.Participants[i].Position = &position
	}
	r.PenaltyTurns = append([]model.PenaltyTurnResult{}, r.PenaltyTurns...)
	return r, true
}
func (e *Engine) emit(kind string) { e.events = append(e.events, PublicEvent{kind, e.Snapshot("")}) }
