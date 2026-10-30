//go:build integration

package application

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"patra-world-social/server/internal/games/blaster"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"patra-world-social/server/internal/world"
)

type loadClient struct {
	windowStart          time.Time
	windows              map[int][2]int
	latest               map[string]json.RawMessage
	conn                 *websocket.Conn
	join                 realtime.JoinContext
	user                 string
	mu                   sync.Mutex
	bytes, activityBytes int
	events               map[string]int
	replies              chan realtime.Event
	seq                  uint64
	lastWrite            time.Time
	err                  error
}

func (c *loadClient) command(t *testing.T, ctx context.Context, kind string, payload any) json.RawMessage {
	t.Helper()
	if delay := 125*time.Millisecond - time.Since(c.lastWrite); delay > 0 {
		time.Sleep(delay)
	}
	c.lastWrite = time.Now()
	c.seq++
	id := fmt.Sprintf("load-%d", c.seq)
	raw, _ := json.Marshal(map[string]any{"v": 2, "worldId": c.join.WorldID, "commandId": id, "seq": c.seq, "type": kind, "payload": payload})
	if err := c.conn.Write(ctx, websocket.MessageText, raw); err != nil {
		t.Fatal(err)
	}
	for {
		select {
		case <-ctx.Done():
			t.Fatal(kind, ctx.Err())
		case ev := <-c.replies:
			var ack struct {
				CommandID string          `json:"commandId"`
				Result    json.RawMessage `json:"result"`
			}
			json.Unmarshal(ev.Payload, &ack)
			if ack.CommandID == id {
				if ev.Type != "command.ack" {
					t.Fatalf("%s: %s", kind, ev.Payload)
				}
				return ack.Result
			}
		}
	}
}
func (c *loadClient) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bytes = 0
	c.activityBytes = 0
	c.events = map[string]int{}
	c.windowStart = time.Now()
	c.windows = map[int][2]int{}
}
func (c *loadClient) metrics(seconds float64) map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	rates := map[string]float64{}
	for k, v := range c.events {
		rates[k] = float64(v) / seconds
	}
	maxTotal, maxActivities := 0, 0
	for second, window := range c.windows {
		if second < int(seconds) {
			if window[0] > maxTotal {
				maxTotal = window[0]
			}
			if window[1] > maxActivities {
				maxActivities = window[1]
			}
		}
	}
	return map[string]any{"maxCompleteSecondTotalKiB": float64(maxTotal) / 1024, "maxCompleteSecondActivitiesKiB": float64(maxActivities) / 1024, "totalKiBps": float64(c.bytes) / 1024 / seconds, "activitiesKiBps": float64(c.activityBytes) / 1024 / seconds, "eventHz": rates, "readError": fmt.Sprint(c.err)}
}

// Opt-in repeatable wall-clock profiling; full integration separately exercises authority and result persistence.
func TestActivitiesCombinedLoad(t *testing.T) {
	if os.Getenv("ACTIVITIES_LOAD") != "1" {
		t.Skip("set ACTIVITIES_LOAD=1 for actual 20/60 websocket wall-clock measurement")
	}
	for _, config := range []struct {
		name     string
		worlds   int
		scenario string
	}{{"A20", 1, "A"}, {"B20", 1, "B"}, {"A60", 3, "A"}} {
		t.Run(config.name, func(t *testing.T) {
			pool := testutil.Database(t)
			users, tokens := testutil.Users(t, pool, 20*config.worlds)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
			defer cancel()
			app, err := New(pool, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true})
			if err != nil {
				t.Fatal(err)
			}
			defer app.Close()
			server := httptest.NewServer(app.Handler)
			defer server.Close()
			clients := make([]*loadClient, len(users))
			worldIDs := []string{}
			for w := 0; w < config.worlds; w++ {
				created, e := app.Worlds.CreateWorld(ctx, users[w*20], fmt.Sprintf("Load %d", w))
				if e != nil {
					t.Fatal(e)
				}
				worldIDs = append(worldIDs, created.ID)
				for n := 0; n < 20; n++ {
					i := w*20 + n
					if n > 0 {
						if _, e = pool.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'member')`, created.ID, users[i]); e != nil {
							t.Fatal(e)
						}
					}
					conn, _, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/realtime?protocolVersion=2&worldId="+created.ID, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://127.0.0.1:4176"}, "Cookie": {"patra_session=" + tokens[i]}}})
					if e != nil {
						t.Fatal(e)
					}
					_, raw, e := conn.Read(ctx)
					if e != nil {
						t.Fatal(e)
					}
					var event realtime.Event
					json.Unmarshal(raw, &event)
					var snap realtime.PublicSnapshot
					json.Unmarshal(event.Payload, &snap)
					if snap.SessionID == "" {
						t.Fatal(string(raw))
					}
					c := &loadClient{conn: conn, join: realtime.JoinContext{Token: tokens[i], WorldID: created.ID, SessionID: snap.SessionID, Epoch: snap.Epoch}, user: users[i], events: map[string]int{}, latest: map[string]json.RawMessage{}, replies: make(chan realtime.Event, 64)}
					clients[i] = c
					defer conn.CloseNow()
					go func() {
						for {
							_, raw, e := conn.Read(ctx)
							if e != nil {
								c.mu.Lock()
								c.err = e
								c.mu.Unlock()
								return
							}
							var ev realtime.Event
							if e = json.Unmarshal(raw, &ev); e != nil {
								return
							}
							c.mu.Lock()
							c.bytes += len(raw)
							c.events[ev.Type]++
							c.latest[ev.Type] = append(json.RawMessage(nil), ev.Payload...)
							incremental := strings.HasPrefix(ev.Type, "activity.") || strings.HasPrefix(ev.Type, "race.") || strings.HasPrefix(ev.Type, "penalty.") || strings.HasPrefix(ev.Type, "pingpong.") || strings.HasPrefix(ev.Type, "blaster.") || strings.HasPrefix(ev.Type, "lobby.") || strings.HasPrefix(ev.Type, "match.")
							if incremental {
								c.activityBytes += len(raw)
							}
							if c.windows != nil {
								second := int(time.Since(c.windowStart).Seconds())
								bin := c.windows[second]
								bin[0] += len(raw)
								if incremental {
									bin[1] += len(raw)
								}
								c.windows[second] = bin
							}
							c.mu.Unlock()
							if ev.Type == "command.ack" || ev.Type == "command.error" {
								select {
								case c.replies <- ev:
								case <-ctx.Done():
									return
								}
							}
						}
					}()
				}
			}
			// Drain admission/resync bursts before the quiet baseline window.
			time.Sleep(500 * time.Millisecond)
			// Baseline includes the existing 20-avatar world plus idle new activity snapshots.
			for _, c := range clients {
				c.reset()
			}
			baseAt := time.Now()
			time.Sleep(2 * time.Second)
			baselineSeconds := time.Since(baseAt).Seconds()
			baseline := clients[19].metrics(baselineSeconds)
			matches := make([]map[string]string, config.worlds)
			for w, worldID := range worldIDs {
				cs := clients[w*20 : (w+1)*20]
				matches[w] = map[string]string{}
				groups := []struct {
					kind     string
					from, to int
				}{{"blaster", 7, 11}, {"pingpong", 11, 13}}
				if config.scenario == "A" {
					groups = append(groups, struct {
						kind     string
						from, to int
					}{"race", 0, 5}, struct {
						kind     string
						from, to int
					}{"penalty", 5, 7})
				}
				for _, group := range groups {
					var l lobby.Lobby
					for n := group.from; n < group.to; n++ {
						raw := cs[n].command(t, ctx, "lobby.join", map[string]any{"arenaId": group.kind})
						if err = json.Unmarshal(raw, &l); err != nil {
							t.Fatal(err)
						}
						cs[n].command(t, ctx, "lobby.ready", map[string]any{"lobbyId": l.ID, "ready": true})
					}
					app.Lobbies.SetStorageBlocked(false)
					if group.kind == "race" {
						cs[group.from].command(t, ctx, "match.start", map[string]any{"lobbyId": l.ID})
					}
					matches[w][group.kind] = l.ID
				}
				// Public placement is a fixture setup shortcut only; all activity intents travel actual authenticated WS.
				var contract world.ActivityContract
				json.Unmarshal(world.ActivitiesAsset, &contract)
				place := func(n int, p world.Point) {
					if e := app.Hub.SetActivity(ctx, worldID, cs[n].user, false, &p); e != nil {
						t.Fatal(e)
					}
				}
				for n, kind := range []string{"pingpong", "blaster"} {
					a, _ := world.ActivityArea(kind)
					b := a.SpectatorStrips[0]
					place(18+n, world.Point{X: b.X, Y: b.Y})
				}
				states, e := app.Hub.ActivitySnapshots(ctx, cs[13].join)
				if e != nil {
					t.Fatal(e)
				}
				for _, s := range states {
					if s.Kind == "claw" {
						place(13, contract.ClawMachines[0].Entrance)
						cs[13].command(t, ctx, "activity.claw", world.CasualIntent{Incarnation: s.Incarnation, Generation: s.Generation, MachineID: "claw-1", Action: "claw.enter"})
					}
				}
				for n, spot := range contract.FishingSpots {
					p := spot.Position
					if spot.ID == "fishing-3" {
						p.X = 54
					}
					place(14+n, p)
					states, e = app.Hub.ActivitySnapshots(ctx, cs[14+n].join)
					if e != nil {
						t.Fatal(e)
					}
					for _, s := range states {
						if s.AreaID == spot.ID {
							cs[14+n].command(t, ctx, "activity.fishing", world.CasualIntent{Incarnation: s.Incarnation, Generation: s.Generation, SpotID: spot.ID, Action: "fishing.cast"})
						}
					}
				}
				if config.scenario == "B" {
					states, e = app.Hub.ActivitySnapshots(ctx, cs[0].join)
					if e != nil {
						t.Fatal(e)
					}
					for _, s := range states {
						if s.Kind == "vehicles" {
							for n, v := range *s.Payload.Vehicles {
								place(n, contract.VehicleSlots[n].Position)
								cs[n].command(t, ctx, "activity.ride", world.CasualIntent{Incarnation: s.Incarnation, Generation: v.Generation, VehicleID: v.ID, Action: "mount"})
							}
						}
						if s.Kind == "football" {
							for n := 4; n < 7; n++ {
								place(n, world.Point{X: 28, Y: 46})
							}
							for n := 4; n < 7; n++ {
								fresh, e := app.Hub.ActivitySnapshots(ctx, cs[n].join)
								if e != nil {
									t.Fatal(e)
								}
								for _, ball := range fresh {
									if ball.Kind == "football" {
										cs[n].command(t, ctx, "activity.kick", world.CasualIntent{Incarnation: ball.Incarnation, Generation: ball.Generation, Action: "kick", DX: 1})
									}
								}
							}
						}
					}
				}
			}
			if config.scenario == "A" {
				time.Sleep(6 * time.Second)
			}
			for w := range worldIDs {
				for _, group := range []struct {
					kind string
					host int
				}{{"penalty", 5}, {"pingpong", 11}, {"blaster", 7}} {
					if id := matches[w][group.kind]; id != "" {
						clients[w*20+group.host].command(t, ctx, "match.start", map[string]any{"lobbyId": id})
					}
				}
			}
			time.Sleep(3200 * time.Millisecond)
			for w, worldID := range worldIDs {
				ls, e := app.Lobbies.List(ctx, worldID)
				if e != nil {
					t.Fatal(e)
				}
				for _, l := range ls {
					if matches[w][l.Arena.ID] != "" && matches[w][l.Arena.ID] == l.ID {
						if l.MatchID == "" || l.StartsAt == 0 {
							t.Fatal("match not active", l)
						}
						matches[w][l.Arena.ID] = l.MatchID
					}
				}
			}
			for _, c := range clients {
				c.reset()
			}
			worldTicks := make(chan time.Duration, 10000)
			gameTicks := make(chan time.Duration, 10000)
			observeWorld := func(d time.Duration) {
				select {
				case worldTicks <- d:
				default:
				}
			}
			for _, id := range worldIDs {
				if e := app.Hub.ObserveWorldTicks(ctx, id, observeWorld); e != nil {
					t.Fatal(e)
				}
			}
			app.Lobbies.ObserveTicks(func(d time.Duration) {
				select {
				case gameTicks <- d:
				default:
				}
			})
			start := time.Now()
			lastCasual := time.Time{}
			latencies := []float64{}
			end := start.Add(8 * time.Second)
			for time.Now().Before(end) {
				for w := range worldIDs {
					cs := clients[w*20 : (w+1)*20]
					if config.scenario == "B" || time.Since(lastCasual) >= time.Second {
						states, e := app.Hub.ActivitySnapshots(ctx, cs[13].join)
						if e != nil {
							t.Fatal(e)
						}
						for _, state := range states {
							if state.Kind == "fishing" && state.Payload.Fishing.OwnerID == "" {
								var n int
								fmt.Sscanf(state.AreaID, "fishing-%d", &n)
								cs[14+n].command(t, ctx, "activity.fishing", world.CasualIntent{Incarnation: state.Incarnation, Generation: state.Generation, SpotID: state.AreaID, Action: "fishing.cast"})
							}
							if state.Kind == "claw" && state.Payload.Claw.OperatorID == "" {
								cs[13].command(t, ctx, "activity.claw", world.CasualIntent{Incarnation: state.Incarnation, Generation: state.Generation, MachineID: "claw-1", Action: "claw.enter"})
							}
							if state.Kind == "vehicles" && config.scenario == "B" {
								for n, v := range *state.Payload.Vehicles {
									cs[n].command(t, ctx, "activity.ride", world.CasualIntent{Incarnation: state.Incarnation, Generation: v.Generation, VehicleID: v.ID, Action: "drive", DX: math.Sin(time.Since(start).Seconds())})
								}
							}
						}
					}
					cs[7].mu.Lock()
					blastRaw := append(json.RawMessage(nil), cs[7].latest["blaster.snapshot"]...)
					cs[7].mu.Unlock()
					var blast blaster.Snapshot
					json.Unmarshal(blastRaw, &blast)
					for n := 7; n < 11; n++ {
						if len(blast.Players) > n-7 && blast.Players[n-7].HP > 0 && blast.Result == nil {
							p := blast.Players[n-7]
							aim := 0.
							if len(blast.Bots) > 0 {
								aim = math.Atan2(blast.Bots[0].Y-p.Y, blast.Bots[0].X-p.X)
							}
							cs[n].command(t, ctx, "blaster.input", map[string]any{"matchId": matches[w]["blaster"], "dx": math.Sin(time.Since(start).Seconds()), "dy": math.Cos(time.Since(start).Seconds()), "aim": aim, "fire": true})
						}
					}
					for n := 11; n < 13; n++ {
						cs[n].command(t, ctx, "pingpong.input", map[string]any{"matchId": matches[w]["pingpong"], "axis": .5})
					}
					if config.scenario == "A" {
						for n := 0; n < 5; n++ {
							cs[n].command(t, ctx, "race.input", map[string]any{"matchId": matches[w]["race"], "throttle": 1, "brake": 0, "steer": 0})
						}
					}
					at := time.Now()
					if _, e := app.Hub.Snapshot(ctx, cs[19].join); e != nil {
						t.Fatal(e)
					}
					latencies = append(latencies, float64(time.Since(at).Nanoseconds())/1e6)
				}
				if time.Since(lastCasual) >= time.Second {
					lastCasual = time.Now()
				}
			}
			seconds := time.Since(start).Seconds()
			for _, id := range worldIDs {
				app.Hub.ObserveWorldTicks(ctx, id, nil)
			}
			app.Lobbies.ObserveTicks(nil)
			distribution := func(ch chan time.Duration) map[string]any {
				values := []float64{}
				for len(ch) > 0 {
					values = append(values, float64((<-ch).Nanoseconds())/1e6)
				}
				sort.Float64s(values)
				if len(values) == 0 {
					t.Fatal("no owner tick samples")
				}
				return map[string]any{"samples": len(values), "p50ms": values[len(values)/2], "p95ms": values[len(values)*95/100], "p99ms": values[len(values)*99/100]}
			}
			worldCosts, gameCosts := distribution(worldTicks), distribution(gameTicks)
			sort.Float64s(latencies)
			rows := []map[string]any{}
			for i, c := range clients {
				m := c.metrics(seconds)
				m["client"] = i
				m["world"] = i / 20
				rows = append(rows, m)
				c.mu.Lock()
				err := c.err
				c.mu.Unlock()
				if err != nil {
					t.Fatal("client lost", i, err)
				}
			}
			report := map[string]any{"scenario": config.name, "durationSeconds": seconds, "clients": len(clients), "worlds": config.worlds, "baselineDurationSeconds": baselineSeconds, "idleBaselineClient": baseline, "snapshotOwnerQueueP95ms": latencies[len(latencies)*95/100], "snapshotOwnerQueueSamples": len(latencies), "measurement": "actual authenticated test WS and isolated PostgreSQL schema; application JSON payload bytes exclude websocket/TCP/TLS overhead; owner queue metric is not pure tick CPU", "rows": rows, "worldOwnerTick": worldCosts, "gameOwnerAdvance": gameCosts}
			raw, _ := json.Marshal(report)
			t.Log("ACTIVITIES_METRICS " + string(raw))
			for i, row := range rows {
				limit := 40.
				if i%20 >= 18 {
					limit = 12
				}
				if row["activitiesKiBps"].(float64) > limit {
					t.Errorf("client %d activity traffic %.3f exceeds %.0fKiB/s", i, row["activitiesKiBps"], limit)
				}
			}
			if worldCosts["p95ms"].(float64) > 25 || gameCosts["p95ms"].(float64) > 25 {
				t.Error("owner tick p95 exceeds25ms")
			}
			for i, row := range rows {
				rates := row["eventHz"].(map[string]float64)
				if i%20 >= 18 {
					for _, kind := range []string{"race.snapshot", "pingpong.snapshot", "blaster.snapshot"} {
						if rates[kind] > 5+1/seconds {
							t.Errorf("spectator %d %s too fast", i, kind)
						}
					}
				}
			}
		})
	}
}
