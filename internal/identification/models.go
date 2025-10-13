package identification

import (
	"time"
)

type IdentificationSession struct {
	ID           string
	VideoID      string
	CurrentFrame int
	Candidates   []FilmCandidate
	UserFeedback map[string]bool
	Confidence   float64
	Status       string
	StartedAt    time.Time
	CompletedAt  *time.Time
	Updates      chan SessionUpdate
}

type FilmCandidate struct {
	Title     string
	Year      int
	TMDbID    string
	Score     float64
	MatchedOn []string
	Source    string
	Snippet   string
}

type SessionUpdate struct {
	Type string
	Data interface{}
}

type ChipData struct {
	SessionID string
	Chips     []Chip
}

type Chip struct {
	Value    string
	Label    string
	Selected bool
	Type     string
}

type IdentificationResult struct {
	SessionID   string
	VideoID     string
	FilmDetails *FilmDetails
	Confidence  float64
	FramesUsed  int
	TimeElapsed time.Duration
}
