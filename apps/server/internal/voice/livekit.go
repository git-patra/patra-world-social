package voice

import (
	"context"
	"errors"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"net/url"
	"strings"
	"time"
)

type LiveKit struct {
	url, key, secret string
	client           *lksdk.RoomServiceClient
}

func NewLiveKit(raw, key, secret string) (*LiveKit, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "wss" || !strings.HasSuffix(u.Hostname(), ".livekit.cloud") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || key == "" || secret == "" {
		return nil, errors.New("voice requires an explicitly configured LiveKit Cloud wss URL and credentials")
	}
	return newLiveKit(raw, key, secret), nil
}
func newLiveKit(url, key, secret string) *LiveKit {
	return &LiveKit{url, key, secret, lksdk.NewRoomServiceClient(url, key, secret)}
}
func (p *LiveKit) Token(a VoiceAdmission) (VoiceToken, error) {
	yes, no := true, false
	grant := &auth.VideoGrant{Room: a.Room, RoomJoin: true, CanPublish: &yes, CanSubscribe: &yes, CanPublishData: &no, CanUpdateOwnMetadata: &no, CanPublishSources: []string{"microphone"}}
	token, e := auth.NewAccessToken(p.key, p.secret).SetIdentity(a.Identity).SetVideoGrant(grant).SetValidFor(60 * time.Second).ToJWT()
	return VoiceToken{token, p.url}, e
}
func (p *LiveKit) Remove(ctx context.Context, r Record, cutoff int64) error {
	if cutoff <= 0 {
		return errors.New("explicit cutoff required")
	}
	_, e := p.client.RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{Room: r.Room, Identity: r.ID, RevokeTokenTs: cutoff})
	return e
}
func (p *LiveKit) Rooms(ctx context.Context) (map[string][]string, error) {
	rooms, e := p.client.ListRooms(ctx, &livekit.ListRoomsRequest{})
	if e != nil {
		return nil, e
	}
	out := map[string][]string{}
	for _, r := range rooms.Rooms {
		out[r.Name] = []string{}
	}
	return out, nil
}

// Participants is fetched only for this application's namespace by the service.
func (p *LiveKit) Participants(ctx context.Context, room string) ([]string, error) {
	all, e := p.client.ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: room})
	if e != nil {
		return nil, e
	}
	ids := []string{}
	for _, v := range all.Participants {
		ids = append(ids, v.Identity)
	}
	return ids, nil
}
func (p *LiveKit) DeleteRoom(ctx context.Context, room string) error {
	_, e := p.client.DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: room})
	return e
}
func (p *LiveKit) Mute(ctx context.Context, r Record) error {
	all, e := p.client.ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: r.Room})
	if e != nil {
		return e
	}
	for _, participant := range all.Participants {
		if participant.Identity != r.ID {
			continue
		}
		for _, track := range participant.Tracks {
			if track.Source == livekit.TrackSource_MICROPHONE {
				if _, e = p.client.MutePublishedTrack(ctx, &livekit.MuteRoomTrackRequest{Room: r.Room, Identity: r.ID, TrackSid: track.Sid, Muted: true}); e != nil {
					return e
				}
			}
		}
	}
	return nil
}
