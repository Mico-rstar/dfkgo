package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetConfig(t *testing.T) {
	LoadConfig("..")
	config := GetConfig()
	assert.NotEmpty(t, config.JwtPriKey)
	assert.Equal(t, "bcuwehjfihafjkbjnacc", config.JwtPriKey)
	assert.Equal(t, ":8888", config.ServerPort)
	assert.Equal(t, 24, config.JwtDurationHours)
}
