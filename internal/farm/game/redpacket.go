package game

import (
	"context"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/redpacketpb"
)

func (a *API) sendRedPacket(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, redPacketService, method, nonNilBody(body))
	return raw, err
}

// GetTodayClaimStatus fetches today's red-packet claim status.
func (a *API) GetTodayClaimStatus(ctx context.Context) (*redpacketpb.GetTodayClaimStatusReply, error) {
	raw, err := a.sendRedPacket(ctx, "GetTodayClaimStatus", (&redpacketpb.GetTodayClaimStatusRequest{}).Marshal())
	if err != nil {
		return nil, err
	}
	reply := &redpacketpb.GetTodayClaimStatusReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// ClaimRedPacket claims a red packet by ID.
func (a *API) ClaimRedPacket(ctx context.Context, id int32) (*redpacketpb.ClaimRedPacketReply, error) {
	req := &redpacketpb.ClaimRedPacketRequest{ID: id}
	raw, err := a.sendRedPacket(ctx, "ClaimRedPacket", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &redpacketpb.ClaimRedPacketReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}
