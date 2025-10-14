package identification

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/kdimtricp/vshazam/internal/ai"
)

func CalculateScore(candidate FilmCandidate, analysis ai.FrameAnalysis, feedback map[string]bool) float64 {
	score := 0.0

	weights := map[string]float64{
		"snippet_similarity": 0.5,
		"actor_match":        0.3,
		"decade_match":       0.1,
		"genre_match":        0.1,
	}

	score += weights["snippet_similarity"] * calculateTextSimilarity(candidate.Title, candidate.Snippet, analysis.Caption)

	if actorMatch(candidate, analysis.Faces) {
		score += weights["actor_match"]
		candidate.MatchedOn = append(candidate.MatchedOn, "actor")
	}

	if decadeMatch(candidate.Year, analysis) {
		score += weights["decade_match"]
		candidate.MatchedOn = append(candidate.MatchedOn, "decade")
	}

	if genreMatch(candidate, analysis) {
		score += weights["genre_match"]
		candidate.MatchedOn = append(candidate.MatchedOn, "genre")
	}

	for chip, selected := range feedback {
		if selected && contains(candidate.MatchedOn, chip) {
			score += 0.2
		}
	}

	return math.Min(score, 1.0)
}

func calculateTextSimilarity(title, snippet, caption string) float64 {
	titleLower := strings.ToLower(title)
	snippetLower := strings.ToLower(snippet)
	captionLower := strings.ToLower(caption)

	titleWords := strings.Fields(titleLower)
	captionWords := strings.Fields(captionLower)

	matchScore := 0.0

	for _, word := range titleWords {
		if len(word) > 3 && (strings.Contains(captionLower, word) || strings.Contains(snippetLower, word)) {
			matchScore += 0.2
		}
	}

	for _, word := range captionWords {
		if len(word) > 5 && strings.Contains(snippetLower, word) {
			matchScore += 0.1
		}
	}

	return math.Min(matchScore, 1.0)
}

func actorMatch(candidate FilmCandidate, faces []ai.FaceDetection) bool {
	return len(faces) > 0 && len(faces) <= 5
}

func decadeMatch(candidateYear int, analysis ai.FrameAnalysis) bool {
	decadeStr := detectDecade(analysis)
	if decadeStr == "" {
		return false
	}

	re := regexp.MustCompile(`\d{4}`)
	matches := re.FindAllString(decadeStr, -1)

	for _, match := range matches {
		year, err := strconv.Atoi(match)
		if err == nil {
			candidateDecade := (candidateYear / 10) * 10
			detectedDecade := (year / 10) * 10
			if math.Abs(float64(candidateDecade-detectedDecade)) <= 10 {
				return true
			}
		}
	}

	return false
}

func detectDecade(analysis ai.FrameAnalysis) string {
	combined := strings.ToLower(analysis.Caption)
	for _, label := range analysis.Labels {
		combined += " " + strings.ToLower(label.Name)
	}

	decades := []string{"1920s", "1930s", "1940s", "1950s", "1960s", "1970s", "1980s", "1990s", "2000s", "2010s", "2020s"}
	for _, decade := range decades {
		if strings.Contains(combined, decade) {
			return decade
		}
	}

	re := regexp.MustCompile(`(19|20)\d{2}`)
	matches := re.FindAllString(combined, -1)
	if len(matches) > 0 {
		return matches[0]
	}

	return ""
}

func genreMatch(candidate FilmCandidate, analysis ai.FrameAnalysis) bool {
	genres := []string{"action", "comedy", "drama", "horror", "sci-fi", "thriller", "romance", "adventure", "fantasy", "mystery"}

	candidateLower := strings.ToLower(candidate.Snippet)
	analysisText := strings.ToLower(analysis.Caption)
	for _, label := range analysis.Labels {
		analysisText += " " + strings.ToLower(label.Name)
	}

	for _, genre := range genres {
		if strings.Contains(candidateLower, genre) && strings.Contains(analysisText, genre) {
			return true
		}
	}

	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func extractKeywords(caption string) []string {
	skipPhrases := []string{
		"i'm unable to identify",
		"unable to identify",
		"unable identify",
		"cannot identify",
		"can't identify",
		"i cannot",
		"specific movie",
		"specific movies",
		"recognize people",
		"however provide",
		"however give",
	}

	captionLower := strings.ToLower(caption)
	for _, phrase := range skipPhrases {
		captionLower = strings.ReplaceAll(captionLower, phrase, "")
	}

	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "is": true, "are": true,
		"was": true, "were": true, "been": true, "be": true, "this": true,
		"that": true, "from": true, "have": true, "has": true, "had": true,
		"will": true, "would": true, "could": true, "should": true,
		"images": true, "image": true, "frame": true, "scene": true, "provide": true,
	}

	words := strings.Fields(captionLower)
	keywords := []string{}

	for _, word := range words {
		word = strings.Trim(word, ".,!?;:'\"*")
		if len(word) > 4 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

func BuildSearchQuery(analysis ai.FrameAnalysis, feedback map[string]bool) string {
	parts := []string{}

	keywords := extractKeywords(analysis.Caption)
	if len(keywords) > 8 {
		keywords = keywords[:8]
	}
	parts = append(parts, keywords...)

	if decade := detectDecade(analysis); decade != "" {
		parts = append(parts, decade)
	}

	genres := extractGenres(analysis)
	if len(genres) > 0 {
		parts = append(parts, genres[0])
	}

	for chip, selected := range feedback {
		if selected {
			parts = append(parts, chip)
		}
	}

	uniqueParts := make(map[string]bool)
	result := []string{}
	for _, part := range parts {
		if !uniqueParts[part] {
			uniqueParts[part] = true
			result = append(result, part)
		}
	}

	query := strings.Join(result, " ")
	return query + " movie site:themoviedb.org"
}
