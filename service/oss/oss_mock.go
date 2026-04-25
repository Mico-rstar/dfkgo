package oss

import (
	"context"
	"fmt"
	"time"
)

type MockOSSService struct{}

func NewMockOSSService() OSSService {
	return &MockOSSService{}
}

func (m *MockOSSService) IssueSTSCredentials(_ context.Context, bucket, objectKeyPrefix string, durationSec int) (*STSCredentials, error) {
	return &STSCredentials{
		AccessKeyID:     "mock-ak-id",
		AccessKeySecret: "mock-ak-secret",
		SecurityToken:   "mock-security-token",
		Expiration:      time.Now().Add(time.Duration(durationSec) * time.Second).UTC().Format(time.RFC3339),
	}, nil
}

func (m *MockOSSService) HeadObject(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

func (m *MockOSSService) BuildOssURL(bucket, objectKey string) string {
	return fmt.Sprintf("https://%s.oss-cn-hangzhou.aliyuncs.com/%s", bucket, objectKey)
}
