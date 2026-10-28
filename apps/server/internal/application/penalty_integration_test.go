//go:build integration

package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	"patra-world-social/server/internal/games/penalty"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"strings"
	"sync"
	"testing"
	"time"
)

type penaltyClock struct {
	mu sync.Mutex
	at time.Time
}

func (c *penaltyClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.at }
func (c *penaltyClock) set(ms int64)   { c.mu.Lock(); c.at = time.UnixMilli(ms); c.mu.Unlock() }
func TestPenaltyActualTransportHistoryAndMutualRematch(t *testing.T) {
	p := testutil.Database(t)
	users, tokens := testutil.Users(t, p, 3)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	clock := &penaltyClock{at: time.Now()}
	app, err := newApplication(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true}, clock)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	w, err := app.Worlds.CreateWorld(ctx, users[0], "Synthetic duel")
	if err != nil {
		t.Fatal(err)
	}
	p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, w.ID, users[1])
	p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, w.ID, users[2])
	other, _ := app.Worlds.CreateWorld(ctx, users[2], "Other")
	server := httptest.NewServer(app.Handler)
	defer server.Close()
	dial := func(i int) (*websocket.Conn, realtime.JoinContext) {
		t.Helper()
		c, _, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/realtime?protocolVersion=2&worldId="+w.ID, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://127.0.0.1:4176"}, "Cookie": {"patra_session=" + tokens[i]}}})
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { c.CloseNow() })
		_, raw, e := c.Read(ctx)
		if e != nil {
			t.Fatal(e)
		}
		var ev realtime.Event
		json.Unmarshal(raw, &ev)
		var snap realtime.PublicSnapshot
		json.Unmarshal(ev.Payload, &snap)
		return c, realtime.JoinContext{Token: tokens[i], WorldID: w.ID, SessionID: snap.SessionID, Epoch: snap.Epoch}
	}
	host, hj := dial(0)
	peer, pj := dial(1)
	spectator, sj := dial(2)
	counter := 0
	command := func(kind string, payload any) realtime.Command {
		counter++
		raw, _ := json.Marshal(payload)
		return realtime.Command{V: 2, CommandID: fmt.Sprint("penalty-", counter), Seq: uint64(counter), WorldID: w.ID, Type: kind, Payload: raw}
	}
	send := func(c *websocket.Conn, cmd realtime.Command) (string, json.RawMessage) {
		t.Helper()
		raw, _ := json.Marshal(cmd)
		if e := c.Write(ctx, websocket.MessageText, raw); e != nil {
			t.Fatal(e)
		}
		for {
			_, raw, e := c.Read(ctx)
			if e != nil {
				t.Fatal(e)
			}
			ev, e := realtime.ValidateEvent(raw)
			if e != nil {
				t.Fatal(e)
			}
			var ack struct {
				CommandID string          `json:"commandId"`
				Result    json.RawMessage `json:"result"`
			}
			json.Unmarshal(ev.Payload, &ack)
			if ack.CommandID == cmd.CommandID {
				return ev.Type, ack.Result
			}
		}
	}
	direct := func(j realtime.JoinContext, kind string, payload any) (any, error) {
		t.Helper()
		if e := app.Hub.Validate(ctx, j); e != nil {
			t.Fatal(e)
		}
		return app.Hub.Execute(ctx, j, command(kind, payload))
	}
	kind, raw := send(host, command("lobby.join", map[string]any{"arenaId": "penalty"}))
	if kind != "command.ack" {
		t.Fatal(kind)
	}
	var l lobby.Lobby
	json.Unmarshal(raw, &l)
	send(peer, command("lobby.join", map[string]any{"arenaId": "penalty"}))
	send(host, command("lobby.ready", map[string]any{"lobbyId": l.ID, "ready": true}))
	send(peer, command("lobby.ready", map[string]any{"lobbyId": l.ID, "ready": true}))
	// Wait only for the startup worker's existing storage gate, without driving it.
	var startErr error
	for i := 0; i < 40; i++ {
		_, startErr = direct(hj, "match.start", map[string]any{"lobbyId": l.ID})
		if startErr == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if startErr != nil {
		t.Fatal(startErr)
	}
	awaitClearedStart(t, ctx, app.Lobbies, app.Hub, clock, w.ID, l.ID)
	all, _ := app.Lobbies.List(ctx, w.ID)
	for _, v := range all {
		if v.ID == l.ID {
			l = v
		}
	}
	mid := l.MatchID
	snapshot := func() penalty.PublicSnapshot {
		t.Helper()
		var s penalty.PublicSnapshot
		if e := app.Lobbies.ReadEngine(ctx, w.ID, mid, func(engine lobby.Engine) error { s = engine.(*penalty.Adapter).Snapshot(); return nil }); e != nil {
			t.Fatal(e)
		}
		return s
	}
	s := snapshot()
	if _, e := direct(hj, "match.return", map[string]any{"matchId": mid}); e == nil {
		t.Fatal("running result released")
	}
	if _, e := direct(hj, "match.rematch", map[string]any{"matchId": mid}); e == nil {
		t.Fatal("running result rematched")
	}
	_, public := send(spectator, command("penalty.snapshot", map[string]any{"matchId": mid}))
	if strings.Contains(string(public), "kickZone") {
		t.Fatal("secret public snapshot", string(public))
	}
	if _, e := direct(sj, "penalty.choose", map[string]any{"matchId": mid, "turnId": s.TurnID, "zone": 0}); e == nil {
		t.Fatal("spectator chose")
	}
	if app.Lobbies.ReadEngine(ctx, other.ID, mid, func(lobby.Engine) error { return nil }) == nil {
		t.Fatal("cross-world snapshot")
	}
	clock.set(s.ChoiceDeadlineAt - 1)
	app.Hub.Validate(ctx, hj)
	choice := command("penalty.choose", map[string]any{"matchId": mid, "turnId": s.TurnID, "zone": 2})
	if kind, _ = send(host, choice); kind != "command.ack" {
		t.Fatal("2999 rejected", kind)
	}
	// Distinct schema-valid rejected IDs pass the real admitted application
	// dispatcher without consuming either player's later valid-choice authority.
	for n := 0; n < 512; n++ {
		turn := s.TurnID
		if n%2 == 0 {
			turn = "stale"
		}
		if _, e := app.Hub.Execute(ctx, hj, command("penalty.choose", map[string]any{"matchId": mid, "turnId": turn, "zone": 1})); e == nil {
			t.Fatal("rejected flood accepted", n)
		}
	}
	if kind, _ = send(host, choice); kind != "command.error" {
		t.Fatal("duplicate changed choice")
	}
	_, public = send(spectator, command("penalty.snapshot", map[string]any{"matchId": mid}))
	if strings.Contains(string(public), "kickZone") {
		t.Fatal("choice leaked", string(public))
	}
	clock.set(s.ChoiceDeadlineAt)
	app.Hub.Validate(ctx, pj)
	if kind, _ = send(peer, command("penalty.choose", map[string]any{"matchId": mid, "turnId": s.TurnID, "zone": 1})); kind != "command.error" {
		t.Fatal("3000 accepted")
	}
	clock.set(s.AnimationEndsAt)
	app.Lobbies.Sync(ctx)
	if _, e := direct(hj, "penalty.choose", map[string]any{"matchId": mid, "turnId": s.TurnID, "zone": 1}); e == nil {
		t.Fatal("old turn accepted")
	}
	// The remaining commands advance virtual time via direct admitted dispatch.
	// Read every socket and wait for each turn's public delivery before advancing
	// again: accelerating 90 seconds into a tight loop otherwise outruns the
	// intentionally bounded 64-event transport queue, even with active readers.
	drainCtx, stopDrain := context.WithCancel(ctx)
	var readers sync.WaitGroup
	drainErrors := make(chan error, 3)
	delivered := make([]chan penalty.PublicSnapshot, 3)
	for i, c := range []*websocket.Conn{host, peer, spectator} {
		delivered[i] = make(chan penalty.PublicSnapshot, 128)
		readers.Add(1)
		go func(i int, c *websocket.Conn) {
			defer readers.Done()
			for {
				_, raw, err := c.Read(drainCtx)
				if err != nil {
					if drainCtx.Err() == nil {
						drainErrors <- fmt.Errorf("reader %d terminated before cleanup: %w", i, err)
					}
					return
				}
				event, err := realtime.ValidateEvent(raw)
				if err != nil {
					drainErrors <- err
					return
				}
				if strings.HasPrefix(event.Type, "penalty.") {
					var snap penalty.PublicSnapshot
					if err = json.Unmarshal(event.Payload, &snap); err != nil {
						drainErrors <- err
						return
					}
					select {
					case delivered[i] <- snap:
					case <-drainCtx.Done():
						return
					}
				}
			}
		}(i, c)
	}
	defer func() {
		stopDrain()
		readers.Wait()
		select {
		case err := <-drainErrors:
			t.Error(err)
		default:
		}
	}()
	awaitPublished := func(expected penalty.PublicSnapshot) {
		t.Helper()
		for _, ch := range delivered {
			for {
				select {
				case err := <-drainErrors:
					t.Fatal(err)
				case <-ctx.Done():
					t.Fatal("public turn was not delivered", ctx.Err())
				case observed := <-ch:
					if observed.MatchID == mid && observed.Turn >= expected.Turn && (expected.Result == nil || observed.Result != nil) {
						goto nextReader
					}
				}
			}
		nextReader:
		}
	}
	// Finish through actual admitted command dispatch with deterministic time.
	for n := 0; n < 20; n++ {
		s = snapshot()
		awaitPublished(s)
		if s.Result != nil {
			break
		}
		for _, j := range []realtime.JoinContext{hj, pj} {
			zone := 0
			for _, a := range s.Actors {
				if a.UserID == map[string]string{hj.SessionID: users[0], pj.SessionID: users[1]}[j.SessionID] && a.ActorID == s.KeeperID {
					zone = 1
				}
			}
			if _, e := direct(j, "penalty.choose", map[string]any{"matchId": mid, "turnId": s.TurnID, "zone": zone}); e != nil {
				all, _ := app.Lobbies.List(ctx, w.ID)
				t.Fatal(e, "loop", n, "snap", s, "lobby", all)
			}
		}
		clock.set(s.AnimationEndsAt)
		app.Lobbies.Sync(ctx)
	}
	s = snapshot()
	if s.Result == nil {
		t.Fatal("no natural result")
	}
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		all, _ = app.Lobbies.List(ctx, w.ID)
		for _, v := range all {
			if v.MatchID == mid {
				l = v
			}
		}
		if l.State == "results" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if l.State != "results" {
		t.Fatal("not durably finalized", l.State)
	}
	record, e := app.History.Detail(ctx, users[0], mid)
	if e != nil || record.Status != "completed" || record.Result == nil {
		t.Fatal(record, e)
	}
	var count int
	p.QueryRow(ctx, `SELECT count(*) FROM penalty_turns WHERE match_id=$1`, mid).Scan(&count)
	if count != len(s.Turns) {
		t.Fatal("turn projection", count, len(s.Turns))
	}
	if _, e = app.History.Finalize(ctx, mid); e != nil {
		t.Fatal("retry", e)
	}
	p.QueryRow(ctx, `SELECT count(*) FROM penalty_turns WHERE match_id=$1`, mid).Scan(&count)
	if count != len(s.Turns) {
		t.Fatal("duplicate projector")
	}
	if _, e = app.History.Detail(ctx, users[2], mid); e == nil {
		t.Fatal("spectator private history")
	}
	var oldHash, oldPayload []byte
	if e = p.QueryRow(ctx, `SELECT result_hash,convert_to(payload::text,'UTF8') FROM result_jobs WHERE match_id=$1`, mid).Scan(&oldHash, &oldPayload); e != nil {
		t.Fatal(e)
	}
	if _, e = direct(hj, "match.rematch", map[string]any{"matchId": mid}); e != nil {
		t.Fatal(e)
	}
	all, _ = app.Lobbies.List(ctx, w.ID)
	for _, v := range all {
		if v.MatchID == mid && v.State != "results" {
			t.Fatal("single consent released")
		}
	}
	if _, e = direct(pj, "match.rematch", map[string]any{"matchId": mid}); e != nil {
		t.Fatal(e)
	}
	all, _ = app.Lobbies.List(ctx, w.ID)
	var rematch lobby.Lobby
	for _, v := range all {
		if v.Arena.ID == "penalty" {
			rematch = v
		}
	}
	if rematch.ID == l.ID || rematch.State != "lobby" || len(rematch.Players) != 2 || rematch.Players[0].Ready || rematch.Players[1].Ready {
		t.Fatal("mutual rematch", rematch)
	}
	var afterHash, afterPayload []byte
	if e = p.QueryRow(ctx, `SELECT result_hash,convert_to(payload::text,'UTF8') FROM result_jobs WHERE match_id=$1`, mid).Scan(&afterHash, &afterPayload); e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(oldHash, afterHash) || !bytes.Equal(oldPayload, afterPayload) {
		t.Fatal("rematch changed durable old result")
	}
	for i, p := range rematch.Players {
		if p.UserID != l.Players[i].UserID || p.ActorID == l.Players[i].ActorID {
			t.Fatal("rematch roster identity", rematch)
		}
	}

}
func TestPenaltyOnlineBotNaturalResultReturnAndAbandonedCleanup(t *testing.T) {
	p := testutil.Database(t)
	users, tokens := testutil.Users(t, p, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	clock := &penaltyClock{at: time.Now()}
	app, err := newApplication(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true}, clock)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	w, err := app.Worlds.CreateWorld(ctx, users[0], "Synthetic bot")
	if err != nil {
		t.Fatal(err)
	}
	snap, err := app.Hub.Join(ctx, realtime.JoinContext{Token: tokens[0], WorldID: w.ID})
	if err != nil {
		t.Fatal(err)
	}
	j := realtime.JoinContext{Token: tokens[0], WorldID: w.ID, SessionID: snap.SessionID, Epoch: snap.Epoch}
	i := 0
	execute := func(kind string, payload any) (any, error) {
		t.Helper()
		i++
		if err := app.Hub.Validate(ctx, j); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(payload)
		return app.Hub.Execute(ctx, j, realtime.Command{V: 2, CommandID: fmt.Sprint("bot-", i), Seq: uint64(i), WorldID: w.ID, Type: kind, Payload: raw})
	}
	l, err := app.Lobbies.JoinLobby(ctx, lobby.JoinContext{UserID: users[0], WorldID: w.ID, DisplayName: "Synthetic"}, "penalty")
	if err != nil {
		t.Fatal(err)
	}
	app.Lobbies.SetReady(ctx, users[0], l.ID, true)
	if _, err = app.Lobbies.StartMatch(ctx, users[0], l.ID, "without-bot"); err != lobby.ErrNotReady && err != lobby.ErrUnconfirmed {
		t.Fatal("bot start not explicit", err)
	}
	app.Lobbies.ChooseBot(ctx, users[0], l.ID, true)
	var mID string
	for n := 0; n < 40; n++ {
		m, e := startClearedMatch(t, ctx, app.Lobbies, app.Hub, clock, users[0], l.ID, "explicit-bot")
		if e == nil {
			mID = m.MatchID
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if mID == "" {
		t.Fatal("bot did not start")
	}
	var result penalty.PublicSnapshot
	// Drain the bounded server delivery queue exactly as an active transport does.
	drained, cancelDrain := context.WithCancel(ctx)
	defer cancelDrain()
	go func() {
		for {
			select {
			case <-drained.Done():
				return
			case <-app.Hub.Events(j):
			}
		}
	}()
	for n := 0; n < 21; n++ {
		if err = app.Lobbies.ReadEngine(ctx, w.ID, mID, func(e lobby.Engine) error { result = e.(*penalty.Adapter).Snapshot(); return nil }); err != nil {
			t.Fatal(err)
		}
		if result.Result != nil {
			break
		}
		if _, err = execute("penalty.choose", map[string]any{"matchId": mID, "turnId": result.TurnID, "zone": 0}); err != nil {
			t.Fatal(err)
		}
		clock.set(result.AnimationEndsAt)
		app.Lobbies.Sync(ctx)
	}
	if result.Result == nil || len(result.Result.Participants) != 2 || result.Result.Participants[1].ActorType != "bot" || result.Result.Participants[1].UserID != "" {
		t.Fatal(result)
	}
	wait := func(check func() bool) {
		t.Helper()
		until := time.Now().Add(4 * time.Second)
		for time.Now().Before(until) {
			if check() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal("timed out")
	}
	wait(func() bool {
		all, _ := app.Lobbies.List(ctx, w.ID)
		for _, v := range all {
			if v.MatchID == mID {
				return v.State == "results"
			}
		}
		return false
	})
	if _, err = execute("match.return", map[string]any{"matchId": mID}); err != nil {
		t.Fatal(err)
	}
	if app.Lobbies.Busy(users[0]) {
		t.Fatal("completed claim retained")
	}
	record, err := app.History.Detail(ctx, users[0], mID)
	if err != nil || record.Status != "completed" {
		t.Fatal(record, err)
	}
	// Explicit departure finishes with forfeit, then the completed all-gone arena
	// releases automatically only after its result is durably finalized.
	l, err = app.Lobbies.JoinLobby(ctx, lobby.JoinContext{UserID: users[0], WorldID: w.ID}, "penalty")
	if err != nil {
		t.Fatal(err)
	}
	app.Lobbies.ChooseBot(ctx, users[0], l.ID, true)
	app.Lobbies.SetReady(ctx, users[0], l.ID, true)
	m, err := startClearedMatch(t, ctx, app.Lobbies, app.Hub, clock, users[0], l.ID, "departing")
	if err != nil {
		t.Fatal(err)
	}
	if err = app.Lobbies.Leave(ctx, users[0], w.ID); err != nil {
		t.Fatal(err)
	}
	wait(func() bool {
		r, e := app.History.Detail(ctx, users[0], m.MatchID)
		return e == nil && r.Status == "completed"
	})
	record, err = app.History.Detail(ctx, users[0], m.MatchID)
	if err != nil || record.Result.Participants[0].Outcome != "forfeit" || record.Result.Participants[1].Outcome != "win" {
		t.Fatal(record, err)
	}
	wait(func() bool {
		all, _ := app.Lobbies.List(ctx, w.ID)
		for _, v := range all {
			if v.MatchID == m.MatchID {
				return false
			}
		}
		return true
	})
	var claims int
	p.QueryRow(ctx, `SELECT count(*) FROM active_match_users WHERE match_id=$1`, m.MatchID).Scan(&claims)
	if claims != 0 || app.Lobbies.Busy(users[0]) {
		t.Fatal("departed completed claim retained", claims)
	}
}
