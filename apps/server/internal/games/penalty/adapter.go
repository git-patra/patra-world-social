package penalty

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"math/rand"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/lobby"
	"time"
)

// Adapter preserves the concrete engine API while keeping factory preparation inert.
type Adapter struct {
	start  model.MatchStart
	clock  model.Clock
	engine *Engine
	events []lobby.GameEvent
	seen   map[string]bool
}

func Factory(clock model.Clock) lobby.Factory {
	return func(m model.MatchStart) (lobby.Engine, error) {
		if len(m.Actors) != 2 {
			return nil, errors.New("penalty requires two actors")
		}
		return &Adapter{start: m, clock: clock}, nil
	}
}
func (a *Adapter) Start(at int64) {
	a.start.StartsAt = at
	a.engine = New(a.start, a.clock, rand.New(rand.NewSource(a.start.Seed)))
}
func (a *Adapter) Tick(at time.Time) bool {
	if a.engine == nil {
		return false
	}
	for _, e := range a.engine.Tick(at) {
		a.events = append(a.events, lobby.GameEvent{WorldID: a.start.WorldID, Type: e.Type, Payload: e.Snapshot})
	}
	_, ok := a.engine.Result()
	return ok
}
func (a *Adapter) Presence(user, kind string) {
	if a.engine != nil {
		a.PresenceAt(user, kind, a.clock.Now())
	}
}
func (a *Adapter) PresenceAt(user, kind string, at time.Time) {
	if a.engine != nil {
		a.engine.Presence(user, kind, at)
	}
}
func (a *Adapter) Close() { a.engine = nil; a.events = nil }
func (a *Adapter) Result() (model.MatchResult, error) {
	if a.engine != nil {
		if r, ok := a.engine.Result(); ok {
			return r, nil
		}
	}
	return model.MatchResult{}, errors.New("penalty result pending")
}
func (a *Adapter) DrainEvents() []lobby.GameEvent { events := a.events; a.events = nil; return events }
func (a *Adapter) HasEvents() bool                { return len(a.events) > 0 }
func (a *Adapter) Snapshot() PublicSnapshot       { return a.engine.Snapshot("") }
func (a *Adapter) Choose(user, command, turn string, zone int, at time.Time) error {
	key := user + ":" + command
	if a.seen == nil {
		a.seen = map[string]bool{}
	}
	if command == "" || a.seen[key] {
		return ErrChoice
	}
	if a.engine == nil {
		return ErrChoice
	}
	for _, actor := range a.start.Actors {
		if actor.ActorType == "human" && actor.UserID == user {
			// Only accepted immutable choices enter replay protection (at most 40).
			err := a.engine.Choose(actor.ActorID, turn, zone, at)
			if err == nil {
				a.seen[key] = true
			}
			return err
		}
	}
	return ErrChoice
}

// Project executes only inside the history finalization transaction.
func Project(ctx context.Context, tx pgx.Tx, r model.MatchResult) error {
	for _, t := range r.PenaltyTurns {
		_, err := tx.Exec(ctx, `INSERT INTO penalty_turns(match_id,turn,kicker_id,keeper_id,kick_zone,keep_zone,kick_timeout,keep_timeout,goal)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, r.MatchID, t.Turn, t.KickerID, t.KeeperID, t.KickZone, t.KeepZone, t.KickTimeout, t.KeepTimeout, t.Goal)
		if err != nil {
			return err
		}
	}
	return nil
}
