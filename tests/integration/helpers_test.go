package integration

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdimtricp/vshazam/internal/api"
	"github.com/kdimtricp/vshazam/internal/database"
	"github.com/kdimtricp/vshazam/internal/storage"
)

type TestServer struct {
	Server      *httptest.Server
	App         *api.App
	DB          *database.DB
	VideoRepo   *database.VideoRepository
	Storage     storage.Storage
	TempDir     string
	OriginalDir string
}

func setupTestServer(t *testing.T) *TestServer {
	// Change to project root directory to find templates
	originalDir, _ := os.Getwd()
	projectRoot := filepath.Join(originalDir, "../..")
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}
	
	// Create temp directory for uploads
	tempDir, err := os.MkdirTemp("", "vshazam_test_*")
	if err != nil {
		os.Chdir(originalDir)
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create upload directory
	uploadDir := filepath.Join(tempDir, "uploads")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		os.Chdir(originalDir)
		t.Fatalf("Failed to create upload dir: %v", err)
	}

	// Initialize storage
	localStorage, err := storage.NewLocalStorage(uploadDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create test database
	dbPath := filepath.Join(tempDir, "test.db")
	dbConfig := database.Config{
		Type:       "sqlite",
		SQLitePath: dbPath,
	}

	db, err := database.NewDB(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	videoRepo := database.NewVideoRepository(db)

	// Create app
	app := &api.App{
		Storage:       localStorage,
		DB:            db,
		VideoRepo:     videoRepo,
		MaxUploadSize: 10 * 1024 * 1024, // 10MB
	}

	// Create router and server
	router := api.NewRouter(app)
	server := httptest.NewServer(router)

	return &TestServer{
		Server:      server,
		App:         app,
		DB:          db,
		VideoRepo:   videoRepo,
		Storage:     localStorage,
		TempDir:     tempDir,
		OriginalDir: originalDir,
	}
}

func (ts *TestServer) Cleanup() {
	ts.Server.Close()
	ts.DB.Close()
	os.RemoveAll(ts.TempDir)
	// Return to original directory
	os.Chdir(ts.OriginalDir)
}

func createMultipartUpload(title, description, filename string, content []byte) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add video file
	part, err := writer.CreateFormFile("video", filename)
	if err != nil {
		return nil, "", err
	}
	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		return nil, "", err
	}

	// Add title
	if err := writer.WriteField("title", title); err != nil {
		return nil, "", err
	}

	// Add description
	if err := writer.WriteField("description", description); err != nil {
		return nil, "", err
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return body, writer.FormDataContentType(), nil
}

func countVideosInDB(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&count)
	return count, err
}

func uploadTestVideo(t *testing.T, server string, title, description string) *http.Response {
	// Create a simple test video content
	content := []byte("fake mp4 content for testing")
	body, contentType, err := createMultipartUpload(title, description, "test.mp4", content)
	if err != nil {
		t.Fatalf("Failed to create multipart upload: %v", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/upload", server), body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to upload video: %v", err)
	}

	return resp
}