package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func newTestMaker() *JwtMaker {
	return NewJwtMakerWithKey("test-secret-key-for-unit-tests")
}

func TestJwtMakeToken(t *testing.T) {
	maker := newTestMaker()
	token, err := maker.MakeToken(1, uuid.NewString(), time.Hour)
	assert.Nil(t, err)
	assert.NotEmpty(t, token)

	token, err = maker.MakeToken(0, "", time.Hour)
	assert.Nil(t, err)
	assert.Empty(t, token)
}

func TestJwtVerifyToken(t *testing.T) {
	maker := newTestMaker()
	username := uuid.NewString()
	var userID uint64 = 42

	// valid case
	token, err := maker.MakeToken(userID, username, time.Hour)
	assert.Nil(t, err)

	payload, err := maker.VerifyToken(token)
	assert.Nil(t, err)
	assert.Equal(t, username, payload.Username)
	assert.Equal(t, userID, payload.UserID)
	assert.Equal(t, time.Hour, payload.ExpiredAt.Time.Sub(payload.IssuedAt.Time))

	// invalid case: token expired
	token, err = maker.MakeToken(userID, username, time.Microsecond)
	assert.Nil(t, err)
	time.Sleep(time.Second)

	_, err = maker.VerifyToken(token)
	assert.NotNil(t, err)

	// invalid case: invalid token
	_, err = maker.VerifyToken("invalid token")
	assert.NotNil(t, err)

	// invalid case: empty token
	_, err = maker.VerifyToken("")
	assert.NotNil(t, err)
}
