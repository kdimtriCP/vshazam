package search

import "context"

type WebSearchResult struct {
	Title   string
	Snippet string
	Link    string
}

type MovieSearchResult struct {
	ID          int
	Title       string
	ReleaseDate string
	Overview    string
	PosterPath  string
	VoteAverage float64
}

type WebSearcher interface {
	WebSearch(ctx context.Context, query string) ([]WebSearchResult, error)
}

type MovieSearcher interface {
	MovieSearch(ctx context.Context, query string) ([]MovieSearchResult, error)
}
