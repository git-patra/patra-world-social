package realtime

import (
	"context"
	"encoding/json"
	"patra-world-social/server/internal/world"
	"strings"
	"time"
)

type Delivery struct {
	Type    string
	Payload any
}

// WithJoined binds storage work to an attached seat. Admission/revocation waits
// at most the short operation deadline; movement and simulation ticks keep running.
func (h *Hub) WithJoined(ctx context.Context, j JoinContext, fn func(context.Context, string) error) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	h.lease.RLock()
	defer h.lease.RUnlock()
	if e := h.Validate(ctx, j); e != nil {
		return e
	}
	h.mu.Lock()
	s := h.seats[j.SessionID]
	if s == nil || !matches(s, j, s.user) || !s.disconnected.IsZero() {
		h.mu.Unlock()
		return ErrFenced
	}
	user := s.user
	h.mu.Unlock()
	return fn(ctx, user)
}

func (h *Hub) Events(j JoinContext) <-chan Delivery {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s := h.seats[j.SessionID]; s != nil && matches(s, j, s.user) {
		return s.events
	}
	return nil
}

// Publish never blocks on a transport. Overflow terminates that connection;
// the next snapshot causes the client to resync the durable world history.
func (h *Hub) Publish(worldID, kind string, payload any) {
	h.publishTo(worldID, nil, kind, payload)
}

// PublishParticipants delivers private result status only to these admitted accounts.
func (h *Hub) PublishParticipants(worldID string, users []string, kind string, payload any) {
	allowed := map[string]bool{}
	for _, id := range users {
		allowed[id] = true
	}
	h.publishTo(worldID, allowed, kind, payload)
}
func (h *Hub) publishTo(worldID string, allowed map[string]bool, kind string, payload any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, s := range h.seats {
		if s.world != worldID || !s.disconnected.IsZero() || (allowed != nil && !allowed[s.user]) {
			continue
		}
		select {
		case s.events <- Delivery{kind, payload}:
		default:
			h.stopSeat(s)
			s.disconnected = h.clock.Now()
			if h.SeatChanged != nil {
				h.SeatChanged("disconnect", s.world, s.user)
			}
			h.rooms[s.world].Idle(context.Background(), s.user)
		}
	}
}
func (h *Hub) Online(userID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.users[userID]
	return s != nil && s.disconnected.IsZero()
}

// Execute keeps durable/control acknowledgements separate from replaceable snapshots.
func (h *Hub) Execute(ctx context.Context, j JoinContext, c Command) (any, error) {
	if c.Type != "chat.send" && !isGameCommand(c.Type) && !strings.HasPrefix(c.Type, "voice.") {
		return nil, h.Submit(ctx, j, c)
	}
	raw, e := json.Marshal(c)
	if e != nil {
		return nil, e
	}
	if _, e = ValidateCommand(raw); e != nil {
		return nil, e
	}
	if c.WorldID != j.WorldID {
		return nil, ErrUnauthorized
	}
	if strings.HasPrefix(c.Type, "voice.") {
		if h.Voice == nil {
			return nil, ErrUnsupported
		}
		return h.Voice(ctx, j, c)
	}
	if isGameCommand(c.Type) {
		if h.Games == nil {
			return nil, ErrUnsupported
		}
		return h.Games(ctx, j, c)
	}
	if h.Chat == nil {
		return nil, ErrUnsupported
	}
	return h.Chat(ctx, j, c)
}
func (h *Hub) NeedsResync(j JoinContext) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.seats[j.SessionID]
	return s != nil && matches(s, j, s.user) && !s.disconnected.IsZero()
}

// WithControl orders chat moderation after every in-flight stored-message
// publication. Callbacks may publish, but must not reenter revocation/admission.
func (h *Hub) WithControl(ctx context.Context, fn func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	h.lease.Lock()
	defer h.lease.Unlock()
	if e := ctx.Err(); e != nil {
		return e
	}
	return fn(ctx)
}

func isGameCommand(kind string) bool {
	switch kind {
	case "duck.snapshot", "duck.shoot", "duck.clock", "duck.clock_reply", "lobby.list", "lobby.join", "lobby.ready", "lobby.bot", "lobby.leave", "match.start", "race.input", "race.snapshot", "penalty.choose", "penalty.snapshot", "match.return", "match.rematch":
		return true
	}
	return false
}

// SetActivity is called outside the lobby owner and all admission callbacks.
func (h *Hub) SetActivity(ctx context.Context, worldID, user string, playing bool, at *world.Point) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if (playing || h.VoiceExcluded != nil && h.VoiceExcluded(user)) && h.VoiceRevoke != nil {
		h.VoiceRevoke(worldID, user, "game entry")
	}
	if r := h.rooms[worldID]; r != nil {
		return r.Activity(ctx, user, playing, at)
	}
	return nil
}

// ReturnActivity evaluates the ownership fence on the world owner at mutation
// time. No hub/lobby callback is made while the fence is held.
func (h *Hub) ReturnActivity(ctx context.Context, worldID, user string, at world.Point, guard world.ActivityGuard) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r := h.rooms[worldID]; r != nil {
		return r.ReturnActivity(ctx, user, at, guard)
	}
	return nil
}

// WithAdmitted is only for latency-sensitive game input. Revocation/admission
// holds the opposite lease; identity/world/token/epoch and expiry remain fenced.
func (h *Hub) WithAdmitted(ctx context.Context, j JoinContext, fn func(context.Context, string) error) error {
	h.lease.RLock()
	defer h.lease.RUnlock()
	h.mu.Lock()
	s := h.seats[j.SessionID]
	if h.closed || s == nil || !matches(s, j, s.user) || !s.disconnected.IsZero() {
		h.mu.Unlock()
		return ErrFenced
	}
	if !h.clock.Now().Before(s.authorizedUntil) {
		h.mu.Unlock()
		return ErrUnauthorized
	}
	user := s.user
	until := s.authorizedUntil
	h.mu.Unlock()
	ctx = context.WithValue(ctx, admissionDeadlineKey{}, until)
	ctx, cancel := context.WithDeadline(ctx, until)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn(ctx, user)
}

type admissionDeadlineKey struct{}

// CheckAdmissionTime repeats expiry at owner ingestion after queue waiting.
func CheckAdmissionTime(ctx context.Context, at time.Time) error {
	until, ok := ctx.Value(admissionDeadlineKey{}).(time.Time)
	if !ok || !at.Before(until) {
		return ErrUnauthorized
	}
	return ctx.Err()
}
