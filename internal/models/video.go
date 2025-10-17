package models

import (
	"time"

	"github.com/google/uuid"
)

type Video struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Title       string    `gorm:"not null"`
	Description string    `gorm:"type:text"`
	Filename    string    `gorm:"not null"`
	ContentType string    `gorm:"not null"`
	Size        int64     `gorm:"not null"`
	UploadTime  time.Time `gorm:"not null;index"`
}

func (Video) TableName() string {
	return "videos"
}

func NewVideo(title, description, filename, contentType string, size int64) *Video {
	return &Video{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
		Filename:    filename,
		ContentType: contentType,
		Size:        size,
		UploadTime:  time.Now(),
	}
}
