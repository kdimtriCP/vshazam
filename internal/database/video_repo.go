package database

import (
	"fmt"

	"github.com/kdimtricp/vshazam/internal/models"
	"gorm.io/gorm"
)

type VideoRepository struct {
	db *DB
}

func NewVideoRepository(db *DB) *VideoRepository {
	return &VideoRepository{db: db}
}

func (r *VideoRepository) InsertVideo(video *models.Video) error {
	result := r.db.GORM().Create(video)
	if result.Error != nil {
		return fmt.Errorf("failed to insert video: %w", result.Error)
	}
	return nil
}

func (r *VideoRepository) GetVideoByID(id string) (*models.Video, error) {
	var video models.Video
	result := r.db.GORM().First(&video, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("video not found")
		}
		return nil, fmt.Errorf("failed to get video: %w", result.Error)
	}
	return &video, nil
}

func (r *VideoRepository) ListVideos() ([]models.Video, error) {
	var videos []models.Video
	result := r.db.GORM().Order("upload_time DESC").Find(&videos)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list videos: %w", result.Error)
	}
	return videos, nil
}

func (r *VideoRepository) SearchVideos(query string) ([]models.Video, error) {
	if query == "" {
		return r.ListVideos()
	}

	var videos []models.Video
	searchPattern := "%" + query + "%"

	db := r.db.GORM()
	if r.db.dbType == "postgres" {
		db = db.Where("title ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	} else {
		db = db.Where("LOWER(title) LIKE LOWER(?) OR LOWER(description) LIKE LOWER(?)", searchPattern, searchPattern)
	}

	result := db.Order("upload_time DESC").Limit(20).Find(&videos)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to search videos: %w", result.Error)
	}

	return videos, nil
}
