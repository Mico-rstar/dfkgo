package model

import "time"

type File struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement"`
	FileUID      string    `gorm:"type:varchar(40);uniqueIndex:uk_file_uid;not null"`
	UserID       uint64    `gorm:"not null;uniqueIndex:uk_user_md5,priority:1;index:idx_user"`
	FileName     string    `gorm:"type:varchar(255);not null"`
	MimeType     string    `gorm:"type:varchar(64);not null"`
	FileSize     int64     `gorm:"not null"`
	Modality     string    `gorm:"type:varchar(16);not null"`
	MD5          string    `gorm:"type:char(32);not null;uniqueIndex:uk_user_md5,priority:2"`
	OSSBucket    string    `gorm:"type:varchar(64);not null"`
	OSSObjectKey string    `gorm:"type:varchar(512);not null"`
	OSSURL       string    `gorm:"type:varchar(1024);not null"`
	UploadStatus string    `gorm:"type:varchar(16);not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (File) TableName() string { return "files" }
