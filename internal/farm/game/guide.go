package game

import (
	"context"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/guidepb"
)

func (a *API) sendGuide(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, guideService, method, nonNilBody(body))
	return raw, err
}

// SetWeakGuideNodeComplete marks a weak-guide node complete.
func (a *API) SetWeakGuideNodeComplete(ctx context.Context, nodeID int64) (*guidepb.SetWeakGuideNodeCompleteReply, error) {
	req := &guidepb.SetWeakGuideNodeCompleteRequest{NodeID: nodeID}
	raw, err := a.sendGuide(ctx, "SetWeakGuideNodeComplete", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &guidepb.SetWeakGuideNodeCompleteReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// ClaimWeakGuideReward claims a weak-guide reward for the given node.
func (a *API) ClaimWeakGuideReward(ctx context.Context, nodeID int64) (*guidepb.ClaimWeakGuideRewardReply, error) {
	req := &guidepb.ClaimWeakGuideRewardRequest{NodeID: nodeID}
	raw, err := a.sendGuide(ctx, "ClaimWeakGuideReward", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &guidepb.ClaimWeakGuideRewardReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}
