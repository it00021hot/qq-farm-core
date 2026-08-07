package dailygifts

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/app/service"
	farmruntime "github.com/it00021hot/qq-farm-core/internal/farm/runtime"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/internal/vars"
	"github.com/gofiber/fiber/v3"
)

type Service struct {
	service.Service
}

var DailyGifts = &Service{}

func (s *Service) Get(ctx fiber.Ctx, req farmtypes.DailyGiftsReq) (farmruntime.DailyGiftOverview, error) {
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return farmruntime.DailyGiftOverview{}, friendlyFarmErr(err)
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := session.DailyGiftOverview(callCtx)
	if err != nil {
		return farmruntime.DailyGiftOverview{}, friendlyFarmErr(err)
	}
	return out, nil
}

func friendlyFarmErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "i/o timeout") {
		return errors.New("游戏连接已断开，请重新启动账号")
	}
	return err
}

func (s *Service) session(ctx fiber.Ctx, accountID uint64) (*farmruntime.Session, error) {
	var account model.FarmAccount
	if err := vars.DB.Where("id = ?", accountID).First(&account).Error; err != nil {
		return nil, errors.New("账号不存在")
	}
	session, ok := farmruntime.Default.Session(accountID)
	if !ok || session.Status() != farmruntime.StatusRunning {
		return nil, errors.New("账号未运行")
	}
	return session, nil
}
