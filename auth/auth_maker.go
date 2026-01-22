package auth

import "time"


type AuthMaker interface {
	MakeToken(username string, duration time.Duration) (string, error)
	VerifyToken(tokenString string) (*Payload, error)
}

