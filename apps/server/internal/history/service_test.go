package history

import (
	"patra-world-social/server/internal/games/model"
	"testing"
)

func TestInvalidResultCannotBeHashed(t *testing.T) {
	if _, _, e := encodeResult(model.MatchResult{}); e == nil {
		t.Fatal("empty result accepted")
	}
}
