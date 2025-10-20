package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/kdimtricp/vshazam/internal/ai"
	"github.com/kdimtricp/vshazam/internal/api"
	"github.com/kdimtricp/vshazam/internal/database"
	"github.com/kdimtricp/vshazam/internal/storage"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	maxUploadSize := os.Getenv("MAX_UPLOAD_SIZE")
	if maxUploadSize == "" {
		maxUploadSize = "104857600"
	}
	maxSize, err := strconv.ParseInt(maxUploadSize, 10, 64)
	if err != nil {
		log.Fatal("Invalid MAX_UPLOAD_SIZE:", err)
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	// Database configuration
	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = "sqlite"
	}

	var dbConfig database.Config
	dbConfig.Type = dbType

	if dbType == "postgres" {
		dbConfig.Host = os.Getenv("DB_HOST")
		if dbConfig.Host == "" {
			dbConfig.Host = "localhost"
		}

		dbPortStr := os.Getenv("DB_PORT")
		if dbPortStr == "" {
			dbPortStr = "5432"
		}
		dbPort, err := strconv.Atoi(dbPortStr)
		if err != nil {
			log.Fatal("Invalid DB_PORT:", err)
		}
		dbConfig.Port = dbPort

		dbConfig.User = os.Getenv("DB_USER")
		if dbConfig.User == "" {
			dbConfig.User = "vshazam"
		}

		dbConfig.Password = os.Getenv("DB_PASSWORD")
		if dbConfig.Password == "" {
			dbConfig.Password = "vshazam_dev"
		}

		dbConfig.Name = os.Getenv("DB_NAME")
		if dbConfig.Name == "" {
			dbConfig.Name = "vshazam"
		}
	} else {
		dbConfig.SQLitePath = os.Getenv("DB_PATH")
		if dbConfig.SQLitePath == "" {
			dbConfig.SQLitePath = "./vshazam.db"
		}
	}

	localStorage, err := storage.NewLocalStorage(uploadDir)
	if err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}

	db, err := database.NewDB(dbConfig)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "./migrations"
	}

	log.Printf("Running database migrations from %s", migrationsPath)
	if err := db.RunMigrations(migrationsPath); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	videoRepo := database.NewVideoRepository(db)
	frameRepo := database.NewFrameAnalysisRepo(db)

	aiConfig := &ai.Config{
		OpenAIAPIKey:               os.Getenv("OPENAI_API_KEY"),
		GoogleVisionKey:            os.Getenv("GOOGLE_VISION_API_KEY"),
		GoogleVisionServiceAccount: os.Getenv("GOOGLE_VISION_SERVICE_ACCOUNT"),
		GoogleSearchAPIKey:         os.Getenv("GOOGLE_SEARCH_API_KEY"),
		GoogleCSEID:                os.Getenv("GOOGLE_CSE_ID"),
		TMDbAPIKey:                 os.Getenv("TMDB_API_KEY"),
	}

	maxFramesStr := os.Getenv("MAX_FRAMES_PER_VIDEO")
	if maxFramesStr != "" {
		if maxFrames, err := strconv.Atoi(maxFramesStr); err == nil {
			aiConfig.MaxFramesPerVideo = maxFrames
		}
	}
	if aiConfig.MaxFramesPerVideo == 0 {
		aiConfig.MaxFramesPerVideo = 5
	}

	frameSizeStr := os.Getenv("FRAME_SIZE")
	if frameSizeStr != "" {
		if frameSize, err := strconv.Atoi(frameSizeStr); err == nil {
			aiConfig.FrameSize = frameSize
		}
	}
	if aiConfig.FrameSize == 0 {
		aiConfig.FrameSize = 512
	}

	var visionService ai.VisionService
	var frameExtractor *ai.FrameExtractor

	if aiConfig.OpenAIAPIKey != "" || aiConfig.GoogleVisionKey != "" || aiConfig.GoogleVisionServiceAccount != "" {
		visionService, err = ai.NewVisionService(aiConfig)
		if err != nil {
			log.Printf("Warning: Failed to initialize vision service: %v", err)
		} else {
			frameExtractor, err = ai.NewFrameExtractor()
			if err != nil {
				log.Printf("Warning: Failed to initialize frame extractor: %v", err)
			}
		}
	} else {
		log.Printf("AI services not configured. Set at least one: OPENAI_API_KEY, GOOGLE_VISION_API_KEY, or GOOGLE_VISION_SERVICE_ACCOUNT")
	}

	app := &api.App{
		Storage:        localStorage,
		DB:             db,
		VideoRepo:      videoRepo,
		FrameRepo:      frameRepo,
		MaxUploadSize:  maxSize,
		VisionService:  visionService,
		FrameExtractor: frameExtractor,
		AIConfig:       aiConfig,
	}

	router := api.NewRouter(app)

	log.Printf("Server starting on port %s", port)
	log.Printf("Upload directory: %s", uploadDir)
	log.Printf("Database type: %s", dbType)
	if dbType == "postgres" {
		log.Printf("Database connection: %s@%s:%d/%s", dbConfig.User, dbConfig.Host, dbConfig.Port, dbConfig.Name)
	} else {
		log.Printf("Database path: %s", dbConfig.SQLitePath)
	}
	log.Printf("Max upload size: %d bytes", maxSize)

	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}
