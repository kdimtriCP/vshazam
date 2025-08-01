package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestVideoListing(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Upload test videos
	testVideos := []struct {
		title       string
		description string
	}{
		{"Alpha Video", "First test video"},
		{"Beta Video", "Second test video"},
		{"Gamma Video", "Third test video"},
	}

	for _, v := range testVideos {
		resp := uploadTestVideo(t, ts.Server.URL, v.title, v.description)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to upload video: %s", v.title)
		}
		resp.Body.Close()
	}

	// Test listing endpoint
	resp, err := http.Get(ts.Server.URL + "/videos")
	if err != nil {
		t.Fatalf("Failed to get videos: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Check that all video titles appear in the response
	bodyStr := string(body)
	for _, v := range testVideos {
		if !strings.Contains(bodyStr, v.title) {
			t.Errorf("Video title '%s' not found in response", v.title)
		}
		if !strings.Contains(bodyStr, v.description) {
			t.Errorf("Video description '%s' not found in response", v.description)
		}
	}
}

func TestEmptyVideoList(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Test listing with no videos
	resp, err := http.Get(ts.Server.URL + "/videos")
	if err != nil {
		t.Fatalf("Failed to get videos: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Should contain "No videos" message
	if !strings.Contains(string(body), "No videos") {
		t.Error("Expected 'No videos' message in empty list")
	}
}

func TestVideoPartialEndpoint(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Upload a test video
	resp := uploadTestVideo(t, ts.Server.URL, "Test Video", "Test Description")
	if resp.StatusCode != http.StatusOK {
		t.Fatal("Failed to upload test video")
	}
	resp.Body.Close()

	// Test partial endpoint (used by HTMX)
	resp, err := http.Get(ts.Server.URL + "/videos/partial")
	if err != nil {
		t.Fatalf("Failed to get partial videos: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Should contain the video title
	if !strings.Contains(string(body), "Test Video") {
		t.Error("Video title not found in partial response")
	}
}

func TestVideoDetailPage(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Upload a test video
	resp := uploadTestVideo(t, ts.Server.URL, "Detail Test Video", "Detailed description")
	if resp.StatusCode != http.StatusOK {
		t.Fatal("Failed to upload test video")
	}
	resp.Body.Close()

	// Get the video ID
	videos, err := ts.VideoRepo.ListVideos()
	if err != nil || len(videos) == 0 {
		t.Fatal("Failed to get uploaded video")
	}

	videoID := videos[0].ID

	// Test video detail page
	resp, err = http.Get(ts.Server.URL + "/videos/" + videoID)
	if err != nil {
		t.Fatalf("Failed to get video detail: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Check for video details
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Detail Test Video") {
		t.Error("Video title not found in detail page")
	}
	if !strings.Contains(bodyStr, "Detailed description") {
		t.Error("Video description not found in detail page")
	}
}

func TestNonExistentVideo(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Try to access non-existent video
	resp, err := http.Get(ts.Server.URL + "/videos/non-existent-id")
	if err != nil {
		t.Fatalf("Failed to get video: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}