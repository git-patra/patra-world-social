package world

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

type Clock interface{ Now() time.Time }
type WallClock struct{}

func (WallClock) Now() time.Time { return time.Now() }

var ErrClosed = errors.New("runtime closed")
var ErrBusy = errors.New("runtime queue full")
var ErrMovement = errors.New("invalid movement")

type Avatar struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Status string  `json:"status"`
	Seq    uint64  `json:"seq"`
}
type Snapshot struct {
	Avatars []Avatar `json:"avatars"`
	Tick    uint64   `json:"tick"`
	TimeMs  int64    `json:"timeMs"`
}
type player struct {
	Avatar
	epoch   uint64
	dx, dy  float64
	expires time.Time
	path    []Point
}
type request struct {
	fn   func()
	done chan struct{}
}

// Runtime owns simulation state on one goroutine. No database/network work runs here.
type Runtime struct {
	vehicles        [4]Vehicle
	vehicleRevision uint64
	incarnation     string
	areaGeneration  uint64
	areas           map[string]*areaState
	clock           Clock
	m               MapDefinition
	requests        chan request
	stop            chan struct{}
	done            chan struct{}
	once            sync.Once
	players         map[string]*player
	last            time.Time
	tick            uint64
}

func NewRuntime(m MapDefinition, c Clock) *Runtime {
	r := &Runtime{incarnation: incarnation(), areas: map[string]*areaState{}, clock: c, m: m, requests: make(chan request, 128), stop: make(chan struct{}), done: make(chan struct{}), players: map[string]*player{}, last: c.Now()}
	r.initVehicles()
	go r.run()
	return r
}
func (r *Runtime) run() {
	defer close(r.done)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-r.stop:
			return
		case <-ticker.C:
			r.advance()
		case q := <-r.requests:
			r.advance()
			q.fn()
			close(q.done)
		}
	}
}
func (r *Runtime) call(ctx context.Context, fn func()) error {
	q := request{fn, make(chan struct{})}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.done:
		return ErrClosed
	default:
	}
	select {
	case r.requests <- q:
	default:
		return ErrBusy
	} // Once accepted, wait for ownership completion; never mutate result after caller returns.
	select {
	case <-q.done:
		return nil
	case <-r.done:
		return ErrClosed
	}
}
func (r *Runtime) Close() { r.once.Do(func() { close(r.stop) }); <-r.done }
func (r *Runtime) advance() {
	now := r.clock.Now()
	r.advanceAreas()
	vehicleSteps := 0
	for r.last.Add(50*time.Millisecond).Compare(now) <= 0 {
		r.last = r.last.Add(50 * time.Millisecond)
		r.tick++
		if vehicleSteps < 4 {
			r.moveVehicles(r.last)
			vehicleSteps++
		}
		for _, p := range r.players {
			r.move(p, r.last)
		} // Skip long stalls after intents expire, never grant elapsed-time speed.
		if now.Sub(r.last) > time.Second {
			r.last = now.Add(-250 * time.Millisecond)
		}
	}
}
func (r *Runtime) Join(ctx context.Context, id, name string, epoch uint64) error {
	return r.call(ctx, func() {
		r.dismount(id)
		p := r.players[id]
		if p == nil {
			p = &player{Avatar: Avatar{ID: id, Name: name, X: float64(r.m.Spawn.X), Y: float64(r.m.Spawn.Y), Status: "available"}}
			r.players[id] = p
		}
		p.epoch = epoch
		p.Seq = 0
		p.dx = 0
		p.dy = 0
		p.path = nil
		p.Status = "available"
	})
}
func (r *Runtime) Remove(ctx context.Context, id string) error {
	return r.call(ctx, func() { r.dismount(id); delete(r.players, id) })
}
func (r *Runtime) Idle(ctx context.Context, id string) error {
	return r.call(ctx, func() {
		r.dismount(id)
		if p := r.players[id]; p != nil {
			p.dx = 0
			p.dy = 0
			p.path = nil
			p.Status = "AFK"
		}
	})
}
func (r *Runtime) Snapshot(ctx context.Context) (Snapshot, error) {
	s := Snapshot{}
	err := r.call(ctx, func() {
		s = Snapshot{Avatars: make([]Avatar, 0, len(r.players)), Tick: r.tick, TimeMs: r.clock.Now().UnixMilli()}
		for _, p := range r.players {
			s.Avatars = append(s.Avatars, p.Avatar)
		}
		sort.Slice(s.Avatars, func(i, j int) bool { return s.Avatars[i].ID < s.Avatars[j].ID })
	})
	return s, err
}
