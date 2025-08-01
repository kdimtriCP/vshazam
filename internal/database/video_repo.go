package database

import (
	"database/sql"
	"fmt"

	"github.com/kdimtricp/vshazam/internal/models"
)

type VideoRepository struct {
	db *DB
}

func NewVideoRepository(db *DB) *VideoRepository {
	return &VideoRepository{db: db}
}

func (r *VideoRepository) InsertVideo(video *models.Video) error {
	query := `
		INSERT INTO videos (id, title, description, filename, content_type, size, upload_time)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.conn.Exec(query,
		video.ID,
		video.Title,
		video.Description,
		video.Filename,
		video.ContentType,
		video.Size,
		video.UploadTime,
	)

	if err != nil {
		return fmt.Errorf("failed to insert video: %w", err)
	}

	return nil
}

func (r *VideoRepository) GetVideoByID(id string) (*models.Video, error) {
	query := `
		SELECT id, title, description, filename, content_type, size, upload_time
		FROM videos
		WHERE id = ?
	`

	video := &models.Video{}
	err := r.db.conn.QueryRow(query, id).Scan(
		&video.ID,
		&video.Title,
		&video.Description,
		&video.Filename,
		&video.ContentType,
		&video.Size,
		&video.UploadTime,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("video not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	return video, nil
}

func (r *VideoRepository) ListVideos() ([]models.Video, error) {
	query := `
		SELECT id, title, description, filename, content_type, size, upload_time
		FROM videos
		ORDER BY upload_time DESC
	`

	rows, err := r.db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list videos: %w", err)
	}
	defer rows.Close()

	var videos []models.Video
	for rows.Next() {
		var video models.Video
		if err := rows.Scan(
			&video.ID,
			&video.Title,
			&video.Description,
			&video.Filename,
			&video.ContentType,
			&video.Size,
			&video.UploadTime,
		); err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}
		videos = append(videos, video)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return videos, nil
}
