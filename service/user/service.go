package user

import (
	"context"
	"crypto/rand"
	"dfkgo/config"
	"dfkgo/entity"
	"dfkgo/errcode"
	"dfkgo/repository"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"dfkgo/service/oss"

	"gorm.io/gorm"
)

const maxAvatarSize = 10 * 1024 * 1024 // 10MB

var allowedAvatarMimeTypes = map[string]string{
	"image/jpg":  "jpg",
	"image/jpeg": "jpeg",
	"image/png":  "png",
}

type UserService struct {
	userRepo   *repository.UserRepo
	ossService oss.OSSService
	config     config.Config
}

func NewUserService(userRepo *repository.UserRepo, ossService oss.OSSService, cfg config.Config) *UserService {
	return &UserService{
		userRepo:   userRepo,
		ossService: ossService,
		config:     cfg,
	}
}

func (s *UserService) GetProfile(userID uint64) (*entity.ProfileResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrUserNotFound
		}
		return nil, errcode.ErrDBError
	}
	return &entity.ProfileResponse{
		Email:     user.Email,
		Nickname:  user.Nickname,
		AvatarUrl: user.AvatarURL,
		Phone:     user.Phone,
	}, nil
}

func (s *UserService) UpdateProfile(userID uint64, req *entity.UpdateProfileRequest) error {
	updates := make(map[string]any)
	if req.Nickname != nil {
		updates["nickname"] = *req.Nickname
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if len(updates) == 0 {
		return nil
	}
	err := s.userRepo.UpdateProfile(userID, updates)
	if err != nil {
		return errcode.ErrDBError
	}
	return nil
}

func (s *UserService) InitAvatarUpload(userID uint64, mimeType string, fileSize int64) (*entity.AvatarUploadInitResponse, error) {
	ext, ok := allowedAvatarMimeTypes[mimeType]
	if !ok {
		return nil, errcode.ErrInvalidMimeType
	}
	if fileSize > maxAvatarSize {
		return nil, errcode.ErrFileTooLarge
	}

	randomHex := generateRandomHex(8)
	objectKey := fmt.Sprintf("avatars/%d/%s.%s", userID, randomHex, ext)

	creds, err := s.ossService.IssueSTSCredentials(context.Background(), s.config.OSSBucketAvatars, objectKey, s.config.OSSStsDurationSeconds)
	if err != nil {
		return nil, errcode.ErrOSSError
	}

	return &entity.AvatarUploadInitResponse{
		AccessKeyID:      creds.AccessKeyID,
		AccessKeySecret:  creds.AccessKeySecret,
		SecurityToken:    creds.SecurityToken,
		Expiration:       creds.Expiration,
		Bucket:           s.config.OSSBucketAvatars,
		Region:           s.config.OSSRegion,
		Endpoint:         buildEndpoint(s.config.OSSRegion),
		ObjectKey:        objectKey,
		MaxFileSize:      maxAvatarSize,
		AllowedFileTypes: []string{"jpg", "jpeg", "png"},
	}, nil
}

func (s *UserService) AvatarUploadCallback(userID uint64, objectKey string) (string, error) {
	exists, err := s.ossService.HeadObject(context.Background(), s.config.OSSBucketAvatars, objectKey)
	if err != nil {
		return "", errcode.ErrOSSError
	}
	if !exists {
		return "", errcode.ErrAvatarOSSNotFound
	}

	avatarURL := s.ossService.BuildOssURL(s.config.OSSBucketAvatars, objectKey)
	err = s.userRepo.UpdateAvatar(userID, avatarURL)
	if err != nil {
		return "", errcode.ErrDBError
	}
	return avatarURL, nil
}

func (s *UserService) FetchAvatar(userID uint64) (string, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errcode.ErrUserNotFound
		}
		return "", errcode.ErrDBError
	}
	return user.AvatarURL, nil
}

func generateRandomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func buildEndpoint(region string) string {
	return fmt.Sprintf("oss-%s.aliyuncs.com", region)
}

func extFromFileName(fileName string) string {
	ext := filepath.Ext(fileName)
	return strings.TrimPrefix(ext, ".")
}
