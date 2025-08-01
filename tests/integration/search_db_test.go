package integration

import (
	"testing"
)

func TestSearchDatabase(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Upload test videos
	testVideos := []struct {
		title       string
		description string
	}{
		{"Take5 Jazz Performance", "A smooth jazz performance of Take Five"},
		{"Take Me Home", "Country roads, take me home to the place I belong"},
		{"The Great Gatsby", "A classic movie adaptation of the novel"},
		{"Star Wars Episode V", "The Empire Strikes Back"},
		{"Documentary: Ocean Life", "Exploring the depths of the ocean"},
	}

	for _, v := range testVideos {
		resp := uploadTestVideo(t, ts.Server.URL, v.title, v.description)
		if resp.StatusCode != 200 {
			t.Fatalf("Failed to upload video: %s", v.title)
		}
		resp.Body.Close()
	}

	// Test search functionality at the database level
	tests := []struct {
		name          string
		query         string
		expectedCount int
		shouldContain []string
	}{
		{
			name:          "Partial match - 'take'",
			query:         "take",
			expectedCount: 2,
			shouldContain: []string{"Take5 Jazz Performance", "Take Me Home"},
		},
		{
			name:          "Case insensitive - 'TAKE'",
			query:         "TAKE",
			expectedCount: 2,
			shouldContain: []string{"Take5 Jazz Performance", "Take Me Home"},
		},
		{
			name:          "Description search - 'jazz'",
			query:         "jazz",
			expectedCount: 1,
			shouldContain: []string{"Take5 Jazz Performance"},
		},
		{
			name:          "Description search - 'ocean'",
			query:         "ocean",
			expectedCount: 1,
			shouldContain: []string{"Documentary: Ocean Life"},
		},
		{
			name:          "Partial word - 'gat'",
			query:         "gat",
			expectedCount: 1,
			shouldContain: []string{"The Great Gatsby"},
		},
		{
			name:          "No results",
			query:         "nonexistent",
			expectedCount: 0,
			shouldContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			videos, err := ts.VideoRepo.SearchVideos(tt.query)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			if len(videos) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(videos))
			}

			// Check that expected videos are in results
			for _, expectedTitle := range tt.shouldContain {
				found := false
				for _, video := range videos {
					if video.Title == expectedTitle {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find video with title '%s'", expectedTitle)
				}
			}
		})
	}
}

func TestSearchPartialWords(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Upload a video
	resp := uploadTestVideo(t, ts.Server.URL, "Programming Tutorial", "Learn programming basics")
	if resp.StatusCode != 200 {
		t.Fatal("Failed to upload video")
	}
	resp.Body.Close()

	// Test partial word searches
	searches := []string{"Prog", "prog", "gram", "Tutorial", "tutor"}
	
	for _, query := range searches {
		t.Run("Search for "+query, func(t *testing.T) {
			videos, err := ts.VideoRepo.SearchVideos(query)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			if len(videos) != 1 {
				t.Errorf("Expected 1 result for query '%s', got %d", query, len(videos))
			}

			if len(videos) > 0 && videos[0].Title != "Programming Tutorial" {
				t.Errorf("Wrong video found: %s", videos[0].Title)
			}
		})
	}
}