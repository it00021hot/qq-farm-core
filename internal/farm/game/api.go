// Package game provides PlantService RPC helpers for own-farm operations.
package game

import (
	"context"
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/gatepb"
	"google.golang.org/protobuf/proto"
)

const plantService = "gamepb.plantpb.PlantService"
const itemService = "gamepb.itempb.ItemService"
const shopService = "gamepb.shoppb.ShopService"
const mallService = "gamepb.mallpb.MallService"
const friendService = "gamepb.friendpb.FriendService"
const visitService = "gamepb.visitpb.VisitService"
const userService = "gamepb.userpb.UserService"
const interactService = "gamepb.interactpb.InteractService"
const interactVisitorService = "gamepb.interactpb.VisitorService"
const guideService = "gamepb.guidepb.GuideService"
const mutantService = "gamepb.mutantpb.MutantService"
const randomDropService = "gamepb.randomdroppb.RandomDropService"
const redPacketService = "gamepb.redpacketpb.RedPacketService"
const skinService = "gamepb.skinpb.SkinService"
const dogService = "gamepb.dogpb.DogService"
const careerService = "gamepb.careerpb.CareerService"
const avatarFrameService = "gamepb.avatarframepb.AvatarFrameService"
const bulletinBoardService = "gamepb.bulletinboardpb.BulletinBoardService"
const marqueeService = "gamepb.marqueepb.MarqueeService"
const payService = "gamepb.paypb.PayService"
const rechargeBonusService = "gamepb.rechargebonuspb.RechargeBonusService"
const uicProxyService = "gamepb.uicproxypb.UicProxyService"
const mysteryShopService = "gamepb.mysteryshoppb.MysteryShopService"

const (
	NormalFertilizerID  int64 = 1011
	OrganicFertilizerID int64 = 1012
)

// Sender is the RPC transport used by game APIs.
type Sender interface {
	Send(ctx context.Context, service, method string, body []byte) ([]byte, *gatepb.Meta, error)
}

// API wraps plant service calls.
type API struct {
	Sender Sender
	GID    int64 // host_gid for own-farm ops
}

func (a *API) requireSender() error {
	if a == nil || a.Sender == nil {
		return fmt.Errorf("game: sender nil")
	}
	return nil
}

func marshalMessage(message proto.Message) []byte {
	body, _ := proto.Marshal(message)
	return body
}

func unmarshalMessage(body []byte, message proto.Message) error {
	return proto.Unmarshal(body, message)
}
