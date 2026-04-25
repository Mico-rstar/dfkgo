package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JwtMaker struct {
	key []byte
}

func NewJwtMaker() *JwtMaker {
	return &JwtMaker{}
}

func NewJwtMakerWithKey(key string) *JwtMaker {
	return &JwtMaker{key: []byte(key)}
}

func (j *JwtMaker) getKey() []byte {
	if len(j.key) > 0 {
		return j.key
	}
	// lazy load from config
	cfg := loadConfigKey()
	j.key = []byte(cfg)
	return j.key
}

func (j *JwtMaker) MakeToken(userID uint64, username string, duration time.Duration) (string, error) {
	if username == "" {
		return "", nil
	}
	payload := NewPayload(userID, username, duration)
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	return t.SignedString(j.getKey())
}

func (j *JwtMaker) VerifyToken(tokenString string) (*Payload, error) {
	key := j.getKey()
	token, err := jwt.ParseWithClaims(tokenString, &Payload{}, func(t *jwt.Token) (any, error) {
		return key, nil
	})
	if err != nil {
		return &Payload{}, err
	}

	payload, ok := token.Claims.(*Payload)
	if !ok || !token.Valid {
		return &Payload{}, jwt.ErrTokenInvalidClaims
	}
	err = payload.Valid()
	if err != nil {
		return &Payload{}, err
	}
	return payload, nil
}

func loadConfigKey() string {
	// Import config lazily to avoid circular init
	// This will be called only when key is not set via constructor
	return getJwtKeyFromConfig()
}
