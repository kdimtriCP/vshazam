package api

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

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

	for {
		select {
		case update, ok := <-session.Updates:
			if !ok {
				return
			}

			data, err := json.Marshal(update.Data)
			if err != nil {
				log.Printf("Error marshaling update: %v", err)
				continue
			}

			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", update.Type, string(data))
			flusher.Flush()

		case <-clientGone:
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

	session, exists := h.identService.GetSession(sessionID)
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	h.renderIdentificationPartial(w, session)
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
