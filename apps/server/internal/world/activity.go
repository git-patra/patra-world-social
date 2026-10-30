package world

import (
	"context"
	"errors"
)

var ErrActivitySuperseded = errors.New("activity ownership superseded")

// ActivityGuard checks ownership and calls apply synchronously under its fence.
// The callback only mutates owner state; it must not block or reenter services.
type ActivityGuard func(apply func()) bool

// Activity neutralizes movement on the world owner. Returns always use a known
// unblocked entrance, never client-supplied teleport coordinates.
func (r *Runtime) Activity(ctx context.Context, user string, playing bool, at *Point) error {
	return r.activity(ctx, user, playing, at, nil)
}

// ResumeCasual is called by the Hub after the simulation movement claim ends.
// It restores stale game presence without canceling a walker's path or held input.
func (r *Runtime) ResumeCasual(ctx context.Context, user string, epoch uint64) error {
	var result error
	err := r.call(ctx, func() {
		if ctx.Err() != nil {
			result = ctx.Err()
			return
		}
		p := r.players[user]
		if p == nil || p.epoch != epoch {
			result = ErrActivitySuperseded
			return
		}
		if p.Status == "playing" {
			p.Status = "available"
		}
	})
	if err != nil {
		return err
	}
	return result
}

func (r *Runtime) ReturnActivity(ctx context.Context, user string, at Point, guard ActivityGuard) error {
	return r.activity(ctx, user, false, &at, guard)
}

func (r *Runtime) activity(ctx context.Context, user string, playing bool, at *Point, guard ActivityGuard) error {
	applied := true
	err := r.call(ctx, func() {
		p := r.players[user]
		if p == nil {
			return
		}
		apply := func() {
			p.dx = 0
			p.dy = 0
			p.path = nil
			if playing {
				r.dismount(user)
				r.releaseClaw(user)
				p.Status = "playing"
			} else {
				p.Status = "available"
			}
			if at != nil && !r.blocked(*at) {
				p.X = float64(at.X)
				p.Y = float64(at.Y)
			}
		}
		if guard == nil {
			apply()
		} else {
			applied = guard(apply)
		}
	})
	if err != nil {
		return err
	}
	if !applied {
		return ErrActivitySuperseded
	}
	return nil
}
