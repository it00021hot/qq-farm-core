package oss

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type Oss struct {
	client *oss.Client
	bucket *oss.Bucket
}

type Config struct {
	EndPoint     string
	AccessId     string
	AccessSecret string
	BucketName   string
}

func New(config *Config, options ...oss.ClientOption) (*Oss, error) {
	client, err := oss.New(config.EndPoint, config.AccessId, config.AccessSecret, options...)
	if err != nil {
		return nil, err
	}
	bucket, err := client.Bucket(config.BucketName)
	if err != nil {
		return nil, err
	}
	return &Oss{
		client: client,
		bucket: bucket,
	}, nil
}

// PutObject 上传字符串
func (o *Oss) PutObject(object string, content []byte) error {
	return o.bucket.PutObject(object, bytes.NewReader(content))
}

// PutObjectFromFile 上传文件
func (o *Oss) PutObjectFromFile(object, localPath string) error {
	return o.bucket.PutObjectFromFile(object, localPath)
}

// GetObject 获取文件
func (o *Oss) GetObject(object string) ([]byte, error) {
	body, err := o.bucket.GetObject(object)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return ioutil.ReadAll(body)
}

// SignGetURL 生成短期预签名 GET URL
func (o *Oss) SignGetURL(object string, expire time.Duration) (string, error) {
	if expire <= 0 {
		expire = time.Hour
	}
	signed, err := o.bucket.SignURL(object, oss.HTTPGet, int64(expire/time.Second))
	if err != nil {
		return "", fmt.Errorf("OSS 预签名失败: %w", err)
	}
	return signed, nil
}
