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

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./vshazam.db"
	}

	localStorage, err := storage.NewLocalStorage(uploadDir)
	if err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}

	db, err := database.NewDB(dbPath)
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
	log.Printf("Database path: %s", dbPath)
	log.Printf("Max upload size: %d bytes", maxSize)

	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}
