package file

import (
	"context"
	"dfkgo/config"
	"dfkgo/entity"
	"dfkgo/errcode"
	"dfkgo/model"
	"dfkgo/repository"
	"dfkgo/service/oss"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxFileSize = 500 * 1024 * 1024 // 500MB

var allowedFileMimeTypes = map[string]string{
	"video/mp4":       "video",
	"video/quicktime": "video",
	"video/x-msvideo": "video",
	"audio/mpeg":      "audio",
	"audio/wav":       "audio",
	"image/jpeg":      "image",
	"image/png":       "image",
}

type FileService struct {
	fileRepo   *repository.FileRepo
	ossService oss.OSSService
	config     config.Config
}

func NewFileService(fileRepo *repository.FileRepo, ossService oss.OSSService, cfg config.Config) *FileService {
	return &FileService{
		fileRepo:   fileRepo,
		ossService: ossService,
		config:     cfg,
	}
}

func (s *FileService) InitUpload(userID uint64, req *entity.UploadInitRequest) (*entity.UploadInitResponse, error) {
	modality, ok := allowedFileMimeTypes[req.MimeType]
	if !ok {
		return nil, errcode.ErrFileTypeNotSupported
	}
	if req.FileSize > maxFileSize {
		return nil, errcode.ErrFileSizeExceeded
	}

	// 查秒传
	existing, err := s.fileRepo.FindByUserAndMD5(userID, req.MD5)
	if err == nil && existing.UploadStatus == "completed" {
		return &entity.UploadInitResponse{
			FileId: existing.FileUID,
			Hit:    true,
			OssUrl: existing.OSSURL,
		}, nil
	}

	// 未命中，创建新记录
	fileUID := "file_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	ext := extFromFileName(req.FileName)
	objectKey := fmt.Sprintf("files/%d/%s.%s", userID, fileUID, ext)
	ossURL := s.ossService.BuildOssURL(s.config.OSSBucketFiles, objectKey)

	file := &model.File{
		FileUID:      fileUID,
		UserID:       userID,
		FileName:     req.FileName,
		MimeType:     req.MimeType,
		FileSize:     req.FileSize,
		Modality:     modality,
		MD5:          req.MD5,
		OSSBucket:    s.config.OSSBucketFiles,
		OSSObjectKey: objectKey,
		OSSURL:       ossURL,
		UploadStatus: "pending",
	}
	if err := s.fileRepo.Create(file); err != nil {
		return nil, errcode.ErrDBError
	}

	creds, err := s.ossService.IssueSTSCredentials(context.Background(), s.config.OSSBucketFiles, objectKey, s.config.OSSStsDurationSeconds)
	if err != nil {
		return nil, errcode.ErrOSSError
	}

	return &entity.UploadInitResponse{
		FileId: fileUID,
		Hit:    false,
		STS: &entity.STSInfo{
			AccessKeyID:     creds.AccessKeyID,
			AccessKeySecret: creds.AccessKeySecret,
			SecurityToken:   creds.SecurityToken,
			Expiration:      creds.Expiration,
			Bucket:          s.config.OSSBucketFiles,
			Region:          s.config.OSSRegion,
			Endpoint:        fmt.Sprintf("oss-%s.aliyuncs.com", s.config.OSSRegion),
			ObjectKey:       objectKey,
		},
	}, nil
}

func (s *FileService) UploadCallback(userID uint64, fileId string) (*entity.UploadCallbackResponse, error) {
	file, err := s.fileRepo.FindByFileUID(fileId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrFileIDNotFound
		}
		return nil, errcode.ErrDBError
	}

	if file.UserID != userID {
		return nil, errcode.ErrFileIDNotFound
	}

	exists, err := s.ossService.HeadObject(context.Background(), file.OSSBucket, file.OSSObjectKey)
	if err != nil {
		return nil, errcode.ErrOSSError
	}
	if !exists {
		return nil, errcode.ErrFileOSSNotFound
	}

	if err := s.fileRepo.UpdateUploadStatus(fileId, "completed"); err != nil {
		return nil, errcode.ErrDBError
	}

	return &entity.UploadCallbackResponse{
		FileId: file.FileUID,
		OssUrl: file.OSSURL,
	}, nil
}

func extFromFileName(fileName string) string {
	ext := filepath.Ext(fileName)
	return strings.TrimPrefix(ext, ".")
}
