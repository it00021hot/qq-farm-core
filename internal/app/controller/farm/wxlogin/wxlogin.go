package wxlogin

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	farmwx "github.com/MQEnergy/go-skeleton/internal/farm/wxlogin"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/fiber/v3"
	"github.com/spf13/cast"
)

type Controller struct {
	controller.Controller
}

var (
	WXLogin = &Controller{}
	store   = farmwx.NewTaskStore()
)

type createTaskReq struct {
	AppID string `json:"app_id"`
}

func ownerKey(ctx fiber.Ctx) string {
	uid := cast.ToString(ctx.Locals(tenant.LocalUID))
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
