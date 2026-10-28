// Package moderation uses world roles and records actions without message/audio content.
package moderation

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/worlds"
	"strconv"
	"time"
)

type Action struct {
	Kind      string `json:"kind"`
	UserID    string `json:"userId"`
	MessageID string `json:"messageId"`
	Seconds   int    `json:"seconds"`
}
type Service struct {
	Pool   *pgxpool.Pool
	Worlds *worlds.Service
	Hub    *realtime.Hub
}

func New(p *pgxpool.Pool, w *worlds.Service, h *realtime.Hub) *Service { return &Service{p, w, h} }
func (s *Service) Moderate(ctx context.Context, actor, world string, a Action) error {
	if a.Kind == "kick" {
		return s.Worlds.KickMember(ctx, actor, world, a.UserID)
	}
	if a.Kind != "mute" && a.Kind != "delete" {
		return worlds.ErrInput
	}
	if a.Kind == "mute" && (a.Seconds < 60 || a.Seconds > 86400) {
		return worlds.ErrInput
	}
	return s.Hub.WithControl(ctx, func(ctx context.Context) error { return s.moderate(ctx, actor, world, a) })
}
func (s *Service) moderate(ctx context.Context, actor, world string, a Action) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var active bool
	e = tx.QueryRow(ctx, `SELECT archived_at IS NULL FROM worlds WHERE id=$1 FOR UPDATE`, world).Scan(&active)
	if e != nil {
		return e
	}
	if !active {
		return worlds.ErrForbidden
	}
	var role string
	e = tx.QueryRow(ctx, `SELECT role FROM world_members WHERE world_id=$1 AND user_id=$2 AND active`, world, actor).Scan(&role)
	if errors.Is(e, pgx.ErrNoRows) {
		return worlds.ErrForbidden
	}
	if e != nil {
		return e
	}
	if role != "owner" && role != "moderator" {
		return worlds.ErrForbidden
	}
	var message any
	if a.Kind == "delete" {
		n, e := strconv.ParseInt(a.MessageID, 10, 64)
		if e != nil || n < 1 {
			return worlds.ErrInput
		}
		message = n
		e = tx.QueryRow(ctx, `SELECT user_id FROM chat_messages WHERE id=$1 AND world_id=$2`, n, world).Scan(&a.UserID)
		if errors.Is(e, pgx.ErrNoRows) {
			return worlds.ErrForbidden
		}
		if e != nil {
			return e
		}
	}
	var targetRole string
	e = tx.QueryRow(ctx, `SELECT role FROM world_members WHERE world_id=$1 AND user_id=$2`, world, a.UserID).Scan(&targetRole)
	if errors.Is(e, pgx.ErrNoRows) {
		return worlds.ErrForbidden
	}
	if e != nil {
		return e
	}
	if targetRole == "owner" && actor != a.UserID {
		return worlds.ErrForbidden
	}
	var expiry any
	if a.Kind == "mute" {
		if targetRole == "owner" || actor == a.UserID {
			return worlds.ErrForbidden
		}
		var until time.Time
		e = tx.QueryRow(ctx, `INSERT INTO chat_mutes(world_id,user_id,expires_at) VALUES($1,$2,now()+$3*interval '1 second') ON CONFLICT(world_id,user_id) DO UPDATE SET expires_at=excluded.expires_at RETURNING expires_at`, world, a.UserID, a.Seconds).Scan(&until)
		if e != nil {
			return e
		}
		expiry = until
	} else {
		if _, e = tx.Exec(ctx, `UPDATE chat_messages SET deleted_at=coalesce(deleted_at,now()) WHERE id=$1 AND world_id=$2`, message, world); e != nil {
			return e
		}
	}
	if _, e = tx.Exec(ctx, `INSERT INTO moderation_audit(world_id,actor_id,target_id,action,message_id,expires_at) VALUES($1,$2,$3,$4,$5,$6)`, world, actor, a.UserID, a.Kind, message, expiry); e != nil {
		return e
	}
	if e = tx.Commit(ctx); e != nil {
		return e
	}
	if a.Kind == "delete" {
		s.Hub.Publish(world, "chat.deleted", map[string]string{"worldId": world, "id": a.MessageID})
	}
	return nil
}
