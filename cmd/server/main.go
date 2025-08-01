package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

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

	videoRepo := database.NewVideoRepository(db)

	app := &api.App{
		Storage:       localStorage,
		DB:            db,
		VideoRepo:     videoRepo,
		MaxUploadSize: maxSize,
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
