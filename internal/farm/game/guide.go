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
	req := &guidepb.SetWeakGuideNodeCompleteRequest{NodeId: nodeID}
	raw, err := a.sendGuide(ctx, "SetWeakGuideNodeComplete", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &guidepb.SetWeakGuideNodeCompleteReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// ClaimWeakGuideReward claims a weak-guide reward for the given node.
func (a *API) ClaimWeakGuideReward(ctx context.Context, nodeID int64) (*guidepb.ClaimWeakGuideRewardReply, error) {
	req := &guidepb.ClaimWeakGuideRewardRequest{NodeId: nodeID}
	raw, err := a.sendGuide(ctx, "ClaimWeakGuideReward", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &guidepb.ClaimWeakGuideRewardReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
