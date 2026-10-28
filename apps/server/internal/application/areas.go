package application

import (
	"context"
	"errors"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/world"
	"time"
)

// Clearing waits outside both owners, and only returns the world owner's ACK.
type areaCoordinator struct{ *realtime.Hub }

func (a areaCoordinator) ClearArea(ctx context.Context, l world.AreaLease) error {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		err := a.Hub.ClearArea(ctx, l)
		if !errors.Is(err, world.ErrAreaClearing) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
