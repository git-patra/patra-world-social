package history

import (
	"patra-world-social/server/internal/games/model"
	"testing"
)

func TestPingpongMetadataIsTypedAndBoundToScores(t *testing.T) {
	score := int64(7)
	other := int64(5)
	r := model.MatchResult{MatchID: "m", WorldID: "w", ArenaID: "pingpong", GameType: "pingpong", PreparedAt: 1000, StartsAt: 1000, FinishedAt: 2000, Pingpong: &model.PingpongResult{Scores: [2]int64{7, 5}, Rallies: 12, Reason: "score"}, Participants: []model.ParticipantResult{{Actor: model.Actor{ActorID: "a", ActorType: "human", UserID: "u"}, Outcome: "win", Score: &score}, {Actor: model.Actor{ActorID: "b", ActorType: "bot"}, Outcome: "loss", Score: &other}}}
	if _, _, err := encodeResult(r); err != nil {
		t.Fatal(err)
	}
	r.Pingpong.Scores[0] = 8
	if _, _, err := encodeResult(r); err == nil {
		t.Fatal("inconsistent score")
	}
	r.Pingpong.Scores[0] = 7
	r.GameType = "penalty"
	if _, _, err := encodeResult(r); err == nil {
		t.Fatal("wrong game metadata")
	}
	if !validFilter(Filter{Game: "pingpong"}) {
		t.Fatal("filter")
	}
}
