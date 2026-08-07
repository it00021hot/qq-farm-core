package card

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/pkg/pagination"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	farmtypes "github.com/MQEnergy/go-skeleton/internal/types/farm"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/fiber/v3"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type Service struct {
	service.Service
}

var Card = &Service{}

func (s *Service) List(ctx fiber.Ctx, req farmtypes.CardListReq) (response.PageData, error) {
	isPlatform, _ := ctx.Locals(tenant.LocalIsPlatform).(bool)
	if !isPlatform {
		return response.PageData{}, errors.New("仅平台用户可查看卡密列表")
	}
	db := tenant.Global(vars.DB, context.Background()).Model(&model.FarmCard{})
	if req.Keyword != "" {
		kw := "%" + req.Keyword + "%"
		db = db.Where("code LIKE ? OR description LIKE ?", kw, kw)
	}
	if req.CardType > 0 {
		db = db.Where("card_type = ?", req.CardType)
	}
	if req.Status > 0 {
		db = db.Where("status = ?", req.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return response.PageData{}, err
	}
	page := pagination.New().ParsePage(req.Current, req.Size)
	page.Total = total
	page.GetLastPage()
	var list []model.FarmCard
	if err := db.Order("id DESC").Offset(page.GetOffset()).Limit(page.GetLimit()).Find(&list).Error; err != nil {
		return response.PageData{}, err
	}
	return response.NewPageData(list, req.Current, req.Size, total), nil
}

func (s *Service) Generate(ctx fiber.Ctx, req farmtypes.CardGenerateReq) ([]model.FarmCard, error) {
	isPlatform, _ := ctx.Locals(tenant.LocalIsPlatform).(bool)
	if !isPlatform {
		return nil, errors.New("仅平台用户可生成卡密")
	}
	if req.CardType == model.FarmCardTypeTime && req.Value == 0 {
		return nil, errors.New("时长卡密面值不能为0")
	}
	if req.CardType == model.FarmCardTypeQuota && req.Value <= 0 {
		return nil, errors.New("额度卡密面值必须大于0")
	}
	now := uint(time.Now().Unix())
	cards := make([]model.FarmCard, 0, req.Count)
	db := tenant.Global(vars.DB, context.Background())
	err := db.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < req.Count; i++ {
			c := model.FarmCard{
				Code:        strings.ToUpper(helper.RandString(16)),
				CardType:    req.CardType,
				Value:       req.Value,
				Description: req.Description,
				Status:      model.FarmCardStatusUnused,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := tx.Create(&c).Error; err != nil {
				return err
			}
			cards = append(cards, c)
		}
		return nil
	})
	return cards, err
}

func (s *Service) Redeem(ctx fiber.Ctx, req farmtypes.CardRedeemReq) (map[string]any, error) {
	tid := cast.ToUint64(ctx.Locals(tenant.LocalTenantID))
	if tid == 0 {
		return nil, errors.New("缺少租户上下文")
	}
	uid := cast.ToUint64(ctx.Locals(tenant.LocalUID))
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	db := tenant.Global(vars.DB, context.Background())
	var tenantRow model.SysTenant
	var card model.FarmCard
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("code = ?", code).First(&card).Error; err != nil {
			return errors.New("卡密不存在")
		}
		if card.Status == model.FarmCardStatusDisabled {
			return errors.New("卡密已被禁用")
		}
		if card.Status != model.FarmCardStatusUnused || card.UsedByTenant > 0 {
			return errors.New("卡密已被使用")
		}
		if err := tx.Where("id = ?", tid).First(&tenantRow).Error; err != nil {
			return errors.New("租户不存在")
		}
		now := uint(time.Now().Unix())
		updates := map[string]any{"updated_at": now}
		switch card.CardType {
		case model.FarmCardTypeTime:
			if card.Value < 0 {
				updates["expire_at"] = uint(0)
			} else {
				base := tenantRow.ExpireAt
				if base == 0 || base < now {
					base = now
				}
				updates["expire_at"] = base + uint(card.Value)*86400
			}
		case model.FarmCardTypeQuota:
			updates["max_accounts"] = tenantRow.MaxAccounts + uint(card.Value)
		default:
			return errors.New("未知卡密类型")
		}
		if err := tx.Model(&tenantRow).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Model(&card).Updates(map[string]any{
			"status":         model.FarmCardStatusUsed,
			"used_by_tenant": tid,
			"used_at":        now,
			"updated_at":     now,
		}).Error; err != nil {
			return err
		}
		claim := model.FarmCardClaim{
			TenantID:  tid,
			CardID:    card.ID,
			CardCode:  card.Code,
			CardType:  card.CardType,
			Value:     card.Value,
			ClaimedBy: uid,
			CreatedAt: now,
		}
		return tx.Create(&claim).Error
	})
	if err != nil {
		return nil, err
	}
	_ = db.Where("id = ?", tid).First(&tenantRow).Error
	return map[string]any{
		"cardType":    card.CardType,
		"value":       card.Value,
		"expireAt":    tenantRow.ExpireAt,
		"maxAccounts": tenantRow.MaxAccounts,
	}, nil
}

func (s *Service) UpdateStatus(ctx fiber.Ctx, req farmtypes.CardStatusReq) error {
	isPlatform, _ := ctx.Locals(tenant.LocalIsPlatform).(bool)
	if !isPlatform {
		return errors.New("仅平台用户可操作卡密")
	}
	db := tenant.Global(vars.DB, context.Background())
	res := db.Model(&model.FarmCard{}).Where("id = ? AND status = ?", req.ID, model.FarmCardStatusUnused).Updates(map[string]any{
		"status":     req.Status,
		"updated_at": uint(time.Now().Unix()),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("卡密不存在或已使用")
	}
	return nil
}
