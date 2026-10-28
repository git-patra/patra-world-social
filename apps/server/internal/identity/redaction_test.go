package identity

import (
	"strings"
	"testing"
)

func TestKnownPublicShapesAndNoArbitraryTextRewrite(t *testing.T) {
	r := &Redactions{}
	r.Add("gone")
	raw := []byte(`{"actors":[{"userId":"gone","displayName":"Private","avatarId":"moon"}],"avatars":[{"id":"gone","name":"Private","status":"available","x":1,"y":2}],"chat":{"id":"gone","text":"gone Private"}}`)
	v, e := r.JSON(raw)
	if e != nil {
		t.Fatal(e)
	}
	s := string(v)
	if strings.Contains(s, `"name":"Private"`) || strings.Contains(s, `"displayName":"Private"`) || !strings.Contains(s, `"text":"gone Private"`) {
		t.Fatal(s)
	}
	if !strings.Contains(string(raw), `"displayName":"Private"`) {
		t.Fatal("mutated source")
	}
}
