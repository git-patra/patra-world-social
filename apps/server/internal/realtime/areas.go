package realtime

import (
	"context"
	"patra-world-social/server/internal/world"
)

type areaRoom interface {
	ReserveArea(context.Context, string, string, string) (world.AreaLease, error)
	ClearArea(context.Context, world.AreaLease) error
	ActivateArea(context.Context, world.AreaLease, int64) error
	ReleaseArea(context.Context, world.AreaLease) error
	ActivitySnapshots(context.Context, string) ([]world.ActivitySnapshot, error)
	HasAreaLeases(context.Context) (bool, error)
}

func (h *Hub) ReserveArea(ctx context.Context, w, a, m string) (world.AreaLease, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return world.AreaLease{}, world.ErrClosed
	}
	r := h.rooms[w]
	if r == nil {
		r = world.NewRuntime(h.m, h.clock)
		h.rooms[w] = r
	}
	owner, ok := r.(areaRoom)
	if !ok {
		return world.AreaLease{}, ErrUnsupported
	}
	l, err := owner.ReserveArea(ctx, w, a, m)
	if err == nil {
		h.areaLifecycleLocked(ctx, w)
	}
	return l, err
}
func (h *Hub) mutateArea(ctx context.Context, l world.AreaLease, fn func(areaRoom) error) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	r, ok := h.rooms[l.WorldID].(areaRoom)
	if !ok {
		return world.ErrActivitySuperseded
	}
	if err := fn(r); err != nil {
		return err
	}
	h.areaLifecycleLocked(ctx, l.WorldID)
	return nil
}
func (h *Hub) ClearArea(ctx context.Context, l world.AreaLease) error {
	return h.mutateArea(ctx, l, func(r areaRoom) error { return r.ClearArea(ctx, l) })
}
func (h *Hub) ActivateArea(ctx context.Context, l world.AreaLease, at int64) error {
	return h.mutateArea(ctx, l, func(r areaRoom) error { return r.ActivateArea(ctx, l, at) })
}
func (h *Hub) ReleaseArea(ctx context.Context, l world.AreaLease) error {
	return h.mutateArea(ctx, l, func(r areaRoom) error { return r.ReleaseArea(ctx, l) })
}
func (h *Hub) areaLifecycleLocked(ctx context.Context, w string) {
	r, ok := h.rooms[w].(areaRoom)
	if !ok {
		return
	}
	states, err := r.ActivitySnapshots(ctx, w)
	if err != nil {
		return
	}
	for _, v := range states {
		key := w + ":" + v.AreaID + ":" + v.Kind
		previous, seen := h.areaStates[key]
		if seen && previous.Incarnation == v.Incarnation && previous.Generation == v.Generation && (v.Kind == "vehicles" || v.Kind == "football" || previous.Payload == v.Payload) {
			continue
		}
		h.areaStates[key] = v
		for _, s := range h.seats {
			if s.world == w && s.disconnected.IsZero() {
				select {
				case s.events <- Delivery{"activity.lifecycle", v}:
				default:
					h.stopSeat(s)
					s.disconnected = h.clock.Now()
					if h.SeatChanged != nil {
						h.SeatChanged("disconnect", s.world, s.user)
					}
					h.rooms[w].Idle(ctx, s.user)
				}
			}
		}
	}
}
func (h *Hub) ActivitySnapshots(ctx context.Context, j JoinContext) ([]world.ActivitySnapshot, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.seats[j.SessionID]
	if s == nil || !matches(s, j, s.user) || !s.disconnected.IsZero() {
		return nil, ErrFenced
	}
	r, ok := h.rooms[s.world].(areaRoom)
	if !ok {
		return nil, ErrUnsupported
	}
	h.areaLifecycleLocked(ctx, s.world)
	return r.ActivitySnapshots(ctx, s.world)
}

// WorldActivities is a trusted coordinator read; transports use the seat-fenced API.
func (h *Hub) WorldActivities(ctx context.Context, worldID string) ([]world.ActivitySnapshot, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r, ok := h.rooms[worldID].(areaRoom)
	if !ok {
		return nil, ErrUnsupported
	}
	return r.ActivitySnapshots(ctx, worldID)
}
