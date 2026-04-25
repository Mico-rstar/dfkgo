package auth

import (
	"dfkgo/auth"
	"dfkgo/config"
	"dfkgo/errcode"
	"dfkgo/repository"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthService(t *testing.T) *AuthService {
	t.Helper()
	db, err := repository.InitTestDB()
	require.NoError(t, err)
	userRepo := repository.NewUserRepo(db)
	maker := auth.NewJwtMakerWithKey("test-secret-key")
	cfg := config.Config{JwtDurationHours: 24}
	return NewAuthService(userRepo, maker, cfg)
}

func TestRegister_Success(t *testing.T) {
	svc := setupAuthService(t)
	err := svc.Register("test@example.com", "password123")
	assert.NoError(t, err)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := setupAuthService(t)
	err := svc.Register("dup@example.com", "password123")
	require.NoError(t, err)

	err = svc.Register("dup@example.com", "password456")
	assert.ErrorIs(t, err, errcode.ErrUserExists)
}

func TestLogin_Success(t *testing.T) {
	svc := setupAuthService(t)
	err := svc.Register("login@example.com", "password123")
	require.NoError(t, err)

	token, expiresAt, err := svc.Login("login@example.com", "password123")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Greater(t, expiresAt, int64(0))
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := setupAuthService(t)
	err := svc.Register("wrong@example.com", "password123")
	require.NoError(t, err)

	_, _, err = svc.Login("wrong@example.com", "wrongpassword")
	assert.ErrorIs(t, err, errcode.ErrWrongPassword)
}

func TestLogin_UserNotFound(t *testing.T) {
	svc := setupAuthService(t)
	_, _, err := svc.Login("noone@example.com", "password123")
	assert.ErrorIs(t, err, errcode.ErrUserNotFound)
}
