package lobby

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"log/slog"
	"patra-world-social/server/internal/games/model"
	"time"
)

func (s *Service) StartMatch(ctx context.Context, host, lobby, command string) (model.MatchStart, error) {
	var a *attempt
	var e error
	err := s.call(ctx, func() {
		l := s.find(lobby)
		if l == nil || l.HostID != host || command == "" {
			e = ErrForbidden
			return
		}
		key := [2]string{host, command}
		if old := s.attempts[key]; old != nil {
			if old.match.WorldID != l.WorldID || old.match.ArenaID != l.Arena.ID {
				e = ErrForbidden
			} else {
				if old.err == ErrUnconfirmed {
					a = s.retryAttempt(l, old)
				} else {
					a = old
				}
			}
			return
		}
		if s.storageBlocked {
			e = ErrUnconfirmed
			return
		}
		if s.factories[l.GameType] == nil {
			e = ErrUnavailable
			return
		}
		if l.State != "lobby" {
			e = ErrBusy
			return
		}
		if len(l.Players) == 0 {
			e = ErrNotReady
			return
		}
		for _, p := range l.Players {
			if !p.Ready || !p.Connected {
				e = ErrNotReady
				return
			}
		}
		if l.GameType == "penalty" && len(l.Players) < 2 && !l.BotChosen {
			e = ErrNotReady
			return
		}
		var seedBytes [8]byte
		rand.Read(seedBytes[:])
		m := model.MatchStart{Version: s.version, MatchID: id(), WorldID: l.WorldID, ArenaID: l.Arena.ID, GameType: l.GameType, Seed: int64(binary.LittleEndian.Uint64(seedBytes[:]) & 0x7fffffffffffffff), PreparedAt: s.clock.Now().UnixMilli()}
		if r, ok := s.rulesMetadata[l.GameType]; ok {
			m.RulesID = r.ID
			m.RulesVersion = r.Version
			m.RulesChecksum = r.Checksum
		}
		for _, p := range l.Players {
			m.Actors = append(m.Actors, p.Actor)
		}
		for len(m.Actors) < l.Capacity {
			m.Actors = append(m.Actors, model.Actor{ActorID: id(), ActorType: "bot", DisplayName: "BOT", AvatarID: "pumpkin"})
		}
		if prepare := s.preparers[l.GameType]; prepare != nil {
			ordered, err := prepare(cloneStart(m))
			if err != nil {
				e = err
				return
			}
			if !sameActors(m.Actors, ordered) {
				e = ErrForbidden
				return
			}
			m.Actors = append([]model.Actor{}, ordered...)
		}
		eng, ee := s.factories[l.GameType](cloneStart(m))
		if ee != nil {
			e = ee
			return
		}
		a = &attempt{match: m, host: host, command: command, done: make(chan struct{}), engine: eng}
		l.start = a
		l.State = "starting"
		s.attempts[key] = a
		s.mark(l.WorldID)
	})
	if err != nil {
		return model.MatchStart{}, err
	}
	if e != nil {
		return model.MatchStart{}, e
	}
	// A single leader persists; followers wait on its completion, never repeat SQL.
	leader := false
	s.control(func() {
		if !a.durable && a.err == nil && a.engine != nil {
			a.durable = true
			leader = true
			s.workers.Add(1)
		}
	})
	if leader {
		go func() { defer s.workers.Done(); s.persistStart(a) }()
	}
	select {
	case <-a.done:
		return cloneStart(a.match), a.err
	case <-ctx.Done():
		return model.MatchStart{}, ctx.Err()
	case <-s.done:
		return model.MatchStart{}, ErrClosed
	}
}
func cloneStart(m model.MatchStart) model.MatchStart {
	m.Actors = append([]model.Actor{}, m.Actors...)
	return m
}
func (s *Service) persistStart(a *attempt) {
	ctx, cancel := context.WithTimeout(s.ctx, 3*time.Second)
	defer cancel()
	var e error
	if a.reconcileOnly {
		var found bool
		found, e = s.store.Lookup(ctx, a.match.MatchID)
		if e != nil {
			e = ErrUnconfirmed
		} else if !found {
			e = s.store.Persist(ctx, a.match, a.host, a.command)
			if errors.Is(e, ErrCommitUnknown) {
				e = ErrUnconfirmed
			}
		}
	} else {
		e = s.store.Persist(ctx, a.match, a.host, a.command)
		if errors.Is(e, ErrCommitUnknown) {
			check, done := context.WithTimeout(s.ctx, 3*time.Second)
			found, lookupErr := s.store.Lookup(check, a.match.MatchID)
			done()
			if lookupErr != nil || !found {
				e = ErrUnconfirmed
			} else {
				e = nil
			}
		}
	}
	activated := false
	s.control(func() {
		l := s.findByMatch(a)
		if e != nil {
			a.err = e
			if e == ErrUnconfirmed {
				a.retryAt = s.clock.Now().Add(time.Second)
			}
			if e != ErrUnconfirmed {
				a.engine.Close()
				a.engine = nil
				delete(s.attempts, [2]string{a.host, a.command})
				if l != nil {
					l.start = nil
					l.State = "lobby"
					for i := len(l.Players) - 1; i >= 0; i-- {
						if l.Players[i].departed {
							s.depart(l, i)
						}
					}
					s.mark(l.WorldID)
				}
			}
			close(a.done)
			return
		}
		if l == nil {
			a.err = ErrClosed
			a.engine.Close()
			close(a.done)
			return
		}
		l.MatchID = a.match.MatchID
		allGone := true
		for _, p := range l.Players {
			if !p.departed {
				allGone = false
			}
		}
		if allGone {
			l.State = "abandoned"
			a.engine.Close()
			a.engine = nil
			a.err = ErrForbidden
		} else {
			a.match.StartsAt = s.clock.Now().Add(l.countdown).UnixMilli()
			a.engine.Start(a.match.StartsAt)
			l.engine = a.engine
			l.StartsAt = a.match.StartsAt
			l.State = "countdown"
			if l.countdown == 0 {
				l.State = "running"
			}
			activated = true
			// A fresh engine needs every original disconnect before an expired
			// departure can evaluate grace, just as in the live presence batch.
			for _, p := range l.Players {
				if !p.disconnected.IsZero() {
					enginePresence(l.engine, p.UserID, "disconnect", p.disconnected)
				}
			}
			for _, p := range l.Players {
				if p.departed {
					enginePresence(l.engine, p.UserID, p.departureKind, p.departedAt)
				}
			}
			if l.State == "running" && l.engine.Tick(s.clock.Now()) {
				l.State = "finalizing"
				l.finalizingAt = s.clock.Now()
			}
		}
		s.mark(l.WorldID)
		close(a.done)
	})
	if activated {
		select {
		case s.storage <- func() {
			ctx, done := context.WithTimeout(s.ctx, 3*time.Second)
			defer done()
			if err := s.store.Schedule(ctx, a.match.MatchID, a.match.StartsAt); err != nil && s.ctx.Err() == nil {
				slog.Error("match schedule metadata not persisted", "match_id", a.match.MatchID)
			}
		}:
		default:
			slog.Error("match schedule metadata queue full", "match_id", a.match.MatchID)
		}
	}
}
func (s *Service) findByMatch(a *attempt) *entry {
	for _, l := range s.entries {
		if l.start == a {
			return l
		}
	}
	return nil
}

func (s *Service) retryAttempt(l *entry, old *attempt) *attempt {
	a := &attempt{match: cloneStart(old.match), host: old.host, command: old.command, done: make(chan struct{}), engine: old.engine, reconcileOnly: true}
	s.attempts[[2]string{a.host, a.command}] = a
	l.start = a
	return a
}
