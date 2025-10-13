package identification

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kdimtricp/vshazam/internal/ai"
	"github.com/kdimtricp/vshazam/internal/database"
	"github.com/kdimtricp/vshazam/internal/storage"
)

type Service struct {
	visionService    ai.VisionService
	searchClient     *ai.GoogleSearchClient
	tmdbClient       *TMDbClient
	frameExtractor   *ai.FrameExtractor
	videoRepo        database.VideoRepository
	frameRepo        database.FrameAnalysisRepository
	storageService   storage.Service
	scoreThreshold   float64
	maxFramesAnalyze int
	sessions         map[string]*IdentificationSession
	sessionsMu       sync.RWMutex
}

type Config struct {
	ScoreThreshold   float64
	MaxFramesAnalyze int
}

func NewService(
	visionService ai.VisionService,
	searchClient *ai.GoogleSearchClient,
	tmdbClient *TMDbClient,
	frameExtractor *ai.FrameExtractor,
	videoRepo database.VideoRepository,
	frameRepo database.FrameAnalysisRepository,
	storageService storage.Service,
	config Config,
) *Service {
	if config.ScoreThreshold == 0 {
		config.ScoreThreshold = 0.9
	}
	if config.MaxFramesAnalyze == 0 {
		config.MaxFramesAnalyze = 10
	}

	return &Service{
		visionService:    visionService,
		searchClient:     searchClient,
		tmdbClient:       tmdbClient,
		frameExtractor:   frameExtractor,
		videoRepo:        videoRepo,
		frameRepo:        frameRepo,
		storageService:   storageService,
		scoreThreshold:   config.ScoreThreshold,
		maxFramesAnalyze: config.MaxFramesAnalyze,
		sessions:         make(map[string]*IdentificationSession),
	}
}

func (s *Service) StartIdentification(ctx context.Context, videoID string) (*IdentificationSession, error) {
	video, err := s.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("getting video: %w", err)
	}

	session := &IdentificationSession{
		ID:           uuid.New().String(),
		VideoID:      videoID,
		CurrentFrame: 0,
		Candidates:   []FilmCandidate{},
		UserFeedback: make(map[string]bool),
		Confidence:   0,
		Status:       "analyzing",
		StartedAt:    time.Now(),
		Updates:      make(chan SessionUpdate, 100),
	}

	s.sessionsMu.Lock()
	s.sessions[session.ID] = session
	s.sessionsMu.Unlock()

	go s.runIdentificationLoop(ctx, session, video)

	return session, nil
}

func (s *Service) GetSession(sessionID string) (*IdentificationSession, bool) {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	session, exists := s.sessions[sessionID]
	return session, exists
}

func (s *Service) UpdateFeedback(sessionID string, chip string, selected bool) error {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.UserFeedback[chip] = selected

	session.Updates <- SessionUpdate{
		Type: "feedback_updated",
		Data: map[string]interface{}{
			"chip":     chip,
			"selected": selected,
		},
	}

	return nil
}

func (s *Service) runIdentificationLoop(ctx context.Context, session *IdentificationSession, video *database.Video) {
	defer close(session.Updates)

	existingAnalyses, err := s.frameRepo.GetByVideoID(ctx, video.ID)
	if err != nil {
		log.Printf("Error getting existing analyses: %v", err)
		session.Status = "error"
		return
	}

	for frameNum := 0; frameNum < s.maxFramesAnalyze && session.Confidence < s.scoreThreshold; frameNum++ {
		session.CurrentFrame = frameNum

		var analysis *ai.FrameAnalysis

		if frameNum < len(existingAnalyses) {
			analysis = &existingAnalyses[frameNum]
		} else {
			videoPath, err := s.storageService.GetPath(video.StorageKey)
			if err != nil {
				log.Printf("Error getting video path: %v", err)
				continue
			}

			frames, err := s.frameExtractor.ExtractFrames(ctx, videoPath, frameNum+1)
			if err != nil {
				log.Printf("Error extracting frame %d: %v", frameNum, err)
				continue
			}

			if len(frames) == 0 {
				continue
			}

			frameAnalysis, err := s.visionService.AnalyzeFrame(ctx, frames[len(frames)-1])
			if err != nil {
				log.Printf("Error analyzing frame %d: %v", frameNum, err)
				continue
			}

			if err := s.frameRepo.Create(ctx, video.ID, frameNum, frameAnalysis); err != nil {
				log.Printf("Error saving frame analysis: %v", err)
			}

			analysis = &frameAnalysis
		}

		query := BuildSearchQuery(*analysis, session.UserFeedback)

		searchResults, err := s.searchClient.SearchFilms(ctx, query)
		if err != nil {
			log.Printf("Error searching films: %v", err)
			continue
		}

		candidates := s.processCandidates(ctx, searchResults, *analysis, session.UserFeedback)

		session.Candidates = candidates
		if len(candidates) > 0 {
			session.Confidence = candidates[0].Score
		}

		chips := s.extractChips(*analysis)
		session.Updates <- SessionUpdate{
			Type: "chips",
			Data: ChipData{
				SessionID: session.ID,
				Chips:     chips,
			},
		}

		session.Updates <- SessionUpdate{
			Type: "candidates",
			Data: map[string]interface{}{
				"candidates": candidates,
				"frame":      frameNum,
				"confidence": session.Confidence,
			},
		}

		if session.Confidence >= s.scoreThreshold && len(candidates) > 0 {
			filmDetails, err := s.tmdbClient.GetFilm(ctx, candidates[0].TMDbID)
			if err != nil {
				log.Printf("Error getting film details: %v", err)
			} else {
				now := time.Now()
				session.CompletedAt = &now
				session.Status = "complete"

				session.Updates <- SessionUpdate{
					Type: "complete",
					Data: IdentificationResult{
						SessionID:   session.ID,
						VideoID:     video.ID,
						FilmDetails: filmDetails,
						Confidence:  session.Confidence,
						FramesUsed:  frameNum + 1,
						TimeElapsed: time.Since(session.StartedAt),
					},
				}
				return
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	session.Status = "needs_input"
	session.Updates <- SessionUpdate{
		Type: "needs_input",
		Data: map[string]interface{}{
			"message": "Could not identify film with high confidence. Please try selecting more chips or uploading a different clip.",
		},
	}
}

func (s *Service) processCandidates(ctx context.Context, searchResults []ai.SearchResult, analysis ai.FrameAnalysis, feedback map[string]bool) []FilmCandidate {
	candidates := []FilmCandidate{}

	for _, result := range searchResults {
		tmdbID := extractTMDbID(result.Link)
		if tmdbID == "" {
			continue
		}

		year := extractYear(result.Title, result.Snippet)

		candidate := FilmCandidate{
			Title:     cleanTitle(result.Title),
			Year:      year,
			TMDbID:    tmdbID,
			Score:     0,
			MatchedOn: []string{},
			Source:    "google",
			Snippet:   result.Snippet,
		}

		candidate.Score = CalculateScore(candidate, analysis, feedback)

		candidates = append(candidates, candidate)
	}

	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].Score > candidates[i].Score {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	if len(candidates) > 5 {
		candidates = candidates[:5]
	}

	return candidates
}

func (s *Service) extractChips(analysis ai.FrameAnalysis) []Chip {
	chips := []Chip{}

	for _, face := range analysis.Faces {
		if face.Celebrity != "" {
			chips = append(chips, Chip{
				Value: face.Celebrity,
				Label: face.Celebrity,
				Type:  "actor",
			})
		}
	}

	if decade := detectDecade(analysis); decade != "" {
		chips = append(chips, Chip{
			Value: decade,
			Label: "Era: " + decade,
			Type:  "decade",
		})
	}

	genres := extractGenres(analysis)
	for _, genre := range genres {
		chips = append(chips, Chip{
			Value: genre,
			Label: strings.Title(genre),
			Type:  "genre",
		})
	}

	objects := extractSignificantObjects(analysis)
	for _, obj := range objects {
		chips = append(chips, Chip{
			Value: obj,
			Label: strings.Title(obj),
			Type:  "object",
		})
	}

	return chips
}

func extractTMDbID(link string) string {
	if strings.Contains(link, "themoviedb.org/movie/") {
		parts := strings.Split(link, "/movie/")
		if len(parts) > 1 {
			idParts := strings.Split(parts[1], "-")
			if len(idParts) > 0 {
				return idParts[0]
			}
		}
	}
	return ""
}

func extractYear(title, snippet string) int {
	combined := title + " " + snippet
	yearPattern := `\b(19|20)\d{2}\b`
	re := regexp.MustCompile(yearPattern)
	matches := re.FindAllString(combined, -1)

	for _, match := range matches {
		year, err := strconv.Atoi(match)
		if err == nil && year >= 1900 && year <= 2030 {
			return year
		}
	}

	return 0
}

func cleanTitle(title string) string {
	title = strings.Split(title, " - ")[0]
	title = strings.TrimSuffix(title, " - IMDb")
	title = strings.TrimSuffix(title, " - Wikipedia")
	title = strings.TrimSuffix(title, " - TMDb")
	return strings.TrimSpace(title)
}

func extractGenres(analysis ai.FrameAnalysis) []string {
	genres := []string{}
	genreKeywords := map[string]bool{
		"action": true, "comedy": true, "drama": true, "horror": true,
		"sci-fi": true, "thriller": true, "romance": true, "adventure": true,
		"fantasy": true, "mystery": true, "crime": true, "animation": true,
	}

	text := strings.ToLower(analysis.Caption)
	for _, label := range analysis.Labels {
		text += " " + strings.ToLower(label.Description)
	}

	for genre := range genreKeywords {
		if strings.Contains(text, genre) {
			genres = append(genres, genre)
		}
	}

	return genres
}

func extractSignificantObjects(analysis ai.FrameAnalysis) []string {
	objects := []string{}
	significantLabels := map[string]bool{
		"car": true, "gun": true, "explosion": true, "spaceship": true,
		"robot": true, "monster": true, "castle": true, "sword": true,
	}

	for _, label := range analysis.Labels {
		labelLower := strings.ToLower(label.Description)
		if significantLabels[labelLower] && label.Score > 0.7 {
			objects = append(objects, labelLower)
		}
	}

	return objects
}
