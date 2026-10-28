package race

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/lobby"
	"time"
)

type Adapter struct {
	start  model.MatchStart
	clock  model.Clock
	engine *Engine
	events []lobby.GameEvent
}

func Factory(clock model.Clock) lobby.Factory {
	return func(m model.MatchStart) (lobby.Engine, error) {
		if len(m.Actors) != 5 {
			return nil, errors.New("race needs five slots")
		}
		track, rules := DefaultTrack(), DefaultRules()
		if rules.TrackID != track.ID || rules.TrackChecksum != track.Checksum {
			return nil, errors.New("race track/rules mismatch")
		}
		return &Adapter{start: m, clock: clock}, nil
	}
}
func (a *Adapter) Start(at int64) {
	a.start.StartsAt = at
	a.engine = New(a.start, DefaultTrack(), DefaultRules())
}
func (a *Adapter) Tick(at time.Time) bool {
	if a.engine == nil {
		return false
	}
	events := a.engine.Tick(at)
	for _, v := range events { // Coalesce race frames; terminal results are recoverable from durable history.
		a.events = []lobby.GameEvent{{WorldID: a.start.WorldID, Type: v.Type, Payload: v.Snapshot}}
	}
	_, ok := a.engine.Result()
	return ok
}
func (a *Adapter) Presence(user, kind string) { a.PresenceAt(user, kind, a.clock.Now()) }
func (a *Adapter) PresenceAt(user, kind string, at time.Time) {
	if a.engine != nil {
		before := a.engine.phase
		a.engine.Presence(user, kind, at)
		if before != "finished" && a.engine.phase == "finished" {
			a.events = []lobby.GameEvent{{WorldID: a.start.WorldID, Type: "race.finished", Payload: a.engine.Snapshot("")}}
		}
	}
}
func (a *Adapter) Close()                         { a.engine = nil; a.events = nil }
func (a *Adapter) DrainEvents() []lobby.GameEvent { events := a.events; a.events = nil; return events }
func (a *Adapter) HasEvents() bool                { return len(a.events) > 0 }
func (a *Adapter) Snapshot() PublicSnapshot       { return a.engine.Snapshot("") }
func (a *Adapter) Input(user string, seq uint64, c Controls, at time.Time) error {
	if a.engine == nil {
		return ErrInput
	}
	for _, actor := range a.start.Actors {
		if actor.UserID == user && actor.ActorType == "human" {
			return a.engine.InputAt(actor.ActorID, seq, c, at)
		}
	}
	return ErrInput
}
func (a *Adapter) Result() (model.MatchResult, error) {
	if a.engine != nil {
		if r, ok := a.engine.Result(); ok {
			return r, nil
		}
	}
	return model.MatchResult{}, errors.New("race result pending")
}
func Project(ctx context.Context, tx pgx.Tx, r model.MatchResult) error {
	if len(r.Participants) != 5 {
		return errors.New("invalid race roster")
	}
	for slot, p := range r.Participants {
		if p.Race == nil || p.Position == nil {
			return errors.New("missing race detail")
		}
		v := p.Race
		if len(v.LapTimesMs) != v.Laps || (!v.DNF && (v.Laps != 5 || v.ElapsedMs == nil)) {
			return errors.New("invalid race laps")
		}
		_, err := tx.Exec(ctx, `INSERT INTO race_results(match_id,actor_id,start_slot,position,laps,lap_times_ms,best_lap_ms,elapsed_ms,dnf,bot_assisted) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, r.MatchID, p.ActorID, slot, *p.Position, v.Laps, append([]int64{}, v.LapTimesMs...), v.BestLapMs, v.ElapsedMs, v.DNF, p.BotAssisted)
		if err != nil {
			return err
		}
	}
	return nil
}
