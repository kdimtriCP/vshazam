package api

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kdimtricp/vshazam/internal/ai"
	"github.com/kdimtricp/vshazam/internal/database"
	"github.com/kdimtricp/vshazam/internal/models"
	"github.com/kdimtricp/vshazam/internal/storage"
)

type App struct {
	Storage        storage.Storage
	DB             *database.DB
	VideoRepo      *database.VideoRepository
	FrameRepo      *database.FrameAnalysisRepo
	MaxUploadSize  int64
	VisionService  ai.VisionService
	FrameExtractor *ai.FrameExtractor
	AIConfig       *ai.Config
}

func PingHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	tmplPath := filepath.Join("web", "templates", "base.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	data := struct {
		Title   string
		Message string
	}{
		Title:   "VShazam",
		Message: "Hello, VShazam!",
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}

func (app *App) UploadPageHandler(w http.ResponseWriter, r *http.Request) {
	tmplPath := filepath.Join("web", "templates", "upload.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}

func (app *App) UploadHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, app.MaxUploadSize)

	if err := r.ParseMultipartForm(app.MaxUploadSize); err != nil {
		app.renderError(w, "File too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		app.renderError(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "video/") && contentType != "application/octet-stream" {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext != ".mp4" {
			app.renderError(w, "Only MP4 video files are allowed", http.StatusBadRequest)
			return
		}
		contentType = "video/mp4"
	}

	title := r.FormValue("title")
	if title == "" {
		app.renderError(w, "Title is required", http.StatusBadRequest)
		return
	}

	description := r.FormValue("description")

	filename, err := app.Storage.SaveFile(file, storage.FileInfo{
		Filename:    header.Filename,
		ContentType: contentType,
		Size:        header.Size,
	})
	if err != nil {
		app.renderError(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	video := models.NewVideo(title, description, filename, contentType, header.Size)
	if err := app.VideoRepo.InsertVideo(video); err != nil {
		app.Storage.DeleteFile(filename)
		app.renderError(w, "Failed to save video information", http.StatusInternalServerError)
		return
	}

	app.renderSuccess(w, "Video uploaded successfully!")
	w.Header().Set("HX-Trigger", "videoUploaded")
}

func (app *App) VideoListPartialHandler(w http.ResponseWriter, r *http.Request) {
	videos, err := app.VideoRepo.ListVideos()
	if err != nil {
		w.Write([]byte("<p>Error loading videos</p>"))
		return
	}

	if len(videos) == 0 {
		w.Write([]byte("<p>No videos uploaded yet</p>"))
		return
	}

	tmplPath := filepath.Join("web", "templates", "_video_item.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		app.renderError(w, "Error loading template", http.StatusInternalServerError)
	}

	for _, video := range videos {
		tmpl.Execute(w, video)
	}
}

func (app *App) ListVideosHandler(w http.ResponseWriter, r *http.Request) {
	videos, err := app.VideoRepo.ListVideos()
	if err != nil {
		http.Error(w, "Error loading videos", http.StatusInternalServerError)
		return
	}

	tmplPath := filepath.Join("web", "templates", "list.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	data := struct {
		Videos   []models.Video
		Query    string
		IsSearch bool
	}{
		Videos:   videos,
		Query:    "",
		IsSearch: false,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}

func (app *App) WatchVideoHandler(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "id")
	if videoID == "" {
		app.renderError(w, "Video ID is required", http.StatusBadRequest)
		return
	}

	video, err := app.VideoRepo.GetVideoByID(videoID)
	if err != nil {
		app.renderError(w, "Error loading video", http.StatusInternalServerError)
		return
	}

	tmplPath := filepath.Join("web", "templates", "video.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	data := struct {
		Video         *models.Video
		FormattedSize string
	}{
		Video:         video,
		FormattedSize: storage.FormatFileSize(video.Size),
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}

func (app *App) StreamVideoHandler(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "id")
	if videoID == "" {
		http.NotFound(w, r)
		return
	}

	video, err := app.VideoRepo.GetVideoByID(videoID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	file, err := app.Storage.OpenFile(video.Filename)
	if err != nil {
		http.Error(w, "Video file not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Get file info for ServeContent
	stat, err := file.(interface{ Stat() (os.FileInfo, error) }).Stat()
	if err != nil {
		http.Error(w, "Error accessing video file", http.StatusInternalServerError)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", video.ContentType)

	// ServeContent handles Range requests automatically
	// It sets proper headers including Accept-Ranges, Content-Length, and handles 206 Partial Content
	http.ServeContent(w, r, video.Filename, stat.ModTime(), file)
}

func (app *App) SearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	videos, err := app.VideoRepo.SearchVideos(query)
	if err != nil {
		http.Error(w, "Error searching videos", http.StatusInternalServerError)
		return
	}

	// If it's an HTMX request, return only the results partial
	if r.Header.Get("HX-Request") == "true" {
		tmplPath := filepath.Join("web", "templates", "search_results.html")
		tmpl, err := template.ParseFiles(tmplPath)
		if err != nil {
			w.Write([]byte("<p>Error loading search results</p>"))
			return
		}

		data := struct {
			Videos []models.Video
			Query  string
		}{
			Videos: videos,
			Query:  query,
		}

		if err := tmpl.Execute(w, data); err != nil {
			w.Write([]byte("<p>Error rendering search results</p>"))
			return
		}
		return
	}

	// Otherwise, render the full page with search results
	tmplPath := filepath.Join("web", "templates", "list.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	data := struct {
		Videos   []models.Video
		Query    string
		IsSearch bool
	}{
		Videos:   videos,
		Query:    query,
		IsSearch: query != "",
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}

func (app *App) renderError(w http.ResponseWriter, message string, code int) {
	w.WriteHeader(code)
	fmt.Fprintf(w, `<div class="alert alert-error">%s</div>`, template.HTMLEscapeString(message))
}

func (app *App) renderSuccess(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<div class="alert alert-success">%s</div>`, template.HTMLEscapeString(message))
}
