package attachment

import (
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/MQEnergy/go-skeleton/internal/app/dao"
	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	"github.com/MQEnergy/go-skeleton/internal/types/attachment"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/upload"
	"github.com/gofiber/fiber/v3"
	"github.com/spf13/cast"
)

type AttachmentService struct {
	service.Service
}

var Attachment = &AttachmentService{}

func (s *AttachmentService) Upload(ctx fiber.Ctx, params attachment.UploadReq) (*upload.FileHeader, *model.Attachment, error) {
	file, err := ctx.FormFile("file")
	if err != nil {
		return nil, nil, fmt.Errorf("获取文件失败: %v", err)
	}
	uploader := upload.New(&vars.Config, 0, []string{})
	fileHeader, err := uploader.Upload(file, params.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("上传文件失败: %v", err)
	}
	attachType := 1
	for i, mimeGroup := range upload.GroupMimeTypes {
		if helper.InAnySlice(mimeGroup, fileHeader.MimeType) {
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

	// 签发短期预签名访问地址
	signedURL, expire, err := uploader.SignURL(fileHeader.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("签发访问地址失败: %v", err)
	}
	fileHeader.SignedURL = signedURL
	fileHeader.Expire = expire
	return fileHeader, attachmentInfo, nil
}

// AccessURL 根据 attach_url / file_path 置换短期访问地址
func (s *AttachmentService) AccessURL(filePath string) (*attachment.AccessURLResp, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("file_path 不能为空")
	}
	signedURL, expire, err := upload.New(&vars.Config, 0, []string{}).SignURL(filePath)
	if err != nil {
		return nil, fmt.Errorf("签发访问地址失败: %v", err)
	}
	return &attachment.AccessURLResp{
		FilePath:  filePath,
		SignedURL: signedURL,
		Expire:    expire,
	}, nil
}

// GetFileByURL 根据URL获取文件内容
func (s *AttachmentService) GetFileByURL(fileURL, xOssProcess string) (*multipart.FileHeader, []byte, error) {
	filePath := upload.ResolveObjectPath(fileURL)
	return upload.New(&vars.Config, 0, []string{}).GetFileInfo(filePath, xOssProcess)
}
