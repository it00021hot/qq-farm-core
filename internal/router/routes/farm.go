package routes

import (
	"github.com/it00021hot/qq-farm-core/internal/app/controller/farm/account"
	"github.com/it00021hot/qq-farm-core/internal/app/controller/farm/activity"
	"github.com/it00021hot/qq-farm-core/internal/app/controller/farm/analytics"
	"github.com/it00021hot/qq-farm-core/internal/app/controller/farm/automation"
	"github.com/it00021hot/qq-farm-core/internal/app/controller/farm/bag"
	"github.com/it00021hot/qq-farm-core/internal/app/controller/farm/commerce"
	"github.com/it00021hot/qq-farm-core/internal/app/controller/farm/dailygifts"
	"github.com/it00021hot/qq-farm-core/internal/app/controller/farm/friend"
	"github.com/it00021hot/qq-farm-core/internal/app/controller/farm/gameconfig"
	"github.com/it00021hot/qq-farm-core/internal/app/controller/farm/lands"
	farmlogs "github.com/it00021hot/qq-farm-core/internal/app/controller/farm/logs"
	"github.com/it00021hot/qq-farm-core/internal/app/controller/farm/status"
	farmws "github.com/it00021hot/qq-farm-core/internal/app/controller/farm/ws"
	farmwxlogin "github.com/it00021hot/qq-farm-core/internal/app/controller/farm/wxlogin"
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
)

// InitFarmGroup 农场业务
func InitFarmGroup(r fiber.Router, handles ...any) {
	router := r.Group("farm", handles...)
	{
		router.Get("/account/list", account.Account.List).Name("农场账号列表")
		router.Get("/account/detail", account.Account.Detail).Name("农场账号详情")
		router.Post("/account/add", account.Account.Create).Name("创建农场账号")
		router.Post("/account/modify", account.Account.Update).Name("更新农场账号")
		router.Post("/account/delete", account.Account.Delete).Name("删除农场账号")
		router.Post("/account/start", account.Account.Start).Name("启动农场账号")
		router.Post("/account/stop", account.Account.Stop).Name("停止农场账号")

		router.Post("/wx-login/tasks", farmwxlogin.WXLogin.CreateTask).Name("创建微信扫码登录任务")
		router.Get("/wx-login/tasks/:taskId/qr", farmwxlogin.WXLogin.QRImage).Name("微信扫码登录二维码")
		router.Get("/wx-login/tasks/:taskId/status", farmwxlogin.WXLogin.Status).Name("微信扫码登录状态")
		router.Post("/wx-login/tasks/:taskId/confirm", farmwxlogin.WXLogin.Confirm).Name("确认微信扫码登录")
		router.Post("/wx-login/tasks/:taskId/code", farmwxlogin.WXLogin.Code).Name("获取微信登录code")

		router.Get("/automation/detail", automation.Automation.Detail).Name("自动化配置详情")
		router.Post("/automation/modify", automation.Automation.Modify).Name("修改自动化配置")

		router.Get("/status/detail", status.Status.Detail).Name("运行状态详情")
		router.Get("/status/list", status.Status.List).Name("运行状态列表")
		router.Get("/logs", farmlogs.Logs.List).Name("运行日志列表")
		router.Delete("/logs", farmlogs.Logs.Clear).Name("清空运行日志")
		router.Get("/lands", lands.Lands.Get).Name("农场土地列表")
		router.Post("/operate", lands.Lands.Operate).Name("执行农场操作")
		router.Get("/bag", bag.Bag.Get).Name("农场背包")
		router.Get("/seeds", bag.Bag.Seeds).Name("可用种子列表")
		router.Post("/bag/sell", bag.Bag.Sell).Name("出售背包物品")
		router.Post("/bag/use", bag.Bag.Use).Name("使用背包物品")
		router.Get("/daily-gifts", dailygifts.DailyGifts.Get).Name("每日礼包与任务总览")
		router.Get("/game-mall", commerce.Commerce.Mall).Name("游戏商城")
		router.Post("/game-mall/purchase", commerce.Commerce.Purchase).Name("购买商城商品")
		router.Get("/mystery-shop", commerce.Commerce.MysteryShop).Name("神秘商人")
		router.Get("/diamond", commerce.Commerce.Diamond).Name("钻石余额")

		router.Get("/friend/list", friend.Friend.List).Name("好友列表")
		router.Post("/friend/sync", friend.Friend.Sync).Name("同步好友列表")
		router.Get("/friend/lands", friend.Friend.Lands).Name("好友土地详情")
		router.Post("/friend/op", friend.Friend.Op).Name("好友互动操作")
		router.Get("/friend/interact-logs", friend.Friend.InteractLogs).Name("互动记录")
		router.Get("/friend/interact-records", friend.Friend.InteractRecords).Name("最近访客")

		router.Get("/activity/snapshot", activity.Activity.Snapshot).Name("活动快照")
		router.Post("/activity/pass/claim", activity.Activity.ClaimPass).Name("领取通行证")
		router.Post("/activity/constellation/light", activity.Activity.LightConstellation).Name("点亮观星")
		router.Post("/activity/shop/exchange", activity.Activity.ShopExchange).Name("星砂兑换")
		router.Post("/activity/solar-terms/claim", activity.Activity.ClaimSolarTerm).Name("领取节令")
		router.Post("/activity/task/claim", activity.Activity.ClaimTask).Name("领取任务")
		router.Post("/activity/gift/claim", activity.Activity.ClaimGift).Name("领取礼包")

		router.Get("/analytics/detail", analytics.Analytics.Detail).Name("分析详情")
		router.Get("/game-config/list", gameconfig.GameConfig.List).Name("游戏配置列表")
		router.Post("/game-config/modify", gameconfig.GameConfig.Modify).Name("修改游戏配置")
		router.Get("/game-config/seeds", gameconfig.GameConfig.Seeds).Name("种子目录")
		router.Get("/game-config/fruits", gameconfig.GameConfig.Fruits).Name("果实目录")
		router.Get("/game-config/items", gameconfig.GameConfig.Items).Name("道具目录")
		router.Get("/game-config/plants", gameconfig.GameConfig.Plants).Name("植物目录")
		router.Get("/game-config/item-types", gameconfig.GameConfig.ItemTypes).Name("物品类型")
		router.Post("/game-config/seed/add", gameconfig.GameConfig.SeedAdd).Name("录入种子")
		router.Post("/game-config/seed/modify", gameconfig.GameConfig.SeedModify).Name("修改种子")
		router.Post("/game-config/seed/delete", gameconfig.GameConfig.SeedDelete).Name("删除种子")
		router.Post("/game-config/fruit/add", gameconfig.GameConfig.FruitAdd).Name("录入果实")
		router.Post("/game-config/fruit/modify", gameconfig.GameConfig.FruitModify).Name("修改果实")
		router.Post("/game-config/fruit/delete", gameconfig.GameConfig.FruitDelete).Name("删除果实")
		router.Post("/game-config/item/add", gameconfig.GameConfig.ItemAdd).Name("录入道具")
		router.Post("/game-config/item/modify", gameconfig.GameConfig.ItemModify).Name("修改道具")
		router.Post("/game-config/item/delete", gameconfig.GameConfig.ItemDelete).Name("删除道具")

		router.Get("/ws", farmws.WS.Upgrade, websocket.New(farmws.WS.Handle)).Name("农场实时通道")
	}
}
