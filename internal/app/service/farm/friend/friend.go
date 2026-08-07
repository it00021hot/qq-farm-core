package friend

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/pkg/pagination"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	"github.com/MQEnergy/go-skeleton/internal/farm/logic"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/friendpb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/interactpb"
	farmruntime "github.com/MQEnergy/go-skeleton/internal/farm/runtime"
	farmtypes "github.com/MQEnergy/go-skeleton/internal/types/farm"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/fiber/v3"
	"github.com/spf13/cast"
)

type Service struct {
	service.Service
}

var Friend = &Service{}

type friendPlantSummary struct {
	StealNum  int64 `json:"stealNum,omitempty"`
	DryNum    int64 `json:"dryNum,omitempty"`
	WeedNum   int64 `json:"weedNum,omitempty"`
	InsectNum int64 `json:"insectNum,omitempty"`
}

type friendView struct {
	ID        uint64              `json:"id,omitempty"`
	AccountID uint64              `json:"accountId"`
	Gid       int64               `json:"gid"`
	Nickname  string              `json:"nickname"`
	Level     int64               `json:"level,omitempty"`
	Gold      int64               `json:"gold,omitempty"`
	Avatar    string              `json:"avatar,omitempty"`
	SyncedAt  uint                `json:"syncedAt,omitempty"`
	Plant     *friendPlantSummary `json:"plant,omitempty"`
}

func friendNickname(f friendpb.GameFriend) string {
	if f.Remark != "" {
		return f.Remark
	}
	return f.Name
}

func enrichFromLive(row model.FarmFriendGid, live *friendpb.GameFriend) friendView {
	view := friendView{
		ID:        row.ID,
		AccountID: row.AccountID,
		Gid:       row.Gid,
		Nickname:  row.Nickname,
		SyncedAt:  row.SyncedAt,
	}
	if live != nil {
		view.Nickname = friendNickname(*live)
		if live.Level > 0 {
			view.Level = live.Level
		}
		if live.Gold > 0 {
			view.Gold = live.Gold
		}
		if live.AvatarURL != "" {
			view.Avatar = live.AvatarURL
		}
		if live.Plant != nil {
			view.Plant = &friendPlantSummary{
				StealNum:  live.Plant.StealPlantNum,
				DryNum:    live.Plant.DryNum,
				WeedNum:   live.Plant.WeedNum,
				InsectNum: live.Plant.InsectNum,
			}
		}
		view.SyncedAt = uint(time.Now().Unix())
	}
	return view
}

func (s *Service) List(ctx fiber.Ctx, req farmtypes.FriendListReq) (response.PageData, error) {
	if req.AccountID == 0 {
		return response.PageData{}, errors.New("accountId 必填")
	}
	tid := cast.ToUint64(ctx.Locals(tenant.LocalTenantID))

	liveMap := map[int64]friendpb.GameFriend{}
	if session, err := s.session(ctx, req.AccountID); err == nil {
		callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		friends, loadErr := session.Friends(callCtx)
		cancel()
		if loadErr == nil {
			farmruntime.SyncFriendsToDB(req.AccountID, tid, friends)
			for _, f := range friends {
				if f.GID > 0 {
					liveMap[f.GID] = f
				}
			}
		}
	}

	db := tenant.Scope(vars.DB, tenant.TenantCtx(ctx)).Model(&model.FarmFriendGid{}).Where("account_id = ?", req.AccountID)
	if req.Keyword != "" {
		db = db.Where("nickname LIKE ?", "%"+req.Keyword+"%")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return response.PageData{}, err
	}
	page := pagination.New().ParsePage(req.Current, req.Size)
	page.Total = total
	page.GetLastPage()
	var list []model.FarmFriendGid
	if err := db.Order("id desc").Offset(page.GetOffset()).Limit(page.GetLimit()).Find(&list).Error; err != nil {
		return response.PageData{}, err
	}
	records := make([]friendView, 0, len(list))
	for _, row := range list {
		var live *friendpb.GameFriend
		if f, ok := liveMap[row.Gid]; ok {
			copy := f
			live = &copy
		}
		records = append(records, enrichFromLive(row, live))
	}
	return response.NewPageData(records, req.Current, req.Size, total), nil
}

func (s *Service) Sync(ctx fiber.Ctx, req farmtypes.FriendSyncReq) (map[string]any, error) {
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return nil, errors.New("账号未在线或功能未就绪")
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	friends, err := session.Friends(callCtx)
	if err != nil {
		return nil, err
	}
	tid := cast.ToUint64(ctx.Locals(tenant.LocalTenantID))
	farmruntime.SyncFriendsToDB(req.AccountID, tid, friends)
	return map[string]any{
		"accountId": req.AccountID,
		"count":     len(friends),
		"synced":    true,
	}, nil
}

func (s *Service) Lands(ctx fiber.Ctx, req farmtypes.FriendLandsReq) (logic.LandsUIResponse, error) {
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return logic.LandsUIResponse{}, errors.New("账号未在线或功能未就绪")
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	lands, err := session.FriendLands(callCtx, req.Gid)
	if err != nil {
		return logic.LandsUIResponse{}, err
	}
	return logic.FormatFriendLandsResponse(lands), nil
}

func (s *Service) Op(ctx fiber.Ctx, req farmtypes.FriendOpReq) (map[string]any, error) {
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return nil, errors.New("账号未在线或功能未就绪")
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := session.FriendOp(callCtx, req.Gid, req.Op); err != nil {
		return nil, err
	}
	return map[string]any{
		"accountId": req.AccountID,
		"gid":       req.Gid,
		"op":        req.Op,
		"ok":        true,
	}, nil
}

func (s *Service) InteractLogs(ctx fiber.Ctx, req farmtypes.FriendListReq) (response.PageData, error) {
	db := tenant.Scope(vars.DB, tenant.TenantCtx(ctx)).Model(&model.FarmInteractLog{})
	if req.AccountID > 0 {
		db = db.Where("account_id = ?", req.AccountID)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return response.PageData{}, err
	}
	page := pagination.New().ParsePage(req.Current, req.Size)
	page.Total = total
	page.GetLastPage()
	var list []model.FarmInteractLog
	if err := db.Order("id desc").Offset(page.GetOffset()).Limit(page.GetLimit()).Find(&list).Error; err != nil {
		return response.PageData{}, err
	}
	return response.NewPageData(list, req.Current, req.Size, total), nil
}

// InteractRecords returns live visitor records from the game (aligned with bot /api/interact-records).
func (s *Service) InteractRecords(ctx fiber.Ctx, req farmtypes.FriendInteractRecordsReq) ([]map[string]any, error) {
	if req.AccountID == 0 {
		return nil, errors.New("accountId 必填")
	}
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return nil, errors.New("账号未在线或功能未就绪")
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	reply, err := session.InteractRecords(callCtx)
	if err != nil {
		return nil, err
	}
	raw := reply.Records
	out := make([]map[string]any, 0, len(raw))
	for i, rec := range raw {
		if rec == nil {
			continue
		}
		out = append(out, formatInteractRecord(rec, i))
	}
	sort.SliceStable(out, func(i, j int) bool {
		ai, _ := out[i]["serverTimeSec"].(int64)
		aj, _ := out[j]["serverTimeSec"].(int64)
		if ai != aj {
			return ai > aj
		}
		gi, _ := out[i]["visitorGid"].(int64)
		gj, _ := out[j]["visitorGid"].(int64)
		if gi != gj {
			return gi > gj
		}
		ti, _ := out[i]["actionType"].(int32)
		tj, _ := out[j]["actionType"].(int32)
		return ti > tj
	})
	return out, nil
}

func formatInteractRecord(rec *interactpb.InteractRecord, index int) map[string]any {
	actionType := rec.ActionType
	visitorGID := rec.VisitorGID
	cropID := int64(rec.CropID)
	cropCount := int64(rec.CropCount)
	times := int64(rec.Times)
	level := int64(rec.Level)
	fromType := int64(rec.FromType)
	serverTimeSec := rec.ServerTime
	if serverTimeSec > 1_000_000_000_000 {
		serverTimeSec = serverTimeSec / 1000
	}
	landID := int64(0)
	flag1 := int64(0)
	flag2 := int64(0)
	if rec.Extra != nil {
		landID = int64(rec.Extra.LandID)
		flag1 = int64(rec.Extra.Flag1)
		flag2 = int64(rec.Extra.Flag2)
	}
	cropName := resolveInteractCropName(cropID)
	nick := strings.TrimSpace(rec.Nick)
	if nick == "" {
		nick = fmt.Sprintf("GID:%d", visitorGID)
	}
	actionLabel := interactActionLabel(actionType)
	row := map[string]any{
		"key":           fmt.Sprintf("%d-%d-%d-%d", serverTimeSec, visitorGID, actionType, index),
		"serverTimeSec": serverTimeSec,
		"serverTimeMs":  serverTimeSec * 1000,
		"actionType":    actionType,
		"actionLabel":   actionLabel,
		"visitorGid":    visitorGID,
		"nick":          nick,
		"avatarUrl":     strings.TrimSpace(rec.AvatarURL),
		"cropId":        cropID,
		"cropName":      cropName,
		"cropCount":     cropCount,
		"times":         times,
		"fromType":      fromType,
		"level":         level,
		"landId":        landID,
		"flag1":         flag1,
		"flag2":         flag2,
	}
	row["actionDetail"] = buildInteractActionDetail(row)
	return row
}

func interactActionLabel(actionType int32) string {
	switch actionType {
	case 1:
		return "偷取作物"
	case 2:
		return "帮忙"
	case 3:
		return "捣乱"
	default:
		return "互动"
	}
}

func resolveInteractCropName(cropID int64) string {
	if cropID <= 0 {
		return ""
	}
	if name := logic.GetPlantName(cropID); name != "" {
		return name
	}
	if plant := logic.GetPlantByFruitID(cropID); plant != nil && strings.TrimSpace(plant.Name) != "" {
		return plant.Name
	}
	if item := logic.GetItemByID(cropID); item != nil && strings.TrimSpace(item.Name) != "" {
		return item.Name
	}
	return ""
}

func buildInteractActionDetail(row map[string]any) string {
	actionType, _ := row["actionType"].(int32)
	cropName, _ := row["cropName"].(string)
	cropCount, _ := row["cropCount"].(int64)
	times, _ := row["times"].(int64)
	landID, _ := row["landId"].(int64)
	parts := make([]string, 0, 3)
	switch actionType {
	case 1:
		switch {
		case cropName != "" && cropCount > 0:
			parts = append(parts, fmt.Sprintf("偷取 %s × %d", cropName, cropCount))
		case cropName != "":
			parts = append(parts, fmt.Sprintf("偷取 %s", cropName))
		case cropCount > 0:
			parts = append(parts, fmt.Sprintf("偷取作物 × %d", cropCount))
		default:
			parts = append(parts, "偷取作物")
		}
	case 2:
		if times > 1 {
			parts = append(parts, fmt.Sprintf("帮忙 %d 次", times))
		} else {
			parts = append(parts, "帮忙")
		}
	case 3:
		if times > 1 {
			parts = append(parts, fmt.Sprintf("捣乱 %d 次", times))
		} else {
			parts = append(parts, "捣乱")
		}
	default:
		if times > 1 {
			parts = append(parts, fmt.Sprintf("互动 %d 次", times))
		} else {
			parts = append(parts, "互动")
		}
	}
	if landID > 0 {
		parts = append(parts, fmt.Sprintf("地块 %d", landID))
	}
	return strings.Join(parts, " · ")
}

func (s *Service) session(ctx fiber.Ctx, accountID uint64) (*farmruntime.Session, error) {
	var account model.FarmAccount
	if err := tenant.Scope(vars.DB, tenant.TenantCtx(ctx)).Where("id = ?", accountID).First(&account).Error; err != nil {
		return nil, errors.New("账号不存在")
	}
	session, ok := farmruntime.Default.Session(accountID)
	if !ok || session.Status() != farmruntime.StatusRunning {
		return nil, errors.New("账号未运行")
	}
	return session, nil
}
