// Package history owns immutable server results and private participation history.
package history

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"patra-world-social/server/internal/games/model"
	"reflect"
)

var ErrInvalid = errors.New("invalid authoritative result")
var ErrConflict = errors.New("immutable result conflict; investigation required")
var ErrNotFound = errors.New("history not found")

// Projector runs within finalization's transaction, once per match. Implementors
// lock their world/ruleset leaderboard row here before PB/champion updates.
// Register concrete projectors before starting workers. Never perform external I/O.
type Projector func(context.Context, pgx.Tx, model.MatchResult) error
type Service struct {
	Pool       *pgxpool.Pool
	Projectors map[string]Projector
}

func New(pool *pgxpool.Pool) *Service {
	return &Service{Pool: pool, Projectors: map[string]Projector{}}
}

func encodeResult(r model.MatchResult) ([]byte, [32]byte, error) {
	if r.MatchID == "" || r.WorldID == "" || r.ArenaID == "" || r.PreparedAt <= 0 || r.StartsAt < r.PreparedAt || r.FinishedAt < r.StartsAt || r.FinishedAt > 9007199254740991 || len(r.Participants) < 1 || len(r.Participants) > 5 {
		return nil, [32]byte{}, ErrInvalid
	}
	if r.GameType != "race" && r.GameType != "penalty" && r.GameType != "duck" && r.GameType != "pingpong" && r.GameType != "blaster" {
		return nil, [32]byte{}, ErrInvalid
	}
	if r.Pingpong != nil {
		if r.GameType != "pingpong" || len(r.Participants) != 2 || r.Pingpong.Rallies < 1 || r.Pingpong.Rallies > 22 || (r.Pingpong.Reason != "score" && r.Pingpong.Reason != "deadline" && r.Pingpong.Reason != "forfeit") {
			return nil, [32]byte{}, ErrInvalid
		}
		for i, score := range r.Pingpong.Scores {
			if score < 0 || score > 11 || r.Participants[i].Score == nil || *r.Participants[i].Score != score {
				return nil, [32]byte{}, ErrInvalid
			}
		}
	} else if r.GameType == "pingpong" {
		return nil, [32]byte{}, ErrInvalid
	}
	if r.Blaster != nil {
		b := r.Blaster
		if r.GameType != "blaster" || len(r.Participants) > 4 || b.Waves < 0 || b.Waves > 3 || b.Kills < 0 || b.Kills > 18 || (b.Reason != "cleared" && b.Reason != "all_down" && b.Reason != "timeout") || (b.Reason == "cleared") != (b.Waves == 3) {
			return nil, [32]byte{}, ErrInvalid
		}
		kills := int64(0)
		for _, p := range r.Participants {
			if p.ActorType != "human" || p.Score == nil || p.BotAssisted {
				return nil, [32]byte{}, ErrInvalid
			}
			kills += *p.Score
		}
		if kills != int64(b.Kills) {
			return nil, [32]byte{}, ErrInvalid
		}
	} else if r.GameType == "blaster" {
		return nil, [32]byte{}, ErrInvalid
	}
	actors, users := map[string]bool{}, map[string]bool{}
	for _, p := range r.Participants {
		if p.ActorID == "" || actors[p.ActorID] || (p.ActorType != "human" && p.ActorType != "bot") || (p.ActorType == "human") != (p.UserID != "") || (p.UserID != "" && users[p.UserID]) || (!validOutcome(r.GameType, p.Outcome)) || (p.Position != nil && (*p.Position < 1 || *p.Position > len(r.Participants))) || (p.Score != nil && (*p.Score < 0 || *p.Score > 9007199254740991)) {
			return nil, [32]byte{}, ErrInvalid
		}
		if p.Race != nil && (r.GameType != "race" || p.Race.Laps < 0 || p.Race.Laps > 5 || (p.Race.ElapsedMs != nil && (*p.Race.ElapsedMs < 0 || *p.Race.ElapsedMs > r.FinishedAt-r.StartsAt))) {
			return nil, [32]byte{}, ErrInvalid
		}
		if p.Duck != nil && (r.GameType != "duck" || p.Duck.Shots < 0 || p.Duck.Hits < 0 || p.Duck.Hits > p.Duck.Shots) {
			return nil, [32]byte{}, ErrInvalid
		}
		if p.Race != nil {
			if len(p.Race.LapTimesMs) > 5 {
				return nil, [32]byte{}, ErrInvalid
			}
			for _, lap := range p.Race.LapTimesMs {
				if lap < 0 || lap > r.FinishedAt-r.StartsAt {
					return nil, [32]byte{}, ErrInvalid
				}
			}
			if p.Race.BestLapMs != nil && (*p.Race.BestLapMs < 0 || *p.Race.BestLapMs > r.FinishedAt-r.StartsAt) {
				return nil, [32]byte{}, ErrInvalid
			}
		}
		actors[p.ActorID] = true
		users[p.UserID] = true
	}
	if len(r.PenaltyTurns) > 1000 || (len(r.PenaltyTurns) > 0 && r.GameType != "penalty") {
		return nil, [32]byte{}, ErrInvalid
	}
	for i, t := range r.PenaltyTurns {
		if t.Turn != i+1 || !actors[t.KickerID] || !actors[t.KeeperID] || t.KickerID == t.KeeperID || t.KickZone < 0 || t.KickZone > 3 || t.KeepZone < 0 || t.KeepZone > 3 || t.Goal != (t.KickZone != t.KeepZone) {
			return nil, [32]byte{}, ErrInvalid
		}
	}
	b, e := json.Marshal(r)
	if e != nil || len(b) > 262144 {
		return nil, [32]byte{}, ErrInvalid
	}
	return b, sha256.Sum256(b), nil
}
func lockMatch(ctx context.Context, tx pgx.Tx, id string) error {
	_, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,7))`, id)
	return e
}
func bindResult(r model.MatchResult, m model.MatchStart, scheduled *int64) bool {
	if r.MatchID != m.MatchID || r.WorldID != m.WorldID || r.ArenaID != m.ArenaID || r.GameType != m.GameType || r.Version != m.Version || r.PreparedAt != m.PreparedAt || len(r.Participants) != len(m.Actors) || (scheduled != nil && *scheduled != r.StartsAt) {
		return false
	}
	for i, p := range r.Participants {
		if !reflect.DeepEqual(p.Actor, m.Actors[i]) {
			return false
		}
	}
	return true
}

func validOutcome(game, outcome string) bool {
	if game == "blaster" {
		return outcome == "win" || outcome == "loss" || outcome == "forfeit"
	}
	if outcome == "finished" || outcome == "dnf" {
		return true
	}
	if game == "duck" && outcome == "incomplete" {
		return true
	}
	return (game == "penalty" || game == "pingpong" || game == "blaster") && (outcome == "win" || outcome == "loss" || outcome == "draw" || outcome == "forfeit" || outcome == "abandoned")
}
