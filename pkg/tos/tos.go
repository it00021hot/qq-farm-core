package tos

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
)

type Tos struct {
	ctx    context.Context
	client *tos.ClientV2
	config *Config
}

type Config struct {
	EndPoint     string
	AccessId     string
	AccessSecret string
	BucketName   string
	Region       string
}

// New 初始化
func New(config *Config, options ...tos.ClientOption) (*Tos, error) {
	credential := tos.NewStaticCredentials(config.AccessId, config.AccessSecret)
	options = append(options, tos.WithCredentials(credential))
	options = append(options, tos.WithRegion(config.Region))
	client, err := tos.NewClientV2(config.EndPoint, options...)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	return &Tos{
		ctx:    context.Background(),
		client: client,
		config: config,
	}, nil
}

// PutObject 上传字符串
func (o *Tos) PutObject(object string, content []byte) error {
	// 上传对象
	_, err := o.client.PutObjectV2(o.ctx, &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket: o.config.BucketName,
			Key:    object,
		},
		Content: bytes.NewReader(content),
	})
	if err != nil {
		return err
	}
	return nil
}

// PutObjectFromStream 上传网络流
func (o *Tos) PutObjectFromStream(object, url string) error {
	// 从网络流中获取数据
	res, _ := http.Get(url)
	defer res.Body.Close()
	_, err := o.client.PutObjectV2(o.ctx, &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket: o.config.BucketName,
			Key:    object,
		},
		Content: res.Body,
	})
	if err != nil {
		return err
	}
	return nil
}

// PutObjectFromFileStream 上传本地文件流
func (o *Tos) PutObjectFromFileStream(object, localPath string) error {
	// 读取本地文件数据
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = o.client.PutObjectV2(o.ctx, &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket: o.config.BucketName,
			Key:    object,
		},
		Content: f,
	})
	if err != nil {
		return err
	}
	return nil
}

// PutObjectFromFile 上传本地文件
func (o *Tos) PutObjectFromFile(object, localPath string) error {
	// 直接使用文件路径上传文件
	_, err := o.client.PutObjectFromFile(o.ctx, &tos.PutObjectFromFileInput{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket: o.config.BucketName,
			Key:    object,
		},
		FilePath: localPath,
	})
	if err != nil {
		return err
	}
	return nil
}

// GetObject 获取对象
func (o *Tos) GetObject(object string) ([]byte, error) {
	output, err := o.client.GetObjectV2(o.ctx, &tos.GetObjectV2Input{
		Bucket: o.config.BucketName,
		Key:    object,
	})
	if err != nil {
		return nil, err
	}
	defer output.Content.Close()
	return ioutil.ReadAll(output.Content)
}
