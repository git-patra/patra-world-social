// Package chat persists world plain-text messages; table channels need actual T13 seats.
package chat

import (
	"context"
	"crypto/sha256"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"patra-world-social/server/internal/db"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/worlds"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrInput = errors.New("invalid chat message")
var ErrMuted = errors.New("world chat muted")
var ErrRate = errors.New("chat rate exceeded")

type Message struct {
	ID              string    `json:"id"`
	WorldID         string    `json:"worldId"`
	UserID          string    `json:"userId"`
	Username        string    `json:"username"`
	ClientMessageID string    `json:"clientMessageId"`
	Text            string    `json:"text"`
	CreatedAt       time.Time `json:"createdAt"`
	Deleted         bool      `json:"deleted"`
}
type Service struct {
	Pool   *pgxpool.Pool
	Hub    *realtime.Hub
	Worlds *worlds.Service
}

func New(p *pgxpool.Pool, h *realtime.Hub, w *worlds.Service) *Service { return &Service{p, h, w} }
func Validate(channel, id, text string) error {
	if channel != "world" || !utf8.ValidString(text) || !utf8.ValidString(id) || utf8.RuneCountInString(text) > 500 || strings.TrimSpace(text) == "" || utf8.RuneCountInString(id) < 1 || utf8.RuneCountInString(id) > 128 {
		return ErrInput
	}
	return nil
}
func (s *Service) SendMessage(ctx context.Context, j realtime.JoinContext, channel, id, text string) (Message, error) {
	if e := Validate(channel, id, text); e != nil {
		return Message{}, e
	}
	var m Message
	e := s.Hub.WithJoined(ctx, j, func(ctx context.Context, user string) error {
		tx, e := s.Pool.Begin(ctx)
		if e != nil {
			return e
		}
		defer tx.Rollback(ctx)
		// Same world-first order as membership/moderation writers; session row fences logout.
		var active bool
		e = tx.QueryRow(ctx, `SELECT archived_at IS NULL FROM worlds WHERE id=$1 FOR UPDATE`, j.WorldID).Scan(&active)
		if e != nil {
			return e
		}
		if !active {
			return worlds.ErrForbidden
		}
		hash := sha256.Sum256([]byte(j.Token))
		var valid bool
		e = tx.QueryRow(ctx, `SELECT revoked_at IS NULL AND expires_at>clock_timestamp() AND idle_expires_at>clock_timestamp() FROM sessions WHERE token_hash=$1 AND user_id=$2 FOR UPDATE`, hash[:], user).Scan(&valid)
		if e != nil {
			return e
		}
		if !valid {
			return realtime.ErrUnauthorized
		}
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM world_members m WHERE world_id=$1 AND user_id=$2 AND active AND NOT EXISTS(SELECT 1 FROM world_bans WHERE world_id=$1 AND user_id=$2))`, j.WorldID, user).Scan(&valid)
		if e != nil {
			return e
		}
		if !valid {
			return worlds.ErrForbidden
		}
		var muted bool
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM chat_mutes WHERE world_id=$1 AND user_id=$2 AND expires_at>clock_timestamp())`, j.WorldID, user).Scan(&muted)
		if e != nil {
			return e
		}
		if muted {
			return ErrMuted
		}
		e = scan(tx.QueryRow(ctx, `SELECT c.id::text,c.world_id,c.user_id,u.username,c.client_message_id,CASE WHEN c.deleted_at IS NULL THEN c.text ELSE '' END,c.created_at,c.deleted_at IS NOT NULL FROM chat_messages c JOIN users u ON u.id=c.user_id WHERE c.world_id=$1 AND c.user_id=$2 AND c.client_message_id=$3`, j.WorldID, user, id), &m)
		if e == nil {
			s.Hub.Publish(j.WorldID, "chat.message", m)
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		// Serialize all this account's worlds to keep the five-token bucket global.
		if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,1))`, user); e != nil {
			return e
		}
		var tokens float64
		e = tx.QueryRow(ctx, `INSERT INTO chat_limits(user_id,tokens,updated_at) VALUES($1,5,clock_timestamp()) ON CONFLICT(user_id) DO UPDATE SET tokens=least(5,chat_limits.tokens+greatest(0,extract(epoch FROM clock_timestamp()-chat_limits.updated_at))),updated_at=clock_timestamp() RETURNING tokens`, user).Scan(&tokens)
		if e != nil {
			return e
		}
		if tokens < 1 {
			return ErrRate
		}
		if _, e = tx.Exec(ctx, `UPDATE chat_limits SET tokens=tokens-1 WHERE user_id=$1`, user); e != nil {
			return e
		}
		e = tx.QueryRow(ctx, `INSERT INTO chat_messages(world_id,user_id,client_message_id,text) VALUES($1,$2,$3,$4) RETURNING id::text,created_at`, j.WorldID, user, id, text).Scan(&m.ID, &m.CreatedAt)
		if e != nil {
			return e
		}
		m.WorldID = j.WorldID
		m.UserID = user
		m.ClientMessageID = id
		m.Text = text
		if e = tx.QueryRow(ctx, `SELECT username FROM users WHERE id=$1`, user).Scan(&m.Username); e != nil {
			return e
		}
		if e = tx.Commit(ctx); e != nil {
			return e
		}
		s.Hub.Publish(j.WorldID, "chat.message", m)
		return nil
	})
	return m, e
}
func scan(row pgx.Row, m *Message) error {
	return row.Scan(&m.ID, &m.WorldID, &m.UserID, &m.Username, &m.ClientMessageID, &m.Text, &m.CreatedAt, &m.Deleted)
}
func (s *Service) History(ctx context.Context, user, world, before string) ([]Message, error) {
	if _, e := s.Worlds.AuthorizeMember(ctx, user, world); e != nil {
		return nil, e
	}
	cursor := int64(9223372036854775807)
	if before != "" {
		var e error
		cursor, e = strconv.ParseInt(before, 10, 64)
		if e != nil || cursor < 1 {
			return nil, ErrInput
		}
	}
	rows, e := db.New(s.Pool).ListWorldChatPage(ctx, db.ListWorldChatPageParams{WorldID: world, UserID: user, BeforeID: cursor})
	if e != nil {
		return nil, e
	}
	out := []Message{}
	for _, row := range rows {
		out = append(out, Message{ID: row.ID, WorldID: row.WorldID, UserID: row.UserID, Username: row.Username, ClientMessageID: row.ClientMessageID, Text: row.Text, CreatedAt: row.CreatedAt.Time, Deleted: row.Deleted})
	}
	return out, nil
}
func (s *Service) Cleanup(ctx context.Context) error { return db.New(s.Pool).CleanupWorldChat(ctx) }
