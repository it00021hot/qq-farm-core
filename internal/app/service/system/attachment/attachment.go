package attachment

import (
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/pkg/pagination"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	"github.com/MQEnergy/go-skeleton/internal/types/attachment"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
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
	now := uint(time.Now().Unix())
	attachmentInfo := &model.Attachment{
		TenantID:         cast.ToUint64(ctx.Locals(tenant.LocalTenantID)),
		UserID:           cast.ToUint64(ctx.GetRespHeader("uid")),
		AttachName:       fileHeader.Filename,
		AttachOriginName: fileHeader.OriginName,
		AttachURL:        fileHeader.FilePath,
		AttachType:       cast.ToUint8(attachType),
		AttachMimetype:   fileHeader.MimeType,
		AttachExtension:  fileHeader.Extension,
		AttachSize:       cast.ToString(fileHeader.FileSize),
		Status:           1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	tctx := tenant.TenantCtx(ctx)
	if err := tenant.Scope(vars.DB, tctx).Create(attachmentInfo).Error; err != nil {
		return nil, nil, fmt.Errorf("创建附件失败: %v", err)
	}
	fileHeader.AttachmentId = attachmentInfo.ID

	signedURL, expire, err := uploader.SignURL(fileHeader.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("签发访问地址失败: %v", err)
	}
	fileHeader.SignedURL = signedURL
	fileHeader.Expire = expire
	return fileHeader, attachmentInfo, nil
}

func (s *AttachmentService) List(ctx fiber.Ctx, req attachment.ListReq) (response.PageData, error) {
	tctx := tenant.TenantCtx(ctx)
	db := tenant.Scope(vars.DB, tctx).Model(&model.Attachment{})
	if req.Keyword != "" {
		kw := "%" + req.Keyword + "%"
		db = db.Where("attach_origin_name ILIKE ? OR attach_name ILIKE ?", kw, kw)
	}
	if req.AttachType > 0 {
		db = db.Where("attach_type = ?", req.AttachType)
	}
	if ctx.Query("status") != "" {
		db = db.Where("status = ?", req.Status)
	} else {
		db = db.Where("status = ?", 1)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return response.PageData{}, err
	}
	page := pagination.New().ParsePage(req.Current, req.Size)
	var list []model.Attachment
	if err := db.Order("id DESC").Offset(page.GetOffset()).Limit(page.GetLimit()).Find(&list).Error; err != nil {
		return response.PageData{}, err
	}
	return response.NewPageData(list, req.Current, req.Size, total), nil
}

func (s *AttachmentService) Detail(ctx fiber.Ctx, id uint64) (*model.Attachment, error) {
	tctx := tenant.TenantCtx(ctx)
	var row model.Attachment
	if err := tenant.Scope(vars.DB, tctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, fmt.Errorf("附件不存在")
	}
	return &row, nil
}

func (s *AttachmentService) Status(ctx fiber.Ctx, req attachment.StatusReq) error {
	tctx := tenant.TenantCtx(ctx)
	res := tenant.Scope(vars.DB, tctx).Model(&model.Attachment{}).Where("id = ?", req.ID).
		Updates(map[string]any{"status": req.Status, "updated_at": uint(time.Now().Unix())})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("附件不存在")
	}
	return nil
}

func (s *AttachmentService) Delete(ctx fiber.Ctx, id uint64) error {
	tctx := tenant.TenantCtx(ctx)
	var row model.Attachment
	if err := tenant.Scope(vars.DB, tctx).Where("id = ?", id).First(&row).Error; err != nil {
		return fmt.Errorf("附件不存在")
	}
	now := uint(time.Now().Unix())
	if err := tenant.Scope(vars.DB, tctx).Model(&row).Updates(map[string]any{"status": 0, "updated_at": now}).Error; err != nil {
		return err
	}
	return nil
}

func (s *AttachmentService) AccessURL(filePath string) (*attachment.AccessURLResp, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("filePath 不能为空")
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

func (s *AttachmentService) GetFileByURL(fileURL, xOssProcess string) (*multipart.FileHeader, []byte, error) {
	filePath := upload.ResolveObjectPath(fileURL)
	return upload.New(&vars.Config, 0, []string{}).GetFileInfo(filePath, xOssProcess)
}
