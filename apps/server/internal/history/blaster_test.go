package history

import (
	"patra-world-social/server/internal/games/model"
	"testing"
)

func TestBlasterResultTypedHumanOnly(t *testing.T) {
	score := int64(0)
	r := model.MatchResult{MatchID: "m", WorldID: "w", ArenaID: "blaster", GameType: "blaster", PreparedAt: 1000, StartsAt: 1000, FinishedAt: 2000, Blaster: &model.BlasterResult{Waves: 0, Kills: 0, Reason: "all_down"}, Participants: []model.ParticipantResult{{Actor: model.Actor{ActorID: "a", ActorType: "human", UserID: "u"}, Outcome: "loss", Score: &score}}}
	if _, _, e := encodeResult(r); e != nil {
		t.Fatal(e)
	}
	for _, mutate := range []func(*model.MatchResult){func(r *model.MatchResult) { r.Blaster.Kills = 1 }, func(r *model.MatchResult) { r.Participants[0].ActorType = "bot"; r.Participants[0].UserID = "" }, func(r *model.MatchResult) { r.Participants[0].Outcome = "draw" }, func(r *model.MatchResult) { r.Blaster.Reason = "cleared" }, func(r *model.MatchResult) { r.Blaster = nil }} {
		copy := r
		meta := *r.Blaster
		copy.Blaster = &meta
		copy.Participants = append([]model.ParticipantResult{}, r.Participants...)
		mutate(&copy)
		if _, _, e := encodeResult(copy); e == nil {
			t.Fatal("invalid accepted", copy)
		}
	}
}
