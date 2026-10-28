// Package lobby owns arena rosters and match admission. Simulation and SQL never
// share an owner loop: starts reserve, persist outside it, then activate.
package lobby

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"patra-world-social/server/internal/games/model"
	"sync"
	"time"
)

var (
	ErrBusy          = errors.New("already busy or arena starting")
	ErrFull          = errors.New("arena full; spectate after start")
	ErrForbidden     = errors.New("lobby action forbidden")
	ErrUnavailable   = errors.New("concrete game engine unavailable")
	ErrNotReady      = errors.New("all humans must be ready; penalty needs two players or explicit bot")
	ErrClosed        = errors.New("lobby closed")
	ErrQueue         = errors.New("lobby queue full")
	ErrCommitUnknown = errors.New("commit acknowledgement unknown")
	ErrUnconfirmed   = errors.New("match start storage unconfirmed; retry same command")
)

type Clock interface{ Now() time.Time }
type Store interface {
	Persist(context.Context, model.MatchStart, string, string) error
	Lookup(context.Context, string) (bool, error)
	Schedule(context.Context, string, int64) error
	Terminal(context.Context, string, string) error
	ReleaseUser(context.Context, string, string) error
}

// JoinContext is constructed only after current session/seat authorization.
type JoinContext struct{ UserID, WorldID, DisplayName, AvatarID string }
type Player struct {
	model.Actor
	Ready         bool `json:"ready"`
	Connected     bool `json:"connected"`
	BotAssisted   bool `json:"botAssisted"`
	disconnected  time.Time
	departed      bool
	departedAt    time.Time
	departureKind string
	releaseQueued bool
	claimReleased bool
	releaseAt     time.Time
}
type Lobby struct {
	Arena
	ID              string   `json:"id"`
	WorldID         string   `json:"worldId"`
	HostID          string   `json:"hostId"`
	State           string   `json:"state"`
	Players         []Player `json:"players"`
	Spectators      []string `json:"spectators"`
	MatchID         string   `json:"matchId,omitempty"`
	StartsAt        int64    `json:"startsAt,omitempty"`
	EngineAvailable bool     `json:"engineAvailable"`
	BotChosen       bool     `json:"botChosen"`
}
type entry struct {
	rematch map[string]bool
	Lobby
	start           *attempt
	engine          Engine
	finalizingAt    time.Time
	releasing       bool
	resultAction    bool
	rematchApproved bool
	releaseAt       time.Time
}
type attempt struct {
	match         model.MatchStart
	host, command string
	done          chan struct{}
	err           error
	engine        Engine
	durable       bool
	reconcileOnly bool
	retryAt       time.Time
}
type request struct {
	fn   func()
	done chan struct{}
}
type notification struct {
	kind, world, user string
	at                time.Time
	disconnectedAt    time.Time
}
type Return struct {
	WorldID, UserID string
	X, Y            int
	Generation      uint64
}
type Changes struct {
	Events  []GameEvent
	Worlds  []string
	Returns []Return
}
type Service struct {
	storageBlocked bool
	store          Store
	clock          Clock
	version        model.Version
	ctx            context.Context
	cancel         context.CancelFunc
	workers        sync.WaitGroup
	storage        chan func()
	requests       chan request
	stop, done     chan struct{}
	once           sync.Once
	entries        map[[2]string]*entry
	factories      map[string]Factory
	preparers      map[string]Prepare
	rulesMetadata  map[string]RulesMetadata
	attempts       map[[2]string]*attempt
	busyMu         sync.RWMutex
	busy           map[string]string
	spectators     map[string]bool
	generations    map[string]uint64
	pendingMu      sync.Mutex
	pending        map[[2]string][]notification
	wake           chan struct{}
	changed        map[string]bool
	returns        map[[2]string]Return
	Changed        chan struct{}
}

func New(store Store, clock Clock, version model.Version) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{ctx: ctx, cancel: cancel, storage: make(chan func(), 64), store: store, clock: clock, version: version, requests: make(chan request, 128), stop: make(chan struct{}), done: make(chan struct{}), entries: map[[2]string]*entry{}, factories: map[string]Factory{}, preparers: map[string]Prepare{}, rulesMetadata: map[string]RulesMetadata{}, attempts: map[[2]string]*attempt{}, busy: map[string]string{}, spectators: map[string]bool{}, generations: map[string]uint64{}, pending: map[[2]string][]notification{}, wake: make(chan struct{}, 1), changed: map[string]bool{}, returns: map[[2]string]Return{}, Changed: make(chan struct{}, 1)}
	s.workers.Add(1)
	go func() {
		defer s.workers.Done()
		for {
			select {
			case <-s.ctx.Done():
				return
			case fn := <-s.storage:
				fn()
			}
		}
	}()
	go s.run()
	return s
}
func (s *Service) run() {
	defer close(s.done)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			for _, e := range s.entries {
				if e.engine != nil {
					e.engine.Close()
				} else if e.start != nil && e.start.engine != nil {
					e.start.engine.Close()
				}
			}
			return
		case <-ticker.C:
			s.drainPresence()
			s.advance()
			s.refreshSpectators()
		case <-s.wake:
			s.drainPresence()
			s.advance()
			s.refreshSpectators()
		case q := <-s.requests:
			s.drainPresence()
			s.advance()
			q.fn()
			s.refreshSpectators()
			close(q.done)
		}
	}
}
func (s *Service) call(ctx context.Context, fn func()) error {
	q := request{fn, make(chan struct{})}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return ErrClosed
	default:
	}
	select {
	case s.requests <- q:
	default:
		return ErrQueue
	}
	select {
	case <-q.done:
		return nil
	case <-s.done:
		return ErrClosed
	}
}
func (s *Service) Close() {
	s.once.Do(func() { s.cancel(); close(s.stop) })
	<-s.done
	s.workers.Wait()
}
func (s *Service) Sync(ctx context.Context) error { return s.call(ctx, func() {}) }
func id() string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
func (s *Service) Register(game string, f Factory) error { return s.RegisterPrepared(game, f, nil) }
func (s *Service) RegisterPrepared(game string, f Factory, prepare Prepare) error {
	var e error
	err := s.call(context.Background(), func() {
		if game != "race" && game != "penalty" && game != "duck" || f == nil || s.factories[game] != nil {
			e = ErrForbidden
			return
		}
		s.factories[game] = f
		s.preparers[game] = prepare
	})
	if err != nil {
		return err
	}
	return e
}
func (s *Service) mark(world string) {
	s.changed[world] = true
	select {
	case s.Changed <- struct{}{}:
	default:
	}
}
func (s *Service) claim(user, lobby string) {
	s.busyMu.Lock()
	if lobby == "" {
		delete(s.busy, user)
	} else {
		s.busy[user] = lobby
		s.generations[user]++
	}
	s.busyMu.Unlock()
}
func (s *Service) Busy(user string) bool {
	s.busyMu.RLock()
	defer s.busyMu.RUnlock()
	return s.busy[user] != ""
}
func copyLobby(l Lobby) Lobby {
	l.Players = append([]Player{}, l.Players...)
	l.Spectators = append([]string{}, l.Spectators...)
	return l
}
func (s *Service) List(ctx context.Context, world string) ([]Lobby, error) {
	out := []Lobby{}
	e := s.call(ctx, func() {
		for _, a := range arenas {
			v := Lobby{Arena: a, WorldID: world, State: "idle", Players: []Player{}, Spectators: []string{}, EngineAvailable: s.factories[a.GameType] != nil}
			if l := s.entries[[2]string{world, a.ID}]; l != nil {
				v = copyLobby(l.Lobby)
				v.EngineAvailable = s.factories[a.GameType] != nil
			}
			out = append(out, v)
		}
	})
	return out, e
}
func (s *Service) find(lobby string) *entry {
	for _, e := range s.entries {
		if e.ID == lobby {
			return e
		}
	}
	return nil
}
func (s *Service) JoinLobby(ctx context.Context, j JoinContext, arenaID string) (Lobby, error) {
	var out Lobby
	var e error
	err := s.call(ctx, func() {
		a, ok := arena(arenaID)
		if !ok || j.UserID == "" || j.WorldID == "" {
			e = ErrForbidden
			return
		}
		key := [2]string{j.WorldID, arenaID}
		l := s.entries[key]
		if l != nil {
			for _, p := range l.Players {
				if p.UserID == j.UserID && !p.departed {
					out = copyLobby(l.Lobby)
					return
				}
			}
		}
		if s.Busy(j.UserID) {
			e = ErrBusy
			return
		}
		if l == nil {
			l = &entry{Lobby: Lobby{Arena: a, ID: id(), WorldID: j.WorldID, State: "lobby", HostID: j.UserID, Players: []Player{}, Spectators: []string{}, EngineAvailable: s.factories[a.GameType] != nil}}
			s.entries[key] = l
		}
		if l.State != "lobby" {
			for _, u := range l.Spectators {
				if u == j.UserID {
					out = copyLobby(l.Lobby)
					return
				}
			}
			if len(l.Spectators) >= 20 {
				e = ErrFull
				return
			}
			l.Spectators = append(l.Spectators, j.UserID)
		} else {
			if len(l.Players) >= a.Capacity {
				e = ErrFull
				return
			}
			l.Players = append(l.Players, Player{Actor: model.Actor{ActorID: id(), ActorType: "human", UserID: j.UserID, DisplayName: j.DisplayName, AvatarID: j.AvatarID}, Connected: true})
			s.claim(j.UserID, l.ID)
		}
		out = copyLobby(l.Lobby)
		s.mark(j.WorldID)
	})
	if err != nil {
		return out, err
	}
	return out, e
}
func (s *Service) SetReady(ctx context.Context, user, lobby string, ready bool) error {
	return s.edit(ctx, user, lobby, func(l *entry, p *Player) error { p.Ready = ready; return nil })
}
func (s *Service) ChooseBot(ctx context.Context, user, lobby string, chosen bool) error {
	return s.edit(ctx, user, lobby, func(l *entry, p *Player) error {
		if l.GameType != "penalty" || l.HostID != user {
			return ErrForbidden
		}
		l.BotChosen = chosen
		return nil
	})
}
func (s *Service) edit(ctx context.Context, user, lobby string, fn func(*entry, *Player) error) error {
	var e = ErrForbidden
	err := s.call(ctx, func() {
		l := s.find(lobby)
		if l == nil || l.State != "lobby" {
			return
		}
		for i := range l.Players {
			if l.Players[i].UserID == user {
				e = fn(l, &l.Players[i])
				s.mark(l.WorldID)
				return
			}
		}
	})
	if err != nil {
		return err
	}
	return e
}

func (s *Service) releaseClaim(user, lobby string) bool {
	s.busyMu.Lock()
	defer s.busyMu.Unlock()
	if s.busy[user] == lobby {
		delete(s.busy, user)
		return true
	}
	return false
}

// control is reserved for completion of accepted operations. Internal workers
// wait for space in the same bounded queue; a transient full queue must never
// drop a durable start or claim-release completion.
func (s *Service) control(fn func()) error {
	q := request{fn, make(chan struct{})}
	select {
	case s.requests <- q:
	case <-s.done:
		return ErrClosed
	}
	select {
	case <-q.done:
		return nil
	case <-s.done:
		return ErrClosed
	}
}

// VoiceExcluded includes spectators while Busy deliberately remains player-only.
// This mirror is published by the owner before each request is acknowledged.
func (s *Service) VoiceExcluded(user string) bool {
	s.busyMu.RLock()
	defer s.busyMu.RUnlock()
	return s.busy[user] != "" || s.spectators[user]
}
func (s *Service) refreshSpectators() {
	next := map[string]bool{}
	for _, l := range s.entries {
		for _, u := range l.Spectators {
			next[u] = true
		}
	}
	s.busyMu.Lock()
	s.spectators = next
	s.busyMu.Unlock()
}
