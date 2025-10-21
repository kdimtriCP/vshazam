package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type TMDbClient struct {
	apiKey     string
	httpClient *http.Client
}

type FilmDetails struct {
	ID           int     `json:"id"`
	Title        string  `json:"title"`
	ReleaseDate  string  `json:"release_date"`
	Overview     string  `json:"overview"`
	PosterPath   string  `json:"poster_path"`
	BackdropPath string  `json:"backdrop_path"`
	VoteAverage  float64 `json:"vote_average"`
	Genres       []Genre `json:"genres"`
	Runtime      int     `json:"runtime"`
	Credits      Credits `json:"credits"`
}

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Credits struct {
	Cast []CastMember `json:"cast"`
	Crew []CrewMember `json:"crew"`
}

type CastMember struct {
	Name      string `json:"name"`
	Character string `json:"character"`
	Order     int    `json:"order"`
}

type CrewMember struct {
	Name       string `json:"name"`
	Job        string `json:"job"`
	Department string `json:"department"`
}

type SearchMovieResult struct {
	Page         int     `json:"page"`
	Results      []Movie `json:"results"`
	TotalResults int     `json:"total_results"`
}

type Movie struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	ReleaseDate string  `json:"release_date"`
	Overview    string  `json:"overview"`
	PosterPath  string  `json:"poster_path"`
	VoteAverage float64 `json:"vote_average"`
}

func NewTMDbClient(apiKey string) *TMDbClient {
	return &TMDbClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *TMDbClient) GetFilm(ctx context.Context, tmdbID string) (*FilmDetails, error) {
	url := fmt.Sprintf("https://api.themoviedb.org/3/movie/%s?api_key=%s&append_to_response=credits",
		tmdbID, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDb API returned status %d", resp.StatusCode)
	}

	var details FilmDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &details, nil
}

func (c *TMDbClient) SearchMovies(ctx context.Context, query string) ([]Movie, error) {
	apiURL := "https://api.themoviedb.org/3/search/movie"

	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("query", query)
	params.Set("page", "1")

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
		return nil, fmt.Errorf("TMDb API returned status %d", resp.StatusCode)
	}

	var searchResult SearchMovieResult
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return searchResult.Results, nil
}

func (c *TMDbClient) GetImageURL(path string, size string) string {
	if path == "" {
		return ""
	}
	return fmt.Sprintf("https://image.tmdb.org/t/p/%s%s", size, path)
}

func (c *TMDbClient) MovieSearch(ctx context.Context, query string) ([]MovieSearchResult, error) {
	movies, err := c.SearchMovies(ctx, query)
	if err != nil {
		return nil, err
	}

	results := make([]MovieSearchResult, 0, len(movies))
	for _, movie := range movies {
		results = append(results, MovieSearchResult{
			ID:          movie.ID,
			Title:       movie.Title,
			ReleaseDate: movie.ReleaseDate,
			Overview:    movie.Overview,
			PosterPath:  movie.PosterPath,
			VoteAverage: movie.VoteAverage,
		})
	}

	return results, nil
}
