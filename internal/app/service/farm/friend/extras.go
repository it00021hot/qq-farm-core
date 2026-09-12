package friend

// 好友面板扩展：删好友（bot deleteFriend 语义：删除 + 移出列表缓存/已知 GID + 自动加黑名单）、
// 特殊互动道具（好友土地批量 / 好友农场整场 / 自己土地批量）。

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	farmruntime "github.com/it00021hot/qq-farm-core/internal/farm/runtime"
	"github.com/it00021hot/qq-farm-core/internal/types/farm"
)

// liveSession resolves the account's running session + API.
func (s *Service) liveSession(accountID uint64) (*farmruntime.Session, error) {
	session, ok := farmruntime.Default.Session(accountID)
	if !ok || session.Status() != farmruntime.StatusRunning {
		return nil, errors.New("账号未运行")
	}
	if session.GameAPI() == nil {
		return nil, errors.New("账号未连接游戏")
	}
	return session, nil
}

// Delete 删除好友并按 bot 语义清理本地状态（含自动加黑名单）。
func (s *Service) Delete(ctx fiber.Ctx, req farm.FriendOpReq) (map[string]any, error) {
	session, err := s.liveSession(req.AccountID)
	if err != nil {
		return nil, err
	}
	api := session.GameAPI()
	callCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	if err := farmruntime.DeleteFriend(callCtx, session, api, req.Gid); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "gid": strconv.FormatInt(req.Gid, 10)}, nil
}

// InteractionItems lists usable friend-land / friend-farm interaction items.
func (s *Service) InteractionItems(ctx fiber.Ctx, req farm.FriendListReq) (map[string]any, error) {
	session, err := s.liveSession(req.AccountID)
	if err != nil {
		return nil, err
	}
	api := session.GameAPI()
	callCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	items, err := farmruntime.GetFriendInteractionItems(callCtx, api)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"items":                    items,
		"count":                    len(items),
		"serverValidationRequired": true,
		"confirmationRequired":     true,
		"message":                  map[bool]string{true: "请选择好友农场或土地使用", false: "背包中暂无可用于好友农场的特殊互动道具"}[len(items) > 0],
	}, nil
}

// SelfInteractionItems lists self-usable interaction items (SELF_USABLE 白名单).
func (s *Service) SelfInteractionItems(ctx fiber.Ctx, req farm.FriendListReq) (map[string]any, error) {
	session, err := s.liveSession(req.AccountID)
	if err != nil {
		return nil, err
	}
	api := session.GameAPI()
	callCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	items, err := farmruntime.GetSelfInteractionItems(callCtx, api)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"items":                    items,
		"count":                    len(items),
		"serverValidationRequired": true,
		"confirmationRequired":     true,
		"message":                  map[bool]string{true: "请选择自己农场中符合条件的土地使用", false: "背包中暂无可对自己农场使用的特殊互动道具"}[len(items) > 0],
	}, nil
}

// InteractionUse 批量使用互动道具：friendGid + itemId + landIds（土地类）或仅 friendGid（农场类 5005）。
func (s *Service) InteractionUse(ctx fiber.Ctx, req farm.InteractionUseReq) (map[string]any, error) {
	session, err := s.liveSession(req.AccountID)
	if err != nil {
		return nil, err
	}
	api := session.GameAPI()
	callCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	myGID := session.GID()
	if req.FriendGid > 0 && req.FriendGid != myGID {
		used, failed, err := farmruntime.UseFriendInteractionItemBatch(callCtx, session, api, req.FriendGid, req.ItemId, req.LandIds)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"used": used, "failed": failed,
			"message": interactionSummary(used, failed, "好友农场"),
		}, nil
	}
	// 自己农场（白名单道具：闪电变异瓶/七夕土地道具）。
	used, failed, err := farmruntime.UseSelfInteractionItemBatch(callCtx, session, api, req.ItemId, req.LandIds)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"used": used, "failed": failed,
		"message": interactionSummary(used, failed, "我的农场"),
	}, nil
}

func interactionSummary(used, failed int, where string) string {
	if failed > 0 {
		return "已在" + where + "按顺序使用 " + strconv.Itoa(used) + " 个道具，跳过 " + strconv.Itoa(failed) + " 块地"
	}
	return "已在" + where + "按顺序使用 " + strconv.Itoa(used) + " 个道具"
}
