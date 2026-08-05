package attachment

import (
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"

	attachment2 "github.com/MQEnergy/go-skeleton/internal/app/service/system/attachment"
	"github.com/MQEnergy/go-skeleton/internal/types/attachment"
	"github.com/MQEnergy/go-skeleton/pkg/response"

	"github.com/MQEnergy/go-skeleton/internal/app/controller"
)

type AttachmentController struct {
	controller.Controller
}

var Attachment = &AttachmentController{}

// AccessURLReq swagger 置换访问地址参数
type AccessURLReq = attachment.AccessURLReq

// Upload 上传资源
//
//	@Summary		上传资源
//	@Description	上传资源文件接口，返回 file_path（入库 key）与短期 signed_url
//	@Tags			附件管理
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			file	formData	file					true	"上传的文件"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Router			/system/attachment/upload [post]
func (c *AttachmentController) Upload(ctx fiber.Ctx) error {
	var reqParams attachment.UploadReq
	if err := c.Validate(ctx, &reqParams); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	fileHeader, _, err := attachment2.Attachment.Upload(ctx, reqParams)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", fileHeader)
}

// AccessURL 置换私有对象临时访问地址
//
//	@Summary		置换临时访问地址
//	@Description	根据 file_path（attach_url）签发短期预签名 GET URL
//	@Tags			附件管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		AccessURLReq			true	"访问参数"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Router			/system/attachment/access-url [post]
func (c *AttachmentController) AccessURL(ctx fiber.Ctx) error {
	var reqParams attachment.AccessURLReq
	if err := c.Validate(ctx, &reqParams); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	resp, err := attachment2.Attachment.AccessURL(reqParams.FilePath)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", resp)
}

// ReadFile 根据文件路径获取并返回文件内容（需登录；云存储场景优先使用 access-url 预签名）
//
//	@Summary		获取文件内容
//	@Description	服务端代理读取对象存储文件内容。云存储场景请优先使用 access-url 预签名直链。
//	@Tags			附件管理
//	@Accept			json
//	@Produce		octet-stream
//	@Security		ApiKeyAuth
//	@Param			file_path	path		string	true	"文件路径"
//	@Success		200			{file}		[]byte	"文件内容(支持gzip压缩)"
//	@Router			/system/attachment/file/{file_path} [get]
func (c *AttachmentController) ReadFile(ctx fiber.Ctx) error {
	var reqParams attachment.ReadFileReq
	if err := c.Validate(ctx, &reqParams); err != nil {
		return ctx.Status(fiber.StatusOK).Send([]byte("参数错误"))
	}
	// 获取文件数据
	fileInfo, fileData, err := attachment2.Attachment.GetFileByURL(reqParams.FilePath, reqParams.XOssProcess)
	if err != nil {
		slog.Error("读取文件失败", "filePath", reqParams.FilePath, "err", err)
		return ctx.Status(fiber.StatusOK).Send([]byte{})
	}
	fmt.Println(fileInfo.Header["Content-Type"][0])
	// 设置响应头
	if len(fileInfo.Header["Content-Type"]) > 0 {
		ctx.Set("Content-Type", fileInfo.Header["Content-Type"][0])
	}
	ctx.Set("Cache-Control", "public, max-age=86400")     // 缓存1天
	ctx.Set("ETag", fmt.Sprintf("%x", md5.Sum(fileData))) // 使用文件内容的MD5作为ETag

	// 检查是否支持gzip压缩
	if strings.Contains(ctx.Get("Accept-Encoding"), "gzip") {
		ctx.Set("Content-Encoding", "gzip")

		gz := gzip.NewWriter(ctx.Response().BodyWriter())
		if _, err := gz.Write(fileData); err != nil {
			slog.Error("压缩文件失败", "err", err)
			return ctx.Status(fiber.StatusInternalServerError).Send([]byte("内部错误"))
		}
		if err := gz.Close(); err != nil {
			slog.Error("关闭gzip写入失败", "err", err)
			return ctx.Status(fiber.StatusInternalServerError).Send([]byte("内部错误"))
		}
		return nil
	}
	if _, err := ctx.Status(fiber.StatusOK).Write(fileData); err != nil {
		slog.Error("写入响应失败", "err", err)
		return ctx.Status(fiber.StatusInternalServerError).Send([]byte("内部错误"))
	}
	return nil
}

// List 附件列表
//
//	@Summary		附件列表
//	@Tags			附件管理
//	@Security		ApiKeyAuth
//	@Router			/system/attachment/list [get]
func (c *AttachmentController) List(ctx fiber.Ctx) error {
	var req attachment.ListReq
	_ = c.Validate(ctx, &req)
	info, err := attachment2.Attachment.List(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Detail 附件详情
//
//	@Summary		附件详情
//	@Tags			附件管理
//	@Security		ApiKeyAuth
//	@Router			/system/attachment/detail [get]
func (c *AttachmentController) Detail(ctx fiber.Ctx) error {
	var req attachment.IDReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := attachment2.Attachment.Detail(ctx, req.ID)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Status 附件启停
//
//	@Summary		附件启停
//	@Tags			附件管理
//	@Security		ApiKeyAuth
//	@Router			/system/attachment/status [post]
func (c *AttachmentController) Status(ctx fiber.Ctx) error {
	var req attachment.StatusReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := attachment2.Attachment.Status(ctx, req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", "")
}

// Delete 删除附件
//
//	@Summary		删除附件
//	@Tags			附件管理
//	@Security		ApiKeyAuth
//	@Router			/system/attachment/delete [post]
func (c *AttachmentController) Delete(ctx fiber.Ctx) error {
	var req attachment.DeleteReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := attachment2.Attachment.Delete(ctx, req.ID); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", "")
}
