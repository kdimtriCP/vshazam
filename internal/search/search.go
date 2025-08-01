package search

import (
	"database/sql"
	"fmt"

	"github.com/kdimtricp/vshazam/internal/models"
)

type SearchService struct {
	db *sql.DB
}

func NewSearchService(db *sql.DB) *SearchService {
	return &SearchService{db: db}
}

func (s *SearchService) Search(query string) ([]models.Video, error) {
	if query == "" {
		return []models.Video{}, nil
	}

	sql := `
		SELECT id, title, description, filename, content_type, size, upload_time
		FROM videos
		WHERE search_vector @@ plainto_tsquery('english', $1)
		ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC
		LIMIT 20
	`

	rows, err := s.db.Query(sql, query)
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

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return videos, nil
}
