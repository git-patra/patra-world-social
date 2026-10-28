package worlds

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"patra-world-social/server/internal/identity"
	"patra-world-social/server/internal/social"
)

var (
	ErrForbidden     = errors.New("world access denied")
	ErrInviteInvalid = errors.New("invite unavailable")
	ErrFull          = errors.New("world member limit reached")
	ErrInput         = errors.New("invalid world input")
	usernamePattern  = regexp.MustCompile(`^[A-Za-z0-9_]{3,24}$`)
)

type World struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	OwnerID  string `json:"ownerId"`
	Archived bool   `json:"archived"`
}
type Membership struct {
	WorldID string `json:"worldId"`
	UserID  string `json:"userId"`
	Role    string `json:"role"`
}
type InviteLink struct {
	ID        string    `json:"id"`
	WorldID   string    `json:"worldId"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}
type Member struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
}
type Invite struct {
	ID             string    `json:"id"`
	TargetUsername string    `json:"targetUsername,omitempty"`
	ExpiresAt      time.Time `json:"expiresAt"`
	Revoked        bool      `json:"revoked"`
	Redeemed       bool      `json:"redeemed"`
}
type Service struct {
	Pool    *pgxpool.Pool
	Revoked func(context.Context, string, string)
}

func New(pool *pgxpool.Pool) *Service { return &Service{Pool: pool} }

func (s *Service) CreateWorld(ctx context.Context, userID, name string) (World, error) {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 64 {
		return World{}, ErrInput
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return World{}, err
	}
	defer tx.Rollback(ctx)
	if err = lockAccounts(ctx, tx, userID); err != nil {
		return World{}, err
	}
	var w World
	err = tx.QueryRow(ctx, `INSERT INTO worlds(name,owner_id) VALUES($1,$2) RETURNING id,name,owner_id`, name, userID).Scan(&w.ID, &w.Name, &w.OwnerID)
	if err != nil {
		return World{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role) VALUES($1,$2,'owner')`, w.ID, userID)
	if err != nil {
		return World{}, err
	}
	return w, tx.Commit(ctx)
}
func (s *Service) ListWorlds(ctx context.Context, userID string) ([]World, error) {
	rows, err := s.Pool.Query(ctx, `SELECT w.id,w.name,w.owner_id,w.archived_at IS NOT NULL FROM worlds w JOIN world_members m ON m.world_id=w.id WHERE m.user_id=$1 AND m.active AND w.archived_at IS NULL ORDER BY w.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []World{}
	for rows.Next() {
		var w World
		if err = rows.Scan(&w.ID, &w.Name, &w.OwnerID, &w.Archived); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
func (s *Service) AuthorizeMember(ctx context.Context, userID, worldID string) (Membership, error) {
	var m Membership
	err := s.Pool.QueryRow(ctx, `SELECT m.world_id,m.user_id,m.role FROM world_members m JOIN worlds w ON w.id=m.world_id WHERE m.user_id=$1 AND m.world_id=$2 AND m.active AND w.archived_at IS NULL AND NOT EXISTS(SELECT 1 FROM world_bans b WHERE b.world_id=m.world_id AND b.user_id=m.user_id)`, userID, worldID).Scan(&m.WorldID, &m.UserID, &m.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrForbidden
	}
	return m, err
}
func (s *Service) GetWorld(ctx context.Context, userID, worldID string) (World, error) {
	if _, err := s.AuthorizeMember(ctx, userID, worldID); err != nil {
		return World{}, err
	}
	var w World
	err := s.Pool.QueryRow(ctx, `SELECT id,name,owner_id,archived_at IS NOT NULL FROM worlds WHERE id=$1`, worldID).Scan(&w.ID, &w.Name, &w.OwnerID, &w.Archived)
	return w, err
}
func (s *Service) ListMembers(ctx context.Context, userID, worldID string) ([]Member, error) {
	if _, err := s.AuthorizeMember(ctx, userID, worldID); err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT u.id,u.username,m.role FROM world_members m JOIN users u ON u.id=m.user_id WHERE m.world_id=$1 AND m.active ORDER BY CASE m.role WHEN 'owner' THEN 0 WHEN 'moderator' THEN 1 ELSE 2 END,u.username`, worldID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Member{}
	for rows.Next() {
		var m Member
		if err = rows.Scan(&m.UserID, &m.Username, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func lockAccounts(ctx context.Context, tx pgx.Tx, ids ...string) error {
	e := identity.LockActiveAccounts(ctx, tx, ids...)
	if errors.Is(e, identity.ErrDeleted) {
		return ErrForbidden
	}
	return e
}
func canModerate(role string) bool { return role == "owner" || role == "moderator" }
func lockWorld(ctx context.Context, tx pgx.Tx, worldID string) (bool, error) {
	var active bool
	err := tx.QueryRow(ctx, `SELECT archived_at IS NULL FROM worlds WHERE id=$1 FOR UPDATE`, worldID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrForbidden
	}
	return active, err
}
func actorRole(ctx context.Context, tx pgx.Tx, actorID, worldID string) (string, error) {
	var role string
	err := tx.QueryRow(ctx, `SELECT role FROM world_members WHERE world_id=$1 AND user_id=$2 AND active`, worldID, actorID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrForbidden
	}
	return role, err
}
func (s *Service) IssueInvite(ctx context.Context, actorID, worldID, targetUserID string) (InviteLink, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return InviteLink{}, err
	}
	defer tx.Rollback(ctx)
	if err = lockAccounts(ctx, tx, actorID, targetUserID); err != nil {
		return InviteLink{}, err
	}
	active, err := lockWorld(ctx, tx, worldID)
	if err != nil {
		return InviteLink{}, err
	}
	if !active {
		return InviteLink{}, ErrForbidden
	}
	role, err := actorRole(ctx, tx, actorID, worldID)
	if err != nil {
		return InviteLink{}, err
	}
	if !canModerate(role) {
		return InviteLink{}, ErrForbidden
	}
	var target any
	if targetUserID != "" {
		if err = social.LockPair(ctx, tx, actorID, targetUserID); err != nil {
			return InviteLink{}, err
		}
		blocked, e := social.Blocked(ctx, tx, actorID, targetUserID)
		if e != nil {
			return InviteLink{}, e
		}
		if blocked {
			return InviteLink{}, ErrForbidden
		}
		target = targetUserID
		var exists bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND deleted_at IS NULL)`, targetUserID).Scan(&exists)
		if err != nil {
			return InviteLink{}, err
		}
		if !exists {
			return InviteLink{}, ErrInput
		}
	}
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return InviteLink{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	var link InviteLink
	link.WorldID = worldID
	link.Token = token
	err = tx.QueryRow(ctx, `INSERT INTO world_invites(world_id,issuer_id,target_user_id,token_hash,expires_at) VALUES($1,$2,$3,$4,now()+interval '24 hours') RETURNING id,expires_at`, worldID, actorID, target, hash[:]).Scan(&link.ID, &link.ExpiresAt)
	if err != nil {
		return InviteLink{}, err
	}
	return link, tx.Commit(ctx)
}
func (s *Service) IssueInviteForUsername(ctx context.Context, actorID, worldID, username string) (InviteLink, error) {
	m, err := s.AuthorizeMember(ctx, actorID, worldID)
	if err != nil {
		return InviteLink{}, err
	}
	if !canModerate(m.Role) {
		return InviteLink{}, ErrForbidden
	}
	if !usernamePattern.MatchString(username) {
		return InviteLink{}, ErrInput
	}
	username = strings.ToLower(username)
	var id string
	err = s.Pool.QueryRow(ctx, `SELECT id FROM users WHERE username=$1 AND deleted_at IS NULL`, username).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return InviteLink{}, ErrInput
	}
	if err != nil {
		return InviteLink{}, err
	}
	return s.IssueInvite(ctx, actorID, worldID, id)
}
func (s *Service) RedeemInvite(ctx context.Context, userID, token string) (Membership, error) {
	if len(token) != 43 {
		return Membership{}, ErrInviteInvalid
	}
	hash := sha256.Sum256([]byte(token))
	return s.redeem(ctx, userID, "token_hash=$1", hash[:], false)
}
func (s *Service) AcceptTargetedInvite(ctx context.Context, userID, inviteID string) (Membership, error) {
	return s.redeem(ctx, userID, "id=$1", inviteID, true)
}
func (s *Service) redeem(ctx context.Context, userID, selector string, key any, targeted bool) (Membership, error) {
	// Discover immutable world/issuer IDs, then accounts -> world -> invite.
	var worldID, issuerID string
	err := s.Pool.QueryRow(ctx, `SELECT world_id,issuer_id FROM world_invites WHERE `+selector, key).Scan(&worldID, &issuerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrInviteInvalid
	}
	if err != nil {
		return Membership{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Membership{}, err
	}
	defer tx.Rollback(ctx)
	if err = lockAccounts(ctx, tx, userID, issuerID); err != nil {
		return Membership{}, err
	}
	active, err := lockWorld(ctx, tx, worldID)
	if err != nil {
		return Membership{}, err
	}
	if !active {
		return Membership{}, ErrInviteInvalid
	}
	var target, redeemed *string
	var valid bool
	err = tx.QueryRow(ctx, `SELECT target_user_id,redeemed_by,revoked_at IS NULL AND expires_at>now() FROM world_invites WHERE `+selector+` FOR UPDATE`, key).Scan(&target, &redeemed, &valid)
	if err != nil {
		return Membership{}, err
	}
	if (targeted && target == nil) || (target != nil && *target != userID) {
		return Membership{}, ErrInviteInvalid
	}
	if redeemed != nil {
		if *redeemed == userID {
			return s.memberInTx(ctx, tx, userID, worldID)
		}
		return Membership{}, ErrInviteInvalid
	}
	if !valid {
		return Membership{}, ErrInviteInvalid
	}
	var banned bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM world_bans WHERE world_id=$1 AND user_id=$2)`, worldID, userID).Scan(&banned)
	if err != nil {
		return Membership{}, err
	}
	if banned {
		return Membership{}, ErrForbidden
	}
	var count int
	err = tx.QueryRow(ctx, `SELECT count(*) FROM world_members WHERE world_id=$1 AND active`, worldID).Scan(&count)
	if err != nil {
		return Membership{}, err
	}
	var existing bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM world_members WHERE world_id=$1 AND user_id=$2 AND active)`, worldID, userID).Scan(&existing)
	if err != nil {
		return Membership{}, err
	}
	if !existing && count >= 50 {
		return Membership{}, ErrFull
	}
	if !existing {
		_, err = tx.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role,active) VALUES($1,$2,'member',true) ON CONFLICT(world_id,user_id) DO UPDATE SET active=true,role='member',joined_at=now()`, worldID, userID)
		if err != nil {
			return Membership{}, err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE world_invites SET redeemed_by=$2,redeemed_at=now() WHERE `+selector, key, userID)
	if err != nil {
		return Membership{}, err
	}
	membership, err := s.memberInTx(ctx, tx, userID, worldID)
	if err != nil {
		return Membership{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Membership{}, err
	}
	return membership, nil
}
func (s *Service) memberInTx(ctx context.Context, tx pgx.Tx, userID, worldID string) (Membership, error) {
	var m Membership
	err := tx.QueryRow(ctx, `SELECT world_id,user_id,role FROM world_members WHERE world_id=$1 AND user_id=$2 AND active`, worldID, userID).Scan(&m.WorldID, &m.UserID, &m.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrInviteInvalid
	}
	return m, err
}
