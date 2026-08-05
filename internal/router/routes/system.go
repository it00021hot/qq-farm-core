package routes

import (
	backendctl "github.com/MQEnergy/go-skeleton/internal/app/controller/backend"
	"github.com/MQEnergy/go-skeleton/internal/app/controller/system/admin"
	"github.com/MQEnergy/go-skeleton/internal/app/controller/system/attachment"
	"github.com/gofiber/fiber/v3"
)

// InitSystemGroup 系统业务（用户/附件，租户上下文）
// 约定：仅 GET/POST；动作 list|add|modify|delete（及 detail/status 等）
func InitSystemGroup(r fiber.Router, handles ...any) {
	router := r.Group("system", handles...)
	{
		router.Get("/ping", backendctl.Backend.Ping)

		router.Get("/admin/list", admin.Admin.List).Name("用户列表")
		router.Post("/admin/add", admin.Admin.Create).Name("创建用户")
		router.Post("/admin/modify", admin.Admin.Update).Name("更新用户")
		router.Post("/admin/status", admin.Admin.Status).Name("启停用户")
		router.Post("/platform-user/add", admin.Admin.CreatePlatform).Name("创建平台用户")

		router.Post("/attachment/upload", attachment.Attachment.Upload).Name("上传资源")
		router.Post("/attachment/access-url", attachment.Attachment.AccessURL).Name("置换临时访问地址")
		router.Get("/attachment/file/:file_path", attachment.Attachment.ReadFile).Name("读取文件数据内容")
		router.Get("/attachment/list", attachment.Attachment.List).Name("附件列表")
		router.Get("/attachment/detail", attachment.Attachment.Detail).Name("附件详情")
		router.Post("/attachment/status", attachment.Attachment.Status).Name("附件启停")
		router.Post("/attachment/delete", attachment.Attachment.Delete).Name("删除附件")
	}
}
