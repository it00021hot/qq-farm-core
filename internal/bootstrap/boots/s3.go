package boots

import (
	"context"
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/vars"
	s3store "github.com/MQEnergy/go-skeleton/pkg/s3"
)

// InitS3 当 uploadType=2 时初始化并确保私有桶就绪
func InitS3() error {
	if vars.Config.GetInt("server.uploadType") != 2 {
		return nil
	}
	client, err := s3store.New(&s3store.Config{
		EndPoint:     vars.Config.GetString("s3.endPoint"),
		AccessId:     vars.Config.GetString("s3.accessKeyId"),
		AccessSecret: vars.Config.GetString("s3.accessKeySecret"),
		BucketName:   vars.Config.GetString("s3.bucketName"),
		UseSSL:       vars.Config.GetBool("s3.useSSL"),
		BaseURL:      vars.Config.GetString("s3.baseUrl"),
		Region:       vars.Config.GetString("s3.region"),
	})
	if err != nil {
		return fmt.Errorf("init s3: %w", err)
	}
	if err := client.EnsureReady(context.Background()); err != nil {
		return fmt.Errorf("ensure s3 ready: %w", err)
	}
	return nil
}
