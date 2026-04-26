package repository

import (
	"dfkgo/model"

	"gorm.io/gorm"
)

type FileRepo struct {
	db *gorm.DB
}

func NewFileRepo(db *gorm.DB) *FileRepo {
	return &FileRepo{db: db}
}

func (r *FileRepo) Create(file *model.File) error {
	return r.db.Create(file).Error
}

func (r *FileRepo) FindByFileUID(fileUID string) (*model.File, error) {
	var file model.File
	err := r.db.Where("file_uid = ?", fileUID).First(&file).Error
	return &file, err
}

func (r *FileRepo) FindByUserAndMD5(userID uint64, md5 string) (*model.File, error) {
	var file model.File
	err := r.db.Where("user_id = ? AND md5 = ?", userID, md5).First(&file).Error
	return &file, err
}

func (r *FileRepo) UpdateUploadStatus(fileUID string, status string) error {
	return r.db.Model(&model.File{}).Where("file_uid = ?", fileUID).Update("upload_status", status).Error
}

func (r *FileRepo) FindByID(id uint64) (*model.File, error) {
	var file model.File
	err := r.db.First(&file, id).Error
	return &file, err
}

func (r *FileRepo) FindByIDs(ids []uint64) ([]model.File, error) {
	var files []model.File
	if len(ids) == 0 {
		return files, nil
	}
	err := r.db.Where("id IN ?", ids).Find(&files).Error
	return files, err
}
