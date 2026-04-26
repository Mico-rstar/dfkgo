package model

import (
	"database/sql"
	"time"
)

type Task struct {
	ID           uint64  `gorm:"primaryKey;autoIncrement"`
	TaskUID      string  `gorm:"type:varchar(40);uniqueIndex:uk_task_uid;not null"`
	UserID       uint64  `gorm:"not null;index:idx_user_history,priority:1"`
	FileID       uint64  `gorm:"not null;index:idx_file"`
	Modality     string  `gorm:"type:varchar(16);not null"`
	Status       string  `gorm:"type:varchar(16);not null"`
	ResultJSON   *string `gorm:"type:json"`
	ErrorMessage string  `gorm:"type:varchar(1024);not null;default:''"`
	CreatedAt    time.Time
	StartedAt    sql.NullTime
	CompletedAt  sql.NullTime
	DeletedAt    *time.Time `gorm:"index:idx_user_history,priority:2"`
}
