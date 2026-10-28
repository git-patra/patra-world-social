package voice

import (
	"context"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLiveKitTokenClaims(t *testing.T) {
	p := newLiveKit("wss://example.livekit.cloud", "test-key", "test-secret-that-is-long-enough")
	a := VoiceAdmission{ID: "opaque", Identity: "opaque", Room: "test-room"}
	token, e := p.Token(a)
	if e != nil {
		t.Fatal(e)
	}
	v, e := auth.ParseAPIToken(token.Token)
	if e != nil {
		t.Fatal(e)
	}
	claims, g, e := v.Verify("test-secret-that-is-long-enough")
	if e != nil {
		t.Fatal(e)
	}
	if claims.Issuer != "test-key" || claims.Subject != "opaque" || claims.ExpiresAt.Time.Sub(claims.NotBefore.Time) != 60*time.Second {
		t.Fatalf("claims %+v", claims)
	}
	vg := g.Video
	if vg == nil || vg.Room != "test-room" || !vg.RoomJoin || vg.CanPublish == nil || !*vg.CanPublish || vg.CanSubscribe == nil || !*vg.CanSubscribe || vg.CanPublishData == nil || *vg.CanPublishData || vg.CanUpdateOwnMetadata == nil || *vg.CanUpdateOwnMetadata || len(vg.CanPublishSources) != 1 || vg.CanPublishSources[0] != "microphone" || vg.RoomAdmin || vg.RoomCreate || vg.RoomList || vg.RoomRecord || vg.IngressAdmin || vg.Agent {
		t.Fatalf("unsafe grant %+v", vg)
	}
}
func TestSDKSerializesExplicitCutoff(t *testing.T) {
	var got livekit.RoomParticipantIdentity
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/twirp/livekit.RoomService/RemoveParticipant" {
			t.Error(r.URL.Path)
		}
		verifier, err := auth.ParseAPIToken(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if err != nil {
			t.Error(err)
		} else {
			_, grants, err := verifier.Verify("secret")
			if err != nil || grants.Video == nil || !grants.Video.RoomAdmin || grants.Video.Room != "room" {
				t.Error("incorrect SDK authorization", err)
			}
		}
		raw, _ := io.ReadAll(r.Body)
		if err := proto.Unmarshal(raw, &got); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/protobuf")
		raw, _ = proto.Marshal(&livekit.RemoveParticipantResponse{})
		w.Write(raw)
	}))
	defer srv.Close()
	p := newLiveKit(srv.URL, "key", "secret")
	cutoff := time.Now().Add(2 * time.Second).Unix()
	if err := p.Remove(context.Background(), Record{"identity", "room"}, cutoff); err != nil {
		t.Fatal(err)
	}
	if got.Identity != "identity" || got.Room != "room" || got.RevokeTokenTs != cutoff {
		t.Fatalf("wire request %+v", &got)
	}
}
