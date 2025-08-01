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
	var query string
	switch r.db.dbType {
	case "postgres":
		query = `
			INSERT INTO videos (id, title, description, filename, content_type, size, upload_time)
			VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
		`
	default:
		query = `
			INSERT INTO videos (id, title, description, filename, content_type, size, upload_time)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`
	}

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
	var query string
	switch r.db.dbType {
	case "postgres":
		query = `
			SELECT id::text, title, description, filename, content_type, size, upload_time
			FROM videos
			WHERE id = $1::uuid
		`
	default:
		query = `
			SELECT id, title, description, filename, content_type, size, upload_time
			FROM videos
			WHERE id = ?
		`
	}

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
	var query string
	if r.db.dbType == "postgres" {
		query = `
			SELECT id::text, title, description, filename, content_type, size, upload_time
			FROM videos
			ORDER BY upload_time DESC
		`
	} else {
		query = `
			SELECT id, title, description, filename, content_type, size, upload_time
			FROM videos
			ORDER BY upload_time DESC
		`
	}

	rows, err := r.db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list videos: %w", err)
	}
	defer rows.Close()

	var videos []models.Video
	count := 0
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
		count++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return videos, nil
}

func (r *VideoRepository) SearchVideos(query string) ([]models.Video, error) {
	if query == "" {
		// Return all videos when query is empty
		return r.ListVideos()
	}

	// For SQLite, use LIKE query with case-insensitive search
	if r.db.dbType != "postgres" {
		sqlQuery := `
			SELECT id, title, description, filename, content_type, size, upload_time
			FROM videos
			WHERE LOWER(title) LIKE LOWER(?) OR LOWER(description) LIKE LOWER(?)
			ORDER BY 
				CASE 
					WHEN LOWER(title) = LOWER(?) THEN 1
					WHEN LOWER(title) LIKE LOWER(?) THEN 2
					WHEN LOWER(description) LIKE LOWER(?) THEN 3
					ELSE 4
				END,
				upload_time DESC
			LIMIT 20
		`
		searchPattern := "%" + query + "%"
		rows, err := r.db.conn.Query(sqlQuery, searchPattern, searchPattern, query, searchPattern, searchPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to search videos: %w", err)
		}
		defer rows.Close()

		var videos []models.Video
		for rows.Next() {
			var v models.Video
			err := rows.Scan(&v.ID, &v.Title, &v.Description, &v.Filename, &v.ContentType, &v.Size, &v.UploadTime)
			if err != nil {
				return nil, fmt.Errorf("failed to scan video: %w", err)
			}
			videos = append(videos, v)
		}
		return videos, nil
	}

	// PostgreSQL search with partial matching
	// Use ILIKE for partial matches and combine with full-text search
	sqlQuery := `
		SELECT id::text, title, description, filename, content_type, size, upload_time,
			CASE 
				WHEN title ILIKE $1 THEN 1  -- Exact match in title
				WHEN title ILIKE '%' || $1 || '%' THEN 2  -- Partial match in title
				WHEN description ILIKE '%' || $1 || '%' THEN 3  -- Partial match in description
				ELSE 4  -- Full-text search match
			END as rank
		FROM videos
		WHERE 
			title ILIKE '%' || $1 || '%'
			OR description ILIKE '%' || $1 || '%'
			OR search_vector @@ plainto_tsquery('english', $1)
		ORDER BY rank, upload_time DESC
		LIMIT 20
	`

	rows, err := r.db.conn.Query(sqlQuery, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search videos: %w", err)
	}
	defer rows.Close()

	var videos []models.Video
	for rows.Next() {
		var v models.Video
		var rank int
		err := rows.Scan(&v.ID, &v.Title, &v.Description, &v.Filename, &v.ContentType, &v.Size, &v.UploadTime, &rank)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}
		videos = append(videos, v)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return videos, nil
}
