package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestVideoUpload(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	tests := []struct {
		name           string
		title          string
		description    string
		filename       string
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:           "Valid video upload",
			title:          "Test Video",
			description:    "This is a test video",
			filename:       "test.mp4",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "Upload without title",
			title:          "",
			description:    "No title video",
			filename:       "test.mp4",
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name:           "Upload with long description",
			title:          "Long Description Video",
			description:    strings.Repeat("This is a very long description. ", 50),
			filename:       "test.mp4",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Count videos before upload
			countBefore, err := countVideosInDB(ts.DB.Conn())
			if err != nil {
				t.Fatalf("Failed to count videos: %v", err)
			}

			// Create upload request
			content := []byte("fake mp4 content")
			body, contentType, err := createMultipartUpload(tt.title, tt.description, tt.filename, content)
			if err != nil {
				t.Fatalf("Failed to create upload: %v", err)
			}

			req, err := http.NewRequest("POST", ts.Server.URL+"/upload", body)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", contentType)

			// Perform upload
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to perform request: %v", err)
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, resp.StatusCode, body)
			}

			// Check video count
			countAfter, err := countVideosInDB(ts.DB.Conn())
			if err != nil {
				t.Fatalf("Failed to count videos after: %v", err)
			}

			if tt.expectSuccess {
				if countAfter != countBefore+1 {
					t.Errorf("Expected video count to increase by 1, but got %d -> %d", countBefore, countAfter)
				}
			} else {
				if countAfter != countBefore {
					t.Errorf("Expected video count to remain the same, but got %d -> %d", countBefore, countAfter)
				}
			}
		})
	}
}

func TestMultipleUploads(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Upload multiple videos
	videos := []struct {
		title       string
		description string
	}{
		{"First Video", "Description 1"},
		{"Second Video", "Description 2"},
		{"Third Video", "Description 3"},
	}

	for _, v := range videos {
		resp := uploadTestVideo(t, ts.Server.URL, v.title, v.description)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Failed to upload video '%s': status %d", v.title, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// Check total count
	count, err := countVideosInDB(ts.DB.Conn())
	if err != nil {
		t.Fatalf("Failed to count videos: %v", err)
	}

	if count != len(videos) {
		t.Errorf("Expected %d videos, but found %d", len(videos), count)
	}
}