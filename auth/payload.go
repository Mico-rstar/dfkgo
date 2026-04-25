package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Payload struct {
	ID       uuid.UUID       `json:"id"`
	UserID   uint64          `json:"user_id"`
	Username string          `json:"username"`
	IssuedAt  jwt.NumericDate `json:"issued_at"`
	ExpiredAt jwt.NumericDate `json:"expired_at"`
}

func NewPayload(userID uint64, username string, duration time.Duration) *Payload {
	return &Payload{
		ID:        uuid.New(),
		UserID:    userID,
		Username:  username,
		IssuedAt:  *jwt.NewNumericDate(time.Now()),
		ExpiredAt: *jwt.NewNumericDate(time.Now().Add(duration)),
	}
}

func (p *Payload) Valid() error {
	if p.IssuedAt.Time.After(p.ExpiredAt.Time) {
		return jwt.ErrTokenExpired
	}
	if time.Now().After(p.ExpiredAt.Time) {
		return jwt.ErrTokenExpired
	}
	return nil
}

func (p *Payload) GetExpirationTime() (*jwt.NumericDate, error) {
	return &p.ExpiredAt, nil
}

func (p *Payload) GetIssuedAt() (*jwt.NumericDate, error) {
	return &p.IssuedAt, nil
}

func (p *Payload) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil
}

func (p *Payload) GetIssuer() (string, error) {
	return "", nil
}

func (p *Payload) GetSubject() (string, error) {
	return "", nil
}

func (p *Payload) GetAudience() (jwt.ClaimStrings, error) {
	return nil, nil
}
