package duck

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
	timing clockEstimator
	epoch  uint64
}

func Factory(clock model.Clock) lobby.Factory {
	return func(m model.MatchStart) (lobby.Engine, error) {
		if len(m.Actors) != 1 || m.Actors[0].ActorType != "human" {
			return nil, errors.New("duck requires one human")
		}
		return &Adapter{start: m, clock: clock}, nil
	}
}
func (a *Adapter) Start(at int64) {
	a.start.StartsAt = at
	a.engine = New(a.start, DefaultRules(), NewRNG(a.start.Seed))
}
func (a *Adapter) Tick(at time.Time) bool {
	if a.engine == nil {
		return false
	}
	for _, e := range a.engine.Tick(at) {
		a.events = []lobby.GameEvent{{WorldID: a.start.WorldID, Type: e.Type, Payload: e.Snapshot}}
	}
	_, ok := a.engine.Result()
	if ok {
		a.timing = clockEstimator{}
	}
	return ok
}
func (a *Adapter) Presence(user, kind string) { a.PresenceAt(user, kind, a.clock.Now()) }
func (a *Adapter) PresenceAt(user, kind string, at time.Time) {
	if a.engine == nil {
		return
	}
	before := a.engine.phase
	a.engine.Presence(user, kind, at)
	if before != "finished" && a.engine.phase == "finished" {
		a.events = []lobby.GameEvent{{WorldID: a.start.WorldID, Type: "duck.finished", Payload: a.engine.Snapshot()}}
		a.timing = clockEstimator{}
	}
	if kind == "disconnect" || kind == "leave" || kind == "expired" {
		a.timing = clockEstimator{}
	}
}
func (a *Adapter) Close()                         { a.engine = nil; a.events = nil; a.timing = clockEstimator{} }
func (a *Adapter) DrainEvents() []lobby.GameEvent { v := a.events; a.events = nil; return v }
func (a *Adapter) HasEvents() bool                { return len(a.events) > 0 }
func (a *Adapter) Snapshot() PublicSnapshot       { return a.engine.Snapshot() }
func (a *Adapter) Result() (model.MatchResult, error) {
	if a.engine != nil {
		if r, ok := a.engine.Result(); ok {
			return r, nil
		}
	}
	return model.MatchResult{}, errors.New("duck result pending")
}
func (a *Adapter) bind(user string, epoch uint64) error {
	if a.engine == nil || a.engine.result != nil || a.start.Actors[0].UserID != user {
		return ErrInput
	}
	if a.epoch != epoch {
		a.epoch = epoch
		a.engine.seq = 0
		a.timing = clockEstimator{epoch: epoch}
	}
	return nil
}
func (a *Adapter) Challenge(user string, epoch uint64, at time.Time) (ClockChallenge, error) {
	if err := a.bind(user, epoch); err != nil {
		return ClockChallenge{}, err
	}
	return a.timing.challenge(epoch, at.UnixMilli()), nil
}
func (a *Adapter) Observe(user string, epoch uint64, nonce string, client float64, at time.Time) error {
	if err := a.bind(user, epoch); err != nil {
		return err
	}
	return a.timing.observe(epoch, nonce, client, at.UnixMilli())
}
func (a *Adapter) Shoot(user string, epoch, seq uint64, aim Aim, client float64, at time.Time) (ShotResult, error) {
	if err := a.bind(user, epoch); err != nil {
		return ShotResult{}, err
	}
	rewind, err := a.timing.rewind(epoch, client, at.UnixMilli())
	if err != nil {
		return ShotResult{}, err
	}
	return a.engine.Shoot(a.start.Actors[0].ActorID, seq, aim, InputTiming{ReceivedAt: at, ClientTimeMs: client, RewindMs: rewind})
}
func Project(ctx context.Context, tx pgx.Tx, r model.MatchResult) error {
	if len(r.Participants) != 1 || r.RulesID != "duck-v1" || r.RulesVersion != 1 || r.RulesChecksum != RulesChecksum() {
		return errors.New("invalid duck rules/roster")
	}
	p := r.Participants[0]
	if p.Duck == nil || p.Score == nil || p.ActorType != "human" || p.Duck.Hits < 0 || p.Duck.Shots < p.Duck.Hits || p.Duck.Shots > 240 || *p.Score < int64(p.Duck.Hits)*100 || *p.Score > int64(p.Duck.Hits)*500 || p.Outcome != "finished" && p.Outcome != "incomplete" || p.Outcome == "finished" && r.FinishedAt-r.StartsAt != 60000 {
		return errors.New("invalid duck detail")
	}
	_, err := tx.Exec(ctx, `INSERT INTO duck_results(match_id,actor_id,score,hits,shots,complete)VALUES($1,$2,$3,$4,$5,$6)`, r.MatchID, p.ActorID, *p.Score, p.Duck.Hits, p.Duck.Shots, p.Outcome == "finished")
	return err
}

func (a *Adapter) Calibration(at time.Time) (ClockCalibration, bool) {
	return a.timing.calibration(at.UnixMilli())
}
