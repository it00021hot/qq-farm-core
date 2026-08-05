package upload

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/config"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/oss"
	s3store "github.com/MQEnergy/go-skeleton/pkg/s3"

	"github.com/samber/lo"
)

const (
	// MaxUploadSize 默认最大上传资源大小是10M
	MaxUploadSize = 10 << 20

	// UploadTypeOSS 阿里云 OSS
	UploadTypeOSS = 1
	// UploadTypeS3 S3 兼容存储
	UploadTypeS3 = 2
)

var (
	// GroupMimeTypes 类型分组
	GroupMimeTypes = map[int][]string{
		1: {"image/jpeg", "image/png", "image/gif", "image/bmp", "image/svg", "image/jpg"},
		2: {"video/mp4", "video/vnd.rn-realmedia-vbr", "video/x-msvideo"},
		3: {
			"application/pdf",
			"application/msword",
			"application/vnd.ms-excel",
			"application/vnd.ms-powerpoint",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		},
	}

	// AllowTypes 默认允许上传的文件类型
	AllowTypes = map[string]string{
		"jpg":  "image/jpg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"svg":  "image/svg",
		"gif":  "image/gif",
		"bmp":  "image/bmp",
		"webp": "image/webp",
		"mp3":  "audio/mpeg",
		"mp4":  "video/mp4",
		"avi":  "video/x-msvideo",
		"rmvb": "video/vnd.rn-realmedia-vbr",
		"pdf":  "application/pdf",
		"xls":  "application/vnd.ms-excel",
		"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"ppt":  "application/vnd.ms-powerpoint",
		"doc":  "application/msword",
		"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	}
)

type File struct {
	FileName *multipart.FileHeader `form:"file"`
}

type Upload struct {
	maxUploadSize int
	allowTypes    map[string]string
	config        *config.Config
}

// FileHeader 文件参数
type FileHeader struct {
	AttachmentId uint64 `json:"attachmentId"`
	Filename     string `json:"fileName"`   // 图片新名称
	FileSize     int64  `json:"fileSize"`   // 图片大小
	FilePath     string `json:"filePath"`   // 相对路径地址（DB 存的 canonical key）
	OriginName   string `json:"originName"` // 图片原名称
	MimeType     string `json:"mimeType"`   // 附件mime类型
	Extension    string `json:"extension"`  // 附件后缀名
	SignedURL    string `json:"signedUrl"`  // 短期预签名访问地址
	Expire       int64  `json:"expire"`     // 预签名有效秒数
}

// New 创建一个新的上传实例
func New(cfg *config.Config, maxSize int, allowTypes []string) *Upload {
	if maxSize <= 0 {
		maxSize = MaxUploadSize
	}

	allowMaps := make(map[string]string)
	if len(allowTypes) == 0 {
		allowMaps = AllowTypes
	} else {
		for _, allowType := range allowTypes {
			if mimeType, ok := AllowTypes[allowType]; ok {
				allowMaps[allowType] = mimeType
			}
		}
	}

	return &Upload{
		maxUploadSize: maxSize,
		allowTypes:    allowMaps,
		config:        cfg,
	}
}

// Upload 根据配置的上传类型选择上传方式
func (u *Upload) Upload(file *multipart.FileHeader, path string) (*FileHeader, error) {
	uploadType := u.config.GetInt("server.uploadType")
	header, err := u.validate(file)
	if err != nil {
		return nil, err
	}

	fileBytes, err := helper.GetUploadFileBytes(file)
	if err != nil {
		return nil, err
	}

	uploadDir := u.makeObjectDir(path)
	switch uploadType {
	case UploadTypeOSS:
		return u.uploadToOss(fileBytes, header, uploadDir)
	case UploadTypeS3:
		return u.uploadToS3(fileBytes, header, uploadDir)
	default:
		return nil, fmt.Errorf("不支持的上传类型: %d（仅支持 1:阿里云OSS 2:S3）", uploadType)
	}
}

// uploadToOss 上传文件到阿里云OSS
func (u *Upload) uploadToOss(fileBytes []byte, header *FileHeader, uploadDir string) (*FileHeader, error) {
	o, err := u.newOss()
	if err != nil {
		return nil, err
	}

	filePath := filepath.ToSlash(filepath.Join(uploadDir, header.Filename))
	if err := o.PutObject(filePath, fileBytes, header.MimeType); err != nil {
		return nil, err
	}
	header.FilePath = toStorageKey(filePath)
	return header, nil
}

// uploadToS3 上传文件到 S3 兼容存储
func (u *Upload) uploadToS3(fileBytes []byte, header *FileHeader, uploadDir string) (*FileHeader, error) {
	o, err := u.newS3()
	if err != nil {
		return nil, err
	}

	filePath := filepath.ToSlash(filepath.Join(uploadDir, header.Filename))
	if err := o.PutObject(filePath, fileBytes, header.MimeType); err != nil {
		return nil, err
	}
	header.FilePath = toStorageKey(filePath)
	return header, nil
}

// SignExpireSeconds 预签名有效秒数
func (u *Upload) SignExpireSeconds() int64 {
	sec := u.config.GetInt64("server.signExpireSeconds")
	if sec <= 0 {
		sec = 3600
	}
	return sec
}

// ResolveObjectPath 将 DB/接口中的 underscore key 还原为对象存储路径
func ResolveObjectPath(fileURL string) string {
	filePath := strings.ReplaceAll(fileURL, "_", "/")
	base := vars.Config.GetString("server.fileUploadPath")
	if base != "" && !strings.HasPrefix(filePath, base) {
		filePath = filepath.ToSlash(filepath.Join(base, filePath))
	}
	return filepath.ToSlash(filePath)
}

// SignURL 按当前上传类型签发短期访问地址
func (u *Upload) SignURL(filePathKey string) (string, int64, error) {
	expireSec := u.SignExpireSeconds()
	objectPath := ResolveObjectPath(filePathKey)
	uploadType := u.config.GetInt("server.uploadType")
	expire := time.Duration(expireSec) * time.Second

	switch uploadType {
	case UploadTypeOSS:
		o, err := u.newOss()
		if err != nil {
			return "", 0, err
		}
		signed, err := o.SignGetURL(objectPath, expire)
		return signed, expireSec, err
	case UploadTypeS3:
		o, err := u.newS3()
		if err != nil {
			return "", 0, err
		}
		signed, err := o.SignGetURL(objectPath, expire)
		return signed, expireSec, err
	default:
		return "", 0, fmt.Errorf("不支持的上传类型: %d（仅支持 1:阿里云OSS 2:S3）", uploadType)
	}
}

// GetFileInfo 根据不同的上传类型获取文件信息
func (u *Upload) GetFileInfo(filePath, _ string) (*multipart.FileHeader, []byte, error) {
	uploadType := u.config.GetInt("server.uploadType")
	var fileBytes []byte
	var err error
	filePath, err = url.QueryUnescape(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("url decode file path failed: %w", err)
	}
	ext := filepath.Ext(filePath)
	if ext != "" {
		ext = ext[1:]
	}
	mimeType, ok := AllowTypes[ext]
	if !ok {
		return nil, nil, fmt.Errorf("不支持的文件格式: %s", ext)
	}

	switch uploadType {
	case UploadTypeOSS:
		o, err1 := u.newOss()
		if err1 != nil {
			return nil, nil, err1
		}
		fileBytes, err = o.GetObject(filepath.ToSlash(filePath))
	case UploadTypeS3:
		o, err1 := u.newS3()
		if err1 != nil {
			return nil, nil, err1
		}
		fileBytes, err = o.GetObject(filepath.ToSlash(filePath))
	default:
		return nil, nil, fmt.Errorf("不支持的上传类型: %d（仅支持 1:阿里云OSS 2:S3）", uploadType)
	}
	if err != nil {
		return nil, nil, err
	}

	fileHeader := &multipart.FileHeader{
		Filename: filepath.Base(filePath),
		Size:     int64(len(fileBytes)),
		Header: map[string][]string{
			"Content-Type": {mimeType},
		},
	}
	return fileHeader, fileBytes, nil
}

// makeObjectDir 生成对象目录前缀（不落本地磁盘）
func (u *Upload) makeObjectDir(path string) string {
	baseUploadPath := vars.Config.GetString("server.fileUploadPath")
	parts := []string{baseUploadPath}
	if strings.TrimSpace(path) != "" {
		parts = append(parts, path)
	}
	parts = append(parts, time.Now().Format(time.DateOnly))
	return filepath.ToSlash(filepath.Join(parts...))
}

func toStorageKey(fullPath string) string {
	pathParts := lo.Compact(strings.Split(filepath.ToSlash(fullPath), "/"))
	base := vars.Config.GetString("server.fileUploadPath")
	if base != "" && len(pathParts) > 0 && pathParts[0] == base {
		pathParts = pathParts[1:]
	}
	return strings.Join(pathParts, "_")
}

func (u *Upload) newOss() (*oss.Oss, error) {
	return oss.New(&oss.Config{
		EndPoint:     u.config.GetString("oss.endPoint"),
		AccessId:     u.config.GetString("oss.accessKeyId"),
		AccessSecret: u.config.GetString("oss.accessKeySecret"),
		BucketName:   u.config.GetString("oss.bucketName"),
	})
}

func (u *Upload) newS3() (*s3store.S3, error) {
	return s3store.New(&s3store.Config{
		EndPoint:     u.config.GetString("s3.endPoint"),
		AccessId:     u.config.GetString("s3.accessKeyId"),
		AccessSecret: u.config.GetString("s3.accessKeySecret"),
		BucketName:   u.config.GetString("s3.bucketName"),
		UseSSL:       u.config.GetBool("s3.useSSL"),
		BaseURL:      u.config.GetString("s3.baseUrl"),
		Region:       u.config.GetString("s3.region"),
	})
}

// validate 验证上传文件的有效性
func (u *Upload) validate(file *multipart.FileHeader) (*FileHeader, error) {
	contentType := file.Header.Get("Content-Type")

	if file.Size > int64(u.maxUploadSize) {
		return nil, fmt.Errorf("文件大小超过限制: %dM", u.maxUploadSize>>20)
	}

	if !helper.InAnyMap(u.allowTypes, contentType) {
		return nil, errors.New("不支持的文件格式")
	}

	filePrefix := helper.GetKeyByMap(u.allowTypes, contentType)
	fileName := fmt.Sprintf("file-%s.%s", helper.GenerateUuid(32), filePrefix)

	return &FileHeader{
		Filename:   fileName,
		FileSize:   file.Size,
		FilePath:   "",
		OriginName: file.Filename,
		MimeType:   normalizeMime(contentType, filePrefix),
		Extension:  filePrefix,
	}, nil
}

func normalizeMime(contentType, ext string) string {
	ct := strings.TrimSpace(contentType)
	if ct == "image/jpg" || (ct == "" && (ext == "jpg" || ext == "jpeg")) {
		return "image/jpeg"
	}
	if ct == "image/svg" {
		return "image/svg+xml"
	}
	return ct
}
