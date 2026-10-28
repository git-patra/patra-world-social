package worlds

import (
	"context"
	"time"
)

func (s *Service) ListInvites(ctx context.Context, actorID, worldID string) ([]Invite, error) {
	m, err := s.AuthorizeMember(ctx, actorID, worldID)
	if err != nil {
		return nil, err
	}
	if !canModerate(m.Role) {
		return nil, ErrForbidden
	}
	rows, err := s.Pool.Query(ctx, `SELECT i.id,COALESCE(u.username,''),i.expires_at,i.revoked_at IS NOT NULL,i.redeemed_at IS NOT NULL FROM world_invites i LEFT JOIN users u ON u.id=i.target_user_id WHERE i.world_id=$1 ORDER BY i.created_at DESC LIMIT 100`, worldID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Invite{}
	for rows.Next() {
		var i Invite
		if err = rows.Scan(&i.ID, &i.TargetUsername, &i.ExpiresAt, &i.Revoked, &i.Redeemed); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
func (s *Service) RevokeInvite(ctx context.Context, actorID, worldID, inviteID string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockAccounts(ctx, tx, actorID); err != nil {
		return err
	}
	active, err := lockWorld(ctx, tx, worldID)
	if err != nil {
		return err
	}
	if !active {
		return ErrForbidden
	}
	role, err := actorRole(ctx, tx, actorID, worldID)
	if err != nil {
		return err
	}
	if !canModerate(role) {
		return ErrForbidden
	}
	tag, err := tx.Exec(ctx, `UPDATE world_invites SET revoked_at=coalesce(revoked_at,now()) WHERE id=$1 AND world_id=$2 AND redeemed_at IS NULL`, inviteID, worldID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInviteInvalid
	}
	return tx.Commit(ctx)
}
func (s *Service) UnbanMember(ctx context.Context, actorID, worldID, targetID string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockAccounts(ctx, tx, actorID, targetID); err != nil {
		return err
	}
	active, err := lockWorld(ctx, tx, worldID)
	if err != nil {
		return err
	}
	if !active {
		return ErrForbidden
	}
	role, err := actorRole(ctx, tx, actorID, worldID)
	if err != nil {
		return err
	}
	if !canModerate(role) {
		return ErrForbidden
	}
	_, err = tx.Exec(ctx, `DELETE FROM world_bans WHERE world_id=$1 AND user_id=$2`, worldID, targetID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type Notification struct {
	ID             string    `json:"id"`
	WorldName      string    `json:"worldName"`
	IssuerUsername string    `json:"issuerUsername"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

// Notification metadata contains no raw token and is visible only to its bound target.
func (s *Service) Notifications(ctx context.Context, user string) ([]Notification, error) {
	rows, e := s.Pool.Query(ctx, `SELECT i.id,w.name,u.username,i.expires_at FROM world_invites i JOIN worlds w ON w.id=i.world_id JOIN users u ON u.id=i.issuer_id WHERE i.target_user_id=$1 AND i.revoked_at IS NULL AND i.redeemed_at IS NULL AND i.expires_at>now() AND w.archived_at IS NULL AND NOT EXISTS(SELECT 1 FROM user_blocks b WHERE(b.actor_id=$1 AND b.target_id=i.issuer_id)OR(b.target_id=$1 AND b.actor_id=i.issuer_id)) ORDER BY i.created_at DESC LIMIT 100`, user)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		if e = rows.Scan(&n.ID, &n.WorldName, &n.IssuerUsername, &n.ExpiresAt); e != nil {
			return nil, e
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
