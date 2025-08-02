package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kdimtricp/vshazam/internal/ai"
	"github.com/kdimtricp/vshazam/internal/database"
)

func main() {
	var videoID = flag.String("id", "", "Video ID to analyze")
	flag.Parse()

	if *videoID == "" {
		log.Fatal("Please provide video ID with -id flag")
	}

	// Database config
	dbConfig := database.Config{
		Type:     getEnv("DB_TYPE", "postgres"),
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "vshazam"),
		Password: getEnv("DB_PASSWORD", "vshazam_dev"),
		Name:     getEnv("DB_NAME", "vshazam"),
	}

	db, err := database.NewDB(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Get video info
	videoRepo := database.NewVideoRepository(db)
	video, err := videoRepo.GetVideoByID(*videoID)
	if err != nil {
		log.Fatal("Failed to get video:", err)
	}

	fmt.Printf("Analyzing video: %s\n", video.Title)

	// Initialize AI
	aiConfig := &ai.Config{
		OpenAIAPIKey:      os.Getenv("OPENAI_API_KEY"),
		GoogleVisionKey:   os.Getenv("GOOGLE_VISION_API_KEY"),
		MaxFramesPerVideo: 3,
		FrameSize:         512,
	}

	if aiConfig.OpenAIAPIKey == "" && aiConfig.GoogleVisionKey == "" {
		log.Fatal("No AI API keys configured")
	}

	visionService, err := ai.NewVisionService(aiConfig)
	if err != nil {
		log.Fatal("Failed to initialize vision service:", err)
	}

	frameExtractor, err := ai.NewFrameExtractor()
	if err != nil {
		log.Fatal("Failed to initialize frame extractor:", err)
	}

	// Extract and analyze frames
	videoPath := filepath.Join(getEnv("UPLOAD_DIR", "./uploads"), video.Filename)
	frames, err := frameExtractor.ExtractFrames(videoPath, aiConfig.MaxFramesPerVideo, aiConfig.FrameSize)
	if err != nil {
		log.Fatal("Failed to extract frames:", err)
	}

	fmt.Printf("Extracted %d frames\n", len(frames))

	frameRepo := database.NewFrameAnalysisRepo(db)
	ctx := context.Background()

	for i, frameData := range frames {
		fmt.Printf("Analyzing frame %d...\n", i)

		analysis, err := visionService.AnalyzeFrame(ctx, frameData)
		if err != nil {
			log.Printf("Failed to analyze frame %d: %v", i, err)
			continue
		}

		dbAnalysis, err := analysis.ToDB(video.ID, i)
		if err != nil {
			log.Printf("Failed to convert analysis: %v", err)
			continue
		}

		if err := frameRepo.Create(ctx, dbAnalysis); err != nil {
			log.Printf("Failed to save analysis: %v", err)
			continue
		}

		fmt.Printf("✓ Frame %d analyzed (confidence: %.2f)\n", i, analysis.Confidence)
		if analysis.Caption != "" {
			fmt.Printf("  Caption: %.100s...\n", analysis.Caption)
		}
	}

	fmt.Println("Analysis complete!")
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
