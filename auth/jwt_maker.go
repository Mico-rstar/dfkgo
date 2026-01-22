package auth

import (
	"dfkgo/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var key []byte

func init() {
	key = []byte(config.GetConfig().JwtPriKey)
	if len(key) == 0 {
		panic("JWT_PRIVATE_KEY is not set")
	}
}

type JwtMaker struct {
}

func NewJwtMaker() *JwtMaker {
	return &JwtMaker{}
}

func (j *JwtMaker) MakeToken(username string, duration time.Duration) (string, error) {
	if username == "" {
		return "", nil
	}
	payload := NewPayload(username, duration)
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	return t.SignedString(key)
}


// Verification failed case: expired or invalid token
func (j *JwtMaker) VerifyToken(tokenString string) (*Payload, error) {
	// verify the signature
	token, err :=jwt.ParseWithClaims(tokenString, &Payload{}, func(t *jwt.Token) (any, error) {
		return key, nil
	})
	if err != nil {
		return &Payload{}, err
	}

	payload, ok := token.Claims.(*Payload)
	if !ok || !token.Valid {
		return &Payload{}, jwt.ErrTokenInvalidClaims
	}
	// verify the payload
	err = payload.Valid()
	if err != nil {
		return &Payload{}, err
	}
	return payload, nil
}