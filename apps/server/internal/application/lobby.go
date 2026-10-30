package application

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"patra-world-social/server/internal/games/blaster"
	"patra-world-social/server/internal/games/duck"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/games/penalty"
	"patra-world-social/server/internal/games/pingpong"
	"patra-world-social/server/internal/games/race"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/world"
	"time"
)

func wireLobbies(pool *pgxpool.Pool, hub *realtime.Hub, m world.MapDefinition) *lobby.Service {
	return wireLobbiesClock(pool, hub, m, world.WallClock{})
}
func wireLobbiesClock(pool *pgxpool.Pool, hub *realtime.Hub, m world.MapDefinition, clock world.Clock) *lobby.Service {
	rules, err := world.LoadRules(world.RulesAsset)
	if err != nil {
		panic(err)
	}
	g := lobby.New(&lobby.PGStore{Pool: pool}, clock, model.Version{MapID: m.ID, MapVersion: m.V, MapChecksum: m.Checksum, RulesID: rules.ID, RulesVersion: rules.V, RulesChecksum: rules.Checksum, ProtocolVersion: 2})
	g.SetAreaCoordinator(areaCoordinator{hub})
	g.Register("penalty", penalty.Factory(clock))
	if err := g.RegisterVersioned("blaster", blaster.Factory(clock), lobby.RulesMetadata{ID: "blaster-v1", Version: 1, Checksum: blaster.RulesChecksum()}); err != nil {
		panic(err)
	}
	if err := g.RegisterVersioned("pingpong", pingpong.Factory(clock), lobby.RulesMetadata{ID: "pingpong-v1", Version: 1, Checksum: pingpong.RulesChecksum()}); err != nil {
		panic(err)
	}
	dr := duck.DefaultRules()
	if err := g.RegisterVersioned("duck", duck.Factory(clock), lobby.RulesMetadata{ID: dr.ID, Version: dr.Version, Checksum: duck.RulesChecksum()}); err != nil {
		panic(err)
	}
	g.RegisterPrepared("race", race.Factory(clock), race.Prepare)
	hub.GameBusy = g.MovementBusy
	hub.VoiceExcluded = g.VoiceExcluded
	// Notify only coalesces presence in a bounded-by-admitted-users mailbox. It
	// never calls storage, publishes, or reenters the hub while its lock is held.
	hub.SeatChanged = g.Notify
	hub.Games = func(ctx context.Context, j realtime.JoinContext, c realtime.Command) (any, error) {
		var result any
		admission := hub.WithJoined
		if c.Type == "blaster.input" || c.Type == "pingpong.input" || c.Type == "pingpong.serve" || c.Type == "penalty.choose" || c.Type == "race.input" || c.Type == "duck.shoot" || c.Type == "duck.clock" || c.Type == "duck.clock_reply" {
			admission = hub.WithAdmitted
		}
		e := admission(ctx, j, func(ctx context.Context, user string) error {

			if c.Type == "blaster.snapshot" || c.Type == "blaster.input" {
				var p struct {
					MatchID string `json:"matchId"`
					blaster.Controls
				}
				if err := json.Unmarshal(c.Payload, &p); err != nil {
					return err
				}
				if c.Type == "blaster.snapshot" {
					return g.ReadEngine(ctx, j.WorldID, p.MatchID, func(engine lobby.Engine) error {
						a, ok := engine.(*blaster.Adapter)
						if !ok {
							return lobby.ErrUnavailable
						}
						result = a.Snapshot()
						return nil
					})
				}
				return g.InputEngine(ctx, j.WorldID, p.MatchID, user, func(engine lobby.Engine, at time.Time) error {
					if err := realtime.CheckAdmissionTime(ctx, at); err != nil {
						return err
					}
					a, ok := engine.(*blaster.Adapter)
					if !ok {
						return lobby.ErrUnavailable
					}
					err := a.Input(user, c.Seq, p.Controls, at)
					result = map[string]any{"matchId": p.MatchID, "accepted": err == nil}
					return err
				})
			}
			if c.Type == "pingpong.snapshot" || c.Type == "pingpong.input" || c.Type == "pingpong.serve" {
				var p struct {
					MatchID string  `json:"matchId"`
					Axis    float64 `json:"axis"`
					RallyID int     `json:"rallyId"`
				}
				if err := json.Unmarshal(c.Payload, &p); err != nil {
					return err
				}
				if c.Type == "pingpong.snapshot" {
					return g.ReadEngine(ctx, j.WorldID, p.MatchID, func(engine lobby.Engine) error {
						a, ok := engine.(*pingpong.Adapter)
						if !ok {
							return lobby.ErrUnavailable
						}
						result = a.Snapshot()
						return nil
					})
				}
				return g.InputEngine(ctx, j.WorldID, p.MatchID, user, func(engine lobby.Engine, at time.Time) error {
					if err := realtime.CheckAdmissionTime(ctx, at); err != nil {
						return err
					}
					a, ok := engine.(*pingpong.Adapter)
					if !ok {
						return lobby.ErrUnavailable
					}
					var err error
					if c.Type == "pingpong.input" {
						err = a.Input(user, c.Seq, p.Axis, at)
					} else {
						err = a.Serve(user, p.RallyID, at)
					}
					result = map[string]any{"matchId": p.MatchID, "accepted": err == nil}
					return err
				})
			}
			if c.Type == "duck.snapshot" || c.Type == "duck.shoot" || c.Type == "duck.clock" || c.Type == "duck.clock_reply" {
				var p struct {
					MatchID      string  `json:"matchId"`
					Nonce        string  `json:"nonce"`
					ClientTimeMs float64 `json:"clientTimeMs"`
					duck.Aim
				}
				if err := json.Unmarshal(c.Payload, &p); err != nil {
					return err
				}
				if c.Type == "duck.snapshot" {
					return g.ReadEngine(ctx, j.WorldID, p.MatchID, func(engine lobby.Engine) error {
						a, ok := engine.(*duck.Adapter)
						if !ok {
							return lobby.ErrUnavailable
						}
						result = a.Snapshot()
						return nil
					})
				}
				return g.InputEngine(ctx, j.WorldID, p.MatchID, user, func(engine lobby.Engine, at time.Time) error {
					if err := realtime.CheckAdmissionTime(ctx, at); err != nil {
						return err
					}
					a, ok := engine.(*duck.Adapter)
					if !ok {
						return lobby.ErrUnavailable
					}
					var err error
					switch c.Type {
					case "duck.clock":
						result, err = a.Challenge(user, j.Epoch, at)
					case "duck.clock_reply":
						err = a.Observe(user, j.Epoch, p.Nonce, p.ClientTimeMs, at)
						calibration, measured := a.Calibration(at)
						result = struct {
							Measured    bool                  `json:"measured"`
							Calibration duck.ClockCalibration `json:"calibration"`
						}{measured && err == nil, calibration}
					case "duck.shoot":
						result, err = a.Shoot(user, j.Epoch, c.Seq, p.Aim, p.ClientTimeMs, at)
					}
					return err
				})
			}
			if c.Type == "race.snapshot" || c.Type == "race.input" {
				var p struct {
					MatchID string `json:"matchId"`
					race.Controls
				}
				if err := json.Unmarshal(c.Payload, &p); err != nil {
					return err
				}
				if c.Type == "race.snapshot" {
					return g.ReadEngine(ctx, j.WorldID, p.MatchID, func(engine lobby.Engine) error {
						a, ok := engine.(*race.Adapter)
						if !ok {
							return lobby.ErrUnavailable
						}
						result = a.Snapshot()
						return nil
					})
				}
				return g.InputEngine(ctx, j.WorldID, p.MatchID, user, func(engine lobby.Engine, at time.Time) error {
					if err := realtime.CheckAdmissionTime(ctx, at); err != nil {
						return err
					}
					a, ok := engine.(*race.Adapter)
					if !ok {
						return lobby.ErrUnavailable
					}
					if err := a.Input(user, c.Seq, p.Controls, at); err != nil {
						return err
					}
					result = map[string]any{"matchId": p.MatchID, "accepted": true}
					return nil
				})
			}
			if c.Type == "blaster.input" || c.Type == "pingpong.input" || c.Type == "pingpong.serve" || c.Type == "penalty.choose" || c.Type == "penalty.snapshot" || c.Type == "match.return" || c.Type == "match.rematch" {
				var p struct {
					MatchID string `json:"matchId"`
					TurnID  string `json:"turnId"`
					Zone    int    `json:"zone"`
				}
				if err := json.Unmarshal(c.Payload, &p); err != nil {
					return err
				}
				if c.Type == "penalty.snapshot" {
					return g.ReadEngine(ctx, j.WorldID, p.MatchID, func(engine lobby.Engine) error {
						a, ok := engine.(*penalty.Adapter)
						if !ok {
							return lobby.ErrUnavailable
						}
						result = a.Snapshot()
						return nil
					})
				}
				if c.Type == "penalty.choose" {
					return g.InputEngine(ctx, j.WorldID, p.MatchID, user, func(engine lobby.Engine, receivedAt time.Time) error {
						if err := realtime.CheckAdmissionTime(ctx, receivedAt); err != nil {
							return err
						}
						a, ok := engine.(*penalty.Adapter)
						if !ok {
							return lobby.ErrUnavailable
						}
						if err := a.Choose(user, c.CommandID, p.TurnID, p.Zone, receivedAt); err != nil {
							return err
						}
						result = map[string]any{"matchId": p.MatchID, "turnId": p.TurnID, "chosen": true}
						return nil
					})
				}
				plan, err := g.ResultAction(ctx, j.WorldID, p.MatchID, user, c.Type == "match.rematch")
				if err != nil {
					return err
				}
				result = map[string]bool{"waiting": !plan.Ready}
				if !plan.Ready {
					return nil
				}
				return g.CompleteResultAction(p.MatchID)
			}

			var p struct {
				ArenaID string `json:"arenaId"`
				LobbyID string `json:"lobbyId"`
				Ready   bool   `json:"ready"`
				Enabled bool   `json:"enabled"`
			}
			if err := json.Unmarshal(c.Payload, &p); err != nil {
				return err
			}
			if c.Type == "lobby.list" {
				v, err := g.List(ctx, j.WorldID)
				result = map[string]any{"worldId": j.WorldID, "lobbies": v}
				return err
			}
			if c.Type == "lobby.join" {
				var name, avatar string
				if err := pool.QueryRow(ctx, `SELECT display_name,avatar FROM users WHERE id=$1`, user).Scan(&name, &avatar); err != nil {
					return err
				}
				v, err := g.JoinLobby(ctx, lobby.JoinContext{UserID: user, WorldID: j.WorldID, DisplayName: name, AvatarID: avatar}, p.ArenaID)
				if err != nil {
					return err
				}
				result = v
				return hub.SetActivity(ctx, j.WorldID, user, g.MovementBusy(user), nil)
			}
			if c.Type == "lobby.leave" {
				return g.Leave(ctx, user, j.WorldID)
			}
			all, err := g.List(ctx, j.WorldID)
			if err != nil {
				return err
			}
			inWorld := false
			for _, l := range all {
				if l.ID == p.LobbyID && l.ID != "" {
					inWorld = true
				}
			}
			if !inWorld {
				return lobby.ErrForbidden
			}
			switch c.Type {
			case "lobby.ready":
				return g.SetReady(ctx, user, p.LobbyID, p.Ready)
			case "lobby.bot":
				return g.ChooseBot(ctx, user, p.LobbyID, p.Enabled)
			case "match.start":
				_, err := g.RequestStart(ctx, user, p.LobbyID, c.CommandID)
				if err != nil {
					return err
				}
				all, err = g.List(ctx, j.WorldID)
				if err != nil {
					return err
				}
				for _, l := range all {
					if l.ID == p.LobbyID {
						result = l
						return nil
					}
				}
				return lobby.ErrForbidden
			}
			return realtime.ErrUnsupported
		})
		return result, e
	}
	return g
}

// publishLobbies drains coalesced changes outside either owner. Slow transports
// use the existing bounded hub delivery queues; lobby.list provides full resync.
func publishLobbies(ctx context.Context, g *lobby.Service, hub *realtime.Hub) {
	started := map[string]string{}
	retry := time.NewTicker(100 * time.Millisecond)
	defer retry.Stop()
	pendingReturns := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-g.Changed:
		case <-retry.C:
			if !pendingReturns {
				continue
			}
		}
		c, e := g.DrainChanges(ctx)
		if e != nil {
			pendingReturns = true
			continue
		}
		for _, event := range c.Events {
			if event.Replaceable {
				hub.PublishGameSnapshot(ctx, event.WorldID, event.AreaID, event.MatchID, event.Participants, event.Spectators, event.Type, event.Payload)
			} else {
				hub.Publish(event.WorldID, event.Type, event.Payload)
			}
		}
		pendingReturns = applyLobbyReturns(ctx, g, hub, c.Returns)
		for _, w := range c.Worlds {
			all, err := g.List(ctx, w)
			if err == nil {
				hub.Publish(w, "lobby.updated", map[string]any{"worldId": w, "lobbies": all})
				for _, l := range all {
					key := w + ":" + l.Arena.ID
					if l.MatchID == "" {
						delete(started, key)
					} else if l.StartsAt > 0 && started[key] != l.MatchID {
						started[key] = l.MatchID
						hub.Publish(w, "match.started", l)
					}
					for _, p := range l.Players {
						hub.SetActivity(ctx, w, p.UserID, g.MovementBusy(p.UserID), nil)
					}
				}
			}
		}
	}
}

type lobbyReturner interface {
	ReturnActivity(context.Context, string, string, world.Point, world.ActivityGuard) error
}

func applyLobbyReturns(ctx context.Context, g *lobby.Service, target lobbyReturner, returns []lobby.Return) bool {
	pending := false
	for _, r := range returns {
		// This prefilter is optional; the guard repeats the generation check atomically
		// with the world mutation after waiting for hub/world queue ownership.
		if !g.WithReturn(r, func() {}) {
			if g.AckReturn(r) != nil {
				pending = true
			}
			continue
		}
		at := world.Point{X: r.X, Y: r.Y}
		err := target.ReturnActivity(ctx, r.WorldID, r.UserID, at, func(apply func()) bool { return g.WithReturn(r, apply) })
		if err != nil && !errors.Is(err, world.ErrActivitySuperseded) {
			pending = true
			continue
		}
		if g.AckReturn(r) != nil {
			pending = true
		}
	}
	return pending
}
