package chat

import (
	"strings"
	"testing"
)

func TestMessageBoundary(t *testing.T) {
	if e := Validate("world", "id", strings.Repeat("🎃", 500)); e != nil {
		t.Fatal(e)
	}
	for _, v := range [][3]string{{"table:0", "id", "hi"}, {"world", "", "hi"}, {"world", "id", " \n "}, {"world", "id", strings.Repeat("🎃", 501)}} {
		if e := Validate(v[0], v[1], v[2]); e == nil {
			t.Fatal("invalid accepted", v[:2])
		}
	}
}
