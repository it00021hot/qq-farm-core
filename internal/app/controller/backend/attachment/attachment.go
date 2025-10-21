package attachment

import (
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"

	attachment2 "github.com/MQEnergy/go-skeleton/internal/app/service/backend/attachment"
	"github.com/MQEnergy/go-skeleton/internal/types/attachment"
	"github.com/MQEnergy/go-skeleton/pkg/response"

	"github.com/MQEnergy/go-skeleton/internal/app/controller"
)

type AttachmentController struct {
	controller.Controller
}

var Attachment = &AttachmentController{}

// Upload 上传资源
//
//	@Summary		上传资源
//	@Description	上传资源文件接口
//	@Tags			附件管理
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			file	formData	file					true	"上传的文件"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Router			/backend/attachment/upload [post]
func (c *AttachmentController) Upload(ctx *fiber.Ctx) error {
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

// ReadFile 根据文件路径获取并返回文件内容
//
// @Summary        获取文件内容
// @Description    根据文件路径获取并返回文件内容,支持图片等文件类型的直接显示和gzip压缩
// @Tags           附件管理
// @Accept         json
// @Produce        octet-stream
// @Param          file_path    path        string    true    "文件路径,支持相对路径和绝对路径"
// @Success        200    {file}        []byte    "文件内容(支持gzip压缩)"
// @Router         /file/{file_path} [get]
func (c *AttachmentController) ReadFile(ctx *fiber.Ctx) error {
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
		// slog.Info("客户端支持gzip压缩,使用压缩传输")

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
