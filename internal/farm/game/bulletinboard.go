package game

import (
	"context"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/bulletinboardpb"
)

func (a *API) sendBulletinBoard(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, bulletinBoardService, method, nonNilBody(body))
	return raw, err
}

// GetBulletinList fetches bulletin list entries.
func (a *API) GetBulletinList(ctx context.Context, count int64) (*bulletinboardpb.GetBulletinListReply, error) {
	req := &bulletinboardpb.GetBulletinListRequest{Count: count}
	raw, err := a.sendBulletinBoard(ctx, "GetBulletinList", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &bulletinboardpb.GetBulletinListReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetBulletinDetail fetches a bulletin by ID.
func (a *API) GetBulletinDetail(ctx context.Context, id int64) (*bulletinboardpb.GetBulletinDetailReply, error) {
	req := &bulletinboardpb.GetBulletinDetailRequest{ID: id}
	raw, err := a.sendBulletinBoard(ctx, "GetBulletinDetail", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &bulletinboardpb.GetBulletinDetailReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}
