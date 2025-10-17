package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kdimtricp/vshazam/internal/ai"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FrameAnalysisRepo struct {
	db *DB
}

func NewFrameAnalysisRepo(db *DB) *FrameAnalysisRepo {
	return &FrameAnalysisRepo{db: db}
}

func (r *FrameAnalysisRepo) Create(ctx context.Context, analysis *ai.FrameAnalysisDB) error {
	if analysis.ID == "" {
		analysis.ID = uuid.New().String()
	}

	if analysis.OCRText == nil {
		analysis.OCRText = []string{}
	}

	result := r.db.GORM().WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "video_id"}, {Name: "frame_number"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"gpt_caption", "vision_labels", "ocr_text",
			"face_count", "analysis_time", "raw_response",
		}),
	}).Create(analysis)

	return result.Error
}

func (r *FrameAnalysisRepo) GetByVideoID(ctx context.Context, videoID string) ([]*ai.FrameAnalysisDB, error) {
	var analyses []*ai.FrameAnalysisDB
	result := r.db.GORM().WithContext(ctx).
		Where("video_id = ?", videoID).
		Order("frame_number").
		Find(&analyses)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to query frame analyses: %w", result.Error)
	}

	return analyses, nil
}

func (r *FrameAnalysisRepo) GetByID(ctx context.Context, id string) (*ai.FrameAnalysisDB, error) {
	var analysis ai.FrameAnalysisDB
	result := r.db.GORM().WithContext(ctx).First(&analysis, "id = ?", id)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}

	return &analysis, nil
}

func (r *FrameAnalysisRepo) DeleteByVideoID(ctx context.Context, videoID string) error {
	result := r.db.GORM().WithContext(ctx).Where("video_id = ?", videoID).Delete(&ai.FrameAnalysisDB{})
	return result.Error
}
