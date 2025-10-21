package attachment

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/MQEnergy/go-skeleton/internal/app/dao"
	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	"github.com/MQEnergy/go-skeleton/internal/types/attachment"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/upload"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/cast"
)

type AttachmentService struct {
	service.Service
}

var Attachment = &AttachmentService{}

func (s *AttachmentService) Upload(ctx *fiber.Ctx, params attachment.UploadReq) (*upload.FileHeader, *model.Attachment, error) {
	file, err := ctx.FormFile("file")
	if err != nil {
		return nil, nil, fmt.Errorf("获取文件失败: %v", err)
	}
	fileHeader, err := upload.New(&vars.Config, 0, []string{}).Upload(file, params.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("上传文件失败: %v", err)
	}
	attachType := 1
	for i, s := range upload.GroupMimeTypes {
		if helper.InAnySlice(s, fileHeader.MimeType) {
			attachType = i
		}
	}
	attachmentInfo := &model.Attachment{
		UserID:           cast.ToUint64(ctx.GetRespHeader("uid")),
		AttachName:       fileHeader.Filename,
		AttachOriginName: fileHeader.OriginName,
		AttachURL:        fileHeader.FilePath,
		AttachType:       cast.ToUint8(attachType),
		AttachMimetype:   fileHeader.MimeType,
		AttachExtension:  fileHeader.Extension,
		AttachSize:       cast.ToString(fileHeader.FileSize),
		Status:           1,
	}
	if err := dao.Attachment.Create(attachmentInfo); err != nil {
		return nil, nil, fmt.Errorf("创建附件失败: %v", err)
	}
	fileHeader.AttachmentId = attachmentInfo.ID
	return fileHeader, attachmentInfo, nil
}

// GetFileByURL 根据URL获取文件内容
func (s *AttachmentService) GetFileByURL(fileURL, xOssProcess string) (*multipart.FileHeader, []byte, error) {
	// 处理文件路径,将下划线替换为斜杠,并确保使用正确的文件上传根路径
	filePath := strings.ReplaceAll(fileURL, "_", "/")
	if !strings.HasPrefix(filePath, vars.Config.GetString("server.fileUploadPath")) {
		filePath = filepath.Join(vars.Config.GetString("server.fileUploadPath"), filePath)
	}
	return upload.New(&vars.Config, 0, []string{}).GetFileInfo(filePath, xOssProcess)
}
