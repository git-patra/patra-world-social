package lobby

import (
	"context"
	"encoding/json"
	"errors"
	"patra-world-social/server/internal/games/model"
	"time"
)

// ResultEngine is optional until the concrete games are registered. Result must
// copy only resolved state, be bounded, and perform no I/O or service reentry.
type ResultEngine interface {
	Result() (model.MatchResult, error)
}
type FinalizingMatch struct {
	MatchID, WorldID string
	Users            []string
	Since            time.Time
}

func (s *Service) Finalizing(ctx context.Context) ([]FinalizingMatch, error) {
	out := []FinalizingMatch{}
	e := s.call(ctx, func() {
		for _, l := range s.entries {
			if l.State == "finalizing" {
				m := FinalizingMatch{MatchID: l.MatchID, WorldID: l.WorldID, Since: l.finalizingAt}
				for _, p := range l.Players {
					if p.UserID != "" {
						m.Users = append(m.Users, p.UserID)
					}
				}
				out = append(out, m)
			}
		}
	})
	return out, e
}
func (s *Service) CopyResult(ctx context.Context, id string) (model.MatchResult, error) {
	var out model.MatchResult
	err := ErrForbidden
	e := s.call(ctx, func() {
		for _, l := range s.entries {
			if l.MatchID != id || l.State != "finalizing" {
				continue
			}
			engine, ok := l.engine.(ResultEngine)
			if !ok {
				err = ErrUnavailable
				return
			}
			r, e := engine.Result()
			if e != nil {
				err = e
				return
			}
			// The runtime's actual start is authoritative even if Schedule never committed.
			if l.start == nil || r.MatchID != id || r.StartsAt != l.StartsAt || r.PreparedAt != l.start.match.PreparedAt {
				err = errors.New("engine result timing or match mismatch")
				return
			}
			raw, e := json.Marshal(r)
			if e != nil {
				err = e
				return
			}
			if len(raw) > 262144 {
				err = errors.New("engine result too large")
				return
			}
			err = json.Unmarshal(raw, &out)
			return
		}
	})
	if e != nil {
		return out, e
	}
	return out, err
}

// Gate admission on the owner, not via a racy external Busy precheck. Existing
// uncertain starts may still reconcile their already-reserved command IDs.
func (s *Service) SetStorageBlocked(blocked bool) error {
	return s.control(func() { s.storageBlocked = blocked })
}

// DetachUnconfirmed releases render state and local arena claims without making
// any assertion about commit success. The worker retains this match ID until a
// locked reconciliation resolves it, and keeps new starts gated throughout.
func (s *Service) DetachUnconfirmed(id string) error { return s.control(func() { s.detach(id) }) }
func (s *Service) detach(id string) {
	for key, l := range s.entries {
		if l.MatchID != id {
			continue
		}
		for _, p := range l.Players {
			if s.releaseClaim(p.UserID, l.ID) {
				s.returns[[2]string{l.WorldID, p.UserID}] = s.arenaReturn(l.WorldID, p.UserID, l.ReturnX, l.ReturnY)
			}
		}
		if l.engine != nil {
			l.engine.Close()
		}
		delete(s.entries, key)
		if l.start != nil {
			delete(s.attempts, [2]string{l.start.host, l.start.command})
		}
		s.mark(l.WorldID)
	}
}
