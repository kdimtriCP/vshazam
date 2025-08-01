package integration

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSearchFunctionality(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Upload test videos with various titles and descriptions
	testVideos := []struct {
		title       string
		description string
	}{
		{"Take5 Jazz Performance", "A smooth jazz performance of Take Five"},
		{"Take Me Home", "Country roads, take me home to the place I belong"},
		{"The Great Gatsby", "A classic movie adaptation of the novel"},
		{"Star Wars Episode V", "The Empire Strikes Back"},
		{"Documentary: Ocean Life", "Exploring the depths of the ocean"},
		{"Tutorial: Go Programming", "Learn Go programming from scratch"},
	}

	for _, v := range testVideos {
		resp := uploadTestVideo(t, ts.Server.URL, v.title, v.description)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to upload video: %s", v.title)
		}
		resp.Body.Close()
		// Small delay to ensure upload completes
		time.Sleep(10 * time.Millisecond)
	}

	// Test cases for search
	tests := []struct {
		name          string
		query         string
		expectedCount int
		shouldContain []string
		shouldNotContain []string
	}{
		{
			name:          "Partial match - 'take'",
			query:         "take",
			expectedCount: 2,
			shouldContain: []string{"Take5 Jazz Performance", "Take Me Home"},
			shouldNotContain: []string{"The Great Gatsby", "Star Wars"},
		},
		{
			name:          "Case insensitive - 'TAKE'",
			query:         "TAKE",
			expectedCount: 2,
			shouldContain: []string{"Take5 Jazz Performance", "Take Me Home"},
		},
		{
			name:          "Partial word - 'gat'",
			query:         "gat",
			expectedCount: 1,
			shouldContain: []string{"The Great Gatsby"},
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
			name:          "Multiple word match - 'Go Programming'",
			query:         "Go Programming",
			expectedCount: 1,
			shouldContain: []string{"Tutorial: Go Programming"},
		},
		{
			name:          "No results",
			query:         "nonexistent",
			expectedCount: 0,
			shouldContain: []string{},
		},
		{
			name:          "Single character search",
			query:         "V",
			expectedCount: 1,
			shouldContain: []string{"Star Wars Episode V"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Perform search
			searchURL := ts.Server.URL + "/search?q=" + url.QueryEscape(tt.query)
			req, err := http.NewRequest("GET", searchURL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("HX-Request", "true")
			
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to perform search: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response: %v", err)
			}

			bodyStr := string(body)
			
			// Log response for debugging
			if resp.StatusCode != http.StatusOK {
				t.Logf("Response body: %s", bodyStr)
			}

			// Check for expected videos
			for _, expected := range tt.shouldContain {
				if !strings.Contains(bodyStr, expected) {
					t.Errorf("Expected to find '%s' in search results for query '%s'", expected, tt.query)
				}
			}

			// Check that unexpected videos are not present
			for _, unexpected := range tt.shouldNotContain {
				if strings.Contains(bodyStr, unexpected) {
					t.Errorf("Did not expect to find '%s' in search results for query '%s'", unexpected, tt.query)
				}
			}

			// If no results expected, check for appropriate message
			if tt.expectedCount == 0 {
				if !strings.Contains(bodyStr, "No videos found") && !strings.Contains(bodyStr, "no videos found") {
					t.Logf("Response body: %s", bodyStr)
					t.Error("Expected 'No videos found' message for empty results")
				}
			}
		})
	}
}

func TestSearchWithHTMXHeaders(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Upload a test video
	resp := uploadTestVideo(t, ts.Server.URL, "HTMX Test Video", "Testing HTMX search functionality")
	if resp.StatusCode != http.StatusOK {
		t.Fatal("Failed to upload test video")
	}
	resp.Body.Close()

	// Test search with HTMX headers
	searchURL := ts.Server.URL + "/search?q=HTMX"
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("HX-Request", "true")

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to perform HTMX search: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// HTMX response should be a partial (not full page)
	bodyStr := string(body)
	if strings.Contains(bodyStr, "<!DOCTYPE html>") {
		t.Error("HTMX response should not contain full HTML document")
	}

	if !strings.Contains(bodyStr, "HTMX Test Video") {
		t.Error("Expected to find video in HTMX search results")
	}
}

func TestEmptySearchQuery(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Upload a test video
	resp := uploadTestVideo(t, ts.Server.URL, "Empty Query Test", "Should not appear in empty search")
	if resp.StatusCode != http.StatusOK {
		t.Fatal("Failed to upload test video")
	}
	resp.Body.Close()

	// Test empty search query
	searchURL := ts.Server.URL + "/search?q="
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("HX-Request", "true")
	
	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to perform search: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Should show all videos or a message about empty query
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Empty Query Test") && !strings.Contains(bodyStr, "Enter a search term") {
		t.Error("Expected to see all videos or empty query message")
	}
}

func TestSearchSpecialCharacters(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Upload videos with special characters
	testVideos := []struct {
		title       string
		description string
		searchTerm  string
	}{
		{"Video with @symbol", "Contains @ symbol", "@symbol"},
		{"C++ Programming", "Learn C++ basics", "C++"},
		{"Movie: The Matrix (1999)", "Sci-fi classic", "Matrix (1999)"},
	}

	for _, v := range testVideos {
		resp := uploadTestVideo(t, ts.Server.URL, v.title, v.description)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to upload video: %s", v.title)
		}
		resp.Body.Close()
	}

	// Test searching for each video
	for _, v := range testVideos {
		t.Run("Search for "+v.searchTerm, func(t *testing.T) {
			searchURL := ts.Server.URL + "/search?q=" + url.QueryEscape(v.searchTerm)
			req, err := http.NewRequest("GET", searchURL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("HX-Request", "true")
			
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to search: %v", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response: %v", err)
			}

			bodyStr := string(body)
			// Check for both the literal title and HTML-encoded version
			found := strings.Contains(bodyStr, v.title)
			if !found {
				// Try with HTML encoding for special characters
				encodedTitle := strings.ReplaceAll(v.title, "+", "&#43;")
				found = strings.Contains(bodyStr, encodedTitle)
			}
			if !found {
				t.Logf("Search response for '%s':\n%s", v.searchTerm, bodyStr)
				t.Errorf("Failed to find video '%s' when searching for '%s'", v.title, v.searchTerm)
			}
		})
	}
}