package lobby

import (
	"context"
	"encoding/hex"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/world"
	"time"
)

// Engine is owned by the lobby goroutine. Methods must be bounded and perform no
// storage/network I/O. Factories prepare inert state; Start activates it only
// after the running record is confirmed durable. Game-specific input stays in
// the concrete engine package, never in the generic lobby serializer.
type Engine interface {
	Start(startsAt int64)
	Tick(time.Time) bool
	Presence(userID, kind string)
	Close()
}

// TimedPresenceEngine receives authoritative notification time, independent of
// owner ingestion. expired is a grace boundary, never an intentional leave.
type TimedPresenceEngine interface {
	PresenceAt(userID, kind string, at time.Time)
}

func enginePresence(engine Engine, user, kind string, at time.Time) {
	if timed, ok := engine.(TimedPresenceEngine); ok {
		timed.PresenceAt(user, kind, at)
		return
	}
	if kind == "expired" {
		kind = "leave"
	}
	engine.Presence(user, kind)
}

type Factory func(model.MatchStart) (Engine, error)

// Prepare may only reorder the cloned roster, before persistence freezes it.
type Prepare func(model.MatchStart) ([]model.Actor, error)

func sameActors(before, after []model.Actor) bool {
	if len(before) != len(after) {
		return false
	}
	actors := map[string]model.Actor{}
	for _, a := range before {
		actors[a.ActorID] = a
	}
	for _, a := range after {
		v, ok := actors[a.ActorID]
		if !ok || v != a {
			return false
		}
		delete(actors, a.ActorID)
	}
	return len(actors) == 0
}

type Arena struct {
	ID        string `json:"arenaId"`
	GameType  string `json:"gameType"`
	Capacity  int    `json:"capacity"`
	ReturnX   int    `json:"returnX"`
	ReturnY   int    `json:"returnY"`
	countdown time.Duration
}

var arenas = []Arena{{"race", "race", 5, 30, 14, 3 * time.Second}, {"penalty", "penalty", 2, 50, 32, 0}, {"duck-1", "duck", 1, 30, 53, 3 * time.Second}, {"duck-2", "duck", 1, 30, 53, 3 * time.Second}, {"duck-3", "duck", 1, 30, 53, 3 * time.Second}, {"duck-4", "duck", 1, 30, 53, 3 * time.Second}}

func init() {
	for i := range arenas {
		if a, ok := world.ActivityArea(arenas[i].ID); ok {
			arenas[i].ReturnX = a.SafeReturn.X
			arenas[i].ReturnY = a.SafeReturn.Y
		} else {
			m, e := world.LoadMap(world.MapAsset)
			if e != nil {
				panic(e)
			}
			for _, b := range m.Buildings {
				if b.ID == "duck" {
					arenas[i].ReturnX = b.Entrance.X
					arenas[i].ReturnY = b.Entrance.Y
				}
			}
		}
	}
}
func arena(id string) (Arena, bool) {
	for _, a := range arenas {
		if a.ID == id {
			return a, true
		}
	}
	return Arena{}, false
}

// RulesMetadata overrides only rules identity before durable start preparation.
// Registration is allowed only before the first lobby is created.
type RulesMetadata struct {
	ID       string
	Version  int
	Checksum string
}

func (s *Service) RegisterVersioned(game string, f Factory, r RulesMetadata) error {
	b, err := hex.DecodeString(r.Checksum)
	if err != nil || len(b) != 32 || r.ID == "" || r.Version < 1 || f == nil || game != "duck" {
		return ErrForbidden
	}
	var result error
	err = s.call(context.Background(), func() {
		if len(s.entries) > 0 || s.factories[game] != nil {
			result = ErrForbidden
			return
		}
		s.factories[game] = f
		s.rulesMetadata[game] = r
	})
	if err != nil {
		return err
	}
	return result
}
