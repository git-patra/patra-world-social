package voice

import (
	"context"
	"errors"
	"fmt"
	"patra-world-social/server/internal/realtime"
	"sync"
	"testing"
	"time"
)

type fakeGate struct {
	mu       sync.Mutex
	subjects map[string]Subject
}

func (g *fakeGate) Within(ctx context.Context, j JoinContext, fn func(Subject) error) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	u, ok := g.subjects[j.Token]
	if !ok {
		return ErrForbidden
	}
	return fn(u)
}

type memoryJournal struct {
	mu        sync.Mutex
	rows      map[string]Record
	save      func() error
	afterSave func() error
}

func (j *memoryJournal) Save(ctx context.Context, r Record) error {
	if j.save != nil {
		if e := j.save(); e != nil {
			return e
		}
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	j.rows[r.ID] = r
	if j.afterSave != nil {
		return j.afterSave()
	}
	return nil
}
func (j *memoryJournal) Load(context.Context) ([]Record, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	var r []Record
	for _, v := range j.rows {
		r = append(r, v)
	}
	return r, nil
}
func (j *memoryJournal) Delete(_ context.Context, id string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	delete(j.rows, id)
	return nil
}

type fakeProvider struct {
	fail    bool
	removed []Record
	rooms   map[string][]string
}

func (p *fakeProvider) Token(a VoiceAdmission) (VoiceToken, error) {
	return VoiceToken{Token: "test-only", URL: "wss://test.invalid"}, nil
}
func (p *fakeProvider) Remove(_ context.Context, r Record, cutoff int64) error {
	if p.fail {
		return errors.New("network blocked")
	}
	if cutoff <= time.Now().Unix() {
		return errors.New("missing strict cutoff")
	}
	p.removed = append(p.removed, r)
	return nil
}
func (p *fakeProvider) Rooms(context.Context) (map[string][]string, error) { return p.rooms, nil }
func (p *fakeProvider) DeleteRoom(context.Context, string) error           { return nil }
func (p *fakeProvider) Mute(context.Context, Record) error                 { return nil }
func setup(t *testing.T) (*Service, *fakeGate, *fakeProvider, *memoryJournal) {
	t.Helper()
	g := &fakeGate{subjects: map[string]Subject{}}
	p := &fakeProvider{rooms: map[string][]string{}}
	j := &memoryJournal{rows: map[string]Record{}}
	s := New(g, p, j, "test-")
	if e := s.ReconcileParticipants(context.Background()); e != nil {
		t.Fatal(e)
	}
	return s, g, p, j
}
func joined(g *fakeGate, user, world string, table int) JoinContext {
	g.subjects[user] = Subject{UserID: user, Name: user, X: TableZones.Tables[table].X, Y: TableZones.Tables[table].Y}
	return realtime.JoinContext{Token: user, WorldID: world, SessionID: user, Epoch: 1}
}
func TestSeatsAndIsolation(t *testing.T) {
	s, g, _, _ := setup(t)
	ctx := context.Background()
	for i := 0; i < 8; i++ {
		j := joined(g, fmt.Sprint(i), "w", 0)
		a, e := s.JoinTable(ctx, j, 0)
		if e != nil || a.Identity == j.Token {
			t.Fatalf("seat %d: %+v %v", i, a, e)
		}
	}
	if _, e := s.JoinTable(ctx, joined(g, "nine", "w", 0), 0); !errors.Is(e, ErrFull) {
		t.Fatal(e)
	}
	if _, e := s.JoinTable(ctx, joined(g, "other", "w", 1), 1); e != nil {
		t.Fatal(e)
	}
	if _, e := s.JoinTable(ctx, joined(g, "another", "w2", 0), 0); e != nil {
		t.Fatal(e)
	}
	if _, e := s.JoinTable(ctx, joined(g, "0", "w2", 0), 0); !errors.Is(e, ErrAlreadyJoined) {
		t.Fatal(e)
	}
}
func TestAdmissionRejectsUntrustedAndOutside(t *testing.T) {
	s, g, _, _ := setup(t)
	ctx := context.Background()
	for _, token := range []string{"guest", "outsider", "banned"} {
		if _, e := s.JoinTable(ctx, JoinContext{Token: token, WorldID: "w"}, 0); e == nil {
			t.Fatal(token)
		}
	}
	j := joined(g, "u", "w", 0)
	g.subjects["u"] = Subject{UserID: "u", X: 32, Y: 32}
	if _, e := s.JoinTable(ctx, j, 0); !errors.Is(e, ErrOutside) {
		t.Fatal(e)
	}
	g.subjects["u"] = Subject{UserID: "u", X: TableZones.Tables[0].X, Y: TableZones.Tables[0].Y, Playing: true}
	if _, e := s.JoinTable(ctx, j, 0); !errors.Is(e, ErrOutside) {
		t.Fatal(e)
	}
}
func TestRevokeRetriesBlockMintAndAdmission(t *testing.T) {
	s, g, p, journal := setup(t)
	ctx := context.Background()
	j := joined(g, "u", "w", 0)
	a, e := s.JoinTable(ctx, j, 0)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.IssueToken(ctx, a.ID); e != nil {
		t.Fatal(e)
	}
	p.fail = true
	s.Revoke("w", "u", "logout")
	if _, e = s.IssueToken(ctx, a.ID); e == nil {
		t.Fatal("revoked token minted")
	}
	if _, e = s.JoinTable(ctx, joined(g, "v", "w", 0), 0); !errors.Is(e, ErrUnavailable) {
		t.Fatal(e)
	}
	if e = s.ReconcileParticipants(ctx); e == nil {
		t.Fatal("removal failure hidden")
	}
	if len(journal.rows) != 1 {
		t.Fatal("lost revocation journal")
	}
	p.fail = false
	if e = s.ReconcileParticipants(ctx); e != nil {
		t.Fatal(e)
	}
	b, e := s.JoinTable(ctx, j, 0)
	if e != nil || a.Identity == b.Identity {
		t.Fatalf("%+v %v", b, e)
	}
}
func TestRestartRevokesNeverConnectedToken(t *testing.T) {
	s, g, p, journal := setup(t)
	ctx := context.Background()
	a, e := s.JoinTable(ctx, joined(g, "u", "w", 0), 0)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.IssueToken(ctx, a.ID); e != nil {
		t.Fatal(e)
	}
	restart := New(g, p, journal, "test-")
	if restart.Status() != "degraded" {
		t.Fatal(restart.Status())
	}
	if e = restart.ReconcileParticipants(ctx); e != nil {
		t.Fatal(e)
	}
	if len(p.removed) != 1 || p.removed[0].ID != a.ID {
		t.Fatal("never-connected identity not revoked")
	}
}
func TestCancellationDuringJournalWriteCannotInstall(t *testing.T) {
	s, g, _, journal := setup(t)
	ctx := context.Background()
	j := joined(g, "u", "w", 0)
	writing, release := make(chan struct{}), make(chan struct{})
	journal.save = func() error { close(writing); <-release; return nil }
	done := make(chan error, 1)
	go func() { _, e := s.JoinTable(ctx, j, 0); done <- e }()
	<-writing
	s.Revoke("w", "u", "game entry")
	close(release)
	if e := <-done; e == nil {
		t.Fatal("join installed after cancellation")
	}
	if _, ok := s.Current("u"); ok {
		t.Fatal("cancelled seat retained")
	}
	if e := s.ReconcileParticipants(ctx); e != nil {
		t.Fatal(e)
	}
}
func TestJournalFailureFailsClosed(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(fmt.Sprint("committed=", committed), func(t *testing.T) {
			s, g, p, journal := setup(t)
			fail := func() error { return errors.New("journal write response failed") }
			if committed {
				journal.afterSave = fail
			} else {
				journal.save = fail
			}
			if _, err := s.JoinTable(context.Background(), joined(g, "u", "w", 0), 0); err == nil {
				t.Fatal("uncertain admission activated")
			}
			if got := len(journal.rows); (got == 1) != committed {
				t.Fatal("fixture persistence outcome", got, committed)
			}
			if s.Status() != "degraded" {
				t.Fatal(s.Status())
			}
			if _, ok := s.Current("u"); ok {
				t.Fatal("uncertain seat active")
			}
			for id := range s.entries {
				if _, err := s.IssueToken(context.Background(), id); err == nil {
					t.Fatal("uncertain token minted")
				}
			}
			journal.save = nil
			journal.afterSave = nil
			if err := s.ReconcileParticipants(context.Background()); err != nil {
				t.Fatal(err)
			}
			if len(p.removed) != 1 || len(journal.rows) != 0 {
				t.Fatal("uncertain identity not explicitly cleaned")
			}
		})
	}
}
func TestChatSeatIsolationDedupeRateAndExpiry(t *testing.T) {
	s, g, _, _ := setup(t)
	ctx := context.Background()
	a, _ := s.JoinTable(ctx, joined(g, "u", "w", 0), 0)
	b, _ := s.JoinTable(ctx, joined(g, "v", "w", 1), 1)
	m, e := s.Chat(ctx, a.ID, a.ID+":1", "<script>plain</script>")
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.Chat(ctx, a.ID, a.ID+":1", "different")
	if e != nil || m.ID != again.ID {
		t.Fatal("dedupe", e)
	}
	state, _ := s.State(ctx, b.ID)
	if len(state.Messages) != 0 {
		t.Fatal("cross-table chat")
	}
	for i := 0; i < 4; i++ {
		if _, e = s.Chat(ctx, a.ID, fmt.Sprintf("%s:%d", a.ID, i+2), "ok"); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = s.Chat(ctx, a.ID, a.ID+":6", "no"); !errors.Is(e, ErrRate) {
		t.Fatal(e)
	}
	s.Revoke("w", "u", "leave")
	if _, e = s.Chat(ctx, a.ID, a.ID+":6", "no"); e == nil {
		t.Fatal("departed chat allowed")
	}
	s.mu.Lock()
	for _, table := range s.tables {
		table.emptySince = time.Now().Add(-6 * time.Minute)
	}
	s.mu.Unlock()
	if e = s.ReconcileParticipants(ctx); e != nil {
		t.Fatal(e)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, table := range s.tables {
		if len(table.messages) > 0 {
			t.Fatal("expired ephemeral chat")
		}
	}
}
func TestLifecycleReasonsFenceSeatAndHostTransfers(t *testing.T) {
	for _, reason := range []string{"logout", "kick", "leave", "game entry", "disconnect", "resume", "session expired", "area exit"} {
		t.Run(reason, func(t *testing.T) {
			s, g, _, _ := setup(t)
			ctx := context.Background()
			a, _ := s.JoinTable(ctx, joined(g, "u", "w", 0), 0)
			b, _ := s.JoinTable(ctx, joined(g, "v", "w", 0), 0)
			if reason == "area exit" {
				s.Observe("w", "u", 32, 32, false)
			} else {
				s.Revoke("w", "u", reason)
			}
			if _, err := s.IssueToken(ctx, a.ID); err == nil {
				t.Fatal("revoked token")
			}
			state, err := s.State(ctx, b.ID)
			if err != nil || state.HostID != "v" {
				t.Fatalf("host not transferred: %+v %v", state, err)
			}
		})
	}
}
func TestUnknownParticipantsAndForeignRooms(t *testing.T) {
	s, _, p, _ := setup(t)
	p.rooms = map[string][]string{"unrelated-room": {"foreign"}, "test-orphan": {"orphan"}}
	if err := s.ReconcileParticipants(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(p.removed) != 1 || p.removed[0].ID != "orphan" {
		t.Fatal("wrong namespace sweep", p.removed)
	}
}
func TestOwnedTransportCannotUseReplacementAdmission(t *testing.T) {
	s, g, _, _ := setup(t)
	ctx := context.Background()
	j := joined(g, "u", "w", 0)
	_, err := s.JoinTable(ctx, j, 0)
	if err != nil {
		t.Fatal(err)
	}
	j.Epoch++
	if _, ok := s.Owned(j, "u"); ok {
		t.Fatal("wrong epoch admitted")
	}
}
func TestModeratorMuteAndKickAreAuditedAndScoped(t *testing.T) {
	s, g, _, _ := setup(t)
	ctx := context.Background()
	j := joined(g, "u", "w", 0)
	u := g.subjects["u"]
	u.Role = "moderator"
	g.subjects["u"] = u
	a, _ := s.JoinTable(ctx, j, 0)
	b, _ := s.JoinTable(ctx, joined(g, "v", "w", 0), 0)
	c, _ := s.JoinTable(ctx, joined(g, "x", "w", 1), 1)
	calls := 0
	s.Audit = func(context.Context, string, string, string, string) error { calls++; return nil }
	if err := s.Moderate(ctx, a.ID, c.ID, "kick"); err == nil {
		t.Fatal("cross table moderation")
	}
	if err := s.Moderate(ctx, b.ID, a.ID, "kick"); err == nil {
		t.Fatal("nonmoderator")
	}
	if err := s.Moderate(ctx, a.ID, b.ID, "mute"); err != nil {
		t.Fatal(err)
	}
	if err := s.Unmute(ctx, b.ID); err == nil {
		t.Fatal("unmute overtook provider mute")
	}
	if err := s.ReconcileParticipants(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.Unmute(ctx, b.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Moderate(ctx, a.ID, b.ID, "kick"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal(calls)
	}
}
func TestChatExpiresBeforeReadmissionWithoutWaitingForWorker(t *testing.T) {
	s, g, _, _ := setup(t)
	ctx := context.Background()
	j := joined(g, "u", "w", 0)
	a, _ := s.JoinTable(ctx, j, 0)
	if _, e := s.Chat(ctx, a.ID, a.ID+":1", "old"); e != nil {
		t.Fatal(e)
	}
	s.Revoke("w", "u", "leave")
	if e := s.ReconcileParticipants(ctx); e != nil {
		t.Fatal(e)
	}
	s.mu.Lock()
	s.tables[a.Room].emptySince = time.Now().Add(-6 * time.Minute)
	s.mu.Unlock()
	b, e := s.JoinTable(ctx, j, 0)
	if e != nil {
		t.Fatal(e)
	}
	state, e := s.State(ctx, b.ID)
	if e != nil || len(state.Messages) != 0 {
		t.Fatalf("expired messages resurrected: %+v %v", state.Messages, e)
	}
}
func TestChatDedupeSurvivesVisibleWindow(t *testing.T) {
	s, g, _, _ := setup(t)
	ctx := context.Background()
	a, _ := s.JoinTable(ctx, joined(g, "u", "w", 0), 0)
	_, err := s.Chat(ctx, a.ID, a.ID+":1", "first")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 205; i++ {
		s.mu.Lock()
		s.limits["u"] = bucket{5, time.Now()}
		s.mu.Unlock()
		if _, err = s.Chat(ctx, a.ID, fmt.Sprintf("%s:%d", a.ID, i+2), "new"); err != nil {
			t.Fatal(err)
		}
	}
	_, err = s.Chat(ctx, a.ID, a.ID+":1", "changed")
	if !errors.Is(err, ErrRetryExpired) {
		t.Fatal("old retry must stay rejected", err)
	}
	state, err := s.State(ctx, a.ID)
	if err != nil || len(state.Messages) != 200 {
		t.Fatal("visible message bound", err, len(state.Messages))
	}
}

func TestUnknownIdentitySaveFailureSurvivesDisappearance(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(fmt.Sprint("committed=", committed), func(t *testing.T) {
			s, g, p, journal := setup(t)
			ctx := context.Background()
			a, err := s.JoinTable(ctx, joined(g, "member", "w", 0), 0)
			if err != nil {
				t.Fatal(err)
			}
			fail := func() error { return errors.New("save failed") }
			if committed {
				journal.afterSave = fail
			} else {
				journal.save = fail
			}
			p.rooms = map[string][]string{"test-orphan": {"unknown"}}
			if err = s.ReconcileParticipants(ctx); err == nil {
				t.Fatal("save failure hidden")
			}
			_, persisted := journal.rows["unknown"]
			if persisted != committed {
				t.Fatalf("fixture commit=%v want%v", persisted, committed)
			}
			p.rooms = map[string][]string{}
			_ = s.ReconcileParticipants(ctx)
			if s.Status() != "degraded" {
				t.Fatal("disappearing participant incorrectly restored readiness")
			}
			if _, err = s.IssueToken(ctx, a.ID); !errors.Is(err, ErrUnavailable) {
				t.Fatal("existing token minted before cleanup", err)
			}
			if _, err = s.JoinTable(ctx, joined(g, "next", "w", 0), 0); !errors.Is(err, ErrUnavailable) {
				t.Fatal("new seat before cleanup", err)
			}
			journal.save = nil
			journal.afterSave = nil
			p.fail = true
			if err = s.ReconcileParticipants(ctx); err == nil {
				t.Fatal("removal failure hidden")
			}
			if s.Status() != "degraded" {
				t.Fatal("removal pending but ready")
			}
			p.fail = false
			if err = s.ReconcileParticipants(ctx); err != nil {
				t.Fatal(err)
			}
			if len(p.removed) != 1 || p.removed[0].ID != "unknown" {
				t.Fatal("missing explicit removal", p.removed)
			}
			if _, ok := journal.rows["unknown"]; ok {
				t.Fatal("settled journal retained")
			}
			if _, err = s.IssueToken(ctx, a.ID); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestChatContinuesBeyond4096WithBoundedRetryWindow(t *testing.T) {
	s, g, _, _ := setup(t)
	ctx := context.Background()
	a, err := s.JoinTable(ctx, joined(g, "u", "w", 0), 0)
	if err != nil {
		t.Fatal(err)
	}
	var last Message
	for n := 1; n <= 4200; n++ {
		s.mu.Lock()
		s.limits["u"] = bucket{5, time.Now()}
		s.mu.Unlock()
		last, err = s.Chat(ctx, a.ID, fmt.Sprintf("%s:%d", a.ID, n), "text")
		if err != nil {
			t.Fatalf("message %d stopped active table chat: %v", n, err)
		}
	}
	retry, err := s.Chat(ctx, a.ID, fmt.Sprintf("%s:4200", a.ID), "changed")
	if err != nil || retry.ID != last.ID || retry.Text != "text" {
		t.Fatal("recent retry changed", err)
	}
	if _, err = s.Chat(ctx, a.ID, a.ID+":1", "old"); !errors.Is(err, ErrRetryExpired) {
		t.Fatal("evicted retry must be explicitly rejected, never reposted")
	}
	s.mu.Lock()
	s.limits["u"] = bucket{5, time.Now()}
	s.mu.Unlock()
	if _, err = s.Chat(ctx, a.ID, a.ID+":4201", "new after old retry"); err != nil {
		t.Fatal("old retry blocked new submission", err)
	}
	state, err := s.State(ctx, a.ID)
	if err != nil || len(state.Messages) != 200 || state.NextMessageSeq != 4202 {
		t.Fatal(err, len(state.Messages), state.NextMessageSeq)
	}
	s.mu.Lock()
	if len(s.entries[a.ID].acknowledgements) != 32 {
		t.Fatal("unbounded acknowledgements")
	}
	s.mu.Unlock()
	s.Revoke("w", "u", "leave")
	s.mu.Lock()
	if len(s.entries[a.ID].acknowledgements) != 0 {
		t.Fatal("revoked ACK payload retained")
	}
	s.mu.Unlock()
}
func TestChatRequiresCanonicalCurrentAdmissionSequence(t *testing.T) {
	for _, suffix := range []string{"0", "01", "+1", "1e0", "1.0", " 1", "9007199254740992", "2"} {
		t.Run(suffix, func(t *testing.T) {
			s, g, _, _ := setup(t)
			ctx := context.Background()
			a, _ := s.JoinTable(ctx, joined(g, "u", "w", 0), 0)
			if _, err := s.Chat(ctx, a.ID, a.ID+":"+suffix, "bad"); err == nil {
				t.Fatal("invalid sequence accepted")
			}
			if _, err := s.Chat(ctx, a.ID, "wrong:1", "bad"); err == nil {
				t.Fatal("wrong admission accepted")
			}
			if _, err := s.Chat(ctx, a.ID, a.ID+":1", "valid"); err != nil {
				t.Fatal(err)
			}
		})
	}
}
