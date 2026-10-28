package voice

import (
	"context"
	"errors"
	"strings"
	"time"
)

// ReconcileParticipants is the sole provider network owner. It never holds mu
// across network/SQL/authorization, and checks current generation before applying.
func (s *Service) ReconcileParticipants(ctx context.Context) error {
	s.worker.Lock()
	defer s.worker.Unlock()
	if s.provider == nil {
		return ErrUnavailable
	}
	s.mu.Lock()
	initialized := s.initialized
	s.mu.Unlock()
	if !initialized {
		rows, err := s.journal.Load(ctx)
		if err != nil {
			return err
		}
		s.mu.Lock()
		for _, r := range rows {
			if s.entries[r.ID] == nil {
				s.entries[r.ID] = &entry{VoiceAdmission: VoiceAdmission{ID: r.ID, Identity: r.ID, Room: r.Room}, revoked: true}
			}
		}
		s.initialized = true
		s.mu.Unlock()
	}
	s.mu.Lock()
	var active []entry
	for _, e := range s.entries {
		if e.active && !e.revoked {
			active = append(active, *e)
		}
	}
	s.mu.Unlock()
	for _, e := range active {
		if err := s.with(ctx, e.ID, func(*entry, Subject) error { return nil }); err != nil {
			s.mu.Lock()
			s.revoke(s.entries[e.ID])
			s.mu.Unlock()
		}
	}
	s.mu.Lock()
	var pending, muting []entry
	for _, e := range s.entries {
		if e.revoked && !e.persisting {
			pending = append(pending, *e)
		} else if e.mutePending {
			muting = append(muting, *e)
		}
	}
	s.mu.Unlock()
	var failures []error
	attempted := map[string]bool{}
	for _, e := range pending {
		attempted[e.ID] = true
		if err := s.cleanupRevocation(ctx, e); err != nil {
			failures = append(failures, err)
		}
	}

	for _, e := range muting {
		call, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := s.provider.Mute(call, Record{e.ID, e.Room})
		cancel()
		if err != nil {
			failures = append(failures, err)
			continue
		}
		s.mu.Lock()
		if current := s.entries[e.ID]; current != nil && current.muteVersion == e.muteVersion {
			current.mutePending = false
		}
		s.mu.Unlock()
	}
	call, cancel := context.WithTimeout(ctx, 2*time.Second)
	rooms, err := s.provider.Rooms(call)
	cancel()
	if err != nil {
		failures = append(failures, err)
	} else {
		for room, ids := range rooms {
			if !strings.HasPrefix(room, s.prefix) {
				continue
			}
			if lister, ok := s.provider.(interface {
				Participants(context.Context, string) ([]string, error)
			}); ok {
				call, cancel = context.WithTimeout(ctx, 2*time.Second)
				ids, err = lister.Participants(call, room)
				cancel()
				if err != nil {
					failures = append(failures, err)
					continue
				}
			}
			for _, id := range ids {
				s.mu.Lock()
				e := s.entries[id]
				known := e != nil && !e.revoked && e.Room == room
				if !known {
					s.healthy = false
					if e == nil {
						e = &entry{VoiceAdmission: VoiceAdmission{ID: id, Identity: id, Room: room}, revoked: true, journalPending: true}
						s.entries[id] = e
					}
				}
				var pending entry
				if e != nil {
					pending = *e
				}
				s.mu.Unlock()
				if known || attempted[id] {
					continue
				}
				attempted[id] = true
				if pending.Room != room || !pending.revoked {
					failures = append(failures, ErrForbidden)
					continue
				}
				if err = s.cleanupRevocation(ctx, pending); err != nil {
					failures = append(failures, err)
				}
			}

			// A new reservation may appear during listing. Empty room deletion is safe
			// only when no local reservation exists; LiveKit deletes no occupied room here.
			s.mu.Lock()
			occupied := false
			for _, e := range s.entries {
				if e.Room == room && !e.revoked {
					occupied = true
				}
			}
			if !occupied && len(ids) == 0 {
				s.cleaning[room] = true
			}
			s.mu.Unlock()
			if !occupied && len(ids) == 0 {
				call, cancel = context.WithTimeout(ctx, 2*time.Second)
				err = s.provider.DeleteRoom(call, room)
				cancel()
				s.mu.Lock()
				delete(s.cleaning, room)
				s.mu.Unlock()
				if err != nil {
					failures = append(failures, err)
				}
			}
		}
	}
	s.mu.Lock()
	s.healthy = len(failures) == 0
	now := time.Now()
	for room, t := range s.tables {
		if !t.emptySince.IsZero() && now.Sub(t.emptySince) >= 5*time.Minute {
			delete(s.tables, room)
		}
	}
	for user, b := range s.limits {
		if now.Sub(b.at) > 5*time.Minute {
			delete(s.limits, user)
		}
	}
	s.mu.Unlock()
	return errors.Join(failures...)
}

// Webhooks are only reconciliation hints. Duplicate or reordered events cannot
// resurrect an admission or revoke a newer generation; periodic sweep is primary.
func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		call, cancel := context.WithTimeout(ctx, 10*time.Second)
		_ = s.ReconcileParticipants(call)
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// The pending entry exists before any uncertain journal write. Listing absence
// never settles an identity: save, explicit removal and deletion must all succeed.
func (s *Service) cleanupRevocation(ctx context.Context, e entry) error {
	r := Record{e.ID, e.Room}
	if e.journalPending {
		if err := s.journal.Save(ctx, r); err != nil {
			return err
		}
		s.mu.Lock()
		if current := s.entries[e.ID]; current != nil {
			current.journalPending = false
		}
		s.mu.Unlock()
	}
	call, cancel := context.WithTimeout(ctx, 2*time.Second)
	err := s.provider.Remove(call, r, time.Now().Add(2*time.Second).Unix())
	cancel()
	if err != nil {
		return err
	}
	if err = s.journal.Delete(ctx, r.ID); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if current := s.entries[e.ID]; current != nil && current.revoked && !current.persisting {
		delete(s.entries, e.ID)
	}
	return nil
}
