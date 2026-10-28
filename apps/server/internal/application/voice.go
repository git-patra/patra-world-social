package application

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/voice"
	"patra-world-social/server/internal/world"
	"patra-world-social/server/internal/worlds"
	"regexp"
)

type voiceGate struct {
	hub    *realtime.Hub
	worlds *worlds.Service
}

func (g voiceGate) Within(ctx context.Context, j voice.JoinContext, fn func(voice.Subject) error) error {
	return g.hub.WithJoined(ctx, j, func(ctx context.Context, user string) error {
		member, err := g.worlds.AuthorizeMember(ctx, user, j.WorldID)
		if err != nil {
			return err
		}
		return g.hub.WithPosition(ctx, j, func(a world.Avatar) error {
			return fn(voice.Subject{UserID: a.ID, Name: a.Name, Role: member.Role, X: a.X, Y: a.Y, Playing: a.Status == "playing"})
		})
	})
}
func wireVoice(pool *pgxpool.Pool, hub *realtime.Hub, w *worlds.Service, m world.MapDefinition) (*voice.Service, *voice.LiveKit, error) {
	if err := voice.ValidateZones(m); err != nil {
		return nil, nil, err
	}
	var p *voice.LiveKit
	var provider voice.Provider
	prefix := os.Getenv("LIVEKIT_ROOM_PREFIX")
	url, key, secret := os.Getenv("LIVEKIT_URL"), os.Getenv("LIVEKIT_API_KEY"), os.Getenv("LIVEKIT_API_SECRET")
	if url != "" || key != "" || secret != "" {
		if !regexp.MustCompile(`^patra-[a-z0-9-]{8,40}-$`).MatchString(prefix) {
			return nil, nil, errors.New("LIVEKIT_ROOM_PREFIX must be a unique patra-<8-40 lowercase namespace>- prefix")
		}
		var err error
		p, err = voice.NewLiveKit(url, key, secret)
		if err != nil {
			return nil, nil, err
		}
		provider = p
	}
	s := wireVoiceService(pool, hub, w, provider, prefix)
	return s, p, nil
}
func wireVoiceService(pool *pgxpool.Pool, hub *realtime.Hub, w *worlds.Service, provider voice.Provider, prefix string) *voice.Service {
	s := voice.New(voiceGate{hub, w}, provider, voice.PGJournal{Pool: pool}, prefix)
	s.Audit = func(ctx context.Context, world, actor, target, action string) error {
		_, err := pool.Exec(ctx, `INSERT INTO moderation_audit(world_id,actor_id,target_id,action) VALUES($1,$2,$3,$4)`, world, actor, target, "voice_"+action)
		return err
	}
	previous := hub.SeatChanged
	hub.SeatChanged = func(kind, world, user string) {
		s.Revoke(world, user, kind)
		if previous != nil {
			previous(kind, world, user)
		}
	}
	hub.VoiceObserve = s.Observe
	hub.VoiceRevoke = s.Revoke
	hub.Voice = func(ctx context.Context, j realtime.JoinContext, c realtime.Command) (any, error) {
		var user string
		err := hub.WithJoined(ctx, j, func(_ context.Context, id string) error { user = id; return nil })
		if err != nil {
			return nil, err
		}
		var payload struct {
			Table           int    `json:"table"`
			Target          string `json:"target"`
			Action          string `json:"action"`
			ClientMessageID string `json:"clientMessageId"`
			Text            string `json:"text"`
		}
		if err = json.Unmarshal(c.Payload, &payload); err != nil {
			return nil, err
		}
		if c.Type == "voice.join" {
			return s.JoinTable(ctx, j, payload.Table)
		}
		a, ok := s.Owned(j, user)
		if !ok {
			if c.Type == "voice.state" || c.Type == "voice.leave" {
				return map[string]any{"status": s.Status()}, nil
			}
			return nil, voice.ErrForbidden
		}
		switch c.Type {
		case "voice.state":
			return s.State(ctx, a.ID)
		case "voice.token":
			return s.IssueToken(ctx, a.ID)
		case "voice.leave":
			return nil, s.LeaveTable(ctx, a.ID, "leave")
		case "voice.chat":
			return s.Chat(ctx, a.ID, payload.ClientMessageID, payload.Text)
		case "voice.moderate":
			return nil, s.Moderate(ctx, a.ID, payload.Target, payload.Action)
		case "voice.unmute":
			return nil, s.Unmute(ctx, a.ID)
		}
		return nil, realtime.ErrUnsupported
	}
	return s
}
