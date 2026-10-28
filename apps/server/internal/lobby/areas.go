package lobby

import (
	"context"
	"errors"
	"patra-world-social/server/internal/world"
	"time"
)

type AreaCoordinator interface {
	ReserveArea(context.Context, string, string, string) (world.AreaLease, error)
	ClearArea(context.Context, world.AreaLease) error
	ActivateArea(context.Context, world.AreaLease, int64) error
	ReleaseArea(context.Context, world.AreaLease) error
}

// Configure before admitting lobbies. No cross-owner calls run on the lobby owner.
func (s *Service) SetAreaCoordinator(a AreaCoordinator) error {
	return s.call(context.Background(), func() {
		if len(s.entries) == 0 {
			s.areas = a
		}
	})
}
func (s *Service) releaseSpatial(l *entry) {
	if s.areas == nil || l.start == nil || l.start.lease.Generation == 0 || l.areaReleaseQueued {
		return
	}
	l.areaReleaseQueued = true
	s.queueAreaRelease(l.start.lease)
}

// Independent of result SQL and entry lifetime: detached results must not strand
// a lease after a transient full world queue. A stale lease is already harmless.
func (s *Service) queueAreaRelease(lease world.AreaLease) {
	if s.areas == nil || lease.Generation == 0 {
		return
	}
	s.workers.Add(1)
	go func() {
		defer s.workers.Done()
		for {
			ctx, cancel := context.WithTimeout(s.ctx, time.Second)
			err := s.areas.ReleaseArea(ctx, lease)
			cancel()
			if err == nil || errors.Is(err, world.ErrActivitySuperseded) {
				return
			}
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
		}
	}()
}
