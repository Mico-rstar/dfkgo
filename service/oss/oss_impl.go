package oss

import (
	"context"
	"dfkgo/config"
	"fmt"
	"time"

	ossv2 "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	osscreds "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/alibabacloud-go/tea/tea"
	credentials "github.com/aliyun/credentials-go/credentials"
)

type ossServiceImpl struct {
	client *ossv2.Client
	cfg    config.Config
}

func NewOSSService(cfg config.Config) (OSSService, error) {
	provider := osscreds.NewStaticCredentialsProvider(
		cfg.OSSAccessKeyID,
		cfg.OSSAccessKeySecret,
	)

	endpoint := cfg.OSSEndpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("oss-%s.aliyuncs.com", cfg.OSSRegion)
	}

	ossCfg := ossv2.LoadDefaultConfig().
		WithCredentialsProvider(provider).
		WithRegion(cfg.OSSRegion).
		WithEndpoint(endpoint)

	client := ossv2.NewClient(ossCfg)
	return &ossServiceImpl{client: client, cfg: cfg}, nil
}

// IssueSTSCredentials 通过 AssumeRole 签发限定 objectKeyPrefix 的临时凭证
func (s *ossServiceImpl) IssueSTSCredentials(_ context.Context, bucket, objectKeyPrefix string, durationSec int) (*STSCredentials, error) {
	if durationSec <= 0 {
		durationSec = 900
	}

	policy := buildSTSPolicy(bucket, objectKeyPrefix)

	credConfig := &credentials.Config{
		Type:                  tea.String("ram_role_arn"),
		AccessKeyId:           tea.String(s.cfg.OSSAccessKeyID),
		AccessKeySecret:       tea.String(s.cfg.OSSAccessKeySecret),
		RoleArn:               tea.String(s.cfg.OSSStsRoleArn),
		RoleSessionName:       tea.String("dfkgo-upload"),
		RoleSessionExpiration: tea.Int(durationSec),
		Policy:                tea.String(policy),
	}

	cred, err := credentials.NewCredential(credConfig)
	if err != nil {
		return nil, fmt.Errorf("create STS credential: %w", err)
	}

	model, err := cred.GetCredential()
	if err != nil {
		return nil, fmt.Errorf("assume role failed: %w", err)
	}

	expiration := time.Now().Add(time.Duration(durationSec) * time.Second).UTC().Format("2006-01-02 15:04:05")

	return &STSCredentials{
		AccessKeyID:     tea.StringValue(model.AccessKeyId),
		AccessKeySecret: tea.StringValue(model.AccessKeySecret),
		SecurityToken:   tea.StringValue(model.SecurityToken),
		Expiration:      expiration,
	}, nil
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
	endpoint := s.cfg.OSSEndpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("oss-%s.aliyuncs.com", s.cfg.OSSRegion)
	}
	return fmt.Sprintf("https://%s.%s/%s", bucket, endpoint, objectKey)
}

// buildSTSPolicy 生成限定 bucket + objectKeyPrefix 的最小权限 STS Policy
func buildSTSPolicy(bucket, objectKeyPrefix string) string {
	return fmt.Sprintf(`{
  "Version": "1",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["oss:PutObject"],
      "Resource": ["acs:oss:*:*:%s/%s*"]
    }
  ]
}`, bucket, objectKeyPrefix)
}
