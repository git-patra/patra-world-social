// Package application owns the shared services and realtime revocation wiring.
package application

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"patra-world-social/server/internal/auth"
	"patra-world-social/server/internal/chat"
	"patra-world-social/server/internal/games/duck"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/games/penalty"
	"patra-world-social/server/internal/games/race"
	"patra-world-social/server/internal/history"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/identity"
	"patra-world-social/server/internal/leaderboard"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/moderation"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/social"
	"patra-world-social/server/internal/voice"
	"patra-world-social/server/internal/world"
	"patra-world-social/server/internal/worlds"
	"time"
)

type Application struct {
	Voice       *voice.Service
	voiceDone   chan struct{}
	Auth        *auth.Service
	Worlds      *worlds.Service
	Hub         *realtime.Hub
	Handler     http.Handler
	Social      *social.Service
	Chat        *chat.Service
	Moderation  *moderation.Service
	History     *history.Service
	Leaderboard *leaderboard.Service
	resultsDone chan struct{}
	Lobbies     *lobby.Service
	lobbiesDone chan struct{}
	cancel      context.CancelFunc
	cleaned     chan struct{}
}

func New(pool *pgxpool.Pool, opt httpapi.Options) (*Application, error) {
	return newApplication(pool, opt, world.WallClock{})
}
func newApplication(pool *pgxpool.Pool, opt httpapi.Options, clock world.Clock) (*Application, error) {
	m, e := world.LoadMap(world.MapAsset)
	if e != nil {
		return nil, e
	}
	if _, e = world.LoadRules(world.RulesAsset); e != nil {
		return nil, e
	}
	results := history.New(pool)
	results.Projectors["penalty"] = penalty.Project
	results.Projectors["race"] = race.Project
	boards := leaderboard.New(pool)
	results.Projectors["duck"] = func(ctx context.Context, tx pgx.Tx, r model.MatchResult) error {
		if err := duck.Project(ctx, tx, r); err != nil {
			return err
		}
		return leaderboard.ProjectResult(ctx, tx, r)
	}
	recoveryCtx, recoveryCancel := context.WithTimeout(context.Background(), 10*time.Second)
	e = boards.Backfill(recoveryCtx)
	if e == nil {
		e = results.RecoverPending(recoveryCtx)
	}
	recoveryCancel()
	if e != nil {
		return nil, e
	}
	a := auth.New(pool)
	w := worlds.New(pool)
	hub := realtime.NewHub(a, w, m, clock)
	g := wireLobbiesClock(pool, hub, m, clock)
	g.SetStorageBlocked(true)
	v, provider, e := wireVoice(pool, hub, w, m)
	if e != nil {
		hub.Close()
		g.Close()
		return nil, e
	}
	// Auth callback is account-scoped: logout closes this user's online seat even
	// if a different valid session owns it; recovery closes all account sessions.
	a.Revoked = func(_ context.Context, user string) { hub.Revoke("", user) }
	a.RemoveAccount = leaderboard.RemoveAccount
	redactions := &identity.Redactions{}
	rows, err := pool.Query(context.Background(), `SELECT id::text FROM users WHERE deleted_at IS NOT NULL`)
	if err != nil {
		hub.Close()
		g.Close()
		return nil, err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			hub.Close()
			g.Close()
			return nil, err
		}
		redactions.Add(id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		hub.Close()
		g.Close()
		return nil, err
	}
	hub.RedactPayload = redactions.JSON
	a.DeleteFence = func(ctx context.Context, user string, commit func(context.Context) error) error {
		return hub.DeleteAccount(ctx, user, g.Sync, commit)
	}
	a.Deleted = func(user string, partitions [][3]string) {
		redactions.Add(user)
		for _, p := range partitions {
			hub.Publish(p[0], "leaderboard.changed", leaderboard.Invalidation{WorldID: p[0], Ruleset: p[1], ResultID: p[2]})
		}
	}

	w.Revoked = func(_ context.Context, world, user string) { hub.Revoke(world, user) }
	s := social.New(pool)
	s.Online = hub.Online
	c := chat.New(pool, hub, w)
	moderationService := moderation.New(pool, w, hub)
	hub.Chat = func(ctx context.Context, j realtime.JoinContext, command realtime.Command) (any, error) {
		var payload struct {
			ChannelID       string `json:"channelId"`
			ClientMessageID string `json:"clientMessageId"`
			Text            string `json:"text"`
		}
		if e := json.Unmarshal(command.Payload, &payload); e != nil {
			return nil, e
		}
		return c.SendMessage(ctx, j, payload.ChannelID, payload.ClientMessageID, payload.Text)
	}
	opt.Social = s
	opt.Chat = c
	opt.Moderation = moderationService
	opt.Worlds = w
	router, e := httpapi.NewRouter(a, opt)
	if e != nil {
		hub.Close()
		g.Close()
		return nil, e
	}
	mux := http.NewServeMux()
	mux.Handle("/api/v1/voice/", v.Handler(provider))
	mux.Handle("/api/v1/realtime", realtime.NewHandler(hub, opt.Origin, opt.Development))
	historyHandler := httpapi.SessionMiddleware(a, opt.Development, results.Handler())
	mux.Handle("/api/v1/history", historyHandler)
	mux.Handle("/api/v1/history/", historyHandler)
	mux.Handle("/api/v1/leaderboards/", httpapi.SessionMiddleware(a, opt.Development, boards.Handler()))
	mux.Handle("/api/", router)
	ctx, cancel := context.WithCancel(context.Background())
	voiceDone := make(chan struct{})
	go func() { defer close(voiceDone); v.Run(ctx) }()
	resultsDone := make(chan struct{})
	worker := history.NewWorker(results, g)
	worker.Notify = func(world string, users []string, event history.StatusEvent) {
		hub.PublishParticipants(world, users, "match.result_status", event)
		if event.Status == "completed" {
			notificationCtx, done := context.WithTimeout(ctx, 3*time.Second)
			defer done()
			invalidation, err := boards.InvalidationFor(notificationCtx, event.MatchID)
			if err != nil {
				slog.Error("podium invalidation unavailable", "match_id", event.MatchID)
				return
			}
			if invalidation != nil {
				hub.Publish(invalidation.WorldID, "leaderboard.changed", invalidation)
			}
		}
	}
	go func() { defer close(resultsDone); worker.Run(ctx) }()
	lobbiesDone := make(chan struct{})
	go func() { defer close(lobbiesDone); publishLobbies(ctx, g, hub) }()
	cleaned := make(chan struct{})
	go func() {
		defer close(cleaned)
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			cleanupCtx, done := context.WithTimeout(ctx, 3*time.Second)
			if err := c.Cleanup(cleanupCtx); err != nil && ctx.Err() == nil {
				slog.Error("chat retention cleanup failed")
			}
			done()
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return &Application{Voice: v, voiceDone: voiceDone, Leaderboard: boards, History: results, resultsDone: resultsDone, Auth: a, Worlds: w, Hub: hub, Handler: mux, Social: s, Chat: c, Moderation: moderationService, Lobbies: g, lobbiesDone: lobbiesDone, cancel: cancel, cleaned: cleaned}, nil
}
func (a *Application) Close() {
	a.cancel()
	<-a.cleaned
	<-a.resultsDone
	<-a.lobbiesDone
	<-a.voiceDone
	a.Hub.Close()
	cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_ = a.Voice.ReconcileParticipants(cleanup)
	cancel()
	a.Lobbies.Close()
}
