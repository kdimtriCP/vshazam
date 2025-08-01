package models

import (
	"time"

	"github.com/google/uuid"
)

type Video struct {
	ID          string
	Title       string
	Description string
	Filename    string
	ContentType string
	Size        int64
	UploadTime  time.Time
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
