package auth

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"unicode/utf8"
)

var ErrOwnsWorld = errors.New("transfer or archive owned worlds first")

type Settings struct {
	Audio         bool   `json:"audio"`
	Controls      string `json:"controls"`
	ReducedMotion bool   `json:"reducedMotion"`
}
type Profile struct {
	DisplayName string   `json:"displayName"`
	Avatar      string   `json:"avatar"`
	Settings    Settings `json:"settings"`
}

func (s *Service) Profile(ctx context.Context, id string) (Profile, error) {
	var p Profile
	var raw []byte
	e := s.Pool.QueryRow(ctx, `SELECT display_name,avatar,settings FROM users WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&p.DisplayName, &p.Avatar, &raw)
	if e == nil {
		e = json.Unmarshal(raw, &p.Settings)
	}
	return p, e
}
func (s *Service) UpdateProfile(ctx context.Context, id string, p Profile) (User, error) {
	if !utf8.ValidString(p.DisplayName) || utf8.RuneCountInString(p.DisplayName) < 1 || utf8.RuneCountInString(p.DisplayName) > 32 || (p.Avatar != "pumpkin" && p.Avatar != "autumn" && p.Avatar != "moon") || (p.Settings.Controls != "both" && p.Settings.Controls != "keyboard" && p.Settings.Controls != "touch") {
		return User{}, ErrInput
	}
	b, _ := json.Marshal(p.Settings)
	var u User
	e := s.Pool.QueryRow(ctx, `UPDATE users SET display_name=$2,avatar=$3,settings=$4 WHERE id=$1 AND deleted_at IS NULL RETURNING id::text,username,display_name,avatar`, id, p.DisplayName, p.Avatar, b).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Avatar)
	if errors.Is(e, pgx.ErrNoRows) {
		e = ErrInvalid
	}
	return u, e
}

// UpdateMotion changes one setting atomically against the latest stored profile.
func (s *Service) UpdateMotion(ctx context.Context, id string, reduced bool) (Profile, error) {
	var p Profile
	var raw []byte
	e := s.Pool.QueryRow(ctx, `UPDATE users SET settings=jsonb_set(settings,'{reducedMotion}',to_jsonb($2::boolean),true) WHERE id=$1 AND deleted_at IS NULL RETURNING display_name,avatar,settings`, id, reduced).Scan(&p.DisplayName, &p.Avatar, &raw)
	if errors.Is(e, pgx.ErrNoRows) {
		return p, ErrInvalid
	}
	if e == nil {
		e = json.Unmarshal(raw, &p.Settings)
	}
	return p, e
}
func (s *Service) credentialChange(ctx context.Context, id, password, next string, rotate bool) (RecoveryCode, error) {
	if !rotate {
		if e := validatePassword(next); e != nil {
			return "", e
		}
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(ctx)
	var hash string
	e = tx.QueryRow(ctx, `SELECT password_hash FROM users WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&hash)
	if errors.Is(e, pgx.ErrNoRows) {
		return "", ErrInvalid
	}
	if e != nil {
		return "", e
	}
	if !verifyPassword(ctx, hash, password) {
		return "", ErrInvalid
	}
	var code string
	if rotate {
		code, e = randomString(32)
		if e == nil {
			_, e = tx.Exec(ctx, `UPDATE users SET recovery_hash=$2 WHERE id=$1`, id, tokenHash(code))
		}
	} else {
		hash, e = hashPassword(ctx, next)
		if e == nil {
			_, e = tx.Exec(ctx, `UPDATE users SET password_hash=$2 WHERE id=$1`, id, hash)
		}
		if e == nil {
			_, e = tx.Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, id)
		}
	}
	if e != nil {
		return "", e
	}
	if e = tx.Commit(ctx); e != nil {
		return "", e
	}
	if !rotate && s.Revoked != nil {
		s.Revoked(ctx, id)
	}
	return RecoveryCode(code), nil
}
func (s *Service) ChangePassword(ctx context.Context, id, password, next string) error {
	_, e := s.credentialChange(ctx, id, password, next, false)
	return e
}
func (s *Service) RotateRecovery(ctx context.Context, id, password string) (RecoveryCode, error) {
	return s.credentialChange(ctx, id, password, "", true)
}
func (s *Service) DeleteAccount(ctx context.Context, id, confirmation string) error {
	var username string
	var owns bool
	e := s.Pool.QueryRow(ctx, `SELECT username,EXISTS(SELECT 1 FROM worlds WHERE owner_id=$1 AND archived_at IS NULL) FROM users WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&username, &owns)
	if errors.Is(e, pgx.ErrNoRows) {
		return ErrInvalid
	}
	if e != nil {
		return e
	}
	if confirmation != username {
		return ErrInput
	}
	if owns {
		return ErrOwnsWorld
	}
	if s.RemoveAccount == nil {
		return errors.New("account removal unavailable")
	}
	work := func(ctx context.Context) error {
		tx, e := s.Pool.Begin(ctx)
		if e != nil {
			return e
		}
		defer tx.Rollback(ctx)
		// Global order: admission lease (application), account advisory 12, user row,
		// ordered board rows (RemoveAccount), then membership/participant projections.
		if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,12))`, id); e != nil {
			return e
		}
		e = tx.QueryRow(ctx, `SELECT username FROM users WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&username)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrInvalid
		}
		if e != nil {
			return e
		}
		if username != confirmation {
			return ErrInput
		}
		if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM worlds WHERE owner_id=$1 AND archived_at IS NULL)`, id).Scan(&owns); e != nil {
			return e
		}
		if owns {
			return ErrOwnsWorld
		}
		// Capture exact partitions before the transactional projector removes entries.
		rows, e := tx.Query(ctx, `SELECT world_id::text,ruleset,result_id::text FROM leaderboard_entries WHERE user_id=$1 ORDER BY world_id,ruleset`, id)
		if e != nil {
			return e
		}
		var partitions [][3]string
		for rows.Next() {
			var p [3]string
			if e = rows.Scan(&p[0], &p[1], &p[2]); e != nil {
				rows.Close()
				return e
			}
			partitions = append(partitions, p)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if e = s.RemoveAccount(ctx, tx, id); e != nil {
			return e
		}
		for _, q := range []string{
			`UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`,
			`UPDATE world_members SET active=false WHERE user_id=$1`,
			`UPDATE world_invites SET revoked_at=coalesce(revoked_at,now()) WHERE issuer_id=$1 OR target_user_id=$1`,
			`DELETE FROM friend_requests WHERE actor_id=$1 OR target_id=$1`,
			`DELETE FROM user_blocks WHERE actor_id=$1 OR target_id=$1`,
			`DELETE FROM chat_mutes WHERE user_id=$1`,
			`DELETE FROM chat_limits WHERE user_id=$1`,
			`UPDATE chat_messages SET text='Pesan dihapus',deleted_at=coalesce(deleted_at,now()) WHERE user_id=$1`,
			`UPDATE match_participants SET display_name='Akun dihapus',avatar_id='pumpkin' WHERE user_id=$1`,
			`UPDATE users SET username='~deleted:'||id::text,display_name='Akun dihapus',avatar='pumpkin',password_hash='',recovery_hash=''::bytea,settings='{}',deleted_at=now() WHERE id=$1`,
		} {
			if _, e = tx.Exec(ctx, q, id); e != nil {
				return e
			}
		}
		// voice_revocations has no cascading FK and is intentionally retained.
		if e = tx.Commit(ctx); e != nil {
			return e
		}
		if s.Deleted != nil {
			s.Deleted(id, partitions)
		}
		return nil
	}
	if s.DeleteFence != nil {
		return s.DeleteFence(ctx, id, work)
	}
	e = work(ctx)
	if e == nil && s.Revoked != nil {
		s.Revoked(ctx, id)
	}
	return e
}
