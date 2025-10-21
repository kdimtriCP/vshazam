package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	envPath := filepath.Join("..", "..", ".env")
	_ = godotenv.Load(envPath)
}

func TestTMDbClient_MovieSearch(t *testing.T) {
	apiKey := os.Getenv("TMDB_API_KEY")

	if apiKey == "" {
		t.Skip("Skipping TMDb integration test: TMDB_API_KEY not set")
	}

	client := NewTMDbClient(apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := "The Matrix"
	results, err := client.MovieSearch(ctx, query)
	if err != nil {
		t.Fatalf("MovieSearch failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least one movie result, got none")
	}

	t.Logf("Found %d movie results for query '%s'", len(results), query)

	firstResult := results[0]
	if firstResult.Title == "" {
		t.Error("Expected first result to have a title")
	}
	if firstResult.ID == 0 {
		t.Error("Expected first result to have a non-zero ID")
	}

	t.Logf("First result: %s (ID: %d, Release: %s, Rating: %.1f)",
		firstResult.Title, firstResult.ID, firstResult.ReleaseDate, firstResult.VoteAverage)

	if firstResult.Overview == "" {
		t.Error("Expected first result to have an overview")
	}
}

func TestTMDbClient_SearchMovies(t *testing.T) {
	apiKey := os.Getenv("TMDB_API_KEY")

	if apiKey == "" {
		t.Skip("Skipping TMDb SearchMovies test: TMDB_API_KEY not set")
	}

	client := NewTMDbClient(apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	movies, err := client.SearchMovies(ctx, "Inception")
	if err != nil {
		t.Fatalf("SearchMovies failed: %v", err)
	}

	if len(movies) == 0 {
		t.Fatal("Expected at least one movie, got none")
	}

	t.Logf("Found %d movies", len(movies))

	firstMovie := movies[0]
	if firstMovie.Title == "" {
		t.Error("Expected first movie to have a title")
	}
	if firstMovie.ID == 0 {
		t.Error("Expected first movie to have a non-zero ID")
	}
}

func TestTMDbClient_GetFilm(t *testing.T) {
	apiKey := os.Getenv("TMDB_API_KEY")

	if apiKey == "" {
		t.Skip("Skipping TMDb GetFilm test: TMDB_API_KEY not set")
	}

	client := NewTMDbClient(apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmdbID := "603"
	film, err := client.GetFilm(ctx, tmdbID)
	if err != nil {
		t.Fatalf("GetFilm failed: %v", err)
	}

	if film.ID != 603 {
		t.Errorf("Expected film ID to be 603, got %d", film.ID)
	}

	if film.Title == "" {
		t.Error("Expected film to have a title")
	}

	t.Logf("Film: %s (%s)", film.Title, film.ReleaseDate)
	t.Logf("Runtime: %d minutes, Rating: %.1f", film.Runtime, film.VoteAverage)
	t.Logf("Genres: %v", film.Genres)

	if len(film.Credits.Cast) == 0 {
		t.Error("Expected film to have cast members")
	}

	if len(film.Genres) == 0 {
		t.Error("Expected film to have genres")
	}
}

func TestTMDbClient_GetImageURL(t *testing.T) {
	client := NewTMDbClient("dummy_key")

	tests := []struct {
		name     string
		path     string
		size     string
		expected string
	}{
		{
			name:     "valid poster path",
			path:     "/8Vt6mWEReuy4Of61Lnj5Xj704m8.jpg",
			size:     "w500",
			expected: "https://image.tmdb.org/t/p/w500/8Vt6mWEReuy4Of61Lnj5Xj704m8.jpg",
		},
		{
			name:     "empty path",
			path:     "",
			size:     "w500",
			expected: "",
		},
		{
			name:     "original size",
			path:     "/poster.jpg",
			size:     "original",
			expected: "https://image.tmdb.org/t/p/original/poster.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.GetImageURL(tt.path, tt.size)
			if result != tt.expected {
				t.Errorf("GetImageURL(%q, %q) = %q, expected %q",
					tt.path, tt.size, result, tt.expected)
			}
		})
	}
}

func TestTMDbClient_ContextCancellation(t *testing.T) {
	apiKey := os.Getenv("TMDB_API_KEY")

	if apiKey == "" {
		t.Skip("Skipping TMDb context test: TMDB_API_KEY not set")
	}

	client := NewTMDbClient(apiKey)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.MovieSearch(ctx, "test movie")
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}
