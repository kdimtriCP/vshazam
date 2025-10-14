package identification

import (
	"context"
	"encoding/json"
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
	"github.com/kdimtricp/vshazam/internal/models"
	"github.com/kdimtricp/vshazam/internal/storage"
)

type Service struct {
	visionService    ai.VisionService
	searchClient     *ai.GoogleSearchClient
	tmdbClient       *TMDbClient
	frameExtractor   *ai.FrameExtractor
	videoRepo        *database.VideoRepository
	frameRepo        *database.FrameAnalysisRepo
	storageService   storage.Storage
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
	videoRepo *database.VideoRepository,
	frameRepo *database.FrameAnalysisRepo,
	storageService storage.Storage,
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
	video, err := s.videoRepo.GetVideoByID(videoID)
	if err != nil {
		return nil, fmt.Errorf("getting video: %w", err)
	}

	loopCtx, cancel := context.WithCancel(context.Background())

	session := &IdentificationSession{
		ID:              uuid.New().String(),
		VideoID:         videoID,
		CurrentFrame:    0,
		Candidates:      []FilmCandidate{},
		UserFeedback:    make(map[string]bool),
		Confidence:      0,
		Status:          "analyzing",
		StartedAt:       time.Now(),
		Updates:         make(chan SessionUpdate, 100),
		FeedbackChanged: make(chan struct{}, 1),
		CancelFunc:      cancel,
	}

	s.sessionsMu.Lock()
	s.sessions[session.ID] = session
	s.sessionsMu.Unlock()

	go s.runIdentificationLoop(loopCtx, session, video)

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

	select {
	case session.FeedbackChanged <- struct{}{}:
		log.Printf("[IDENT] Signaled feedback change for session %s", sessionID)
	default:
	}

	return nil
}

func (s *Service) StopIdentification(sessionID string) error {
	s.sessionsMu.Lock()
	session, exists := s.sessions[sessionID]
	s.sessionsMu.Unlock()

	if !exists {
		return fmt.Errorf("session not found")
	}

	if session.CancelFunc != nil {
		log.Printf("[IDENT] Stopping identification for session %s", sessionID)
		session.CancelFunc()
	}

	return nil
}

func (s *Service) runIdentificationLoop(ctx context.Context, session *IdentificationSession, video *models.Video) {
	defer close(session.Updates)

	log.Printf("[IDENT] Starting identification for video %s, session %s", video.ID, session.ID)

	existingAnalysesDB, err := s.frameRepo.GetByVideoID(ctx, video.ID)
	if err != nil {
		log.Printf("[IDENT] Error getting existing analyses: %v", err)
		session.Status = "error"
		return
	}

	log.Printf("[IDENT] Found %d existing frame analyses", len(existingAnalysesDB))

	for frameNum := 0; frameNum < s.maxFramesAnalyze && session.Confidence < s.scoreThreshold; frameNum++ {
		session.CurrentFrame = frameNum
		log.Printf("[IDENT] Processing frame %d/%d", frameNum+1, s.maxFramesAnalyze)

		var analysis ai.FrameAnalysis

		if frameNum < len(existingAnalysesDB) {
			dbAnalysis := existingAnalysesDB[frameNum]
			analysis = ai.FrameAnalysis{
				Caption:    dbAnalysis.GPTCaption,
				TextOCR:    dbAnalysis.OCRText,
				Faces:      []ai.FaceDetection{},
				Confidence: 0.8,
			}

			if len(dbAnalysis.VisionLabels) > 0 {
				if err := json.Unmarshal(dbAnalysis.VisionLabels, &analysis.Labels); err != nil {
					log.Printf("Error unmarshaling labels: %v", err)
				}
			}
		} else {
			videoPath := s.storageService.GetFilePath(video.Filename)
			frames, err := s.frameExtractor.ExtractFrames(videoPath, frameNum+1, 512)
			if err != nil {
				log.Printf("[IDENT] Error extracting frame %d: %v", frameNum, err)
				continue
			}

			if len(frames) == 0 {
				continue
			}

			frameAnalysisPtr, err := s.visionService.AnalyzeFrame(ctx, frames[len(frames)-1])
			if err != nil {
				log.Printf("Error analyzing frame %d: %v", frameNum, err)
				continue
			}

			dbAnalysis, err := frameAnalysisPtr.ToDB(video.ID, frameNum)
			if err != nil {
				log.Printf("Error converting analysis to DB format: %v", err)
			} else if err := s.frameRepo.Create(ctx, dbAnalysis); err != nil {
				log.Printf("Error saving frame analysis: %v", err)
			}

			analysis = *frameAnalysisPtr
		}

	feedbackReactiveLoop:
		for {
			select {
			case <-ctx.Done():
				session.Status = "cancelled"
				session.Updates <- SessionUpdate{
					Type: "cancelled",
					Data: map[string]interface{}{
						"message": "Identification cancelled by user",
					},
				}
				return
			case <-session.FeedbackChanged:
				log.Printf("[IDENT] Feedback changed, re-searching with updated parameters")
			default:
			}

			query := BuildSearchQuery(analysis, session.UserFeedback)
			log.Printf("[IDENT] Search query: %s", query)

			searchResults, err := s.searchClient.SearchFilms(ctx, query)
			if err != nil {
				log.Printf("[IDENT] Error searching films: %v", err)
				break feedbackReactiveLoop
			}

			log.Printf("[IDENT] Found %d search results", len(searchResults))

			candidates := s.processCandidates(ctx, searchResults, analysis, session.UserFeedback)

			session.Candidates = candidates
			if len(candidates) > 0 {
				session.Confidence = candidates[0].Score
				log.Printf("[IDENT] Generated %d candidates, top candidate: %s (score: %.2f)",
					len(candidates), candidates[0].Title, candidates[0].Score)
			} else {
				log.Printf("[IDENT] No candidates generated from search results")
			}

			chips := s.extractChips(analysis, session.UserFeedback)
			log.Printf("[IDENT] Extracted %d chips from frame analysis", len(chips))
			session.Updates <- SessionUpdate{
				Type: "chips",
				Data: ChipData{
					SessionID: session.ID,
					Chips:     chips,
				},
			}
			log.Printf("[IDENT] Sent chips update via SSE")

			session.Updates <- SessionUpdate{
				Type: "candidates",
				Data: map[string]interface{}{
					"candidates": candidates,
					"frame":      frameNum,
					"confidence": session.Confidence,
				},
			}
			log.Printf("[IDENT] Sent candidates update via SSE")

			if session.Confidence >= s.scoreThreshold && len(candidates) > 0 {
				log.Printf("[IDENT] Confidence threshold reached (%.2f >= %.2f), fetching TMDb details for: %s",
					session.Confidence, s.scoreThreshold, candidates[0].Title)
				filmDetails, err := s.tmdbClient.GetFilm(ctx, candidates[0].TMDbID)
				if err != nil {
					log.Printf("[IDENT] Error getting film details: %v", err)
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
					log.Printf("[IDENT] Identification complete! Film: %s, Confidence: %.2f, Frames: %d, Time: %v",
						filmDetails.Title, session.Confidence, frameNum+1, time.Since(session.StartedAt))
					return
				}
			}

			select {
			case <-ctx.Done():
				session.Status = "cancelled"
				return
			case <-session.FeedbackChanged:
				log.Printf("[IDENT] Feedback changed while waiting, re-searching immediately")
				continue
			case <-time.After(500 * time.Millisecond):
				break feedbackReactiveLoop
			}
		}
	}

	log.Printf("[IDENT] Identification loop completed without reaching threshold. Final confidence: %.2f, Frames used: %d",
		session.Confidence, s.maxFramesAnalyze)

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
			log.Printf("[IDENT] Skipping result without TMDb ID: %s", result.Link)
			continue
		}

		log.Printf("[IDENT] Processing candidate: %s (TMDb ID: %s)", result.Title, tmdbID)

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

func (s *Service) extractChips(analysis ai.FrameAnalysis, userFeedback map[string]bool) []Chip {
	chips := []Chip{}

	if decade := detectDecade(analysis); decade != "" {
		chips = append(chips, Chip{
			Value:    decade,
			Label:    "Era: " + decade,
			Type:     "decade",
			Selected: userFeedback[decade],
		})
	}

	genres := extractGenres(analysis)
	for _, genre := range genres {
		chips = append(chips, Chip{
			Value:    genre,
			Label:    strings.Title(genre),
			Type:     "genre",
			Selected: userFeedback[genre],
		})
	}

	objects := extractSignificantObjects(analysis)
	for _, obj := range objects {
		chips = append(chips, Chip{
			Value:    obj,
			Label:    strings.Title(obj),
			Type:     "object",
			Selected: userFeedback[obj],
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
		text += " " + strings.ToLower(label.Name)
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
		labelLower := strings.ToLower(label.Name)
		if significantLabels[labelLower] && label.Confidence > 0.7 {
			objects = append(objects, labelLower)
		}
	}

	return objects
}
