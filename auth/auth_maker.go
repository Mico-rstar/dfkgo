package auth

import "time"

type AuthMaker interface {
	MakeToken(userID uint64, username string, duration time.Duration) (string, error)
	VerifyToken(tokenString string) (*Payload, error)
}
