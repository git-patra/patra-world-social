//go:build integration

package application

import (
	"context"
	"encoding/json"
	"errors"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"patra-world-social/server/internal/voice"
	"patra-world-social/server/internal/world"
	"sync"
	"testing"
	"time"
)

type voiceProviderFixture struct {
	mu      sync.Mutex
	removed []voice.Record
}

func (p *voiceProviderFixture) Token(voice.VoiceAdmission) (voice.VoiceToken, error) {
	return voice.VoiceToken{Token: "isolated-backend-only"}, nil
}
func (p *voiceProviderFixture) Remove(_ context.Context, r voice.Record, cutoff int64) error {
	if cutoff <= time.Now().Unix() {
		return errors.New("cutoff missing")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.removed = append(p.removed, r)
	return nil
}
func (p *voiceProviderFixture) Rooms(context.Context) (map[string][]string, error) {
	return map[string][]string{}, nil
}
func (p *voiceProviderFixture) DeleteRoom(context.Context, string) error { return nil }
func (p *voiceProviderFixture) Mute(context.Context, voice.Record) error { return nil }
func TestVoiceUsesActualAdmissionLifecycleAndJournal(t *testing.T) {
	pool := testutil.Database(t)
	users, tokens := testutil.Users(t, pool, 3)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	app, err := New(pool, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	provider := &voiceProviderFixture{}
	s := wireVoiceService(pool, app.Hub, app.Worlds, provider, "test-app-")
	if err = s.ReconcileParticipants(ctx); err != nil {
		t.Fatal(err)
	}
	w, err := app.Worlds.CreateWorld(ctx, users[0], "Voice fixture")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, w.ID, users[1]); err != nil {
		t.Fatal(err)
	}
	joins := make([]realtime.JoinContext, 2)
	for i := 0; i < 2; i++ {
		j := realtime.JoinContext{Token: tokens[i], WorldID: w.ID}
		snap, e := app.Hub.Join(ctx, j)
		if e != nil {
			t.Fatal(e)
		}
		j.SessionID = snap.SessionID
		j.Epoch = snap.Epoch
		joins[i] = j
		if e = app.Hub.SetActivity(ctx, w.ID, users[i], false, &world.Point{X: int(voice.TableZones.Tables[0].X), Y: int(voice.TableZones.Tables[0].Y)}); e != nil {
			t.Fatal(e)
		}
	}
	execute := func(j realtime.JoinContext, kind, payload string) (any, error) {
		return app.Hub.Execute(ctx, j, realtime.Command{V: 2, CommandID: "voice-test", Seq: 1, WorldID: j.WorldID, Type: kind, Payload: json.RawMessage(payload)})
	}
	if _, err = execute(realtime.JoinContext{Token: tokens[2], WorldID: w.ID}, "voice.join", `{"table":0}`); err == nil {
		t.Fatal("outsider joined")
	}
	result, err := execute(joins[0], "voice.join", `{"table":0}`)
	if err != nil {
		t.Fatal(err)
	}
	a := result.(voice.VoiceAdmission)
	if _, err = execute(joins[0], "voice.token", `{}`); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM voice_revocations WHERE identity=$1`, a.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("token lacks journal", err, count)
	}
	// Authoritative movement observation removes the table, without waiting on a provider.
	if err = app.Hub.SetActivity(ctx, w.ID, users[0], false, &world.Point{X: 32, Y: 32}); err != nil {
		t.Fatal(err)
	}
	if _, err = app.Hub.Snapshot(ctx, joins[0]); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Current(users[0]); ok {
		t.Fatal("area exit kept voice")
	}
	if err = s.ReconcileParticipants(ctx); err != nil {
		t.Fatal(err)
	}
	if err = app.Hub.SetActivity(ctx, w.ID, users[0], false, &world.Point{X: int(voice.TableZones.Tables[0].X), Y: int(voice.TableZones.Tables[0].Y)}); err != nil {
		t.Fatal(err)
	}
	if _, err = execute(joins[0], "voice.join", `{"table":0}`); err != nil {
		t.Fatal(err)
	}
	if _, err = execute(joins[0], "lobby.join", `{"arenaId":"duck-1"}`); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Current(users[0]); ok {
		t.Fatal("game entry kept voice")
	}
	if err = s.ReconcileParticipants(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = execute(joins[1], "voice.join", `{"table":0}`); err != nil {
		t.Fatal(err)
	}
	if err = app.Worlds.BanMember(ctx, users[0], w.ID, users[1]); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Current(users[1]); ok {
		t.Fatal("ban kept voice")
	}
	if _, err = execute(joins[1], "voice.token", `{}`); err == nil {
		t.Fatal("banned token")
	}
	if err = s.ReconcileParticipants(ctx); err != nil {
		t.Fatal(err)
	}
	// Use an independent account so lobby return delivery cannot race the
	// fixture's test-only teleport into the table zone.
	if _, err = pool.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, w.ID, users[2]); err != nil {
		t.Fatal(err)
	}
	next := realtime.JoinContext{Token: tokens[2], WorldID: w.ID}
	snapshot, err := app.Hub.Join(ctx, next)
	if err != nil {
		t.Fatal(err)
	}
	next.SessionID = snapshot.SessionID
	next.Epoch = snapshot.Epoch
	joins[0] = next
	users[0] = users[2]
	tokens[0] = tokens[2]
	if err = app.Hub.SetActivity(ctx, w.ID, users[0], false, &world.Point{X: int(voice.TableZones.Tables[0].X), Y: int(voice.TableZones.Tables[0].Y)}); err != nil {
		t.Fatal(err)
	}
	if _, err = execute(joins[0], "voice.join", `{"table":0}`); err != nil {
		t.Fatal(err)
	}
	snap, err := app.Hub.Resume(ctx, joins[0], 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Current(users[0]); ok {
		t.Fatal("resume kept old voice")
	}
	if err = s.ReconcileParticipants(ctx); err != nil {
		t.Fatal(err)
	}
	joins[0].SessionID = snap.SessionID
	joins[0].Epoch = snap.Epoch
	if err = app.Hub.SetActivity(ctx, w.ID, users[0], false, &world.Point{X: int(voice.TableZones.Tables[0].X), Y: int(voice.TableZones.Tables[0].Y)}); err != nil {
		t.Fatal(err)
	}
	if _, err = execute(joins[0], "voice.join", `{"table":0}`); err != nil {
		t.Fatal(err)
	}
	if err = app.Auth.Logout(ctx, tokens[0]); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Current(users[0]); ok {
		t.Fatal("logout kept voice")
	}
	if err = s.ReconcileParticipants(ctx); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM voice_revocations`).Scan(&count); err != nil || count != 0 {
		t.Fatal("settled journal retained", err, count)
	}
}

func TestVoiceExcludesRunningMatchSpectatorsInBothOrders(t *testing.T) {
	pool := testutil.Database(t)
	users, tokens := testutil.Users(t, pool, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	clock := &penaltyClock{at: time.Now()}
	app, err := newApplication(pool, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true}, clock)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	provider := &voiceProviderFixture{}
	s := wireVoiceService(pool, app.Hub, app.Worlds, provider, "test-spectator-")
	if err = s.ReconcileParticipants(ctx); err != nil {
		t.Fatal(err)
	}
	w, err := app.Worlds.CreateWorld(ctx, users[0], "Spectator fixture")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, w.ID, users[1]); err != nil {
		t.Fatal(err)
	}
	joins := make([]realtime.JoinContext, 2)
	for i := range joins {
		j := realtime.JoinContext{Token: tokens[i], WorldID: w.ID}
		snap, e := app.Hub.Join(ctx, j)
		if e != nil {
			t.Fatal(e)
		}
		j.SessionID = snap.SessionID
		j.Epoch = snap.Epoch
		joins[i] = j
	}
	execute := func(i int, kind string, payload any) (any, error) {
		raw, _ := json.Marshal(payload)
		return app.Hub.Execute(ctx, joins[i], realtime.Command{V: 2, CommandID: kind, Seq: 1, WorldID: w.ID, Type: kind, Payload: raw})
	}
	if _, err = execute(0, "lobby.join", map[string]any{"arenaId": "duck-1"}); err != nil {
		t.Fatal(err)
	}
	all, err := app.Lobbies.List(ctx, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	var lobbyID string
	for _, l := range all {
		if l.Arena.ID == "duck-1" {
			lobbyID = l.ID
		}
	}
	if _, err = execute(0, "lobby.ready", map[string]any{"lobbyId": lobbyID, "ready": true}); err != nil {
		t.Fatal(err)
	}
	if err = app.Lobbies.SetStorageBlocked(false); err != nil {
		t.Fatal(err)
	}
	if _, err = execute(0, "match.start", map[string]any{"lobbyId": lobbyID}); err != nil {
		t.Fatal(err)
	}
	awaitClearedStart(t, ctx, app.Lobbies, app.Hub, clock, w.ID, lobbyID)
	all, err = app.Lobbies.List(ctx, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range all {
		if l.ID == lobbyID {
			clock.set(l.StartsAt + 1)
		}
	}
	if err = app.Lobbies.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	all, _ = app.Lobbies.List(ctx, w.ID)
	for _, l := range all {
		if l.ID == lobbyID && l.State != "running" {
			t.Fatal("fixture match is not running", l.State)
		}
	}
	if err = app.Hub.SetActivity(ctx, w.ID, users[1], false, &world.Point{X: int(voice.TableZones.Tables[0].X), Y: int(voice.TableZones.Tables[0].Y)}); err != nil {
		t.Fatal(err)
	}
	initial, err := s.JoinTable(ctx, joins[1], 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = execute(1, "lobby.join", map[string]any{"arenaId": "duck-1"}); err != nil {
		t.Fatal(err)
	}
	if app.Lobbies.Busy(users[1]) {
		t.Fatal("spectator incorrectly gained player-only busy state")
	}
	if _, ok := s.Current(users[1]); ok {
		t.Fatal("spectator entry kept existing voice admission")
	}
	if _, err = s.IssueToken(ctx, initial.ID); err == nil {
		t.Fatal("spectator kept token")
	}
	if _, err = s.Chat(ctx, initial.ID, initial.ID+":1", "text"); err == nil {
		t.Fatal("spectator kept table chat")
	}
	if err = s.ReconcileParticipants(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = execute(1, "voice.join", map[string]any{"table": 0}); !errors.Is(err, voice.ErrOutside) {
		t.Fatal("spectator late voice join allowed", err)
	}
	if _, err = execute(1, "voice.token", map[string]any{}); err == nil {
		t.Fatal("spectator late token allowed")
	}
	if _, err = execute(1, "lobby.leave", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if _, err = execute(1, "voice.join", map[string]any{"table": 0}); err != nil {
		t.Fatal("voice exclusion did not release after spectating", err)
	}
}
