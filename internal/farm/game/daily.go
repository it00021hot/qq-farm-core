package game

import (
	"context"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/emailpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/mallpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/qqvippb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/sharepb"
)

const emailService = "gamepb.emailpb.EmailService"
const shareService = "gamepb.sharepb.ShareService"
const qqVipService = "gamepb.qqvippb.QQVipService"

func (a *API) sendEmail(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, emailService, method, nonNilBody(body))
	return raw, err
}

func (a *API) sendShare(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, shareService, method, nonNilBody(body))
	return raw, err
}

func (a *API) sendQQVip(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, qqVipService, method, nonNilBody(body))
	return raw, err
}

// GetEmailList lists emails in a box.
func (a *API) GetEmailList(ctx context.Context, boxType int32) (*emailpb.GetEmailListReply, error) {
	req := &emailpb.GetEmailListRequest{BoxType: boxType}
	raw, err := a.sendEmail(ctx, "GetEmailList", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &emailpb.GetEmailListReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// ClaimEmail claims one email reward.
func (a *API) ClaimEmail(ctx context.Context, boxType int32, emailID string) (*emailpb.ClaimEmailReply, error) {
	req := &emailpb.ClaimEmailRequest{BoxType: boxType, EmailId: emailID}
	raw, err := a.sendEmail(ctx, "ClaimEmail", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &emailpb.ClaimEmailReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// BatchClaimEmail batch-claims email rewards.
func (a *API) BatchClaimEmail(ctx context.Context, boxType int32, emailID string) (*emailpb.BatchClaimEmailReply, error) {
	req := &emailpb.BatchClaimEmailRequest{BoxType: boxType, EmailIds: []string{emailID}}
	raw, err := a.sendEmail(ctx, "BatchClaimEmail", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &emailpb.BatchClaimEmailReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// CheckCanShare checks whether daily share is available.
func (a *API) CheckCanShare(ctx context.Context) (*sharepb.CheckCanShareReply, error) {
	raw, err := a.sendShare(ctx, "CheckCanShare", marshalMessage(&sharepb.CheckCanShareRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &sharepb.CheckCanShareReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// ReportShare reports a share action. The wire protocol distinguishes share
// scenes: daily gift uses field_1=1/field_4=42, 青梅酿造 uses 11/215.
func (a *API) ReportShare(ctx context.Context) (*sharepb.ReportShareReply, error) {
	return a.ReportShareScene(ctx, 1, 42)
}

// ReportShareScene reports a share action for an explicit scene pair.
func (a *API) ReportShareScene(ctx context.Context, field1, field4 int32) (*sharepb.ReportShareReply, error) {
	req := &sharepb.ReportShareRequest{Field_1: field1, Field_4: field4}
	raw, err := a.sendShare(ctx, "ReportShare", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &sharepb.ReportShareReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// ClaimShareReward claims the daily share reward.
func (a *API) ClaimShareReward(ctx context.Context) (*sharepb.ClaimShareRewardReply, error) {
	req := &sharepb.ClaimShareRewardRequest{Field_1: true}
	raw, err := a.sendShare(ctx, "ClaimShareReward", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &sharepb.ClaimShareRewardReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetInviteInfo fetches invite/share relationship info.
func (a *API) GetInviteInfo(ctx context.Context) (*sharepb.GetInviteInfoReply, error) {
	raw, err := a.sendShare(ctx, "GetInviteInfo", marshalMessage(&sharepb.GetInviteInfoRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &sharepb.GetInviteInfoReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetInviteAward claims an invite award for the given share config ID.
func (a *API) GetInviteAward(ctx context.Context, shareCfgID int64) (*sharepb.GetInviteAwardReply, error) {
	req := &sharepb.GetInviteAwardRequest{ShareCfgId: shareCfgID}
	raw, err := a.sendShare(ctx, "GetInviteAward", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &sharepb.GetInviteAwardReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// DailyGiftStatus is a panel-facing summary of QQ VIP rewards.
type DailyGiftStatus struct {
	HasGift     bool
	CanClaim    bool
	RewardTypes []int32
}

// GetDailyGiftStatus fetches QQ VIP reward status and collapses it into a
// simple has/can-claim view used by daily automation and the gifts panel.
func (a *API) GetDailyGiftStatus(ctx context.Context) (*DailyGiftStatus, error) {
	_, _ = a.sendQQVip(ctx, "RefreshVipInfo", marshalMessage(&qqvippb.RefreshVipInfoRequest{}))
	raw, err := a.sendQQVip(ctx, "GetQQVipRewardsStatus", marshalMessage(&qqvippb.GetQQVipRewardsStatusRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &qqvippb.GetQQVipRewardsStatusReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	out := &DailyGiftStatus{}
	for _, item := range reply.RewardStatuses {
		if item == nil {
			continue
		}
		if item.Enabled {
			out.HasGift = true
		}
		if item.Enabled && item.CanClaim && item.RewardType > 0 {
			out.CanClaim = true
			out.RewardTypes = append(out.RewardTypes, item.RewardType)
		}
	}
	return out, nil
}

// ClaimDailyGift claims every currently claimable QQ VIP reward type.
func (a *API) ClaimDailyGift(ctx context.Context) (*qqvippb.ClaimQQVipRewardsReply, error) {
	status, err := a.GetDailyGiftStatus(ctx)
	if err != nil {
		return nil, err
	}
	if status == nil || len(status.RewardTypes) == 0 {
		return &qqvippb.ClaimQQVipRewardsReply{}, nil
	}
	raw, err := a.sendQQVip(ctx, "ClaimQQVipRewards", marshalMessage(&qqvippb.ClaimQQVipRewardsRequest{
		RewardTypes: status.RewardTypes,
	}))
	if err != nil {
		return nil, err
	}
	reply := &qqvippb.ClaimQQVipRewardsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetMonthCardInfos fetches month card claim status.
func (a *API) GetMonthCardInfos(ctx context.Context) (*mallpb.GetMonthCardInfosReply, error) {
	raw, err := a.sendMall(ctx, "GetMonthCardInfos", marshalMessage(&mallpb.GetMonthCardInfosRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &mallpb.GetMonthCardInfosReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// ClaimMonthCardReward claims one month card daily reward.
func (a *API) ClaimMonthCardReward(ctx context.Context, goodsID int32) (*mallpb.ClaimMonthCardRewardReply, error) {
	req := &mallpb.ClaimMonthCardRewardRequest{GoodsId: goodsID}
	raw, err := a.sendMall(ctx, "ClaimMonthCardReward", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &mallpb.ClaimMonthCardRewardReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// BuyFreeGifts purchases all free mall goods in slot type 1.
func (a *API) BuyFreeGifts(ctx context.Context) (int, error) {
	goods, err := a.GetMallGoods(ctx, 1)
	if err != nil {
		return 0, err
	}
	bought := 0
	for _, g := range goods {
		if g == nil || !g.IsFree || g.GoodsId <= 0 {
			continue
		}
		if _, err := a.Purchase(ctx, g.GoodsId, 1); err != nil {
			continue
		}
		bought++
	}
	return bought, nil
}
