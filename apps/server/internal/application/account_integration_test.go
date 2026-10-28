//go:build integration

package application

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/coder/websocket"
	"github.com/jackc/pgx/v5"
	"net/http"
	"net/http/httptest"
	"patra-world-social/server/internal/auth"
	"patra-world-social/server/internal/games/duck"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/history"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/leaderboard"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"patra-world-social/server/internal/worlds"
	"strings"
	"testing"
	"time"
)

func TestDeletedAccountPendingResultHTTPAndLatePersist(t *testing.T) {
	p := testutil.Database(t)
	users, tokens := testutil.Users(t, p, 2)
	ctx := context.Background()
	a := auth.New(p)
	a.RemoveAccount = leaderboard.RemoveAccount
	w, e := worlds.New(p).CreateWorld(ctx, users[0], "Synthetic")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, w.ID, users[1]); e != nil {
		t.Fatal(e)
	}
	m := model.MatchStart{MatchID: "b1000000-0000-4000-8000-000000000001", WorldID: w.ID, ArenaID: "penalty", GameType: "penalty", PreparedAt: 1000, Version: model.Version{MapID: "fixture", MapVersion: 1, RulesID: "fixture", RulesVersion: 1, ProtocolVersion: 1}, Actors: []model.Actor{{ActorID: "b2000000-0000-4000-8000-000000000001", ActorType: "human", UserID: users[0], DisplayName: "Survivor", AvatarID: "pumpkin"}, {ActorID: "b2000000-0000-4000-8000-000000000002", ActorType: "human", UserID: users[1], DisplayName: "Private old name", AvatarID: "moon"}}}
	store := &lobby.PGStore{Pool: p}
	if e = store.Persist(ctx, m, users[0], "deletion"); e != nil {
		t.Fatal(e)
	}
	r := model.MatchResult{Version: m.Version, MatchID: m.MatchID, WorldID: m.WorldID, ArenaID: m.ArenaID, GameType: m.GameType, PreparedAt: 1000, StartsAt: 2000, FinishedAt: 5000, Participants: []model.ParticipantResult{{Actor: m.Actors[0], Outcome: "win"}, {Actor: m.Actors[1], Outcome: "loss"}}}
	h := history.New(p)
	if e = h.Enqueue(ctx, r); e != nil {
		t.Fatal(e)
	}
	var before []byte
	if e = p.QueryRow(ctx, `SELECT result_hash FROM result_jobs WHERE match_id=$1`, m.MatchID).Scan(&before); e != nil {
		t.Fatal(e)
	}
	if _, e = p.Exec(ctx, `INSERT INTO voice_revocations(identity,room)VALUES('synthetic-cleanup','synthetic-room')`); e != nil {
		t.Fatal(e)
	}
	if e = a.DeleteAccount(ctx, users[1], "synthetic_1"); e != nil {
		t.Fatal(e)
	}
	if _, e = h.Finalize(ctx, m.MatchID); e != nil {
		t.Fatal(e)
	}
	if e = h.Enqueue(ctx, r); e != nil {
		t.Fatal("immutable retry rejected", e)
	}
	var after []byte
	if e = p.QueryRow(ctx, `SELECT result_hash FROM result_jobs WHERE match_id=$1`, m.MatchID).Scan(&after); e != nil || !bytes.Equal(before, after) {
		t.Fatal("frozen result hash changed", e)
	}
	req := httptest.NewRequest("GET", "/api/v1/history/"+m.MatchID, nil)
	req.AddCookie(&http.Cookie{Name: "patra_session", Value: tokens[0]})
	response := httptest.NewRecorder()
	httpapi.SessionMiddleware(a, true, h.Handler()).ServeHTTP(response, req)
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	body := response.Body.String()
	if strings.Contains(body, "Private old name") || strings.Contains(body, users[1]) || strings.Contains(body, `"avatarId":"moon"`) || !strings.Contains(body, "Akun dihapus") {
		t.Fatal("HTTP leaked erased identity", body)
	}
	late := m
	late.MatchID = "b1000000-0000-4000-8000-000000000002"
	if e = store.Persist(ctx, late, users[0], "late"); e == nil {
		t.Fatal("late prepared roster resurrected account")
	}
	var cleanups int
	if e = p.QueryRow(ctx, `SELECT count(*) FROM voice_revocations`).Scan(&cleanups); e != nil || cleanups != 1 {
		t.Fatal("cleanup journal lost", e)
	}
}
func TestDeletedAccountActualWebsocketSerializationAndRollback(t *testing.T) {
	p := testutil.Database(t)
	users, tokens := testutil.Users(t, p, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	app, e := New(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true})
	if e != nil {
		t.Fatal(e)
	}
	defer app.Close()
	w, e := app.Worlds.CreateWorld(ctx, users[0], "Synthetic")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, w.ID, users[1]); e != nil {
		t.Fatal(e)
	}
	srv := httptest.NewServer(app.Handler)
	defer srv.Close()
	dial := func(i int) *websocket.Conn {
		c, _, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/api/v1/realtime?worldId="+w.ID, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://127.0.0.1:4176"}, "Cookie": {"patra_session=" + tokens[i]}}})
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { c.CloseNow() })
		if _, _, e = c.Read(ctx); e != nil {
			t.Fatal(e)
		}
		return c
	}
	peer := dial(0)
	dial(1)
	// A failing transactional hook must roll back all durable erasure; revocation
	// is conservative and intentionally can disconnect before a failed deletion.
	app.Auth.RemoveAccount = func(context.Context, pgx.Tx, string) error { return context.Canceled }
	if e = app.Auth.DeleteAccount(ctx, users[1], "synthetic_1"); e == nil {
		t.Fatal("failed deletion reported success")
	}
	if _, e = app.Auth.RequireSession(ctx, tokens[1]); e != nil {
		t.Fatal("rollback erased credential", e)
	}
	if app.Hub.Online(users[1]) {
		t.Fatal("deletion fence did not revoke")
	}
	app.Auth.RemoveAccount = leaderboard.RemoveAccount
	if e = app.Auth.DeleteAccount(ctx, users[1], "synthetic_1"); e != nil {
		t.Fatal(e)
	}
	stale := map[string]any{"matchId": "synthetic", "actors": []any{map[string]any{"actorId": "fixed-actor", "actorType": "human", "userId": users[1], "displayName": "Private live name", "avatarId": "moon"}}}
	app.Hub.Publish(w.ID, "penalty.snapshot", stale)
	for {
		_, raw, e := peer.Read(ctx)
		if e != nil {
			t.Fatal(e)
		}
		var event realtime.Event
		if e = json.Unmarshal(raw, &event); e != nil {
			t.Fatal(e)
		}
		if event.Type != "penalty.snapshot" {
			continue
		}
		if bytes.Contains(event.Payload, []byte("Private live name")) || bytes.Contains(event.Payload, []byte(users[1])) || !bytes.Contains(event.Payload, []byte("Akun dihapus")) {
			t.Fatalf("WS leaked identity %s", event.Payload)
		}
		break
	}
	if stale["actors"].([]any)[0].(map[string]any)["displayName"] != "Private live name" {
		t.Fatal("internal frozen actor mutated")
	}
}

func TestDeletionFencesPersistWaitingOnAccountLock(t *testing.T) {
	p := testutil.Database(t)
	users, _ := testutil.Users(t, p, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	a := auth.New(p)
	a.RemoveAccount = leaderboard.RemoveAccount
	w, e := worlds.New(p).CreateWorld(ctx, users[0], "Synthetic")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, w.ID, users[1]); e != nil {
		t.Fatal(e)
	}
	reached := make(chan struct{})
	resume := make(chan struct{})
	a.RemoveAccount = func(ctx context.Context, tx pgx.Tx, id string) error {
		close(reached)
		<-resume
		return leaderboard.RemoveAccount(ctx, tx, id)
	}
	removed := make(chan error, 1)
	go func() { removed <- a.DeleteAccount(ctx, users[1], "synthetic_1") }()
	<-reached
	m := model.MatchStart{MatchID: "b3000000-0000-4000-8000-000000000001", WorldID: w.ID, ArenaID: "duck-1", GameType: "duck", PreparedAt: 1000, Actors: []model.Actor{{ActorID: "b4000000-0000-4000-8000-000000000001", ActorType: "human", UserID: users[1], DisplayName: "Must not return"}}}
	persisted := make(chan error, 1)
	go func() { persisted <- (&lobby.PGStore{Pool: p}).Persist(ctx, m, users[1], "waiting") }()
	select {
	case e := <-persisted:
		t.Fatal("Persist escaped deletion account lock", e)
	case <-time.After(25 * time.Millisecond):
	}
	close(resume)
	if e = <-removed; e != nil {
		t.Fatal(e)
	}
	if e = <-persisted; e == nil {
		t.Fatal("late persist resurrected account")
	}
	var n int
	if e = p.QueryRow(ctx, `SELECT count(*) FROM match_participants WHERE user_id=$1`, users[1]).Scan(&n); e != nil || n != 0 {
		t.Fatal("late personal projection", n, e)
	}
}

func TestAccountDeleteRecomputesBoardAndExcludesPendingProjection(t *testing.T) {
	p := testutil.Database(t)
	users, _ := testutil.Users(t, p, 2)
	ctx := context.Background()
	a := auth.New(p)
	a.RemoveAccount = leaderboard.RemoveAccount
	w, e := worlds.New(p).CreateWorld(ctx, users[0], "Synthetic")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, w.ID, users[1]); e != nil {
		t.Fatal(e)
	}
	h := history.New(p)
	h.Projectors["duck"] = func(ctx context.Context, tx pgx.Tx, r model.MatchResult) error {
		if e := duck.Project(ctx, tx, r); e != nil {
			return e
		}
		return leaderboard.ProjectResult(ctx, tx, r)
	}
	ids := []string{"b5000000-0000-4000-8000-000000000001", "b5000000-0000-4000-8000-000000000002", "b5000000-0000-4000-8000-000000000003"}
	for i, id := range ids {
		u := 0
		if i > 0 {
			u = 1
		}
		m := model.MatchStart{MatchID: id, WorldID: w.ID, ArenaID: "duck-1", GameType: "duck", PreparedAt: int64(1000 + i*100000), Version: model.Version{MapID: "fixture", MapVersion: 1, RulesID: "duck-v1", RulesVersion: 1, RulesChecksum: duck.RulesChecksum(), ProtocolVersion: 1}, Actors: []model.Actor{{ActorID: id, ActorType: "human", UserID: users[u], DisplayName: "Synthetic"}}}
		if e = (&lobby.PGStore{Pool: p}).Persist(ctx, m, users[u], id); e != nil {
			t.Fatal(e)
		}
		score := int64(100 + i*200)
		r := model.MatchResult{Version: m.Version, MatchID: id, WorldID: w.ID, ArenaID: m.ArenaID, GameType: "duck", PreparedAt: m.PreparedAt, StartsAt: m.PreparedAt + 3000, FinishedAt: m.PreparedAt + 63000, Participants: []model.ParticipantResult{{Actor: m.Actors[0], Outcome: "finished", Score: &score, Duck: &model.DuckResult{Hits: 1, Shots: 1}}}}
		if e = h.Enqueue(ctx, r); e != nil {
			t.Fatal(e)
		}
		if i < 2 {
			if _, e = h.Finalize(ctx, id); e != nil {
				t.Fatal(e)
			}
			if e = (&lobby.PGStore{Pool: p}).Terminal(ctx, id, "completed"); e != nil {
				t.Fatal(e)
			}
		}
	}
	before, e := leaderboard.New(p).Board(ctx, users[0], w.ID, leaderboard.CurrentRuleset)
	if e != nil || before.Top[0].UserID != users[1] {
		t.Fatal("fixture champion", e)
	}
	invalidated := false
	a.Deleted = func(_ string, partitions [][3]string) { invalidated = len(partitions) == 1 && partitions[0][0] == w.ID }
	if e = a.DeleteAccount(ctx, users[1], "synthetic_1"); e != nil {
		t.Fatal(e)
	}
	if !invalidated {
		t.Fatal("board invalidation partitions lost")
	}
	if _, e = h.Finalize(ctx, ids[2]); e != nil {
		t.Fatal(e)
	}
	after, e := leaderboard.New(p).Board(ctx, users[0], w.ID, leaderboard.CurrentRuleset)
	if e != nil || len(after.Top) != 1 || after.Top[0].UserID != users[0] {
		t.Fatal("deleted champion resurrected", e)
	}
	var changes int
	if e = p.QueryRow(ctx, `SELECT count(*) FROM champion_changes WHERE user_id=$1`, users[0]).Scan(&changes); e != nil || changes != 1 {
		t.Fatal("recompute counted as victory", changes, e)
	}
}
