package model

import "time"

type User struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement"`
	Email        string     `gorm:"type:varchar(128);uniqueIndex:uk_email;not null"`
	PasswordHash string     `gorm:"type:varchar(72);not null"`
	Nickname     string     `gorm:"type:varchar(64);not null;default:''"`
	AvatarURL    string     `gorm:"type:varchar(512);not null;default:''"`
	Phone        string     `gorm:"type:varchar(32);not null;default:''"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time `gorm:"index"`
}
