package api

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kdimtricp/vshazam/internal/database"
	"github.com/kdimtricp/vshazam/internal/models"
	"github.com/kdimtricp/vshazam/internal/storage"
)

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

type App struct {
	Storage       storage.Storage
	DB            *database.DB
	VideoRepo     *database.VideoRepository
	MaxUploadSize int64
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
		app.renderError(w, "File too large")
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		app.renderError(w, "Failed to get file")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "video/") && contentType != "application/octet-stream" {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext != ".mp4" {
			app.renderError(w, "Only MP4 video files are allowed")
			return
		}
		contentType = "video/mp4"
	}

	title := r.FormValue("title")
	if title == "" {
		app.renderError(w, "Title is required")
		return
	}

	description := r.FormValue("description")

	filename, err := app.Storage.SaveFile(file, storage.FileInfo{
		Filename:    header.Filename,
		ContentType: contentType,
		Size:        header.Size,
	})
	if err != nil {
		app.renderError(w, "Failed to save file")
		return
	}

	video := models.NewVideo(title, description, filename, contentType, header.Size)
	if err := app.VideoRepo.InsertVideo(video); err != nil {
		app.Storage.DeleteFile(filename)
		app.renderError(w, "Failed to save video information")
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
		for _, video := range videos {
			fmt.Fprintf(w, `<div class="video-item">
				<h4>%s</h4>
				<p>%s</p>
				<small>Size: %s | Uploaded: %s</small>
			</div>`,
				template.HTMLEscapeString(video.Title),
				template.HTMLEscapeString(video.Description),
				formatFileSize(video.Size),
				video.UploadTime.Format("Jan 2, 2006 15:04"))
		}
		return
	}

	for _, video := range videos {
		tmpl.Execute(w, video)
	}
}

func (app *App) renderError(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, `<div class="alert alert-error">%s</div>`, template.HTMLEscapeString(message))
}

func (app *App) renderSuccess(w http.ResponseWriter, message string) {
	fmt.Fprintf(w, `<div class="alert alert-success">%s</div>`, template.HTMLEscapeString(message))
}

func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
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
		Videos []models.Video
	}{
		Videos: videos,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}

func (app *App) WatchVideoHandler(w http.ResponseWriter, r *http.Request) {
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
		FormattedSize: formatFileSize(video.Size),
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
