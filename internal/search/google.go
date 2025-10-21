package search

import (
	"context"
	"fmt"

	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/option"
)

type GoogleSearchClient struct {
	cseID   string
	service *customsearch.Service
}

func NewGoogleSearchClient(apiKey, cseID string) (*GoogleSearchClient, error) {
	if apiKey == "" || cseID == "" {
		return nil, fmt.Errorf("Google Search API key or CSE ID not configured")
	}

	svc, err := customsearch.NewService(context.Background(), option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("creating custom search service: %w", err)
	}

	return &GoogleSearchClient{
		cseID:   cseID,
		service: svc,
	}, nil
}

func (c *GoogleSearchClient) WebSearch(ctx context.Context, query string) ([]WebSearchResult, error) {
	call := c.service.Cse.List().Cx(c.cseID).Q(query).Num(10).Context(ctx)
	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("executing search: %w", err)
	}

	results := make([]WebSearchResult, 0, len(resp.Items))
	for _, item := range resp.Items {
		results = append(results, WebSearchResult{
			Title:   item.Title,
			Link:    item.Link,
			Snippet: item.Snippet,
		})
	}

	return results, nil
}
