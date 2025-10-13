package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type GoogleSearchClient struct {
	apiKey         string
	searchEngineID string
	httpClient     *http.Client
}

type SearchResult struct {
	Title   string
	Link    string
	Snippet string
}

type googleSearchResponse struct {
	Items []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"items"`
}

func NewGoogleSearchClient(apiKey, searchEngineID string) *GoogleSearchClient {
	return &GoogleSearchClient{
		apiKey:         apiKey,
		searchEngineID: searchEngineID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *GoogleSearchClient) SearchFilms(ctx context.Context, query string) ([]SearchResult, error) {
	apiURL := "https://www.googleapis.com/customsearch/v1"

	params := url.Values{}
	params.Set("key", c.apiKey)
	params.Set("cx", c.searchEngineID)
	params.Set("q", query)
	params.Set("num", "10")

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search API returned status %d", resp.StatusCode)
	}

	var searchResp googleSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	results := make([]SearchResult, 0, len(searchResp.Items))
	for _, item := range searchResp.Items {
		results = append(results, SearchResult{
			Title:   item.Title,
			Link:    item.Link,
			Snippet: item.Snippet,
		})
	}

	return results, nil
}
