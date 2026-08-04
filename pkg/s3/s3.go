package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3 S3 兼容存储（MinIO / RustFS / AWS S3）
type S3 struct {
	client  *minio.Client
	bucket  string
	baseURL string
}

// Config S3 配置
type Config struct {
	EndPoint     string
	AccessId     string
	AccessSecret string
	BucketName   string
	UseSSL       bool
	BaseURL      string
	Region       string
}

// New 初始化 S3 客户端
func New(config *Config) (*S3, error) {
	endpoint := strings.TrimSpace(config.EndPoint)
	accessId := strings.TrimSpace(config.AccessId)
	accessSecret := strings.TrimSpace(config.AccessSecret)
	bucket := strings.TrimSpace(config.BucketName)
	if endpoint == "" || accessId == "" || accessSecret == "" || bucket == "" {
		return nil, fmt.Errorf("s3 endPoint/accessKeyId/accessKeySecret/bucketName 未配置完整")
	}

	// 去掉 scheme，minio 客户端自行按 UseSSL 处理
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimRight(endpoint, "/")

	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(accessId, accessSecret, ""),
		Secure: config.UseSSL,
	}
	if region := strings.TrimSpace(config.Region); region != "" {
		opts.Region = region
	}

	client, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("初始化 S3 客户端失败: %w", err)
	}

	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		scheme := "http"
		if config.UseSSL {
			scheme = "https"
		}
		baseURL = scheme + "://" + endpoint + "/" + bucket
	}

	return &S3{
		client:  client,
		bucket:  bucket,
		baseURL: baseURL,
	}, nil
}

// EnsureReady 确保 bucket 存在且为私有（清空公共读策略）
func (s *S3) EnsureReady(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("检查 S3 bucket 失败: %w", err)
	}
	if !exists {
		if err = s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("创建 S3 bucket %s 失败: %w", s.bucket, err)
		}
	}
	// 私有桶：清除可能存在的公共读策略（忽略失败）
	_ = s.client.SetBucketPolicy(ctx, s.bucket, "")
	return nil
}

// PutObject 上传对象
func (s *S3) PutObject(object string, content []byte) error {
	object = normalizeKey(object)
	_, err := s.client.PutObject(context.Background(), s.bucket, object, bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("S3 上传失败: %w", err)
	}
	return nil
}

// GetObject 获取对象内容
func (s *S3) GetObject(object string) ([]byte, error) {
	object = normalizeKey(object)
	obj, err := s.client.GetObject(context.Background(), s.bucket, object, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("S3 下载失败: %w", err)
	}
	defer obj.Close()
	if _, err = obj.Stat(); err != nil {
		return nil, fmt.Errorf("S3 对象不存在或不可读: %w", err)
	}
	return io.ReadAll(obj)
}

// SignGetURL 生成短期预签名 GET URL
func (s *S3) SignGetURL(object string, expire time.Duration) (string, error) {
	object = normalizeKey(object)
	if expire <= 0 {
		expire = time.Hour
	}
	u, err := s.client.PresignedGetObject(context.Background(), s.bucket, object, expire, nil)
	if err != nil {
		return "", fmt.Errorf("S3 预签名失败: %w", err)
	}
	return u.String(), nil
}

// URL 返回规范访问地址（不含签名）
func (s *S3) URL(object string) string {
	return s.baseURL + "/" + normalizeKey(object)
}

func normalizeKey(key string) string {
	return strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "/")
}
