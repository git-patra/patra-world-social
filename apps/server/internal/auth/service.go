package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/argon2"
	"patra-world-social/server/internal/db"
)

var (
	ErrInvalid      = errors.New("invalid credentials or recovery code")
	ErrDuplicate    = errors.New("username unavailable")
	ErrInput        = errors.New("invalid input")
	hashSlots       = make(chan struct{}, 4)
	usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,24}$`)
)

type User struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Avatar      string `json:"avatar"`
}
type RecoveryCode string
type Session struct {
	User          User      `json:"user"`
	Token         string    `json:"-"`
	ExpiresAt     time.Time `json:"expiresAt"`
	IdleExpiresAt time.Time `json:"idleExpiresAt"`
}
type Service struct {
	Pool          *pgxpool.Pool
	Revoked       func(context.Context, string)
	RemoveAccount func(context.Context, pgx.Tx, string) error
	DeleteFence   func(context.Context, string, func(context.Context) error) error
	Deleted       func(string, [][3]string)
}

func New(pool *pgxpool.Pool) *Service { return &Service{Pool: pool} }

func canonicalUsername(raw string) (string, error) {
	if !usernamePattern.MatchString(raw) {
		return "", ErrInput
	}
	return strings.ToLower(raw), nil
}
func validatePassword(v string) error {
	n := utf8.RuneCountInString(v)
	if n < 12 || n > 128 || !utf8.ValidString(v) {
		return ErrInput
	}
	return nil
}
func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func tokenHash(v string) []byte { h := sha256.Sum256([]byte(v)); return h[:] }
func uuidString(v pgtype.UUID) string {
	b := v.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
func userFromRow(id pgtype.UUID, username, displayName, avatar string) User {
	return User{uuidString(id), username, displayName, avatar}
}

func hashPassword(ctx context.Context, password string) (string, error) {
	select {
	case hashSlots <- struct{}{}:
		defer func() { <-hashSlots }()
	case <-ctx.Done():
		return "", ctx.Err()
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 2, 19456, 1, 32)
	return fmt.Sprintf("$argon2id$v=19$m=19456,t=2,p=1$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}
func verifyPassword(ctx context.Context, encoded, password string) bool {
	var m, t uint32
	var p uint8
	var salt64, hash64 string
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false
	}
	salt64, hash64 = parts[4], parts[5]
	if m < 19456 || m > 262144 || t < 2 || t > 10 || p < 1 || p > 8 {
		return false
	}
	salt, e1 := base64.RawStdEncoding.DecodeString(salt64)
	expected, e2 := base64.RawStdEncoding.DecodeString(hash64)
	if e1 != nil || e2 != nil || len(salt) < 16 || len(expected) != 32 {
		return false
	}
	select {
	case hashSlots <- struct{}{}:
		defer func() { <-hashSlots }()
	case <-ctx.Done():
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, t, m, p, 32)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func (s *Service) Register(ctx context.Context, username, password string) (User, RecoveryCode, error) {
	username, err := canonicalUsername(username)
	if err != nil {
		return User{}, "", err
	}
	if err = validatePassword(password); err != nil {
		return User{}, "", err
	}
	hash, err := hashPassword(ctx, password)
	if err != nil {
		return User{}, "", err
	}
	code, err := randomString(32)
	if err != nil {
		return User{}, "", err
	}
	row, err := db.New(s.Pool).CreateUser(ctx, db.CreateUserParams{Username: username, DisplayName: username, PasswordHash: hash, RecoveryHash: tokenHash(code)})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, "", ErrDuplicate
		}
		return User{}, "", err
	}
	return userFromRow(row.ID, row.Username, row.DisplayName, row.Avatar), RecoveryCode(code), nil
}
func (s *Service) Authenticate(ctx context.Context, username, password string) (Session, error) {
	username, err := canonicalUsername(username)
	if err != nil {
		return Session{}, ErrInvalid
	}
	row, err := db.New(s.Pool).GetUserCredentials(ctx, username)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Session{}, err
	}
	hash := "$argon2id$v=19$m=19456,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	if err == nil {
		hash = row.PasswordHash
	}
	matched := verifyPassword(ctx, hash, password)
	if err != nil || !matched {
		return Session{}, ErrInvalid
	}
	token, err := randomString(32)
	if err != nil {
		return Session{}, err
	}
	now := time.Now().UTC()
	expiry := now.Add(7 * 24 * time.Hour)
	idle := now.Add(24 * time.Hour)
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(ctx)
	var currentHash string
	err = tx.QueryRow(ctx, `SELECT password_hash FROM users WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, row.ID).Scan(&currentHash)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && currentHash != row.PasswordHash {
		return Session{}, ErrInvalid
	}
	if err != nil {
		return Session{}, err
	}
	if err = db.New(tx).CreateSession(ctx, db.CreateSessionParams{TokenHash: tokenHash(token), UserID: row.ID, ExpiresAt: pgtype.Timestamptz{Time: expiry, Valid: true}, IdleExpiresAt: pgtype.Timestamptz{Time: idle, Valid: true}}); err != nil {
		return Session{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Session{}, err
	}
	return Session{User: userFromRow(row.ID, row.Username, row.DisplayName, row.Avatar), Token: token, ExpiresAt: expiry, IdleExpiresAt: idle}, nil
}
func (s *Service) RequireSession(ctx context.Context, token string) (Session, error) {
	if len(token) != 43 {
		return Session{}, ErrInvalid
	}
	row, err := db.New(s.Pool).RefreshSession(ctx, tokenHash(token))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrInvalid
		}
		return Session{}, err
	}
	// Validation and idle extension are one database operation, so a revoked row cannot be returned by a stale read.
	return Session{User: userFromRow(row.UserID, row.Username, row.DisplayName, row.Avatar), Token: token, ExpiresAt: row.ExpiresAt.Time, IdleExpiresAt: row.IdleExpiresAt.Time}, nil
}
func (s *Service) Logout(ctx context.Context, token string) error {
	if len(token) != 43 {
		return nil
	}
	session, err := s.RequireSession(ctx, token)
	if err != nil {
		if errors.Is(err, ErrInvalid) {
			return nil
		}
		return err
	}
	if err = db.New(s.Pool).RevokeSession(ctx, tokenHash(token)); err != nil {
		return err
	}
	if s.Revoked != nil {
		s.Revoked(ctx, session.User.ID)
	}
	return nil
}
func (s *Service) Recover(ctx context.Context, username, code, newPassword string) (RecoveryCode, error) {
	username, err := canonicalUsername(username)
	if err != nil {
		return "", ErrInvalid
	}
	if err = validatePassword(newPassword); err != nil {
		return "", err
	}
	hash, err := hashPassword(ctx, newPassword)
	if err != nil {
		return "", err
	}
	next, err := randomString(32)
	if err != nil {
		return "", err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var id pgtype.UUID
	err = tx.QueryRow(ctx, `UPDATE users SET password_hash=$1,recovery_hash=$2 WHERE username=$3 AND recovery_hash=$4 AND deleted_at IS NULL RETURNING id`, hash, tokenHash(next), username, tokenHash(code)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvalid
	}
	if err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, id); err != nil {
		return "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	if s.Revoked != nil {
		s.Revoked(ctx, uuidString(id))
	}
	return RecoveryCode(next), nil
}
