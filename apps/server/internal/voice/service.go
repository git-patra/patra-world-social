// Package voice owns ephemeral table seats. Lock order is hub admission -> mu;
// no code holding mu calls the hub, SQL, or a provider network method.
package voice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"patra-world-social/server/internal/realtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var (
	ErrForbidden     = errors.New("voice admission denied")
	ErrUnavailable   = errors.New("Voice belum tersedia atau isolasi sedang dipulihkan. Chat dunia tetap tersedia.")
	ErrOutside       = errors.New("Berjalan ke area meja dan keluar dari permainan dahulu.")
	ErrFull          = errors.New("Meja penuh: maksimal 8 peserta.")
	ErrAlreadyJoined = errors.New("Keluar meja lama sebelum bergabung ke meja lain.")
	ErrRetryExpired  = chatError{code: "voice_retry_expired", message: "Pesan ini sudah diproses; konfirmasi lama telah kedaluwarsa. Sinkronkan meja sebelum mengirim pesan baru."}
	ErrRate          = errors.New("Maksimal 5 pesan per 5 detik.")
	ErrInput         = errors.New("Pesan harus 1–500 karakter.")
)

type chatError struct{ code, message string }

func (e chatError) Error() string       { return e.message }
func (e chatError) CommandCode() string { return e.code }

type JoinContext = realtime.JoinContext
type Subject struct {
	UserID, Name, Role string
	X, Y               float64
	Playing            bool
}
type Gate interface {
	Within(context.Context, JoinContext, func(Subject) error) error
}
type Record struct{ ID, Room string }
type Journal interface {
	Save(context.Context, Record) error
	Load(context.Context) ([]Record, error)
	Delete(context.Context, string) error
}
type Provider interface {
	Token(VoiceAdmission) (VoiceToken, error)
	Remove(context.Context, Record, int64) error
	Rooms(context.Context) (map[string][]string, error)
	DeleteRoom(context.Context, string) error
	Mute(context.Context, Record) error
}
type VoiceAdmission struct {
	ID       string    `json:"id"`
	Identity string    `json:"identity"`
	Room     string    `json:"-"`
	WorldID  string    `json:"worldId"`
	UserID   string    `json:"userId"`
	Name     string    `json:"name"`
	Table    int       `json:"table"`
	JoinedAt time.Time `json:"joinedAt"`
	Muted    bool      `json:"muted"`
}
type VoiceToken struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}
type Message struct {
	ID              string    `json:"id"`
	UserID          string    `json:"userId"`
	Name            string    `json:"name"`
	ClientMessageID string    `json:"clientMessageId"`
	Text            string    `json:"text"`
	CreatedAt       time.Time `json:"createdAt"`
}
type State struct {
	NextMessageSeq uint64           `json:"nextMessageSeq"`
	Admission      VoiceAdmission   `json:"admission"`
	Participants   []VoiceAdmission `json:"participants"`
	Messages       []Message        `json:"messages"`
	HostID         string           `json:"hostId"`
	CanModerate    bool             `json:"canModerate"`
}
type entry struct {
	lastMessageSeq   uint64
	acknowledgements map[uint64]Message
	VoiceAdmission
	j                                        JoinContext
	active, persisting, revoked, mutePending bool
	journalPending                           bool
	muteVersion                              uint64
}
type table struct {
	messages   []Message
	emptySince time.Time
}
type bucket struct {
	tokens float64
	at     time.Time
}
type Service struct {
	mu                   sync.Mutex
	worker               sync.Mutex
	gate                 Gate
	provider             Provider
	journal              Journal
	prefix               string
	entries              map[string]*entry
	users                map[string]string
	tables               map[string]*table
	limits               map[string]bucket
	initialized, healthy bool
	cleaning             map[string]bool
	// Audit must persist only actor/target/action metadata, never chat/audio.
	Audit func(context.Context, string, string, string, string) error
}

func New(g Gate, p Provider, j Journal, prefix string) *Service {
	return &Service{gate: g, provider: p, journal: j, prefix: prefix, entries: map[string]*entry{}, users: map[string]string{}, tables: map[string]*table{}, limits: map[string]bucket{}, cleaning: map[string]bool{}}
}
func randomID() string {
	var b [24]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}
func (s *Service) room(world string, t int) string {
	h := sha256.Sum256([]byte(world))
	return fmt.Sprintf("%s%x-%d", s.prefix, h[:16], t)
}
func (s *Service) available() bool {
	if s.provider == nil || !s.initialized || !s.healthy {
		return false
	}
	for _, e := range s.entries {
		if e.revoked {
			return false
		}
	}
	return true
}
func (s *Service) Status() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.provider == nil {
		return "unavailable"
	}
	if !s.available() {
		return "degraded"
	}
	return "ready"
}
func (s *Service) JoinTable(ctx context.Context, j JoinContext, t int) (VoiceAdmission, error) {
	var a VoiceAdmission
	// Reserve under the same admission/position fence used by token issuance.
	err := s.gate.Within(ctx, j, func(u Subject) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		if !s.available() {
			return ErrUnavailable
		}
		if !InTable(t, u.X, u.Y) || u.Playing {
			return ErrOutside
		}
		if _, ok := s.users[u.UserID]; ok {
			return ErrAlreadyJoined
		}
		room := s.room(j.WorldID, t)
		if s.cleaning[room] {
			return ErrUnavailable
		}
		count := 0
		for _, e := range s.entries {
			if !e.revoked && e.Room == room {
				count++
			}
		}
		if count >= 8 {
			return ErrFull
		}
		id := randomID()
		a = VoiceAdmission{ID: id, Identity: id, Room: room, WorldID: j.WorldID, UserID: u.UserID, Name: u.Name, Table: t, JoinedAt: time.Now()}
		s.entries[id] = &entry{VoiceAdmission: a, j: j, persisting: true}
		s.users[u.UserID] = id
		return nil
	})
	if err != nil {
		return a, err
	}
	// Commit-unknown must never yield a token. The reserved identity is retained
	// until an explicit provider revocation and journal deletion both succeed.
	storeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	err = s.journal.Save(storeCtx, Record{a.ID, a.Room})
	cancel()
	s.mu.Lock()
	e := s.entries[a.ID]
	e.persisting = false
	if err != nil {
		e.journalPending = true
		s.revoke(e)
	}
	s.mu.Unlock()
	if err != nil {
		return VoiceAdmission{}, err
	}
	err = s.gate.Within(ctx, j, func(u Subject) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		e := s.entries[a.ID]
		if e == nil || e.revoked || !s.available() {
			return ErrUnavailable
		}
		if !InTable(t, u.X, u.Y) || u.Playing {
			return ErrOutside
		}
		e.active = true
		e.JoinedAt = time.Now()
		a = e.VoiceAdmission
		if previous := s.tables[a.Room]; previous == nil || (!previous.emptySince.IsZero() && time.Since(previous.emptySince) >= 5*time.Minute) {
			s.tables[a.Room] = &table{messages: []Message{}}
		}
		s.tables[a.Room].emptySince = time.Time{}
		return nil
	})
	if err != nil {
		s.mu.Lock()
		s.revoke(s.entries[a.ID])
		s.mu.Unlock()
		return VoiceAdmission{}, err
	}
	return a, nil
}
func (s *Service) with(ctx context.Context, id string, fn func(*entry, Subject) error) error {
	s.mu.Lock()
	e := s.entries[id]
	if e == nil || e.revoked || !e.active {
		s.mu.Unlock()
		return ErrForbidden
	}
	j := e.j
	s.mu.Unlock()
	return s.gate.Within(ctx, j, func(u Subject) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		e := s.entries[id]
		if e == nil || e.revoked || !e.active || e.UserID != u.UserID {
			return ErrForbidden
		}
		if u.Playing || !InTable(e.Table, u.X, u.Y) {
			s.revoke(e)
			return ErrOutside
		}
		return fn(e, u)
	})
}
func (s *Service) IssueToken(ctx context.Context, id string) (VoiceToken, error) {
	var token VoiceToken
	err := s.with(ctx, id, func(e *entry, _ Subject) error {
		if !s.available() {
			return ErrUnavailable
		}
		var err error
		token, err = s.provider.Token(e.VoiceAdmission)
		return err
	})
	return token, err
}
func (s *Service) Current(user string) (VoiceAdmission, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entries[s.users[user]]
	if e == nil || !e.active || e.revoked {
		return VoiceAdmission{}, false
	}
	return e.VoiceAdmission, true
}
func (s *Service) revoke(e *entry) {
	if e == nil || e.revoked {
		return
	}
	e.revoked = true
	e.acknowledgements = nil
	e.lastMessageSeq = 0
	e.j = JoinContext{}
	e.active = false
	delete(s.users, e.UserID)
	if t := s.tables[e.Room]; t != nil {
		occupied := false
		for _, other := range s.entries {
			if other.Room == e.Room && !other.revoked {
				occupied = true
			}
		}
		if !occupied && t.emptySince.IsZero() {
			t.emptySince = time.Now()
		}
	}
}

// Revoke is synchronous local invalidation only; safe under hub.mu.
func (s *Service) Revoke(world, user, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.entries {
		if (world == "" || e.WorldID == world) && (user == "" || e.UserID == user) {
			s.revoke(e)
		}
	}
}
func (s *Service) Observe(world, user string, x, y float64, playing bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entries[s.users[user]]
	if e != nil && e.WorldID == world && (playing || !InTable(e.Table, x, y)) {
		s.revoke(e)
	}
}
func (s *Service) LeaveTable(_ context.Context, id, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revoke(s.entries[id])
	return nil
}
func (s *Service) State(ctx context.Context, id string) (State, error) {
	state := State{Participants: []VoiceAdmission{}, Messages: []Message{}}
	err := s.with(ctx, id, func(e *entry, u Subject) error {
		state.Admission = e.VoiceAdmission
		state.NextMessageSeq = e.lastMessageSeq + 1
		state.CanModerate = u.Role == "owner" || u.Role == "moderator"
		for _, p := range s.entries {
			if p.active && !p.revoked && p.Room == e.Room {
				state.Participants = append(state.Participants, p.VoiceAdmission)
			}
		}
		sort.Slice(state.Participants, func(i, j int) bool { return state.Participants[i].JoinedAt.Before(state.Participants[j].JoinedAt) })
		if len(state.Participants) > 0 {
			state.HostID = state.Participants[0].UserID
		}
		state.Messages = append(state.Messages, s.tables[e.Room].messages...)
		return nil
	})
	return state, err
}
func (s *Service) Chat(ctx context.Context, id, clientID, text string) (Message, error) {
	var m Message
	if !utf8.ValidString(text) || !utf8.ValidString(clientID) || len(clientID) < 1 || len(clientID) > 128 || utf8.RuneCountInString(text) > 500 || strings.TrimSpace(text) == "" {
		return m, ErrInput
	}
	err := s.with(ctx, id, func(e *entry, _ Subject) error {
		t := s.tables[e.Room]
		if !strings.HasPrefix(clientID, e.ID+":") {
			return ErrInput
		}
		suffix := strings.TrimPrefix(clientID, e.ID+":")
		seq, err := strconv.ParseUint(suffix, 10, 64)
		if err != nil || seq == 0 || seq > 9007199254740991 || strconv.FormatUint(seq, 10) != suffix {
			return ErrInput
		}
		if seq <= e.lastMessageSeq {
			if old, ok := e.acknowledgements[seq]; ok {
				m = old
				return nil
			}
			return ErrRetryExpired
		}
		if seq != e.lastMessageSeq+1 {
			return ErrInput
		}

		now := time.Now()
		b, ok := s.limits[e.UserID]
		if !ok {
			b = bucket{5, now}
		}
		b.tokens = min(5, b.tokens+now.Sub(b.at).Seconds())
		b.at = now
		if b.tokens < 1 {
			s.limits[e.UserID] = b
			return ErrRate
		}
		b.tokens--
		s.limits[e.UserID] = b
		m = Message{randomID(), e.UserID, e.Name, clientID, text, now}
		if e.acknowledgements == nil {
			e.acknowledgements = map[uint64]Message{}
		}
		e.lastMessageSeq = seq
		e.acknowledgements[seq] = m
		if seq > 32 {
			delete(e.acknowledgements, seq-32)
		}
		t.messages = append(t.messages, m)
		if len(t.messages) > 200 {
			t.messages = t.messages[len(t.messages)-200:]
		}
		return nil
	})
	return m, err
}

// Moderation is authorized twice, with auditing outside every owner/mutex.
func (s *Service) Moderate(ctx context.Context, id, target, action string) error {
	if action != "mute" && action != "kick" {
		return ErrForbidden
	}
	var actor, world, user string
	check := func(e *entry, u Subject) error {
		p := s.entries[target]
		if (u.Role != "owner" && u.Role != "moderator") || p == nil || !p.active || p.revoked || p.Room != e.Room {
			return ErrForbidden
		}
		actor = u.UserID
		world = e.WorldID
		user = p.UserID
		return nil
	}
	if err := s.with(ctx, id, check); err != nil {
		return err
	}
	if s.Audit == nil {
		return ErrUnavailable
	}
	auditCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := s.Audit(auditCtx, world, actor, user, action); err != nil {
		return err
	}
	return s.with(ctx, id, func(e *entry, u Subject) error {
		if err := check(e, u); err != nil {
			return err
		}
		p := s.entries[target]
		if action == "kick" {
			s.revoke(p)
		} else {
			p.Muted = true
			p.mutePending = true
			p.muteVersion++
		}
		return nil
	})
}
func (s *Service) Unmute(ctx context.Context, id string) error {
	return s.with(ctx, id, func(e *entry, _ Subject) error {
		if e.mutePending {
			return ErrUnavailable
		}
		e.Muted = false
		return nil
	})
}

// Owned prevents an old transport from operating on a replacement admission.
func (s *Service) Owned(j JoinContext, user string) (VoiceAdmission, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entries[s.users[user]]
	if e == nil || !e.active || e.revoked || e.j.Token != j.Token || e.j.SessionID != j.SessionID || e.j.Epoch != j.Epoch || e.j.WorldID != j.WorldID {
		return VoiceAdmission{}, false
	}
	return e.VoiceAdmission, true
}
