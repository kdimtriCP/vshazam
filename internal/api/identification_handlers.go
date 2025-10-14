package api

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kdimtricp/vshazam/internal/identification"
)

type IdentificationHandlers struct {
	identService *identification.Service
	templatePath string
}

func NewIdentificationHandlers(identService *identification.Service) *IdentificationHandlers {
	return &IdentificationHandlers{
		identService: identService,
		templatePath: filepath.Join("web", "templates"),
	}
}

func (h *IdentificationHandlers) StartIdentificationHandler(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "id")

	session, err := h.identService.StartIdentification(r.Context(), videoID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start identification: %v", err), http.StatusInternalServerError)
		return
	}

	tmplPath := filepath.Join(h.templatePath, "identify.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	data := struct {
		VideoID   string
		SessionID string
		Status    string
		Progress  int
	}{
		VideoID:   videoID,
		SessionID: session.ID,
		Status:    "Starting analysis...",
		Progress:  0,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}

func (h *IdentificationHandlers) IdentificationStreamHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	session, exists := h.identService.GetSession(sessionID)
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	clientGone := r.Context().Done()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	log.Printf("[SSE] Client connected to session %s", sessionID)

	for {
		select {
		case update, ok := <-session.Updates:
			if !ok {
				log.Printf("[SSE] Session %s updates channel closed", sessionID)
				return
			}

			data, err := json.Marshal(update.Data)
			if err != nil {
				log.Printf("[SSE] Error marshaling update: %v", err)
				continue
			}

			log.Printf("[SSE] Sending event '%s' to session %s", update.Type, sessionID)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", update.Type, string(data))
			flusher.Flush()

		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()

		case <-clientGone:
			log.Printf("[SSE] Client disconnected from session %s", sessionID)
			return
		}
	}
}

func (h *IdentificationHandlers) UpdateFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	chip := r.FormValue("chip")
	selected := r.FormValue("selected") == "true"

	if err := h.identService.UpdateFeedback(sessionID, chip, selected); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update feedback: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *IdentificationHandlers) StopIdentificationHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if err := h.identService.StopIdentification(sessionID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to stop: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Identification stopped")
}

func (h *IdentificationHandlers) renderIdentificationPartial(w http.ResponseWriter, session *identification.IdentificationSession) {
	tmplPath := filepath.Join(h.templatePath, "partials", "identification_container.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	progress := int(session.Confidence * 100)
	if progress > 100 {
		progress = 100
	}

	data := struct {
		SessionID  string
		VideoID    string
		Status     string
		Progress   int
		Chips      []identification.Chip
		Candidates []identification.FilmCandidate
		Confidence float64
	}{
		SessionID:  session.ID,
		VideoID:    session.VideoID,
		Status:     session.Status,
		Progress:   progress,
		Candidates: session.Candidates,
		Confidence: session.Confidence,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}
