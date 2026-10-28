// Package model defines shared integer simulation units, independent of rendering.
package model

type Subunits int64

const SubunitsPerTile Subunits = 1024

type Version struct {
	MapID           string `json:"mapId"`
	MapVersion      int    `json:"mapVersion"`
	RulesID         string `json:"rulesId"`
	RulesVersion    int    `json:"rulesVersion"`
	ProtocolVersion int    `json:"protocolVersion"`
	MapChecksum     string `json:"mapChecksum"`
	RulesChecksum   string `json:"rulesChecksum"`
}

type Actor struct {
	ActorID     string `json:"actorId"`
	ActorType   string `json:"actorType"`
	UserID      string `json:"userId,omitempty"`
	DisplayName string `json:"displayName"`
	AvatarID    string `json:"avatarId"`
}

// MatchStart is internal. Public game serializers must omit seed and secrets.
type MatchStart struct {
	Version
	MatchID    string  `json:"matchId"`
	WorldID    string  `json:"worldId"`
	ArenaID    string  `json:"arenaId"`
	GameType   string  `json:"gameType"`
	Seed       int64   `json:"seed"`
	Actors     []Actor `json:"actors"`
	PreparedAt int64   `json:"preparedAt"`
	StartsAt   int64   `json:"startsAt"`
}
