package lobby

import (
	"context"
	"sort"
	"time"
)

// Notify preserves alternating transitions while bounding a stalled owner's queue.
// Repeated identical states add no information; overflow leaves the match safely.
func (s *Service) Notify(kind, world, user string) {
	if kind != "disconnect" && kind != "resume" && kind != "leave" {
		return
	}
	s.pendingMu.Lock()
	key := [2]string{world, user}
	queue := s.pending[key]
	if len(queue) == 0 || (queue[len(queue)-1].kind != "leave" && queue[len(queue)-1].kind != kind) {
		n := notification{kind: kind, world: world, user: user, at: s.clock.Now()}
		if len(queue) >= 32 {
			n.kind = "leave"
			queue = queue[:31]
		}
		s.pending[key] = append(queue, n)
	}
	s.pendingMu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}
func (s *Service) drainPresence() {
	s.pendingMu.Lock()
	pending := s.pending
	s.pending = map[[2]string][]notification{}
	s.pendingMu.Unlock()
	ordered := []notification{}
	for _, queue := range pending {
		ordered = append(ordered, queue...)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].at.Equal(ordered[j].at) {
			return ordered[i].user < ordered[j].user
		}
		return ordered[i].at.Before(ordered[j].at)
	})
	// Engines need the earliest disconnect of every player before catch-up may
	// evaluate grace. Later cycles are applied in order after an intervening resume.
	for _, n := range ordered {
		s.recordDisconnect(n)
	}
	for _, n := range ordered {
		s.recordDisconnect(n)
		s.presence(n)
	}
}
func (s *Service) presence(n notification) {
	for _, l := range s.entries {
		if l.WorldID != n.world {
			continue
		}
		if n.kind == "leave" {
			for i, u := range l.Spectators {
				if u == n.user {
					l.Spectators = append(l.Spectators[:i], l.Spectators[i+1:]...)
					s.mark(l.WorldID)
					break
				}
			}
		}
		for i := range l.Players {
			p := &l.Players[i]
			if p.UserID != n.user || p.departed {
				continue
			}
			if n.kind == "leave" {
				s.departAt(l, i, n.at, "leave")
				break
			}
			if n.kind == "resume" {
				if !p.disconnected.IsZero() && n.at.Sub(p.disconnected) >= 15*time.Second {
					// Keep the original disconnect for advance; expire only after
					// every pending player notification has reached the engine.
					continue
				}
				p.Connected = true
				p.disconnected = time.Time{}
				if l.engine != nil && (l.State == "countdown" || l.State == "running") {
					enginePresence(l.engine, n.user, "resume", n.at)
				}
			}
			s.mark(l.WorldID)
		}
	}
}
func (s *Service) Leave(ctx context.Context, user, world string) error {
	return s.call(ctx, func() {
		s.presence(notification{kind: "leave", world: world, user: user, at: s.clock.Now()})
		for _, l := range s.entries {
			if l.WorldID == world {
				for i, u := range l.Spectators {
					if u == user {
						l.Spectators = append(l.Spectators[:i], l.Spectators[i+1:]...)
						s.mark(world)
						break
					}
				}
			}
		}
	})
}
func (s *Service) depart(l *entry, i int) { s.departAt(l, i, s.clock.Now(), "leave") }
func (s *Service) departAt(l *entry, i int, at time.Time, kind string) {
	p := &l.Players[i]
	departingUser := p.UserID

	if l.engine != nil && (l.State == "countdown" || l.State == "running") {
		enginePresence(l.engine, p.UserID, kind, at)
		// Later queued cycles are not installed yet; catch up only to this
		// departure, then let advance consume the clock after the whole batch.
		if l.engine.Tick(at) {
			l.State = "finalizing"
			l.finalizingAt = s.clock.Now()
		}
	}
	if l.State == "lobby" {
		s.releaseClaim(p.UserID, l.ID)
		s.returns[[2]string{l.WorldID, p.UserID}] = s.arenaReturn(l.WorldID, p.UserID, l.ReturnX, l.ReturnY)
		l.Players = append(l.Players[:i], l.Players[i+1:]...)
		if len(l.Players) == 0 {
			delete(s.entries, [2]string{l.WorldID, l.Arena.ID})
		} else if l.HostID == departingUser {
			l.HostID = l.Players[0].UserID
		}
	} else {
		p.departed = true
		p.departedAt = at
		p.departureKind = kind
		p.Connected = false
		all := true
		for _, p := range l.Players {
			if !p.departed {
				all = false
			}
		}
		if all && (l.State == "countdown" || l.State == "running") {
			l.State = "abandoned"
		}
	}
	s.mark(l.WorldID)
}
func (s *Service) advance() {
	now := s.clock.Now()
	for _, l := range s.entries {
		if l.State == "starting" && l.start != nil && l.start.err == ErrUnconfirmed && !now.Before(l.start.retryAt) {
			old := l.start
			a := s.retryAttempt(l, old)
			a.durable = true
			select {
			case s.storage <- func() { s.persistStart(a) }:
			default:
				l.start = old
				s.attempts[[2]string{old.host, old.command}] = old
			}
		}
		if l.State == "countdown" && now.UnixMilli() >= l.StartsAt {
			l.State = "running"
			s.mark(l.WorldID)
		}
		if l.engine != nil && (l.State == "countdown" || l.State == "running") {
			if l.engine.Tick(now) && l.State == "running" {
				l.State = "finalizing"
				l.finalizingAt = now
				s.mark(l.WorldID)
			}
		}
		if l.State == "finalizing" || l.State == "results" || l.State == "abandoned" {
			s.releaseSpatial(l)
		}
		if ev, ok := l.engine.(EventEngine); ok && ev.HasEvents() {
			s.mark(l.WorldID)
		}
		for i := len(l.Players) - 1; i >= 0; i-- {
			p := &l.Players[i]
			if !p.departed && !p.disconnected.IsZero() && now.Sub(p.disconnected) >= 15*time.Second {
				s.departAt(l, i, p.disconnected.Add(15*time.Second), "expired")
			}
		}

		for i := range l.Players {
			p := &l.Players[i]
			if p.departed && !l.resultAction && l.MatchID != "" && !p.releaseQueued && !p.claimReleased && !now.Before(p.releaseAt) {
				user, matchID, lobbyID, worldID := p.UserID, l.MatchID, l.ID, l.WorldID
				p.releaseQueued = true
				select {
				case s.storage <- func() {
					ctx, cancel := context.WithTimeout(s.ctx, 3*time.Second)
					err := s.store.ReleaseUser(ctx, matchID, user)
					cancel()
					s.control(func() {
						current := s.find(lobbyID)
						if current == nil {
							return
						}
						for i := range current.Players {
							if current.Players[i].UserID == user {
								current.Players[i].releaseQueued = false
								current.Players[i].releaseAt = s.clock.Now().Add(time.Second)
								if err == nil {
									current.Players[i].claimReleased = true
									if s.releaseClaim(user, lobbyID) {
										s.returns[[2]string{worldID, user}] = s.arenaReturn(worldID, user, current.ReturnX, current.ReturnY)
									}
									s.mark(worldID)
								}
							}
						}
					})
				}:
				default:
					p.releaseQueued = false
				}
			}
		}
		if (l.GameType == "pingpong" || l.GameType == "penalty" || l.GameType == "race" || l.GameType == "duck") && l.State == "results" && !l.releasing && !now.Before(l.releaseAt) {
			allGone := true
			for _, p := range l.Players {
				if !p.departed {
					allGone = false
				}
			}
			if allGone || l.GameType == "race" || l.GameType == "duck" {
				l.releasing = true
				matchID := l.MatchID
				select {
				case s.storage <- func() {
					if err := s.ReleaseMatch(matchID, "completed"); err != nil {
						s.control(func() {
							if current := s.find(l.ID); current != nil {
								current.releasing = false
								current.releaseAt = s.clock.Now().Add(time.Second)
							}
						})
					}
				}:
				default:
					l.releasing = false
				}
			}
		}
		if l.State == "abandoned" && !l.releasing && !now.Before(l.releaseAt) {
			l.releasing = true
			matchID := l.MatchID
			select {
			case s.storage <- func() {
				if e := s.ReleaseMatch(matchID, "abandoned"); e != nil {
					s.control(func() {
						if current := s.find(l.ID); current != nil {
							current.releasing = false
							current.releaseAt = s.clock.Now().Add(time.Second)
						}
					})
				}
			}:
			default:
				l.releasing = false
			}
		}
	}
}

// Transition is an internal engine/finalizer hook. completed is released only
// after T8 confirms durable results; this service never fabricates saved results.
func (s *Service) Transition(ctx context.Context, matchID, state string) error {
	var e = ErrForbidden
	err := s.call(ctx, func() {
		for _, l := range s.entries {
			if l.MatchID == matchID {
				if l.State == "running" && state == "finalizing" || l.State == "finalizing" && state == "results" {
					l.State = state
					if state == "results" && (l.GameType == "race" || l.GameType == "duck") {
						l.releaseAt = s.clock.Now().Add(10 * time.Second)
					}
					if state == "finalizing" {
						l.finalizingAt = s.clock.Now()
					}
					s.mark(l.WorldID)
					e = nil
				}
				return
			}
		}
	})
	if err != nil {
		return err
	}
	return e
}
func (s *Service) ReleaseMatch(matchID, status string) error {
	if status != "completed" && status != "abandoned" && status != "aborted_server" && status != "aborted_storage" {
		return ErrForbidden
	}
	reserved := false
	if err := s.control(func() {
		for _, l := range s.entries {
			if l.MatchID == matchID && l.resultAction {
				reserved = true
				return
			}
		}
	}); err != nil {
		return err
	}
	if reserved {
		return ErrBusy
	}
	ctx, cancel := context.WithTimeout(s.ctx, 3*time.Second)
	defer cancel()
	if e := s.store.Terminal(ctx, matchID, status); e != nil {
		return e
	}
	return s.call(ctx, func() { s.detach(matchID) })
}

func (s *Service) DrainChanges(ctx context.Context) (Changes, error) {
	var c Changes
	err := s.call(ctx, func() {
		for _, l := range s.entries {
			if l.State == "finalizing" || l.State == "results" || l.State == "abandoned" {
				s.releaseSpatial(l)
			}
			if ev, ok := l.engine.(EventEngine); ok {
				c.Events = append(c.Events, ev.DrainEvents()...)
			}
		}
		for w := range s.changed {
			c.Worlds = append(c.Worlds, w)
		}
		for _, r := range s.returns {
			c.Returns = append(c.Returns, r)
		}
		s.changed = map[string]bool{}
	})
	sort.Strings(c.Worlds)
	return c, err
}

// WithEngine runs game input/result extraction on the owner. user is empty only
// for trusted internal finalizers; transports must supply the admitted user.
func (s *Service) WithEngine(ctx context.Context, matchID, user string, fn func(Engine) error) error {
	var result = ErrForbidden
	err := s.call(ctx, func() {
		for _, l := range s.entries {
			if l.MatchID != matchID || l.engine == nil {
				continue
			}
			if user != "" {
				allowed := false
				for _, p := range l.Players {
					if p.UserID == user && !p.departed && p.Connected {
						allowed = true
					}
				}
				if !allowed {
					return
				}
			}
			result = fn(l.engine)
			return
		}
	})
	if err != nil {
		return err
	}
	return result
}

// arenaReturn is constructed on the lobby owner after releasing its claim.
func (s *Service) arenaReturn(world, user string, x, y int) Return {
	s.busyMu.RLock()
	defer s.busyMu.RUnlock()
	return Return{WorldID: world, UserID: user, X: x, Y: y, Generation: s.generations[user]}
}

// WithReturn fences the actual world mutation against a newer lobby claim.
// apply must only mutate the current world-owner state; it must not call hub,
// lobby, storage, or network methods while the ownership read lock is held.
func (s *Service) WithReturn(r Return, apply func()) bool {
	s.busyMu.RLock()
	defer s.busyMu.RUnlock()
	if s.busy[r.UserID] != "" || s.generations[r.UserID] != r.Generation {
		return false
	}
	apply()
	return true
}

// AckReturn removes only the delivered/obsolete generation. DrainChanges keeps
// pending returns until this acknowledgement, so transient delivery errors retry.
func (s *Service) AckReturn(r Return) error {
	return s.control(func() {
		key := [2]string{r.WorldID, r.UserID}
		if current, ok := s.returns[key]; ok && current.Generation == r.Generation {
			delete(s.returns, key)
		}
	})
}

// Record the complete batch of known disconnect times before any catch-up or
// resume can evaluate a grace boundary against only part of that batch.
func (s *Service) recordDisconnect(n notification) {
	at := n.disconnectedAt
	if at.IsZero() && n.kind == "disconnect" {
		at = n.at
	}
	if at.IsZero() {
		return
	}
	for _, l := range s.entries {
		if l.WorldID != n.world {
			continue
		}
		for i := range l.Players {
			p := &l.Players[i]
			if p.UserID != n.user || p.departed || !p.disconnected.IsZero() {
				continue
			}
			p.disconnected = at
			p.Connected = false
			if l.engine != nil && (l.State == "running" || l.State == "countdown") {
				p.BotAssisted = l.GameType == "race"
				enginePresence(l.engine, n.user, "disconnect", at)
			}
			s.mark(l.WorldID)
		}
	}
}
