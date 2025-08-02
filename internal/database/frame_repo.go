package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/kdimtricp/vshazam/internal/ai"
)

type FrameAnalysisRepo struct {
	db *DB
}

func NewFrameAnalysisRepo(db *DB) *FrameAnalysisRepo {
	return &FrameAnalysisRepo{db: db}
}

func (r *FrameAnalysisRepo) Create(ctx context.Context, analysis *ai.FrameAnalysisDB) error {
	id := uuid.New().String()
	analysis.ID = id

	if r.db.dbType == "postgres" {
		query := `
			INSERT INTO frame_analyses (
				id, video_id, frame_number, gpt_caption, vision_labels, 
				ocr_text, face_count, analysis_time, raw_response
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (video_id, frame_number) 
			DO UPDATE SET 
				gpt_caption = EXCLUDED.gpt_caption,
				vision_labels = EXCLUDED.vision_labels,
				ocr_text = EXCLUDED.ocr_text,
				face_count = EXCLUDED.face_count,
				analysis_time = EXCLUDED.analysis_time,
				raw_response = EXCLUDED.raw_response`

		// Ensure OCRText is not nil for PostgreSQL
		if analysis.OCRText == nil {
			analysis.OCRText = []string{}
		}
		
		ocrTextJSON, err := json.Marshal(analysis.OCRText)
		if err != nil {
			return fmt.Errorf("failed to marshal OCR text: %w", err)
		}

		_, err = r.db.conn.ExecContext(ctx, query,
			analysis.ID,
			analysis.VideoID,
			analysis.FrameNumber,
			analysis.GPTCaption,
			analysis.VisionLabels,
			ocrTextJSON,
			analysis.FaceCount,
			analysis.AnalysisTime,
			analysis.RawResponse,
		)
		return err
	}

	// Ensure OCRText is not nil
	if analysis.OCRText == nil {
		analysis.OCRText = []string{}
	}
	
	ocrTextJSON, err := json.Marshal(analysis.OCRText)
	if err != nil {
		return fmt.Errorf("failed to marshal OCR text: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO frame_analyses (
			id, video_id, frame_number, gpt_caption, vision_labels, 
			ocr_text, face_count, analysis_time, raw_response
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = r.db.conn.ExecContext(ctx, query,
		analysis.ID,
		analysis.VideoID,
		analysis.FrameNumber,
		analysis.GPTCaption,
		string(analysis.VisionLabels),
		string(ocrTextJSON),
		analysis.FaceCount,
		analysis.AnalysisTime,
		string(analysis.RawResponse),
	)
	return err
}

func (r *FrameAnalysisRepo) GetByVideoID(ctx context.Context, videoID string) ([]*ai.FrameAnalysisDB, error) {
	query := `
		SELECT id, video_id, frame_number, gpt_caption, vision_labels, 
			   ocr_text, face_count, analysis_time, raw_response
		FROM frame_analyses
		WHERE video_id = $1
		ORDER BY frame_number`

	rows, err := r.db.conn.QueryContext(ctx, query, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query frame analyses: %w", err)
	}
	defer rows.Close()

	var analyses []*ai.FrameAnalysisDB
	for rows.Next() {
		analysis := &ai.FrameAnalysisDB{}
		var ocrTextStr string

		if r.db.dbType == "postgres" {
			var ocrTextJSON []byte
			err = rows.Scan(
				&analysis.ID,
				&analysis.VideoID,
				&analysis.FrameNumber,
				&analysis.GPTCaption,
				&analysis.VisionLabels,
				&ocrTextJSON,
				&analysis.FaceCount,
				&analysis.AnalysisTime,
				&analysis.RawResponse,
			)
			if err == nil && len(ocrTextJSON) > 0 {
				if err := json.Unmarshal(ocrTextJSON, &analysis.OCRText); err != nil {
					analysis.OCRText = []string{}
				}
			}
		} else {
			var visionLabelsStr, rawResponseStr string
			err = rows.Scan(
				&analysis.ID,
				&analysis.VideoID,
				&analysis.FrameNumber,
				&analysis.GPTCaption,
				&visionLabelsStr,
				&ocrTextStr,
				&analysis.FaceCount,
				&analysis.AnalysisTime,
				&rawResponseStr,
			)
			if err == nil {
				analysis.VisionLabels = json.RawMessage(visionLabelsStr)
				analysis.RawResponse = json.RawMessage(rawResponseStr)
				
				if ocrTextStr != "" {
					if err := json.Unmarshal([]byte(ocrTextStr), &analysis.OCRText); err != nil {
						analysis.OCRText = []string{}
					}
				}
			}
		}

		if err != nil {
			return nil, fmt.Errorf("failed to scan frame analysis: %w", err)
		}

		analyses = append(analyses, analysis)
	}

	return analyses, nil
}

func (r *FrameAnalysisRepo) GetByID(ctx context.Context, id string) (*ai.FrameAnalysisDB, error) {
	query := `
		SELECT id, video_id, frame_number, gpt_caption, vision_labels, 
			   ocr_text, face_count, analysis_time, raw_response
		FROM frame_analyses
		WHERE id = $1`

	analysis := &ai.FrameAnalysisDB{}
	var ocrTextData interface{}

	if r.db.dbType == "postgres" {
		var ocrTextJSON []byte
		err := r.db.conn.QueryRowContext(ctx, query, id).Scan(
			&analysis.ID,
			&analysis.VideoID,
			&analysis.FrameNumber,
			&analysis.GPTCaption,
			&analysis.VisionLabels,
			&ocrTextJSON,
			&analysis.FaceCount,
			&analysis.AnalysisTime,
			&analysis.RawResponse,
		)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if len(ocrTextJSON) > 0 {
			if err := json.Unmarshal(ocrTextJSON, &analysis.OCRText); err != nil {
				analysis.OCRText = []string{}
			}
		}
		return analysis, nil
	}

	var visionLabelsStr, rawResponseStr string
	err := r.db.conn.QueryRowContext(ctx, query, id).Scan(
		&analysis.ID,
		&analysis.VideoID,
		&analysis.FrameNumber,
		&analysis.GPTCaption,
		&visionLabelsStr,
		&ocrTextData,
		&analysis.FaceCount,
		&analysis.AnalysisTime,
		&rawResponseStr,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	analysis.VisionLabels = json.RawMessage(visionLabelsStr)
	analysis.RawResponse = json.RawMessage(rawResponseStr)
	
	if ocrTextStr, ok := ocrTextData.(string); ok && ocrTextStr != "" {
		if err := json.Unmarshal([]byte(ocrTextStr), &analysis.OCRText); err != nil {
			analysis.OCRText = []string{}
		}
	}

	return analysis, nil
}

func (r *FrameAnalysisRepo) DeleteByVideoID(ctx context.Context, videoID string) error {
	query := `DELETE FROM frame_analyses WHERE video_id = $1`
	_, err := r.db.conn.ExecContext(ctx, query, videoID)
	return err
}