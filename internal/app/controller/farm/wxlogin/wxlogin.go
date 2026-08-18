package wxlogin

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/it00021hot/qq-farm-core/internal/app/controller"
	farmwx "github.com/it00021hot/qq-farm-core/internal/farm/wxlogin"
	"github.com/it00021hot/qq-farm-core/pkg/response"
	"github.com/spf13/cast"
)

type Controller struct {
	controller.Controller
}

var (
	WXLogin = &Controller{}
	store   = farmwx.NewTaskStore()
	quick   = farmwx.NewQuickStore()
)

type createTaskReq struct {
	AppID string `json:"app_id"`
}

type quickConfirmReq struct {
	RedirectURL string `json:"redirect_url"`
}

type quickAuthorizeReq struct {
	Port          uint16 `json:"port"`
	AuthorizeUUID string `json:"authorize_uuid"`
	X             int    `json:"x"`
	Y             int    `json:"y"`
}

func ownerKey(ctx fiber.Ctx) string {
	uid := cast.ToString(ctx.Locals("uid"))
	if uid == "" {
		uid = cast.ToString(ctx.Get("uid"))
	}
	if uid == "" {
		uid = "anon"
	}
	return uid
}

// CreateTask POST /farm/wx-login/tasks
func (c *Controller) CreateTask(ctx fiber.Ctx) error {
	var req createTaskReq
	_ = ctx.Bind().Body(&req)
	if req.AppID != "" && req.AppID != farmwx.TargetMiniProgramID {
		return response.BadRequestException(ctx, "Unsupported app_id")
	}
	task, err := store.Create(ownerKey(ctx))
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	data := task.PublicView()
	data["qr_url"] = "/farm/wx-login/tasks/" + task.ID + "/qr"
	return response.SuccessJSON(ctx, "", data)
}

// CreateQuickTask POST /farm/wx-login/quick-tasks
func (c *Controller) CreateQuickTask(ctx fiber.Ctx) error {
	session, err := quick.Create(ownerKey(ctx))
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", quick.PublicView(session))
}

// DetectQuickTask POST /farm/wx-login/quick-tasks/:sessionId/detect
func (c *Controller) DetectQuickTask(ctx fiber.Ctx) error {
	profile, err := quick.Detect(context.Background(), ownerKey(ctx), ctx.Params("sessionId"))
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", map[string]any{
		"port":           profile.Port,
		"authorize_uuid": profile.AuthorizeUUID,
		"nickname":       profile.Nickname,
		"headimgurl":     profile.Headimgurl,
	})
}

// AuthorizeQuickTask POST /farm/wx-login/quick-tasks/:sessionId/authorize
func (c *Controller) AuthorizeQuickTask(ctx fiber.Ctx) error {
	var req quickAuthorizeReq
	_ = ctx.Bind().Body(&req)
	redirectURL, err := quick.Authorize(context.Background(), ownerKey(ctx), ctx.Params("sessionId"), req.Port, req.AuthorizeUUID, req.X, req.Y)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", map[string]any{
		"redirect_url": redirectURL,
	})
}

// ConfirmQuickTask POST /farm/wx-login/quick-tasks/:sessionId/confirm
func (c *Controller) ConfirmQuickTask(ctx fiber.Ctx) error {
	var req quickConfirmReq
	_ = ctx.Bind().Body(&req)
	code, creds, err := quick.Confirm(ownerKey(ctx), ctx.Params("sessionId"), req.RedirectURL)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	farmwx.StorePendingAuth(code, creds.ToWxAuth())
	return response.SuccessJSON(ctx, "", map[string]any{
		"openid":  creds.OpenID,
		"app_id":  farmwx.TargetMiniProgramID,
		"code":    code,
		"err_msg": "login:ok",
	})
}

// QRImage GET /farm/wx-login/tasks/:taskId/qr
func (c *Controller) QRImage(ctx fiber.Ctx) error {
	task, err := store.Find(ownerKey(ctx), ctx.Params("taskId"))
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).SendString(err.Error())
	}
	ctx.Set("Content-Type", "image/jpeg")
	return ctx.Send(task.QR)
}

// Status GET /farm/wx-login/tasks/:taskId/status
func (c *Controller) Status(ctx fiber.Ctx) error {
	task, err := store.Find(ownerKey(ctx), ctx.Params("taskId"))
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := store.Poll(task); err != nil {
		task.Status = farmwx.StatusFailed
		store.Destroy(task)
		return response.BadRequestException(ctx, err.Error())
	}
	data := task.PublicView()
	if task.Status == farmwx.StatusCancelled || task.Status == farmwx.StatusExpired {
		store.Destroy(task)
	}
	return response.SuccessJSON(ctx, "", data)
}

// Confirm POST /farm/wx-login/tasks/:taskId/confirm
func (c *Controller) Confirm(ctx fiber.Ctx) error {
	task, err := store.Find(ownerKey(ctx), ctx.Params("taskId"))
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := store.Confirm(task); err != nil {
		task.Status = farmwx.StatusFailed
		store.Destroy(task)
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", task.PublicView())
}

// Code POST /farm/wx-login/tasks/:taskId/code
func (c *Controller) Code(ctx fiber.Ctx) error {
	task, err := store.Find(ownerKey(ctx), ctx.Params("taskId"))
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	code, err := store.IssueCode(task)
	if err != nil {
		task.Status = farmwx.StatusFailed
		store.Destroy(task)
		return response.BadRequestException(ctx, err.Error())
	}
	openid := ""
	if task.Session != nil {
		openid = task.Session.OpenID
		farmwx.StorePendingAuth(code, farmwx.WxAuth{
			OpenID:       task.Session.OpenID,
			LoginBuffer:  task.Session.LoginBuffer,
			AccessToken:  task.Session.AccessToken,
			RefreshToken: task.Session.RefreshToken,
			ExpiresAt:    task.Session.ExpiresAt,
		})
	}
	data := map[string]any{
		"openid":  openid,
		"app_id":  farmwx.TargetMiniProgramID,
		"code":    code,
		"err_msg": "login:ok",
	}
	store.Destroy(task)
	return response.SuccessJSON(ctx, "", data)
}
