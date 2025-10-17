package database

import (
	"testing"
	"time"

	"github.com/kdimtricp/vshazam/internal/models"
)

func TestVideoRepository_InsertVideo(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewVideoRepository(db)

	video := models.NewVideo("Test Video", "A test video", "test.mp4", "video/mp4", 1024)

	err := repo.InsertVideo(video)
	if err != nil {
		t.Fatalf("Failed to insert video: %v", err)
	}

	retrieved, err := repo.GetVideoByID(video.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve video: %v", err)
	}

	if retrieved.Title != video.Title {
		t.Errorf("Expected title %s, got %s", video.Title, retrieved.Title)
	}
	if retrieved.Filename != video.Filename {
		t.Errorf("Expected filename %s, got %s", video.Filename, retrieved.Filename)
	}
}

func TestVideoRepository_GetVideoByID_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewVideoRepository(db)

	_, err := repo.GetVideoByID("00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Error("Expected error for non-existent video, got nil")
	}
}

func TestVideoRepository_ListVideos(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewVideoRepository(db)

	video1 := models.NewVideo("Video 1", "First video", "video1.mp4", "video/mp4", 1024)
	video2 := models.NewVideo("Video 2", "Second video", "video2.mp4", "video/mp4", 2048)

	time.Sleep(10 * time.Millisecond)
	video2.UploadTime = time.Now()

	err := repo.InsertVideo(video1)
	if err != nil {
		t.Fatalf("Failed to insert video1: %v", err)
	}

	err = repo.InsertVideo(video2)
	if err != nil {
		t.Fatalf("Failed to insert video2: %v", err)
	}

	videos, err := repo.ListVideos()
	if err != nil {
		t.Fatalf("Failed to list videos: %v", err)
	}

	if len(videos) != 2 {
		t.Errorf("Expected 2 videos, got %d", len(videos))
	}

	if videos[0].ID != video2.ID {
		t.Errorf("Expected first video to be most recent (video2), got %s", videos[0].ID)
	}
}

func TestVideoRepository_SearchVideos(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewVideoRepository(db)

	video1 := models.NewVideo("Action Movie", "An exciting action film", "action.mp4", "video/mp4", 1024)
	video2 := models.NewVideo("Comedy Show", "A funny comedy", "comedy.mp4", "video/mp4", 2048)
	video3 := models.NewVideo("Drama", "An action-packed drama", "drama.mp4", "video/mp4", 3072)

	for _, v := range []*models.Video{video1, video2, video3} {
		if err := repo.InsertVideo(v); err != nil {
			t.Fatalf("Failed to insert video: %v", err)
		}
	}

	results, err := repo.SearchVideos("action")
	if err != nil {
		t.Fatalf("Failed to search videos: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'action', got %d", len(results))
	}

	results, err = repo.SearchVideos("comedy")
	if err != nil {
		t.Fatalf("Failed to search videos: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'comedy', got %d", len(results))
	}
	if len(results) > 0 && results[0].ID != video2.ID {
		t.Errorf("Expected comedy video, got %s", results[0].Title)
	}
}

func TestVideoRepository_SearchVideos_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewVideoRepository(db)

	video := models.NewVideo("Test Video", "A test", "test.mp4", "video/mp4", 1024)
	if err := repo.InsertVideo(video); err != nil {
		t.Fatalf("Failed to insert video: %v", err)
	}

	results, err := repo.SearchVideos("")
	if err != nil {
		t.Fatalf("Failed to search with empty query: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result for empty query, got %d", len(results))
	}
}
