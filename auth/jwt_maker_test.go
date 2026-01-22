package auth

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestJwtMakeToken(t *testing.T) {
	maker := NewJwtMaker()
	token, err :=maker.MakeToken(uuid.NewString(), time.Hour)
	assert.Nil(t, err)
	fmt.Println(token)

	token, err = maker.MakeToken("", time.Hour)
	assert.Nil(t, err)
	fmt.Println(token)
}

func TestJwtVerifyToken(t *testing.T) {
	// valid case
	maker := NewJwtMaker()
	username := uuid.NewString()
	token, err :=maker.MakeToken(username, time.Hour)

	payload, err := maker.VerifyToken(token)
	assert.Nil(t, err)
	assert.Equal(t, username, payload.Username)
	assert.Equal(t, time.Hour, payload.ExpiredAt.Time.Sub(payload.IssuedAt.Time))

	// invalid case
	// case 1: token expired
	token, err = maker.MakeToken(username, time.Microsecond)
	time.Sleep(time.Second)
	
	payload, err = maker.VerifyToken(token)
	assert.NotNil(t, err)

	// case 2: invalid token
	token = "invalid token"
	payload, err = maker.VerifyToken(token)
	assert.NotNil(t, err)

	// case 3: empty token
	payload, err = maker.VerifyToken("")
	assert.NotNil(t, err)
}