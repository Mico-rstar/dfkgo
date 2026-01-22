package auth

import (
	"dfkgo/config"

	"github.com/golang-jwt/jwt/v5"
)

var key string

func init() {
	key = config.GetConfig().JwtPriKey
	if len(key) == 0 {
		panic("JWT_PRIVATE_KEY is not set")
	}
}

type JwtMaker struct {
}

func NewJwtMaker() *JwtMaker {
	return &JwtMaker{}
}

func (j *JwtMaker) MakeToken(userId int64) (string, error) {
	
	t := jwt.NewWithClaims()
	return t.SignedString(key)
}


func (j *JwtMaker) VerifyToken(token string) (int64, error) {
	return 0, nil
}