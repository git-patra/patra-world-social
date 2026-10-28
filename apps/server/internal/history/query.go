package history

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"patra-world-social/server/internal/games/model"
	"strings"
	"time"
)

type Filter struct {
	Game, World, Status string
	From, To            *time.Time
	Limit               int
}
type Record struct {
	MatchID      string             `json:"matchId"`
	WorldID      string             `json:"worldId"`
	WorldName    string             `json:"worldName"`
	GameType     string             `json:"gameType"`
	Status       string             `json:"status"`
	Version      model.Version      `json:"version"`
	PreparedAt   time.Time          `json:"preparedAt"`
	StartsAt     *time.Time         `json:"startsAt"`
	FinishedAt   *time.Time         `json:"finishedAt"`
	Result       *model.MatchResult `json:"result,omitempty"`
	Participants []model.Actor      `json:"participants"`
}
type Page struct {
	Items      []Record `json:"items"`
	NextCursor string   `json:"nextCursor,omitempty"`
}

func uuid(s string) bool { var v pgtype.UUID; return v.Scan(s) == nil && v.Valid }
func validFilter(f Filter) bool {
	return (f.Game == "" || f.Game == "race" || f.Game == "penalty" || f.Game == "duck") && (f.World == "" || uuid(f.World)) && (f.Status == "" || f.Status == "running" || f.Status == "finalizing" || f.Status == "completed" || f.Status == "aborted_storage" || f.Status == "aborted_server" || f.Status == "abandoned") && (f.From == nil || f.To == nil || !f.To.Before(*f.From))
}

const recordSelect = `SELECT m.id::text,m.world_id::text,w.name,m.game_type,m.status,m.version,m.prepared_at,m.starts_at,m.finished_at,m.result_payload,
 (SELECT jsonb_agg(jsonb_build_object('actorId',actor_id,'actorType',actor_type,'userId',CASE WHEN EXISTS(SELECT 1 FROM users u WHERE u.id=match_participants.user_id AND u.deleted_at IS NOT NULL) THEN NULL ELSE user_id END,'displayName',display_name,'avatarId',avatar_id) ORDER BY roster_slot) FROM match_participants WHERE match_id=m.id)
 FROM matches m JOIN worlds w ON w.id=m.world_id WHERE EXISTS(SELECT 1 FROM match_participants p WHERE p.match_id=m.id AND p.user_id=$1)`

func readRecord(row interface{ Scan(...any) error }) (Record, error) {
	var r Record
	var version, result, actors []byte
	e := row.Scan(&r.MatchID, &r.WorldID, &r.WorldName, &r.GameType, &r.Status, &version, &r.PreparedAt, &r.StartsAt, &r.FinishedAt, &result, &actors)
	if e != nil {
		return r, e
	}
	if e = json.Unmarshal(version, &r.Version); e != nil {
		return r, e
	}
	if e = json.Unmarshal(actors, &r.Participants); e != nil {
		return r, e
	}
	if result != nil {
		e = json.Unmarshal(result, &r.Result)
	}
	// Public identities come from participant projections, never immutable jobs.
	// T14 can anonymize these rows without rewriting competitive result hashes.
	if r.Result != nil {
		actors := map[string]model.Actor{}
		for _, a := range r.Participants {
			actors[a.ActorID] = a
		}
		for i := range r.Result.Participants {
			r.Result.Participants[i].Actor = actors[r.Result.Participants[i].ActorID]
		}
	}
	return r, e
}
func (s *Service) ListHistory(ctx context.Context, user string, f Filter, cursor string) (Page, error) {
	page := Page{Items: []Record{}}
	if !uuid(user) || !validFilter(f) {
		return page, ErrInvalid
	}
	if f.Limit == 0 {
		f.Limit = 20
	}
	if f.Limit < 1 || f.Limit > 50 {
		return page, ErrInvalid
	}
	var before *time.Time
	id := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	if cursor != "" {
		b, e := base64.RawURLEncoding.DecodeString(cursor)
		parts := strings.Split(string(b), "|")
		if e != nil || len(parts) != 2 || !uuid(parts[1]) {
			return page, ErrInvalid
		}
		at, e := time.Parse(time.RFC3339Nano, parts[0])
		if e != nil {
			return page, ErrInvalid
		}
		before = &at
		id = parts[1]
	}
	rows, e := s.Pool.Query(ctx, recordSelect+` AND ($2='' OR m.game_type=$2) AND ($3='' OR m.world_id::text=$3) AND ($4='' OR m.status=$4) AND ($5::timestamptz IS NULL OR m.prepared_at >= $5) AND ($6::timestamptz IS NULL OR m.prepared_at <= $6) AND ($7::timestamptz IS NULL OR (m.prepared_at,m.id)<($7,$8::uuid)) ORDER BY m.prepared_at DESC,m.id DESC LIMIT $9`, user, f.Game, f.World, f.Status, f.From, f.To, before, id, f.Limit+1)
	if e != nil {
		return page, e
	}
	defer rows.Close()
	for rows.Next() {
		r, e := readRecord(rows)
		if e != nil {
			return page, e
		}
		page.Items = append(page.Items, r)
	}
	if e = rows.Err(); e != nil {
		return page, e
	}
	if len(page.Items) > f.Limit {
		page.Items = page.Items[:f.Limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(last.PreparedAt.Format(time.RFC3339Nano) + "|" + last.MatchID))
	}
	return page, nil
}
func (s *Service) Detail(ctx context.Context, user, id string) (Record, error) {
	if !uuid(user) || !uuid(id) {
		return Record{}, ErrNotFound
	}
	r, e := readRecord(s.Pool.QueryRow(ctx, recordSelect+` AND m.id=$2`, user, id))
	if errors.Is(e, pgx.ErrNoRows) {
		e = ErrNotFound
	}
	return r, e
}
