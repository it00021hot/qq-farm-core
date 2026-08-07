package friendpb_test

import (
	"bytes"
	"testing"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/friendpb"
	"google.golang.org/protobuf/encoding/protowire"
)

func roundTripRequest[T interface {
	Marshal() []byte
	Unmarshal([]byte) error
}](t *testing.T, name string, orig T, clone T) {
	t.Helper()
	raw := orig.Marshal()
	if err := clone.Unmarshal(raw); err != nil {
		t.Fatalf("%s unmarshal: %v", name, err)
	}
	raw2 := clone.Marshal()
	if !bytes.Equal(raw, raw2) {
		t.Fatalf("%s round-trip bytes differ:\n  orig=% x\n  back=% x", name, raw, raw2)
	}
}

func TestGetAllRequestRoundTrip(t *testing.T) {
	roundTripRequest(t, "GetAllRequest", &friendpb.GetAllRequest{}, &friendpb.GetAllRequest{})
}

func TestGetGameFriendsRequestRoundTrip(t *testing.T) {
	req := &friendpb.GetGameFriendsRequest{GIDs: []int64{100, 200, 300}}
	roundTripRequest(t, "GetGameFriendsRequest", req, &friendpb.GetGameFriendsRequest{})
}

func TestAcceptFriendsRequestRoundTrip(t *testing.T) {
	req := &friendpb.AcceptFriendsRequest{FriendGIDs: []int64{42, 99}}
	roundTripRequest(t, "AcceptFriendsRequest", req, &friendpb.AcceptFriendsRequest{})
}

func TestGetApplicationsRequestRoundTrip(t *testing.T) {
	roundTripRequest(t, "GetApplicationsRequest", &friendpb.GetApplicationsRequest{}, &friendpb.GetApplicationsRequest{})
}

func TestSyncAllRequestRoundTrip(t *testing.T) {
	req := &friendpb.SyncAllRequest{OpenIDs: []string{"a", "b", "c"}}
	roundTripRequest(t, "SyncAllRequest", req, &friendpb.SyncAllRequest{})
}

func TestRejectFriendsRequestRoundTrip(t *testing.T) {
	req := &friendpb.RejectFriendsRequest{FriendGIDs: []int64{1, 2}}
	roundTripRequest(t, "RejectFriendsRequest", req, &friendpb.RejectFriendsRequest{})
}

func TestSetBlockApplicationsRequestRoundTrip(t *testing.T) {
	req := &friendpb.SetBlockApplicationsRequest{Block: true}
	roundTripRequest(t, "SetBlockApplicationsRequest", req, &friendpb.SetBlockApplicationsRequest{})
}

func TestGetShareKeyRequestRoundTrip(t *testing.T) {
	req := &friendpb.GetShareKeyRequest{ShareCfgID: 42}
	roundTripRequest(t, "GetShareKeyRequest", req, &friendpb.GetShareKeyRequest{})
}

func TestGameFriendRoundTrip(t *testing.T) {
	orig := &friendpb.GameFriend{
		GID:       12345,
		OpenID:    "openid-1",
		Name:      "Alice",
		AvatarURL: "https://example.com/a.png",
		Remark:    "note",
		Level:     10,
		Gold:      5000,
		Tags:      &friendpb.Tags{IsNew: true, IsFollow: false},
		Plant: &friendpb.Plant{
			DryTimeSec:    100,
			WeedTimeSec:   200,
			InsectTimeSec: 300,
			RipeTimeSec:   400,
			RipeFruitID:   501,
			StealPlantNum: 3,
			DryNum:        1,
			WeedNum:       2,
			InsectNum:     1,
		},
		AuthorizedStatus: 1,
	}
	raw := orig.Marshal()
	var back friendpb.GameFriend
	if err := back.Unmarshal(raw); err != nil {
		t.Fatalf("unmarshal GameFriend: %v", err)
	}
	raw2 := back.Marshal()
	if !bytes.Equal(raw, raw2) {
		t.Fatalf("GameFriend round-trip bytes differ:\n  orig=% x\n  back=% x", raw, raw2)
	}
}

func TestGetAllReplyDecode(t *testing.T) {
	friend := &friendpb.GameFriend{
		GID: 99, Name: "Bob", Level: 5,
		Plant: &friendpb.Plant{StealPlantNum: 2, DryNum: 1},
	}
	friendRaw := friend.Marshal()

	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendBytes(b, friendRaw)
	b = protowire.AppendTag(b, 3, protowire.VarintType)
	b = protowire.AppendVarint(b, 3)

	var reply friendpb.GetAllReply
	if err := reply.Unmarshal(b); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(reply.GameFriends) != 1 || reply.GameFriends[0].GID != 99 {
		t.Fatalf("friends=%+v", reply.GameFriends)
	}
	if reply.ApplicationCount != 3 {
		t.Fatalf("application_count=%d", reply.ApplicationCount)
	}
}
