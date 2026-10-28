package realtime

import (
	"context"
	"patra-world-social/server/internal/world"
)

// WithPosition runs only local state work under hub.mu, within WithJoined's
// admission lease. The callback must not call storage, providers, or the hub.
func (h *Hub) WithPosition(ctx context.Context, j JoinContext, fn func(world.Avatar) error) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.seats[j.SessionID]
	if s == nil || !matches(s, j, s.user) || !s.disconnected.IsZero() {
		return ErrFenced
	}
	snapshot, err := h.rooms[s.world].Snapshot(ctx)
	if err != nil {
		return err
	}
	for _, a := range snapshot.Avatars {
		if a.ID == s.user {
			if h.GameBusy != nil && h.GameBusy(a.ID) || h.VoiceExcluded != nil && h.VoiceExcluded(a.ID) {
				a.Status = "playing"
			}
			return fn(a)
		}
	}
	return ErrFenced
}
