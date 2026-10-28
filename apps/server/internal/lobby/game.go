package lobby

import (
	"context"
	"time"
)

type GameEvent struct {
	WorldID, Type string
	Payload       any
}
type EventEngine interface {
	DrainEvents() []GameEvent
	HasEvents() bool
}

// ReadEngine is for already-admitted world members, including spectators. It
// binds the requested match to that world; callbacks must return public data.
func (s *Service) ReadEngine(ctx context.Context, world, match string, fn func(Engine) error) error {
	result := ErrForbidden
	err := s.call(ctx, func() {
		for _, l := range s.entries {
			if l.WorldID == world && l.MatchID == match && l.engine != nil {
				result = fn(l.engine)
				return
			}
		}
	})
	if err != nil {
		return err
	}
	return result
}

type ResultReturn struct{ Ready bool }

// ResultAction reserves one release only after the durable result transition.
func (s *Service) ResultAction(ctx context.Context, world, match, user string, rematch bool) (ResultReturn, error) {
	var out ResultReturn
	result := ErrForbidden
	err := s.call(ctx, func() {
		for _, l := range s.entries {
			if l.WorldID != world || l.MatchID != match || l.State != "results" || l.releasing {
				continue
			}
			allowed := false
			for _, p := range l.Players {
				if p.UserID == user && !p.departed && p.Connected {
					allowed = true
				}
			}
			if !allowed {
				return
			}
			if rematch && l.GameType != "penalty" {
				return
			}
			result = nil
			if l.GameType == "race" {
				if l.rematch == nil {
					l.rematch = map[string]bool{}
				}
				l.rematch[user] = true
				for _, p := range l.Players {
					if !p.departed && !l.rematch[p.UserID] {
						return
					}
				}
			}
			if rematch {
				if l.rematch == nil {
					l.rematch = map[string]bool{}
				}
				l.rematch[user] = true
				for _, p := range l.Players {
					if p.departed || !p.Connected || !l.rematch[p.UserID] {
						return
					}
				}
			}
			l.releasing = true
			l.resultAction = true
			l.rematchApproved = rematch
			out.Ready = true
			return
		}
	})
	if err != nil {
		return out, err
	}
	return out, result
}
func (s *Service) retryResultAction(match string) {
	s.control(func() {
		for _, l := range s.entries {
			if l.MatchID == match && l.State == "results" {
				l.releasing = false
				l.resultAction = false
				l.rematchApproved = false
			}
		}
	})
}

// InputEngine stamps receipt on owner ingestion, after its bounded queue. A raw
// socket arrival is never proof of acceptance before the game deadline.
func (s *Service) InputEngine(ctx context.Context, world, match, user string, fn func(Engine, time.Time) error) error {
	result := ErrForbidden
	err := s.call(ctx, func() {
		for _, l := range s.entries {
			if l.WorldID != world || l.MatchID != match || l.engine == nil || l.State != "running" {
				continue
			}
			for _, p := range l.Players {
				if p.UserID == user && !p.departed && p.Connected {
					if err := ctx.Err(); err != nil {
						result = err
						return
					}
					at := s.clock.Now()
					result = fn(l.engine, at)
					return
				}
			}
		}
	})
	if err != nil {
		return err
	}
	return result
}

// CompleteResultAction holds all in-memory arena/account claims through the
// guarded durable release, then replaces the consented pair on the owner once.
func (s *Service) CompleteResultAction(match string) error {
	var reserved *entry
	if err := s.control(func() {
		for _, l := range s.entries {
			if l.MatchID == match && l.State == "results" && l.resultAction && l.releasing {
				reserved = l
				return
			}
		}
	}); err != nil {
		return err
	}
	if reserved == nil {
		return ErrForbidden
	}
	ctx, cancel := context.WithTimeout(s.ctx, 3*time.Second)
	defer cancel()
	if err := s.store.Terminal(ctx, match, "completed"); err != nil {
		s.retryResultAction(match)
		return err
	}
	var result error
	err := s.control(func() {
		current := s.entries[[2]string{reserved.WorldID, reserved.Arena.ID}]
		if current != reserved || !current.resultAction || !current.releasing {
			result = ErrForbidden
			return
		}
		if !current.rematchApproved {
			s.detach(match)
			return
		}
		for _, p := range current.Players {
			if p.departed || !p.Connected {
				result = ErrForbidden
				s.detach(match)
				return
			}
		}
		next := &entry{Lobby: Lobby{Arena: current.Arena, ID: id(), WorldID: current.WorldID, HostID: current.HostID, State: "lobby", Players: []Player{}, Spectators: []string{}, EngineAvailable: s.factories[current.GameType] != nil, BotChosen: current.BotChosen}}
		for _, p := range current.Players {
			actor := p.Actor
			actor.ActorID = id()
			next.Players = append(next.Players, Player{Actor: actor, Connected: true})
		}
		// Busy never becomes empty; new generations fence already-queued returns.
		for _, p := range next.Players {
			s.claim(p.UserID, next.ID)
		}
		if current.engine != nil {
			current.engine.Close()
		}
		if current.start != nil {
			delete(s.attempts, [2]string{current.start.host, current.start.command})
		}
		s.entries[[2]string{current.WorldID, current.Arena.ID}] = next
		s.mark(current.WorldID)
	})
	if err != nil {
		return err
	}
	return result
}
