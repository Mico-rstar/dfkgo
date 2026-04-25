package auth

import (
	"dfkgo/auth"
	"dfkgo/config"
	"dfkgo/errcode"
	"dfkgo/model"
	"dfkgo/repository"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo *repository.UserRepo
	maker    auth.AuthMaker
	config   config.Config
}

func NewAuthService(userRepo *repository.UserRepo, maker auth.AuthMaker, cfg config.Config) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		maker:    maker,
		config:   cfg,
	}
}

func (s *AuthService) Register(email, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return errcode.ErrInternal
	}

	user := &model.User{
		Email:        email,
		PasswordHash: string(hash),
	}
	err = s.userRepo.Create(user)
	if err != nil {
		if isDuplicateKeyError(err) {
			return errcode.ErrUserExists
		}
		return errcode.ErrDBError
	}
	return nil
}

func (s *AuthService) Login(email, password string) (string, int64, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", 0, errcode.ErrUserNotFound
		}
		return "", 0, errcode.ErrDBError
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", 0, errcode.ErrWrongPassword
	}

	duration := time.Duration(s.config.JwtDurationHours) * time.Hour
	token, err := s.maker.MakeToken(user.ID, user.Email, duration)
	if err != nil {
		return "", 0, errcode.ErrInternal
	}

	expiresAt := time.Now().Add(duration).Unix()
	return token, expiresAt, nil
}

func isDuplicateKeyError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "Duplicate entry") ||
		strings.Contains(msg, "duplicate key")
}
