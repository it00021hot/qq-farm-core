package visitpb_test

import (
	"bytes"
	"testing"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/plantpb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/visitpb"
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

func TestEnterRequestRoundTrip(t *testing.T) {
	req := &visitpb.EnterRequest{HostGID: 99999, Reason: visitpb.EnterReasonFriend}
	roundTripRequest(t, "EnterRequest", req, &visitpb.EnterRequest{})
}

func TestLeaveRequestRoundTrip(t *testing.T) {
	req := &visitpb.LeaveRequest{HostGID: 88888}
	roundTripRequest(t, "LeaveRequest", req, &visitpb.LeaveRequest{})
}

func TestEnterReplyDecodeLands(t *testing.T) {
	land := &plantpb.LandInfo{ID: 7, Unlocked: true, Level: 2}
	landRaw := land.Marshal()

	var b []byte
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendBytes(b, landRaw)

	var reply visitpb.EnterReply
	if err := reply.Unmarshal(b); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(reply.Lands) != 1 || reply.Lands[0].ID != 7 {
		t.Fatalf("lands=%+v", reply.Lands)
	}
}
