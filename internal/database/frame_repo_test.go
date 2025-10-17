package database

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kdimtricp/vshazam/internal/models"
	"github.com/kdimtricp/vshazam/internal/models/frame_analysis"
)

func TestFrameAnalysisRepo_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	videoRepo := NewVideoRepository(db)
	frameRepo := NewFrameAnalysisRepo(db)

	video := models.NewVideo("Test Video", "Test", "test.mp4", "video/mp4", 1024)
	if err := videoRepo.InsertVideo(video); err != nil {
		t.Fatalf("Failed to insert video: %v", err)
	}

	ctx := context.Background()
	analysis := &frame_analysis.FrameAnalysisDB{
		VideoID:      video.ID,
		FrameNumber:  1,
		GPTCaption:   "A test frame",
		VisionLabels: json.RawMessage(`[{"name": "test", "confidence": 0.9}]`),
		OCRText:      []string{"Sample text"},
		FaceCount:    2,
		AnalysisTime: time.Now(),
		RawResponse:  json.RawMessage(`{"raw": "data"}`),
	}

	err := frameRepo.Create(ctx, analysis)
	if err != nil {
		t.Fatalf("Failed to create frame analysis: %v", err)
	}

	if analysis.ID == "" {
		t.Error("Expected ID to be set after create")
	}

	retrieved, err := frameRepo.GetByID(ctx, analysis.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve frame analysis: %v", err)
	}

	if retrieved.GPTCaption != analysis.GPTCaption {
		t.Errorf("Expected caption %s, got %s", analysis.GPTCaption, retrieved.GPTCaption)
	}
	if retrieved.FaceCount != analysis.FaceCount {
		t.Errorf("Expected face count %d, got %d", analysis.FaceCount, retrieved.FaceCount)
	}
}

func TestFrameAnalysisRepo_Create_Upsert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	videoRepo := NewVideoRepository(db)
	frameRepo := NewFrameAnalysisRepo(db)

	video := models.NewVideo("Test Video", "Test", "test.mp4", "video/mp4", 1024)
	if err := videoRepo.InsertVideo(video); err != nil {
		t.Fatalf("Failed to insert video: %v", err)
	}

	ctx := context.Background()
	analysis1 := &frame_analysis.FrameAnalysisDB{
		VideoID:      video.ID,
		FrameNumber:  1,
		GPTCaption:   "Original caption",
		VisionLabels: json.RawMessage(`[]`),
		OCRText:      []string{},
		FaceCount:    1,
		AnalysisTime: time.Now(),
		RawResponse:  json.RawMessage(`{}`),
	}

	if err := frameRepo.Create(ctx, analysis1); err != nil {
		t.Fatalf("Failed to create first analysis: %v", err)
	}

	analysis2 := &frame_analysis.FrameAnalysisDB{
		VideoID:      video.ID,
		FrameNumber:  1,
		GPTCaption:   "Updated caption",
		VisionLabels: json.RawMessage(`[]`),
		OCRText:      []string{"New text"},
		FaceCount:    2,
		AnalysisTime: time.Now(),
		RawResponse:  json.RawMessage(`{}`),
	}

	if err := frameRepo.Create(ctx, analysis2); err != nil {
		t.Fatalf("Failed to create second analysis (upsert): %v", err)
	}

	analyses, err := frameRepo.GetByVideoID(ctx, video.ID)
	if err != nil {
		t.Fatalf("Failed to get analyses: %v", err)
	}

	if len(analyses) != 1 {
		t.Errorf("Expected 1 analysis after upsert, got %d", len(analyses))
	}

	if analyses[0].GPTCaption != "Updated caption" {
		t.Errorf("Expected updated caption, got %s", analyses[0].GPTCaption)
	}
	if analyses[0].FaceCount != 2 {
		t.Errorf("Expected face count 2, got %d", analyses[0].FaceCount)
	}
}

func TestFrameAnalysisRepo_GetByVideoID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	videoRepo := NewVideoRepository(db)
	frameRepo := NewFrameAnalysisRepo(db)

	video := models.NewVideo("Test Video", "Test", "test.mp4", "video/mp4", 1024)
	if err := videoRepo.InsertVideo(video); err != nil {
		t.Fatalf("Failed to insert video: %v", err)
	}

	ctx := context.Background()

	for i := 0; i < 3; i++ {
		analysis := &frame_analysis.FrameAnalysisDB{
			VideoID:      video.ID,
			FrameNumber:  i,
			GPTCaption:   "Frame " + string(rune('0'+i)),
			VisionLabels: json.RawMessage(`[]`),
			OCRText:      []string{},
			FaceCount:    i,
			AnalysisTime: time.Now(),
			RawResponse:  json.RawMessage(`{}`),
		}

		if err := frameRepo.Create(ctx, analysis); err != nil {
			t.Fatalf("Failed to create analysis %d: %v", i, err)
		}
	}

	analyses, err := frameRepo.GetByVideoID(ctx, video.ID)
	if err != nil {
		t.Fatalf("Failed to get analyses: %v", err)
	}

	if len(analyses) != 3 {
		t.Errorf("Expected 3 analyses, got %d", len(analyses))
	}

	for i, analysis := range analyses {
		if analysis.FrameNumber != i {
			t.Errorf("Expected frame number %d at index %d, got %d", i, i, analysis.FrameNumber)
		}
	}
}

func TestFrameAnalysisRepo_GetByID_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	frameRepo := NewFrameAnalysisRepo(db)
	ctx := context.Background()

	result, err := frameRepo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Errorf("Expected no error for non-existent ID, got %v", err)
	}
	if result != nil {
		t.Error("Expected nil result for non-existent ID")
	}
}

func TestFrameAnalysisRepo_DeleteByVideoID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	videoRepo := NewVideoRepository(db)
	frameRepo := NewFrameAnalysisRepo(db)

	video := models.NewVideo("Test Video", "Test", "test.mp4", "video/mp4", 1024)
	if err := videoRepo.InsertVideo(video); err != nil {
		t.Fatalf("Failed to insert video: %v", err)
	}

	ctx := context.Background()

	for i := 0; i < 3; i++ {
		analysis := &frame_analysis.FrameAnalysisDB{
			VideoID:      video.ID,
			FrameNumber:  i,
			GPTCaption:   "Frame",
			VisionLabels: json.RawMessage(`[]`),
			OCRText:      []string{},
			FaceCount:    0,
			AnalysisTime: time.Now(),
			RawResponse:  json.RawMessage(`{}`),
		}

		if err := frameRepo.Create(ctx, analysis); err != nil {
			t.Fatalf("Failed to create analysis: %v", err)
		}
	}

	err := frameRepo.DeleteByVideoID(ctx, video.ID)
	if err != nil {
		t.Fatalf("Failed to delete analyses: %v", err)
	}

	analyses, err := frameRepo.GetByVideoID(ctx, video.ID)
	if err != nil {
		t.Fatalf("Failed to get analyses: %v", err)
	}

	if len(analyses) != 0 {
		t.Errorf("Expected 0 analyses after delete, got %d", len(analyses))
	}
}

func TestFrameAnalysisRepo_OCRTextHandling(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	videoRepo := NewVideoRepository(db)
	frameRepo := NewFrameAnalysisRepo(db)

	video := models.NewVideo("Test Video", "Test", "test.mp4", "video/mp4", 1024)
	if err := videoRepo.InsertVideo(video); err != nil {
		t.Fatalf("Failed to insert video: %v", err)
	}

	ctx := context.Background()

	analysis := &frame_analysis.FrameAnalysisDB{
		VideoID:      video.ID,
		FrameNumber:  1,
		GPTCaption:   "Test",
		VisionLabels: json.RawMessage(`[]`),
		OCRText:      nil,
		FaceCount:    0,
		AnalysisTime: time.Now(),
		RawResponse:  json.RawMessage(`{}`),
	}

	err := frameRepo.Create(ctx, analysis)
	if err != nil {
		t.Fatalf("Failed to create analysis with nil OCRText: %v", err)
	}

	retrieved, err := frameRepo.GetByID(ctx, analysis.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve analysis: %v", err)
	}

	if retrieved.OCRText == nil {
		t.Error("Expected OCRText to be initialized as empty slice, got nil")
	}
	if len(retrieved.OCRText) != 0 {
		t.Errorf("Expected empty OCRText slice, got %d items", len(retrieved.OCRText))
	}
}
