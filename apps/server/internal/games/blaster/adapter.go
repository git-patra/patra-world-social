package blaster

import (
	"errors"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/lobby"
	"time"
)

type Adapter struct {
	start  model.MatchStart
	clock  model.Clock
	engine *Engine
	events []lobby.GameEvent
	last   int64
}

func Factory(clock model.Clock) lobby.Factory {
	return func(m model.MatchStart) (lobby.Engine, error) {
		if len(m.Actors) < 1 || len(m.Actors) > 4 {
			return nil, errors.New("blaster requires one to four humans")
		}
		for _, actor := range m.Actors {
			if actor.ActorType != "human" || actor.UserID == "" {
				return nil, errors.New("blaster requires humans")
			}
		}
		return &Adapter{start: m, clock: clock, last: -1}, nil
	}
}
func (a *Adapter) Start(at int64) { a.start.StartsAt = at; a.engine = New(a.start) }
func (a *Adapter) Tick(at time.Time) bool {
	if a.engine == nil {
		return false
	}
	a.engine.Tick(at)
	s := a.engine.Snapshot()
	if s.Frame != a.last && (a.last < 0 || s.Frame-a.last >= 2 || s.Result != nil) {
		a.last = s.Frame
		a.events = []lobby.GameEvent{{WorldID: a.start.WorldID, Type: "blaster.snapshot", Payload: s}}
	}
	return s.Result != nil
}
func (a *Adapter) Presence(user, kind string) { a.PresenceAt(user, kind, a.clock.Now()) }
func (a *Adapter) PresenceAt(user, kind string, at time.Time) {
	if a.engine != nil {
		a.engine.Presence(user, kind, at)
	}
}
func (a *Adapter) Close()             { a.engine = nil; a.events = nil }
func (a *Adapter) Snapshot() Snapshot { return a.engine.Snapshot() }
func (a *Adapter) Input(user string, seq uint64, controls Controls, at time.Time) error {
	if a.engine == nil {
		return ErrInput
	}
	return a.engine.Input(user, seq, controls, at)
}
func (a *Adapter) Result() (model.MatchResult, error) {
	if a.engine != nil {
		if r := a.engine.Snapshot().Result; r != nil {
			return *r, nil
		}
	}
	return model.MatchResult{}, errors.New("blaster result pending")
}
func (a *Adapter) DrainEvents() []lobby.GameEvent { out := a.events; a.events = nil; return out }
func (a *Adapter) HasEvents() bool                { return len(a.events) > 0 }
