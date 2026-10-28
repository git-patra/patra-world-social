package model

// MatchResult is server-authored and deliberately has no seed or hidden choices.
// Engines supply resolved details only; the foundation never computes winners.
type PingpongResult struct {
	Scores  [2]int64 `json:"scores"`
	Rallies int      `json:"rallies"`
	Reason  string   `json:"reason"`
}
type MatchResult struct {
	Pingpong *PingpongResult `json:"pingpong,omitempty"`
	Version
	MatchID      string              `json:"matchId"`
	WorldID      string              `json:"worldId"`
	ArenaID      string              `json:"arenaId"`
	GameType     string              `json:"gameType"`
	PreparedAt   int64               `json:"preparedAt"`
	StartsAt     int64               `json:"startsAt"`
	FinishedAt   int64               `json:"finishedAt"`
	Participants []ParticipantResult `json:"participants"`
	PenaltyTurns []PenaltyTurnResult `json:"penaltyTurns,omitempty"`
}
type ParticipantResult struct {
	Actor
	BotAssisted bool        `json:"botAssisted"`
	Outcome     string      `json:"outcome"`
	Score       *int64      `json:"score,omitempty"`
	Position    *int        `json:"position,omitempty"`
	Race        *RaceResult `json:"race,omitempty"`
	Duck        *DuckResult `json:"duck,omitempty"`
}
type RaceResult struct {
	Laps       int     `json:"laps"`
	LapTimesMs []int64 `json:"lapTimesMs,omitempty"`
	BestLapMs  *int64  `json:"bestLapMs,omitempty"`
	ElapsedMs  *int64  `json:"elapsedMs,omitempty"`
	DNF        bool    `json:"dnf"`
}
type DuckResult struct {
	Hits  int `json:"hits"`
	Shots int `json:"shots"`
}
type PenaltyTurnResult struct {
	Turn        int    `json:"turn"`
	KickerID    string `json:"kickerId"`
	KeeperID    string `json:"keeperId"`
	KickZone    int    `json:"kickZone"`
	KeepZone    int    `json:"keepZone"`
	KickTimeout bool   `json:"kickTimeout"`
	KeepTimeout bool   `json:"keepTimeout"`
	Goal        bool   `json:"goal"`
}
