package upload

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"mime/multipart"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/config"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/oss"
	"github.com/MQEnergy/go-skeleton/pkg/tos"

	"github.com/disintegration/imaging"
	"github.com/samber/lo"
	"github.com/spf13/cast"
)

const (
	// MaxUploadSize 默认最大上传资源大小是10M
	MaxUploadSize = 10 << 20 // 使用位运算更清晰
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
	AttachmentId uint64 `json:"attachment_id"`
	Filename     string `json:"file_name"`   // 图片新名称
	FileSize     int64  `json:"file_size"`   // 图片大小
	FilePath     string `json:"file_path"`   // 相对路径地址
	OriginName   string `json:"origin_name"` // 图片原名称
	MimeType     string `json:"mime_type"`   // 附件mime类型
	Extension    string `json:"extension"`   // 附件后缀名
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

	uploadDir, err := u.makeUploadDir(path)
	if err != nil {
		return nil, err
	}
	switch uploadType {
	case 1:
		return u.uploadToLocal(fileBytes, header, uploadDir)
	case 2:
		return u.uploadToOss(fileBytes, header, uploadDir)
	case 3:
		return u.uploadToTos(fileBytes, header, uploadDir)
	default:
		return u.uploadToLocal(fileBytes, header, uploadDir)
	}
}

// uploadToLocal 上传文件到本地服务器
func (u *Upload) uploadToLocal(fileBytes []byte, header *FileHeader, uploadDir string) (*FileHeader, error) {
	filePath := filepath.Join(uploadDir, header.Filename)
	if err := helper.WriteBytesToFile(fileBytes, filePath); err != nil {
		return nil, err
	}
	pathParts := lo.Compact[string](strings.Split(filepath.ToSlash(filePath), "/"))
	// 数组中去除server.fileUploadPath的值
	pathParts = pathParts[len(vars.Config.GetStringSlice("server.fileUploadPath")):]
	header.FilePath = strings.Join(pathParts, "_")
	return header, nil
}

// uploadToOss 上传文件到阿里云OSS
func (u *Upload) uploadToOss(fileBytes []byte, header *FileHeader, uploadDir string) (*FileHeader, error) {

	o, err := oss.New(&oss.Config{
		EndPoint:     u.config.GetString("oss.endPoint"),
		AccessId:     u.config.GetString("oss.accessKeyId"),
		AccessSecret: u.config.GetString("oss.accessKeySecret"),
		BucketName:   u.config.GetString("oss.bucketName"),
	})
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(uploadDir, header.Filename)
	if err := o.PutObject(filePath, fileBytes); err != nil {
		return nil, err
	}
	pathParts := lo.Compact[string](strings.Split(filepath.ToSlash(filePath), "/"))
	pathParts = pathParts[len(vars.Config.GetStringSlice("server.fileUploadPath")):]
	header.FilePath = strings.Join(pathParts, "_")
	return header, nil
}

// uploadToTos 上传文件到火山引擎TOS
func (u *Upload) uploadToTos(fileBytes []byte, header *FileHeader, uploadDir string) (*FileHeader, error) {
	o, err := tos.New(&tos.Config{
		EndPoint:     u.config.GetString("tos.endPoint"),
		AccessId:     u.config.GetString("tos.accessKeyId"),
		AccessSecret: u.config.GetString("tos.accessKeySecret"),
		BucketName:   u.config.GetString("tos.bucketName"),
		Region:       u.config.GetString("tos.region"),
	})
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(uploadDir, header.Filename)
	if err := o.PutObject(filePath, fileBytes); err != nil {
		return nil, err
	}
	pathParts := lo.Compact[string](strings.Split(filepath.ToSlash(filePath), "/"))
	pathParts = pathParts[len(vars.Config.GetStringSlice("server.fileUploadPath")):]
	header.FilePath = strings.Join(pathParts, "_")
	return header, nil
}

// GetFileInfo 根据不同的上传类型获取文信息
func (u *Upload) GetFileInfo(filePath, xOssProcess string) (*multipart.FileHeader, []byte, error) {
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
	case 2: // OSS存储
		o, err1 := oss.New(&oss.Config{
			EndPoint:     u.config.GetString("oss.endPoint"),
			AccessId:     u.config.GetString("oss.accessKeyId"),
			AccessSecret: u.config.GetString("oss.accessKeySecret"),
			BucketName:   u.config.GetString("oss.bucketName"),
		})
		if err1 != nil {
			return nil, nil, err1
		}
		fileBytes, err = o.GetObject(filePath)
	case 3: // TOS存储
		o, err1 := tos.New(&tos.Config{
			EndPoint:     u.config.GetString("tos.endPoint"),
			AccessId:     u.config.GetString("tos.accessKeyId"),
			AccessSecret: u.config.GetString("tos.accessKeySecret"),
			BucketName:   u.config.GetString("tos.bucketName"),
			Region:       u.config.GetString("tos.region"),
		})
		if err1 != nil {
			return nil, nil, err1
		}
		fileBytes, err = o.GetObject(filePath)
	case 1: // 本地存储
		fallthrough
	default:
		if strings.HasPrefix(mimeType, "image/") && xOssProcess != "" {
			processedBytes, procErr := ProcessImageWithXOssProcess(filePath, xOssProcess)
			if procErr == nil {
				fileBytes = processedBytes
			} else {
				// 如果处理失败，则回退到读取原始文件
				fileBytes, err = helper.ReadLocalFile(filePath)
			}
		} else {
			fileBytes, err = helper.ReadLocalFile(filePath)
		}
	}
	if err != nil {
		return nil, nil, err
	}

	// 创建 multipart.FileHeader
	fileHeader := &multipart.FileHeader{
		Filename: filepath.Base(filePath),
		Size:     int64(len(fileBytes)),
		Header: map[string][]string{
			"Content-Type": {mimeType},
		},
	}
	return fileHeader, fileBytes, nil
}

// ParseXOssProcess 解析x-oss-process参数
// 示例: image/resize,m_lfit,h_200,w_200
func ParseXOssProcess(xOssProcess string) map[string]string {
	if xOssProcess == "" {
		return nil
	}

	params := make(map[string]string)

	// 按逗号分割参数
	parts := strings.Split(xOssProcess, ",")

	for i, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if trimmedPart == "" {
			continue
		}

		if i == 0 {
			// 第一个参数永远是image/*类型，设置为type
			params["type"] = trimmedPart
		} else {
			// 处理其他参数
			if strings.Contains(trimmedPart, "_") {
				// 处理类似 m_lfit 的参数，将下划线前的部分作为key
				kv := strings.SplitN(trimmedPart, "_", 2)
				if len(kv) == 2 {
					params[kv[0]] = kv[1]
				}
			} else if strings.Contains(trimmedPart, "=") {
				// 处理类似 h=200 的参数
				kv := strings.SplitN(trimmedPart, "=", 2)
				if len(kv) == 2 {
					params[kv[0]] = kv[1]
				}
			}
		}
	}

	return params
}

// ProcessImageWithXOssProcess 根据x-oss-process参数处理图片
// aliyun oss doc: https://help.aliyun.com/document_detail/44688.html
func ProcessImageWithXOssProcess(filePath, xOssProcess string) ([]byte, error) {
	params := ParseXOssProcess(xOssProcess)
	if len(params) == 0 {
		return nil, fmt.Errorf("无效的x-oss-process参数")
	}

	img, err := imaging.Open(filePath, imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("打开图片失败: %w", err)
	}

	format, err := imaging.FormatFromExtension(filepath.Ext(filePath))
	if err != nil {
		// 如果无法从扩展名确定格式，则回退到PNG
		format = imaging.PNG
	}

	opType, ok := params["type"]
	if !ok {
		return nil, fmt.Errorf("无效的参数 type")
	}

	var processedImage image.Image
	switch opType {
	case "image/resize":
		width, wOk := params["w"]
		height, hOk := params["h"]

		w := cast.ToInt(width)
		h := cast.ToInt(height)

		if (!wOk && !hOk) || (w == 0 && h == 0) {
			return nil, fmt.Errorf("无效的参数")
		}

		mode, mOk := params["m"]
		if !mOk {
			mode = "lfit"
		}

		switch mode {
		case "lfit":
			if w == 0 {
				// h=200, w=0 => 高度为200，宽度等比缩放
				processedImage = imaging.Resize(img, 0, h, imaging.Lanczos)
			} else if h == 0 {
				// h=0, w=200 => 宽度为200，高度等比缩放
				processedImage = imaging.Resize(img, w, 0, imaging.Lanczos)
			} else {
				// h=200, w=200 => 宽高最大为200，等比缩放
				processedImage = imaging.Fit(img, w, h, imaging.Lanczos)
			}
		case "fixed":
			// h=0或w=0时，等比缩放
			processedImage = imaging.Resize(img, w, h, imaging.Lanczos)
		default:
			processedImage = img
		}
	default:
		return nil, fmt.Errorf("不支持的操作类型: %s", opType)
	}

	if processedImage == nil {
		return nil, fmt.Errorf("图片处理失败")
	}

	buf := new(bytes.Buffer)
	err = imaging.Encode(buf, processedImage, format)
	if err != nil {
		return nil, fmt.Errorf("编码处理后的图片失败: %w", err)
	}
	return buf.Bytes(), nil
}

// makeUploadDir 创建上传目录
func (u *Upload) makeUploadDir(path string) (string, error) {
	baseUploadPath := vars.Config.GetString("server.fileUploadPath")
	filePath, err := helper.MakeTimeFormatDir(baseUploadPath, path, time.DateOnly)
	if err != nil {
		return "", fmt.Errorf("创建上传目录失败: %w", err)
	}
	return filePath, nil
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
		MimeType:   contentType,
		Extension:  filePrefix,
	}, nil
}
