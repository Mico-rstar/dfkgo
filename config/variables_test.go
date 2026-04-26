package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetConfig(t *testing.T) {
	LoadConfig("..")
	config := GetConfig()
	assert.NotEmpty(t, config.JwtPriKey)
	// 不硬编码具体密钥值，只校验加载成功
	assert.Equal(t, ":8888", config.ServerPort)
	assert.Equal(t, 24, config.JwtDurationHours)
}
