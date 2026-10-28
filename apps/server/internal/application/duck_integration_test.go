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
	"os/exec"
	"patra-world-social/server/internal/auth"
	"patra-world-social/server/internal/games/duck"
	"patra-world-social/server/internal/games/model"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"patra-world-social/server/internal/world"
	"patra-world-social/server/internal/worlds"
	"strings"
	"testing"
	"time"
)

func TestDuckActualTransportMeasuredProfilesAndAuthority(t *testing.T) {
	for _, profile := range []struct{ rtt, jitter int64 }{{50, 0}, {50, 12}, {100, 0}, {100, 12}, {200, 0}, {200, 12}, {320, 0}, {320, 12}} {
		rtt, networkJitter := profile.rtt, profile.jitter
		t.Run(fmt.Sprintf("RTT%d/jitter%d", rtt, networkJitter), func(t *testing.T) {
			p := testutil.Database(t)
			users, tokens := testutil.Users(t, p, 2)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			clock := &penaltyClock{at: time.Now()}
			app, err := newApplication(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true}, clock)
			if err != nil {
				t.Fatal(err)
			}
			defer app.Close()
			w, err := app.Worlds.CreateWorld(ctx, users[0], "Duck synthetic")
			if err != nil {
				t.Fatal(err)
			}
			p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, w.ID, users[1])
			server := httptest.NewServer(app.Handler)
			defer server.Close()
			c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/realtime?worldId="+w.ID, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://127.0.0.1:4176"}, "Cookie": {"patra_session=" + tokens[0]}}})
			if err != nil {
				t.Fatal(err)
			}
			defer c.CloseNow()
			_, raw, err := c.Read(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var first realtime.Event
			json.Unmarshal(raw, &first)
			var seat realtime.PublicSnapshot
			json.Unmarshal(first.Payload, &seat)
			j := realtime.JoinContext{Token: tokens[0], WorldID: w.ID, SessionID: seat.SessionID, Epoch: seat.Epoch}
			counter := uint64(0)
			var resolutions []duck.ShotResolved
			recordResolution := func(ev realtime.Event) {
				if ev.Type != "duck.shot_resolved" {
					return
				}
				var resolved duck.ShotResolved
				if err := json.Unmarshal(ev.Payload, &resolved); err != nil {
					t.Fatal(err)
				}
				if resolved.WorldID != w.ID {
					t.Fatal("wrong event world", resolved)
				}
				resolutions = append(resolutions, resolved)
			}
			send := func(kind string, payload any) (string, json.RawMessage) {
				t.Helper()
				counter++
				b, _ := json.Marshal(payload)
				cmd := realtime.Command{V: 1, CommandID: fmt.Sprint(counter), Seq: counter, WorldID: w.ID, Type: kind, Payload: b}
				b, _ = json.Marshal(cmd)
				if e := c.Write(ctx, websocket.MessageText, b); e != nil {
					t.Fatal(e)
				}
				for {
					_, b, e := c.Read(ctx)
					if e != nil {
						t.Fatal(e)
					}
					ev, e := realtime.ValidateEvent(b)
					if e != nil {
						t.Fatal(e)
					}
					recordResolution(ev)
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
			l, err := app.Lobbies.JoinLobby(ctx, lobby.JoinContext{UserID: users[0], WorldID: w.ID}, "duck-1")
			if err != nil {
				t.Fatal(err)
			}
			app.Lobbies.SetReady(ctx, users[0], l.ID, true)
			app.Lobbies.SetStorageBlocked(false)
			m, err := app.Lobbies.StartMatch(ctx, users[0], l.ID, "duck")
			if err != nil {
				t.Fatal(err)
			}
			var persisted model.MatchStart
			p.QueryRow(ctx, `SELECT start_payload FROM matches WHERE id=$1`, m.MatchID).Scan(&raw)
			json.Unmarshal(raw, &persisted)
			if persisted.RulesID != "duck-v1" || persisted.RulesChecksum != duck.RulesChecksum() || m.StartsAt-m.PreparedAt != 3000 {
				t.Fatal(persisted, m)
			}
			// Find a newly visible fast target only through the public snapshot
			// endpoint, never the engine or its seed. This leaves enough lifetime for
			// the measured exchanges and delayed presentation below.
			var public duck.PublicSnapshot
			targetID := 0
			for n := int64(0); n < 40 && targetID == 0; n++ {
				clock.set(m.StartsAt + n*700 + 100)
				value, e := app.Hub.Execute(ctx, j, realtime.Command{V: 1, CommandID: "public-probe", WorldID: w.ID, Type: "duck.snapshot", Payload: json.RawMessage(fmt.Sprintf(`{"matchId":%q}`, m.MatchID))})
				if e != nil {
					t.Fatal(e)
				}
				public = value.(duck.PublicSnapshot)
				for _, target := range public.Targets {
					if target.Kind == "fast" && target.ExpiresAt-public.ServerTimeMs >= 2200 {
						targetID = target.ID
					}
				}
			}
			if targetID == 0 {
				t.Fatal("no public fast target in fixture session")
			}
			base := public.ServerTimeMs + 50
			var kind string
			clock.set(base)
			kind, raw = send("duck.clock", map[string]any{"matchId": m.MatchID})
			if kind != "command.ack" {
				t.Fatal("challenge", kind)
			}
			var ch duck.ClockChallenge
			json.Unmarshal(raw, &ch)
			clock.set(base + rtt)
			kind, _ = send("duck.clock_reply", map[string]any{"matchId": m.MatchID, "nonce": ch.Nonce, "clientTimeMs": float64(base) + float64(rtt)/2 - 9000})
			if kind != "command.ack" {
				t.Fatal("reply", kind)
			}
			// A second observed exchange includes +12ms synthetic jitter. The
			// conservative recent minimum retains the first measured RTT.
			second := base + rtt + 200
			clock.set(second)
			kind, raw = send("duck.clock", map[string]any{"matchId": m.MatchID})
			if kind != "command.ack" {
				t.Fatal(kind)
			}
			var jitter duck.ClockChallenge
			json.Unmarshal(raw, &jitter)
			clock.set(second + rtt + 12)
			kind, raw = send("duck.clock_reply", map[string]any{"matchId": m.MatchID, "nonce": jitter.Nonce, "clientTimeMs": float64(second) + float64(rtt+12)/2 - 9000})
			if kind != "command.ack" {
				t.Fatal(kind)
			}
			calibrationReply := append(json.RawMessage{}, raw...)
			publication := second + rtt + 300
			clock.set(publication)
			kind, raw = send("duck.snapshot", map[string]any{"matchId": m.MatchID})
			if kind != "command.ack" || strings.Contains(string(raw), "seed") || strings.Contains(string(raw), "born") {
				t.Fatal(kind, string(raw))
			}
			var snapshot duck.PublicSnapshot
			if err = json.Unmarshal(raw, &snapshot); err != nil {
				t.Fatal(err)
			}
			delivery := publication + rtt/2 + networkJitter
			click := delivery + 55 // R1 exact200ms/+55ms case is one of these profiles.
			received := click + rtt/2 + networkJitter
			expectedRewind := min(int64(150), rtt/2)
			input := map[string]any{"snapshot": snapshot, "challenge": jitter, "clockReply": calibrationReply, "calibrationClientAt": float64(second) + float64(rtt+12)/2 - 9000, "receivedClientAt": delivery - 9000, "clickClientAt": click - 9000, "targetId": targetID}
			shown := duckPresentedInput(t, input)
			if !shown.Clicked || !shown.Calibrated || shown.Stale {
				t.Fatal("view did not show live calibrated target", shown)
			}
			clock.set(received)
			kind, raw = send("duck.shoot", shown.Payload)
			var shot duck.ShotResult
			json.Unmarshal(raw, &shot)
			if kind != "command.ack" || shot.Points != 200 || shot.TargetID != targetID || shot.RewindMs != expectedRewind || shot.Shots != 1 {
				t.Fatal(kind, shot, string(raw))
			}
			t.Logf("public-view RTT=%dms path jitter=%dms (+12ms clock sample) click55ms after delivery; renderAt-P=%.1fms rewind=%dms fast hit=%d score=%d", rtt, networkJitter, shown.At-float64(publication), shot.RewindMs, shot.TargetID, shot.Score)
			input["resolution"] = shot
			input["afterClickClientAt"] = click + 1 - 9000
			deadView := duckPresentedInput(t, input)
			if deadView.After == nil || deadView.After.Clicked {
				t.Fatal("actual acknowledged dead target rendered again", deadView)
			}
			delete(input, "resolution")
			delete(input, "afterClickClientAt")
			var expires int64
			for _, target := range snapshot.Targets {
				if target.ID == targetID {
					expires = target.ExpiresAt
				}
			}
			input["clickClientAt"] = expires - 9000 // at or beyond the known lifetime, including capped prediction lead.
			expiredView := duckPresentedInput(t, input)
			if expiredView.Clicked {
				t.Fatal("expired target remained clickable", expiredView)
			}
			// An outside-hitbox aim uses that same SVG pointer callback. It is sent
			// after the receipt-time cooldown; y=.95 is outside every current lane.
			input["mode"] = "outside"
			input["clickClientAt"] = click + 250 - 9000
			outside := duckPresentedInput(t, input)
			if !outside.Clicked {
				t.Fatal("outside fixture target unexpectedly hidden")
			}
			clock.set(received + 250)
			kind, raw = send("duck.shoot", outside.Payload)
			var miss duck.ShotResult
			json.Unmarshal(raw, &miss)
			if kind != "command.ack" || miss.Points != 0 || miss.TargetID != 0 || miss.Shots != 2 {
				t.Fatal("outside displayed hitbox scored", kind, miss)
			}

			// The publisher can follow the ACK. Read the real websocket event,
			// then compare its identity with that same authoritative ACK.
			for len(resolutions) < 2 {
				_, b, e := c.Read(ctx)
				if e != nil {
					t.Fatal(e)
				}
				ev, e := realtime.ValidateEvent(b)
				if e != nil {
					t.Fatal(e)
				}
				recordResolution(ev)
			}
			if len(resolutions) != 2 || resolutions[0].ShotResult != shot || resolutions[1].ShotResult != miss || resolutions[0].ServerTimeMs != received || resolutions[1].ServerTimeMs != received+250 {
				t.Fatal("accepted hit/miss transport events", resolutions, shot, miss)
			}
			clock.set(received + 600)
			kind, _ = send("duck.shoot", map[string]any{"matchId": m.MatchID, "x": 0.5, "y": 0.5, "clientTimeMs": float64(received + 999999)})
			if kind != "command.error" {
				t.Fatal("fabricated timestamp accepted")
			}
			kind, _ = send("duck.clock_reply", map[string]any{"matchId": m.MatchID, "nonce": ch.Nonce, "clientTimeMs": 100})
			if kind != "command.error" {
				t.Fatal("nonce replay accepted")
			}

			// Flush another publisher interval after rejected commands: neither
			// a rejection nor the next snapshot may create another resolution.
			time.Sleep(100 * time.Millisecond)
			send("duck.snapshot", map[string]any{"matchId": m.MatchID})
			if len(resolutions) != 2 {
				t.Fatal("rejected command emitted shot", resolutions)
			}
			other, _ := app.Hub.Join(ctx, realtime.JoinContext{Token: tokens[1], WorldID: w.ID})
			oj := realtime.JoinContext{Token: tokens[1], WorldID: w.ID, SessionID: other.SessionID, Epoch: other.Epoch}
			payload := json.RawMessage(fmt.Sprintf(`{"matchId":%q,"x":0.5,"y":0.5,"clientTimeMs":100}`, m.MatchID))
			if _, err = app.Hub.Execute(ctx, oj, realtime.Command{V: 1, CommandID: "spectator", Seq: 90, WorldID: w.ID, Type: "duck.shoot", Payload: payload}); err == nil {
				t.Fatal("spectator shot")
			}
			bad := j
			bad.Epoch++
			if _, err = app.Hub.Execute(ctx, bad, realtime.Command{V: 1, CommandID: "epoch", Seq: 91, WorldID: w.ID, Type: "duck.shoot", Payload: payload}); err == nil {
				t.Fatal("stale generation accepted")
			}
			clock.set(m.StartsAt + 60000)
			app.Lobbies.Sync(ctx)
			waitDuckSaved(t, ctx, app, m.MatchID)
			record, err := app.History.Detail(ctx, users[0], m.MatchID)
			if err != nil || record.Result == nil || record.Result.Participants[0].Duck.Hits != 1 || record.Result.Participants[0].Duck.Shots != 2 || *record.Result.Participants[0].Score != shot.Score {
				t.Fatal(record, err)
			}
			if _, err = app.History.Detail(ctx, users[1], m.MatchID); err == nil {
				t.Fatal("spectator private history")
			}
			var count int
			p.QueryRow(ctx, `SELECT count(*) FROM duck_results WHERE match_id=$1 AND complete`, m.MatchID).Scan(&count)
			if count != 1 {
				t.Fatal("missing projection")
			}
			if _, err = app.History.Finalize(ctx, m.MatchID); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func waitDuckSaved(t *testing.T, ctx context.Context, app *Application, id string) {
	t.Helper()
	until := time.Now().Add(4 * time.Second)
	for time.Now().Before(until) {
		all, _ := app.Lobbies.Finalizing(ctx)
		pending := false
		for _, m := range all {
			if m.MatchID == id {
				pending = true
			}
		}
		if !pending {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	r, e := app.Lobbies.CopyResult(ctx, id)
	enq := app.History.Enqueue(ctx, r)
	_, fin := app.History.Finalize(ctx, id)
	t.Fatal("duck not saved", e, enq, fin)
}
func TestDuckWiredSchedulerAndIncompleteHistory(t *testing.T) {
	for _, tc := range []struct {
		name, kind         string
		disconnect, ingest int64
		complete           bool
	}{{"grace", "disconnect", 59000, 61000, true}, {"equal", "disconnect", 45000, 61000, true}, {"expired", "disconnect", 40000, 61000, false}, {"leave", "leave", 59000, 61000, false}} {
		t.Run(tc.name, func(t *testing.T) {
			p := testutil.Database(t)
			users, _ := testutil.Users(t, p, 1)
			ctx := context.Background()
			clock := &penaltyClock{at: time.Now()}
			app, err := newApplication(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true}, clock)
			if err != nil {
				t.Fatal(err)
			}
			defer app.Close()
			w, _ := app.Worlds.CreateWorld(ctx, users[0], "Duck lifecycle")
			l, _ := app.Lobbies.JoinLobby(ctx, lobby.JoinContext{UserID: users[0], WorldID: w.ID}, "duck-1")
			app.Lobbies.SetReady(ctx, users[0], l.ID, true)
			app.Lobbies.SetStorageBlocked(false)
			m, err := app.Lobbies.StartMatch(ctx, users[0], l.ID, "start")
			if err != nil {
				t.Fatal(err)
			}
			clock.set(m.StartsAt)
			app.Lobbies.DrainChanges(ctx)
			events := 0
			for at := m.StartsAt + 50; at <= m.StartsAt+1000; at += 50 {
				clock.set(at)
				changes, _ := app.Lobbies.DrainChanges(ctx)
				for _, ev := range changes.Events {
					if ev.Type == "duck.snapshot" {
						events++
					}
				}
			}
			var snap duck.PublicSnapshot
			app.Lobbies.ReadEngine(ctx, w.ID, m.MatchID, func(e lobby.Engine) error { snap = e.(*duck.Adapter).Snapshot(); return nil })
			if snap.Frame != 30 {
				t.Fatal("actual owner cadence", snap.Frame, events)
			}
			entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
			go func() {
				done <- app.Lobbies.ReadEngine(ctx, w.ID, m.MatchID, func(lobby.Engine) error { close(entered); <-release; return nil })
			}()
			<-entered
			clock.set(m.StartsAt + tc.disconnect)
			app.Lobbies.Notify(tc.kind, w.ID, users[0])
			clock.set(m.StartsAt + tc.ingest)
			close(release)
			if err = <-done; err != nil {
				t.Fatal(err)
			}
			app.Lobbies.Sync(ctx)
			waitDuckSaved(t, ctx, app, m.MatchID)
			r, err := app.History.Detail(ctx, users[0], m.MatchID)
			if err != nil || r.Result == nil || r.Result.Participants[0].BotAssisted || (r.Result.Participants[0].Outcome == "finished") != tc.complete {
				t.Fatal(r, err)
			}
			var complete bool
			var hits, shots int
			if err = p.QueryRow(ctx, `SELECT complete,hits,shots FROM duck_results WHERE match_id=$1`, m.MatchID).Scan(&complete, &hits, &shots); err != nil || complete != tc.complete || hits != 0 || shots != 0 {
				t.Fatal(complete, hits, shots, err)
			}
		})
	}
}

func TestDuckActualWiredScheduler30Hz15Hz(t *testing.T) {
	p := testutil.Database(t)
	users, _ := testutil.Users(t, p, 1)
	ctx := context.Background()
	clock := &penaltyClock{at: time.Now()}
	m, _ := world.LoadMap(world.MapAsset)
	ws := worlds.New(p)
	w, _ := ws.CreateWorld(ctx, users[0], "Duck scheduler")
	h := realtime.NewHub(auth.New(p), ws, m, clock)
	defer h.Close()
	g := wireLobbiesClock(p, h, m, clock)
	defer g.Close()
	l, _ := g.JoinLobby(ctx, lobby.JoinContext{UserID: users[0], WorldID: w.ID}, "duck-1")
	g.SetReady(ctx, users[0], l.ID, true)
	start, err := g.StartMatch(ctx, users[0], l.ID, "scheduler")
	if err != nil {
		t.Fatal(err)
	}
	clock.set(start.StartsAt)
	g.DrainChanges(ctx)
	published := 0
	for at := start.StartsAt + 50; at <= start.StartsAt+1000; at += 50 {
		clock.set(at)
		changes, err := g.DrainChanges(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, ev := range changes.Events {
			if ev.Type == "duck.snapshot" {
				published++
			}
		}
	}
	var snap duck.PublicSnapshot
	g.ReadEngine(ctx, w.ID, start.MatchID, func(e lobby.Engine) error { snap = e.(*duck.Adapter).Snapshot(); return nil })
	if snap.Frame != 30 || published != 15 {
		t.Fatal(snap.Frame, published)
	}
}

// Integration-only: existing Node + npm/esbuild are explicit test prerequisites.
// The bounded child executes deterministic React/controller logic, no browser.
type duckPresented struct {
	Payload    json.RawMessage `json:"payload"`
	At         float64         `json:"at"`
	Clicked    bool            `json:"clicked"`
	Calibrated bool            `json:"calibrated"`
	Stale      bool            `json:"stale"`
	After      *struct {
		Clicked bool `json:"clicked"`
	} `json:"after"`
}

func duckPresentedInput(t *testing.T, input map[string]any) duckPresented {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	payload, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, "node", "../../../web/scripts/duck-online-fixture.mjs")
	command.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err = command.Run(); err != nil {
		t.Fatalf("public-view bridge requires installed Node/npm tooling: %v %s", err, stderr.String())
	}
	var out duckPresented
	if err = json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatal(err, stdout.String())
	}
	return out
}
