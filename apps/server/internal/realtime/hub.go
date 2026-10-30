package realtime

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"patra-world-social/server/internal/auth"
	"patra-world-social/server/internal/world"
	"patra-world-social/server/internal/worlds"
	"sync"
	"time"
)

var (
	ErrFull         = errors.New("world full")
	ErrTakeover     = errors.New("takeover required")
	ErrFenced       = errors.New("connection fenced")
	ErrUnauthorized = errors.New("access revoked")
	ErrUnsupported  = errors.New("command unavailable")
)

type SessionChecker interface {
	RequireSession(context.Context, string) (auth.Session, error)
}
type MembershipChecker interface {
	AuthorizeMember(context.Context, string, string) (worlds.Membership, error)
}
type JoinContext struct {
	Token     string
	WorldID   string
	SessionID string
	Epoch     uint64
	Takeover  bool
}
type PublicSnapshot struct {
	world.Snapshot
	WorldID         string `json:"worldId"`
	SelfID          string `json:"selfId"`
	SessionID       string `json:"sessionId"`
	Epoch           uint64 `json:"epoch"`
	MapID           string `json:"mapId"`
	MapChecksum     string `json:"mapChecksum"`
	RulesID         string `json:"rulesId"`
	RulesChecksum   string `json:"rulesChecksum"`
	ProtocolVersion int    `json:"protocolVersion"`
}
type snapshotDelivery struct {
	interested       bool
	match            string
	at, seen, refill time.Time
	tokens           float64
	spectator        bool
}
type seat struct {
	gameSnapshots          map[string]snapshotDelivery
	authorizedUntil        time.Time
	id, user, token, world string
	epoch                  uint64
	disconnected           time.Time
	done                   chan struct{}
	events                 chan Delivery
}

// room separates admission ownership from the world simulation owner.
type room interface {
	Join(context.Context, string, string, uint64) error
	Snapshot(context.Context) (world.Snapshot, error)
	Input(context.Context, string, uint64, uint64, float64, float64) error
	Destination(context.Context, string, uint64, uint64, float64, float64) error
	Idle(context.Context, string) error
	Activity(context.Context, string, bool, *world.Point) error
	ReturnActivity(context.Context, string, world.Point, world.ActivityGuard) error
	Remove(context.Context, string) error
	Close()
}

// Hub serializes admission and cross-world ownership; per-world runtimes own movement.
// Authentication runs before locking, fenced by revision so a committed revoke wins.
type Hub struct {
	areaStates map[string]world.ActivitySnapshot
	mu         sync.Mutex
	// lease is acquired before mu or storage; runtime ticks never acquire it.
	lease         sync.RWMutex
	RedactPayload func([]byte) ([]byte, error)
	Voice         func(context.Context, JoinContext, Command) (any, error)
	VoiceObserve  func(worldID, userID string, x, y float64, playing bool)
	VoiceExcluded func(userID string) bool
	VoiceRevoke   func(worldID, userID, reason string)
	Chat          func(context.Context, JoinContext, Command) (any, error)
	Games         func(context.Context, JoinContext, Command) (any, error)
	SeatChanged   func(kind, worldID, userID string)
	GameBusy      func(userID string) bool
	auth          SessionChecker
	members       MembershipChecker
	m             world.MapDefinition
	rules         world.Rules
	clock         world.Clock
	rooms         map[string]room
	seats         map[string]*seat
	users         map[string]*seat
	revision      uint64
	accountRev    map[string]uint64
	worldRev      map[string]uint64
	memberRev     map[[2]string]uint64
	allRev        uint64
	closed        bool
	stop, done    chan struct{}
}

func NewHub(a SessionChecker, m MembershipChecker, definition world.MapDefinition, c world.Clock) *Hub {
	rules, err := world.LoadRules(world.RulesAsset)
	if err != nil {
		panic(err)
	}
	h := &Hub{areaStates: map[string]world.ActivitySnapshot{}, rules: rules, accountRev: map[string]uint64{}, worldRev: map[string]uint64{}, memberRev: map[[2]string]uint64{}, auth: a, members: m, m: definition, clock: c, rooms: map[string]room{}, seats: map[string]*seat{}, users: map[string]*seat{}, stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(h.done)
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-h.stop:
				return
			case <-t.C:
				h.expireTick()
			}
		}
	}()
	return h
}
func (h *Hub) expireTick() {
	h.lease.Lock()
	defer h.lease.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.expire()
	for w := range h.rooms {
		h.areaLifecycleLocked(context.Background(), w)
		h.closeEmptyRoom(w)
	}
}
func (h *Hub) Close() {
	h.lease.Lock()
	h.mu.Lock()
	if !h.closed {
		h.closed = true
		close(h.stop)
		for _, s := range h.seats {
			h.remove(s)
		}
		for _, r := range h.rooms {
			r.Close()
		}
	}
	h.mu.Unlock()
	// A selected expiry tick may be waiting for this lease before it can stop.
	h.lease.Unlock()
	<-h.done
}
func (h *Hub) authorize(ctx context.Context, j JoinContext) (auth.Session, uint64, error) {
	h.mu.Lock()
	rev := h.revision
	closed := h.closed
	authChecker, membershipChecker := h.auth, h.members
	h.mu.Unlock()
	if closed {
		return auth.Session{}, rev, world.ErrClosed
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	s, e := authChecker.RequireSession(ctx, j.Token)
	if e != nil {
		return s, rev, ErrUnauthorized
	}
	if _, e = membershipChecker.AuthorizeMember(ctx, s.User.ID, j.WorldID); e != nil {
		return s, rev, ErrUnauthorized
	}
	return s, rev, nil
}
func randomID() string {
	b := make([]byte, 24)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func (h *Hub) Join(ctx context.Context, j JoinContext) (PublicSnapshot, error) {
	user, rev, e := h.authorize(ctx, j)
	if e != nil {
		return PublicSnapshot{}, e
	}
	h.lease.Lock()
	defer h.lease.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed || h.revokedSince(rev, user.User.ID, j.WorldID) {
		return PublicSnapshot{}, ErrUnauthorized
	}
	h.expire()
	old := h.users[user.User.ID]
	if old != nil && !j.Takeover {
		return PublicSnapshot{}, ErrTakeover
	}
	count := 0
	for _, s := range h.seats {
		if s.world == j.WorldID && s != old {
			count++
		}
	}
	if count >= 20 {
		return PublicSnapshot{}, ErrFull
	}

	epoch := uint64(1)
	if old != nil {
		epoch = old.epoch + 1
	}
	return h.admit(ctx, j, user, old, epoch)
}
func (h *Hub) Resume(ctx context.Context, j JoinContext, lastEventID uint64) (PublicSnapshot, error) {
	u, rev, e := h.authorize(ctx, j)
	if e != nil {
		return PublicSnapshot{}, e
	}
	h.lease.Lock()
	defer h.lease.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	h.expire()
	old := h.seats[j.SessionID]
	if h.closed || h.revokedSince(rev, u.User.ID, j.WorldID) || !matches(old, j, u.User.ID) {
		return PublicSnapshot{}, ErrFenced
	}
	// T5 has no durable events: every successful resume is a full resync.
	return h.admit(ctx, j, u, old, old.epoch+1)
}

// admit runs while admission is locked. No new opaque seat is published until
// both runtime mutation and the caller's full snapshot have succeeded.
func (h *Hub) admit(ctx context.Context, j JoinContext, u auth.Session, old *seat, epoch uint64) (PublicSnapshot, error) {
	if e := ctx.Err(); e != nil {
		return PublicSnapshot{}, e
	}
	r := h.rooms[j.WorldID]
	if r == nil {
		r = world.NewRuntime(h.m, h.clock)
		h.rooms[j.WorldID] = r
	}
	candidate := &seat{authorizedUntil: h.authorizationDeadline(u), id: randomID(), user: u.User.ID, token: j.Token, world: j.WorldID, epoch: epoch, done: make(chan struct{}), events: make(chan Delivery, 64)}
	cleanup := func() {
		if old != nil && old.world == j.WorldID {
			h.remove(old)
			return
		}
		r.Remove(context.Background(), u.User.ID)
		h.closeEmptyRoom(j.WorldID)
	}
	if e := r.Join(ctx, u.User.ID, u.User.DisplayName, epoch); e != nil {
		cleanup()
		return PublicSnapshot{}, e
	}
	snapshot, e := h.snapshot(ctx, candidate)
	if e != nil {
		cleanup()
		return PublicSnapshot{}, e
	}
	if old != nil {
		if old.world == j.WorldID {
			h.stopSeat(old)
			delete(h.seats, old.id)
			delete(h.users, old.user)
		} else {
			h.remove(old)
		}
	}
	h.seats[candidate.id] = candidate
	h.users[candidate.user] = candidate
	if h.GameBusy != nil && h.GameBusy(candidate.user) {
		r.Activity(ctx, candidate.user, true, nil)
	}
	if h.SeatChanged != nil {
		h.SeatChanged("resume", candidate.world, candidate.user)
	}
	return snapshot, nil
}

func matches(s *seat, j JoinContext, user string) bool {
	return s != nil && s.user == user && s.token == j.Token && s.world == j.WorldID && s.epoch == j.Epoch
}
func (h *Hub) snapshot(ctx context.Context, s *seat) (PublicSnapshot, error) {
	v, e := h.rooms[s.world].Snapshot(ctx)
	if e == nil && h.GameBusy != nil {
		for i := range v.Avatars {
			if h.GameBusy(v.Avatars[i].ID) {
				v.Avatars[i].Status = "playing"
			} else if v.Avatars[i].Status == "playing" {
				h.rooms[s.world].Activity(ctx, v.Avatars[i].ID, false, nil)
				v.Avatars[i].Status = "available"
			}
		}
	}
	if e == nil && h.VoiceObserve != nil {
		for _, a := range v.Avatars {
			h.VoiceObserve(s.world, a.ID, a.X, a.Y, a.Status == "playing" || h.VoiceExcluded != nil && h.VoiceExcluded(a.ID))
		}
	}
	return PublicSnapshot{Snapshot: v, WorldID: s.world, SelfID: s.user, SessionID: s.id, Epoch: s.epoch, MapID: h.m.ID, MapChecksum: h.m.Checksum, RulesID: h.rules.ID, RulesChecksum: h.rules.Checksum, ProtocolVersion: 2}, e
}
func (h *Hub) Snapshot(ctx context.Context, j JoinContext) (PublicSnapshot, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.seats[j.SessionID]
	if s == nil || !matches(s, j, s.user) || !s.disconnected.IsZero() {
		return PublicSnapshot{}, ErrFenced
	}
	return h.snapshot(ctx, s)
}
func (h *Hub) Validate(ctx context.Context, j JoinContext) error {
	u, rev, e := h.authorize(ctx, j)
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.seats[j.SessionID]
	if e != nil {
		if s != nil && s.token == j.Token {
			h.remove(s)
		}
		return e
	}
	if h.revokedSince(rev, u.User.ID, j.WorldID) || !matches(s, j, u.User.ID) {
		return ErrFenced
	}
	s.authorizedUntil = h.authorizationDeadline(u)
	return nil
}
func (h *Hub) Submit(ctx context.Context, j JoinContext, c Command) error {
	if c.Type == "world.leave" {
		h.lease.Lock()
		defer h.lease.Unlock()
	}
	raw, e := json.Marshal(c)
	if e != nil {
		return e
	}
	c, e = ValidateCommand(raw)
	if e != nil {
		return e
	}
	if c.WorldID != j.WorldID {
		return ErrUnauthorized
	}
	if c.Type == "chat.send" {
		if h.Chat == nil {
			return ErrUnsupported
		}
		_, e := h.Chat(ctx, j, c)
		return e
	}
	if e = h.Validate(ctx, j); e != nil {
		return e
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.seats[j.SessionID]
	if !matches(s, j, func() string {
		if s != nil {
			return s.user
		}
		return ""
	}()) || !s.disconnected.IsZero() {
		return ErrFenced
	}
	r := h.rooms[s.world]
	switch c.Type {
	case "activity.ride", "activity.kick", "activity.claw", "activity.fishing":
		if h.GameBusy != nil && h.GameBusy(s.user) {
			return ErrUnsupported
		}
		owner, ok := r.(interface {
			ResumeCasual(context.Context, string, uint64) error
			ApplyCasualIntent(context.Context, string, uint64, uint64, world.CasualIntent) error
		})
		if !ok {
			return ErrUnsupported
		}
		if h.GameBusy != nil {
			if err := owner.ResumeCasual(ctx, s.user, s.epoch); err != nil {
				return err
			}
		}
		var intent world.CasualIntent
		if err := json.Unmarshal(c.Payload, &intent); err != nil {
			return err
		}
		err := owner.ApplyCasualIntent(ctx, s.user, s.epoch, c.Seq, intent)
		if err == nil {
			if (intent.Action == "mount" || intent.Action == "claw.enter" || intent.Action == "fishing.cast") && h.VoiceRevoke != nil {
				h.VoiceRevoke(s.world, s.user, "ride entry")
			}
			h.areaLifecycleLocked(ctx, s.world)
		}
		return err
	case "player.input":
		if h.GameBusy != nil && h.GameBusy(s.user) {
			return ErrUnsupported
		}
		if h.GameBusy != nil {
			r.Activity(ctx, s.user, false, nil)
		}
		var p struct{ DX, DY float64 }
		json.Unmarshal(c.Payload, &p)
		return r.Input(ctx, s.user, s.epoch, c.Seq, p.DX, p.DY)
	case "player.destination":
		if h.GameBusy != nil && h.GameBusy(s.user) {
			return ErrUnsupported
		}
		if h.GameBusy != nil {
			r.Activity(ctx, s.user, false, nil)
		}
		var p struct{ X, Y float64 }
		json.Unmarshal(c.Payload, &p)
		return r.Destination(ctx, s.user, s.epoch, c.Seq, p.X, p.Y)
	case "world.leave":
		h.remove(s)
		return nil
	default:
		return ErrUnsupported
	}
}
func (h *Hub) Done(j JoinContext) <-chan struct{} {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.seats[j.SessionID]
	if s != nil && s.epoch == j.Epoch {
		return s.done
	}
	c := make(chan struct{})
	close(c)
	return c
}
func (h *Hub) stopSeat(s *seat) {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}
func (h *Hub) Disconnect(sessionID, reason string) {
	h.lease.Lock()
	defer h.lease.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	if s := h.seats[sessionID]; s != nil && s.disconnected.IsZero() {
		h.stopSeat(s)
		s.disconnected = h.clock.Now()
		if h.SeatChanged != nil {
			h.SeatChanged("disconnect", s.world, s.user)
		}
		h.rooms[s.world].Idle(context.Background(), s.user)
	}
}
func (h *Hub) remove(s *seat) {
	if h.SeatChanged != nil {
		h.SeatChanged("leave", s.world, s.user)
	}
	h.stopSeat(s)
	h.rooms[s.world].Remove(context.Background(), s.user)
	delete(h.seats, s.id)
	delete(h.users, s.user)
	h.closeEmptyRoom(s.world)
}
func (h *Hub) closeEmptyRoom(worldID string) {
	for _, other := range h.seats {
		if other.world == worldID {
			return
		}
	}
	if r := h.rooms[worldID]; r != nil {
		if owner, ok := r.(areaRoom); ok {
			active, err := owner.HasAreaLeases(context.Background())
			if err != nil || active {
				return
			}
		}
		r.Close()
		delete(h.rooms, worldID)
	}
}
func (h *Hub) expire() {
	for _, s := range h.seats {
		if !s.disconnected.IsZero() && h.clock.Now().Sub(s.disconnected) >= 30*time.Second {
			h.remove(s)
		}
	}
}
func (h *Hub) Revoke(worldID, userID string) {
	h.lease.Lock()
	defer h.lease.Unlock()
	h.revokeLocked(worldID, userID)
}
func (h *Hub) revokeLocked(worldID, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.revision++
	switch {
	case worldID == "" && userID == "":
		h.allRev = h.revision
	case worldID == "":
		h.accountRev[userID] = h.revision
	case userID == "":
		h.worldRev[worldID] = h.revision
	default:
		h.memberRev[[2]string{worldID, userID}] = h.revision
	}
	for _, s := range h.seats {
		if (worldID == "" || s.world == worldID) && (userID == "" || s.user == userID) {
			h.remove(s)
		}
	}
}

// Only a revoke affecting this subject invalidates an authorization read in flight.
func (h *Hub) revokedSince(revision uint64, userID, worldID string) bool {
	return h.allRev > revision || h.accountRev[userID] > revision || h.worldRev[worldID] > revision || h.memberRev[[2]string{worldID, userID}] > revision
}

// Leave releases only the seat bound to this authenticated transport, including
// after its peer has closed. A stale transport can never release a rotated seat.
func (h *Hub) Leave(j JoinContext) {
	h.lease.Lock()
	defer h.lease.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.seats[j.SessionID]
	if s != nil && matches(s, j, s.user) {
		h.remove(s)
	}
}

// A short authorization lease allows deadline input without synchronous SQL.
// The existing transport revalidates every second; known expiry wins immediately.
func (h *Hub) authorizationDeadline(u auth.Session) time.Time {
	until := h.clock.Now().Add(2 * time.Second)
	for _, expiry := range []time.Time{u.ExpiresAt, u.IdleExpiresAt} {
		if !expiry.IsZero() && expiry.Before(until) {
			until = expiry
		}
	}
	return until
}

// DeleteAccount serializes against admission/input, then drains permanent game
// departures outside mu before bounded SQL. Failure may disconnect the caller,
// but must not erase identity or boards. No callback may reacquire lease.
func (h *Hub) DeleteAccount(ctx context.Context, user string, syncPresence func(context.Context) error, commit func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	h.lease.Lock()
	defer h.lease.Unlock()
	h.revokeLocked("", user)
	if e := syncPresence(ctx); e != nil {
		return e
	}
	return commit(ctx)
}
