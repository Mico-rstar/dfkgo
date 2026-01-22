package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)


func TestGetConfig(t *testing.T) {
	config := GetConfig()
	// assert.Equal(t, config.DBDriver, "postgres")
	// assert.Equal(t, config.DBSource, "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable")
	// assert.Equal(t, config.ServerPort, "8080")
	assert.NotNil(t, config.JwtPriKey)
	assert.Equal(t, "bcuwehjfihafjkbjnacc", config.JwtPriKey)
}