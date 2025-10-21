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

func TestGoogleSearchClient_WebSearch(t *testing.T) {
	apiKey := os.Getenv("GOOGLE_SEARCH_API_KEY")
	cseID := os.Getenv("GOOGLE_CSE_ID")

	if apiKey == "" || cseID == "" {
		t.Skip("Skipping Google Search integration test: GOOGLE_SEARCH_API_KEY or GOOGLE_CSE_ID not set")
	}

	client, err := NewGoogleSearchClient(apiKey, cseID)
	if err != nil {
		t.Fatalf("Failed to create Google Search client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := "The Matrix 1999 film"
	results, err := client.WebSearch(ctx, query)
	if err != nil {
		t.Fatalf("WebSearch failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least one search result, got none")
	}

	t.Logf("Found %d search results for query '%s'", len(results), query)

	firstResult := results[0]
	if firstResult.Title == "" {
		t.Error("Expected first result to have a title")
	}
	if firstResult.Link == "" {
		t.Error("Expected first result to have a link")
	}
	if firstResult.Snippet == "" {
		t.Error("Expected first result to have a snippet")
	}

	t.Logf("First result: %s - %s", firstResult.Title, firstResult.Link)
}

func TestGoogleSearchClient_WebSearch_EmptyCredentials(t *testing.T) {
	client, err := NewGoogleSearchClient("", "")
	if err == nil {
		t.Fatal("Expected error when creating client with empty credentials")
	}
	if client != nil {
		t.Fatal("Expected nil client when credentials are invalid")
	}
}

func TestGoogleSearchClient_WebSearch_ContextCancellation(t *testing.T) {
	apiKey := os.Getenv("GOOGLE_SEARCH_API_KEY")
	cseID := os.Getenv("GOOGLE_CSE_ID")

	if apiKey == "" || cseID == "" {
		t.Skip("Skipping Google Search context test: GOOGLE_SEARCH_API_KEY or GOOGLE_CSE_ID not set")
	}

	client, err := NewGoogleSearchClient(apiKey, cseID)
	if err != nil {
		t.Fatalf("Failed to create Google Search client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = client.WebSearch(ctx, "test query")
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}
