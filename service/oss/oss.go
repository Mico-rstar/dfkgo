package oss

import "context"

type STSCredentials struct {
	AccessKeyID     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	SecurityToken   string `json:"securityToken"`
	Expiration      string `json:"expiration"`
}

type OSSService interface {
	IssueSTSCredentials(ctx context.Context, bucket, objectKeyPrefix string, durationSec int) (*STSCredentials, error)
	HeadObject(ctx context.Context, bucket, objectKey string) (bool, error)
	BuildOssURL(bucket, objectKey string) string
}
