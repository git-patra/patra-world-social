package worlds

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

func (s *Service) ListBans(ctx context.Context, actorID, worldID string) ([]Member, error) {
	m, err := s.AuthorizeMember(ctx, actorID, worldID)
	if err != nil {
		return nil, err
	}
	if !canModerate(m.Role) {
		return nil, ErrForbidden
	}
	rows, err := s.Pool.Query(ctx, `SELECT u.id,u.username FROM world_bans b JOIN users u ON u.id=b.user_id WHERE b.world_id=$1 ORDER BY u.username`, worldID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Member{}
	for rows.Next() {
		var m Member
		if err = rows.Scan(&m.UserID, &m.Username); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Service) ChangeRole(ctx context.Context, actorID, worldID, targetID, role string) error {
	if role != "moderator" && role != "member" {
		return ErrInput
	}
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
	actor, err := actorRole(ctx, tx, actorID, worldID)
	if err != nil {
		return err
	}
	if actor != "owner" || actorID == targetID {
		return ErrForbidden
	}
	target, err := actorRole(ctx, tx, targetID, worldID)
	if err != nil {
		return err
	}
	if target == "owner" {
		return ErrForbidden
	}
	_, err = tx.Exec(ctx, `UPDATE world_members SET role=$3 WHERE world_id=$1 AND user_id=$2 AND active`, worldID, targetID, role)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Service) KickMember(ctx context.Context, actorID, worldID, targetID string) error {
	return s.removeMember(ctx, actorID, worldID, targetID, false)
}
func (s *Service) BanMember(ctx context.Context, actorID, worldID, targetID string) error {
	return s.removeMember(ctx, actorID, worldID, targetID, true)
}
func (s *Service) removeMember(ctx context.Context, actorID, worldID, targetID string, ban bool) error {
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
	actor, err := actorRole(ctx, tx, actorID, worldID)
	if err != nil {
		return err
	}
	if !canModerate(actor) || actorID == targetID {
		return ErrForbidden
	}
	var targetRole string
	err = tx.QueryRow(ctx, `SELECT role FROM world_members WHERE world_id=$1 AND user_id=$2 AND active`, worldID, targetID).Scan(&targetRole)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	// Moderators may kick or ban any non-owner, including another moderator.
	// Owner role changes and ownership transfer remain owner-only operations.
	if targetRole == "owner" {
		return ErrForbidden
	}
	if ban {
		_, err = tx.Exec(ctx, `INSERT INTO world_bans(world_id,user_id,banned_by) VALUES($1,$2,$3) ON CONFLICT(world_id,user_id) DO UPDATE SET banned_by=$3,created_at=now()`, worldID, targetID, actorID)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE world_members SET active=false WHERE world_id=$1 AND user_id=$2`, worldID, targetID)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO moderation_audit(world_id,actor_id,target_id,action) VALUES($1,$2,$3,$4)`, worldID, actorID, targetID, map[bool]string{false: "kick", true: "ban"}[ban]); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	if s.Revoked != nil {
		s.Revoked(ctx, worldID, targetID)
	}
	return nil
}
func (s *Service) TransferWorld(ctx context.Context, ownerID, worldID, targetID string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockAccounts(ctx, tx, ownerID, targetID); err != nil {
		return err
	}
	active, err := lockWorld(ctx, tx, worldID)
	if err != nil {
		return err
	}
	if !active {
		return ErrForbidden
	}
	owner, err := actorRole(ctx, tx, ownerID, worldID)
	if err != nil {
		return err
	}
	if owner != "owner" || ownerID == targetID {
		return ErrForbidden
	}
	if _, err = actorRole(ctx, tx, targetID, worldID); err != nil {
		return err
	}
	// Demote first so the unique active-owner index always holds.
	if _, err = tx.Exec(ctx, `UPDATE world_members SET role='member' WHERE world_id=$1 AND user_id=$2`, worldID, ownerID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE world_members SET role='owner' WHERE world_id=$1 AND user_id=$2`, worldID, targetID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE worlds SET owner_id=$2 WHERE id=$1`, worldID, targetID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Service) ArchiveWorld(ctx context.Context, ownerID, worldID string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockAccounts(ctx, tx, ownerID); err != nil {
		return err
	}
	active, err := lockWorld(ctx, tx, worldID)
	if err != nil {
		return err
	}
	if !active {
		return ErrForbidden
	}
	role, err := actorRole(ctx, tx, ownerID, worldID)
	if err != nil {
		return err
	}
	if role != "owner" {
		return ErrForbidden
	}
	_, err = tx.Exec(ctx, `UPDATE worlds SET archived_at=now() WHERE id=$1`, worldID)
	if err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT user_id FROM world_members WHERE world_id=$1 AND active`, worldID)
	if err != nil {
		return err
	}
	var members []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		members = append(members, id)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	if s.Revoked != nil {
		for _, id := range members {
			s.Revoked(ctx, worldID, id)
		}
	}
	return nil
}
