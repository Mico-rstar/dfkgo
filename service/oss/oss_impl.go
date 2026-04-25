package oss

import (
	"context"
	"dfkgo/config"
	"fmt"

	ossv2 "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

type ossServiceImpl struct {
	client *ossv2.Client
	cfg    config.Config
}

func NewOSSService(cfg config.Config) (OSSService, error) {
	provider := credentials.NewStaticCredentialsProvider(
		cfg.OSSAccessKeyID,
		cfg.OSSAccessKeySecret,
	)
	ossCfg := ossv2.LoadDefaultConfig().
		WithCredentialsProvider(provider).
		WithRegion(cfg.OSSRegion).
		WithEndpoint(cfg.OSSEndpoint)

	client := ossv2.NewClient(ossCfg)
	return &ossServiceImpl{client: client, cfg: cfg}, nil
}

func (s *ossServiceImpl) IssueSTSCredentials(_ context.Context, _, _ string, _ int) (*STSCredentials, error) {
	// STS credential issuance requires Alibaba Cloud STS SDK (AssumeRole).
	// This is a placeholder; real implementation would call STS AssumeRole API.
	return nil, fmt.Errorf("STS credential issuance not yet implemented")
}

func (s *ossServiceImpl) HeadObject(ctx context.Context, bucket, objectKey string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &ossv2.HeadObjectRequest{
		Bucket: &bucket,
		Key:    &objectKey,
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (s *ossServiceImpl) BuildOssURL(bucket, objectKey string) string {
	return fmt.Sprintf("https://%s.%s/%s", bucket, s.cfg.OSSEndpoint, objectKey)
}
