package automation

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/app/service"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	farmruntime "github.com/it00021hot/qq-farm-core/internal/farm/runtime"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/internal/vars"
)

type Service struct {
	service.Service
}

var Automation = &Service{}

func parseAccountConfig(raw string) logic.AccountConfig {
	return logic.ParseAccountConfigJSON(raw)
}

func (s *Service) Detail(ctx fiber.Ctx, req farmtypes.AutomationDetailReq) (map[string]any, error) {
	db := vars.DB
	var row model.FarmAccountConfig
	if err := db.Where("account_id = ?", req.AccountID).First(&row).Error; err != nil {
		// 无配置时返回默认
		cfg := logic.DefaultAccountConfig()
		return detailMap(req.AccountID, cfg, mustJSON(cfg)), nil
	}
	cfg := parseAccountConfig(row.ConfigJSON)
	return detailMap(row.AccountID, cfg, row.ConfigJSON), nil
}

func detailMap(accountID uint64, cfg logic.AccountConfig, configJSON string) map[string]any {
	return map[string]any{
		"accountId":                          accountID,
		"automation":                         cfg.Automation,
		"intervals":                          cfg.Intervals,
		"plantingStrategy":                   cfg.PlantingStrategy,
		"preferredSeedId":                    cfg.PreferredSeedID,
		"bagSeedPriority":                    cfg.BagSeedPriority,
		"bagSeedFallbackStrategy":            cfg.BagSeedFallbackStrategy,
		"plantOrderRandom":                   cfg.PlantOrderRandom,
		"plantDelaySeconds":                  cfg.PlantDelaySeconds,
		"stealDelaySeconds":                  cfg.StealDelaySeconds,
		"friendQuietHours":                   cfg.FriendQuietHours,
		"friendBlacklist":                    cfg.FriendBlacklist,
		"plantBlacklist":                     cfg.PlantBlacklist,
		"fertilizerBuyOrganicCount":          cfg.FertilizerBuyOrganicCount,
		"fertilizerBuyOrganicThresholdHours": cfg.FertilizerBuyOrganicThresholdHours,
		"fertilizerBuyNormalCount":           cfg.FertilizerBuyNormalCount,
		"fertilizerBuyNormalThresholdHours":  cfg.FertilizerBuyNormalThresholdHours,
		"fertilizerBuyCheckIntervalMinutes":  cfg.FertilizerBuyCheckIntervalMinutes,
		// 兜种土地类型与好友申请自动接受过滤（runtime/friend_application.go 消费）。
		"bagSeedLandTypes":              cfg.BagSeedLandTypes,
		"autoAcceptFriendMinLevel":      cfg.AutoAcceptFriendMinLevel,
		"autoAcceptRequireOwnLevel":     cfg.AutoAcceptRequireOwnLevel,
		"autoAcceptHarvestStealEnabled": cfg.AutoAcceptHarvestStealEnabled,
		"autoAcceptHarvestStealHarvest": cfg.AutoAcceptHarvestStealHarvest,
		"autoAcceptHarvestStealSteal":   cfg.AutoAcceptHarvestStealSteal,
		// 运行时管理状态（好友同步维护），仅读回显，不走 Modify。
		"knownFriendGids":               cfg.KnownFriendGids,
		"knownFriendGidSyncCooldownSec": cfg.KnownFriendGidSyncCooldownSec,
		"friendsListCacheTtlSec":        cfg.FriendsListCacheTtlSec,
		"configJson":                    configJSON,
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (s *Service) Modify(ctx fiber.Ctx, req farmtypes.AutomationModifyReq) error {
	db := vars.DB
	var row model.FarmAccountConfig
	err := db.Where("account_id = ?", req.AccountID).First(&row).Error
	cfg := logic.DefaultAccountConfig()
	if err == nil && row.ConfigJSON != "" {
		cfg = parseAccountConfig(row.ConfigJSON)
	} else if err != nil {
		return errors.New("配置不存在，请先创建农场账号")
	}

	// 兼容旧客户端整包 JSON
	if req.ConfigJSON != "" {
		if e := json.Unmarshal([]byte(req.ConfigJSON), &cfg); e != nil {
			return errors.New("配置 JSON 无效")
		}
	} else {
		if req.Automation != nil {
			b, _ := json.Marshal(req.Automation)
			_ = json.Unmarshal(b, &cfg.Automation)
		}
		if req.Intervals != nil {
			b, _ := json.Marshal(req.Intervals)
			_ = json.Unmarshal(b, &cfg.Intervals)
		}
		if req.FriendQuietHours != nil {
			b, _ := json.Marshal(req.FriendQuietHours)
			_ = json.Unmarshal(b, &cfg.FriendQuietHours)
		}
		if req.PlantingStrategy != nil {
			cfg.PlantingStrategy = *req.PlantingStrategy
		}
		if req.PreferredSeedID != nil {
			cfg.PreferredSeedID = *req.PreferredSeedID
		}
		if req.BagSeedPriority != nil {
			cfg.BagSeedPriority = req.BagSeedPriority
		}
		if req.BagSeedLandTypes != nil {
			cfg.BagSeedLandTypes = req.BagSeedLandTypes
		}
		if req.BagSeedFallbackStrategy != nil {
			cfg.BagSeedFallbackStrategy = *req.BagSeedFallbackStrategy
		}
		if req.PlantOrderRandom != nil {
			cfg.PlantOrderRandom = *req.PlantOrderRandom
		}
		if req.PlantDelaySeconds != nil {
			cfg.PlantDelaySeconds = *req.PlantDelaySeconds
		}
		if req.StealDelaySeconds != nil {
			cfg.StealDelaySeconds = *req.StealDelaySeconds
		}
		if req.FertilizerBuyOrganicCount != nil {
			cfg.FertilizerBuyOrganicCount = *req.FertilizerBuyOrganicCount
		}
		if req.FertilizerBuyOrganicThresholdHours != nil {
			cfg.FertilizerBuyOrganicThresholdHours = *req.FertilizerBuyOrganicThresholdHours
		}
		if req.FertilizerBuyNormalCount != nil {
			cfg.FertilizerBuyNormalCount = *req.FertilizerBuyNormalCount
		}
		if req.FertilizerBuyNormalThresholdHours != nil {
			cfg.FertilizerBuyNormalThresholdHours = *req.FertilizerBuyNormalThresholdHours
		}
		if req.FertilizerBuyCheckIntervalMinutes != nil {
			cfg.FertilizerBuyCheckIntervalMinutes = *req.FertilizerBuyCheckIntervalMinutes
		}
		if req.FriendBlacklist != nil {
			cfg.FriendBlacklist = req.FriendBlacklist
		}
		if req.PlantBlacklist != nil {
			cfg.PlantBlacklist = req.PlantBlacklist
		}
		// 好友申请自动接受过滤字段。
		if req.AutoAcceptFriendMinLevel != nil {
			cfg.AutoAcceptFriendMinLevel = *req.AutoAcceptFriendMinLevel
		}
		if req.AutoAcceptRequireOwnLevel != nil {
			cfg.AutoAcceptRequireOwnLevel = *req.AutoAcceptRequireOwnLevel
		}
		if req.AutoAcceptHarvestStealEnabled != nil {
			cfg.AutoAcceptHarvestStealEnabled = *req.AutoAcceptHarvestStealEnabled
		}
		if req.AutoAcceptHarvestStealHarvest != nil {
			cfg.AutoAcceptHarvestStealHarvest = *req.AutoAcceptHarvestStealHarvest
		}
		if req.AutoAcceptHarvestStealSteal != nil {
			cfg.AutoAcceptHarvestStealSteal = *req.AutoAcceptHarvestStealSteal
		}
	}
	// 合并后统一走 logic 的 clamp/normalize 路径（等价 Load+Normalize），
	// 保证落库与运行时都拿到归一化后的值。
	cfg = logic.ParseAccountConfigJSON(mustJSON(cfg))

	raw := mustJSON(cfg)
	if err := db.Model(&row).Updates(map[string]any{
		"config_json": raw,
		"updated_at":  uint(time.Now().Unix()),
	}).Error; err != nil {
		return err
	}
	farmruntime.Default.ApplyConfig(req.AccountID, cfg)
	return nil
}
