package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kdimtricp/vshazam/internal/identification"
)

func NewRouter(app *App) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", HomeHandler)
	r.Get("/ping", PingHandler)

	r.Get("/upload", app.UploadPageHandler)
	r.Post("/upload", app.UploadHandler)
	r.Get("/videos/partial", app.VideoListPartialHandler)

	r.Get("/videos", app.ListVideosHandler)
	r.Get("/videos/{id}", app.WatchVideoHandler)
	r.Get("/stream/{id}", app.StreamVideoHandler)

	r.Get("/search", app.SearchHandler)

	// Film identification routes
	if app.IdentificationService != nil {
		identService := app.IdentificationService.(*identification.Service)
		identHandlers := NewIdentificationHandlers(identService)

		r.Get("/identify/{id}", identHandlers.StartIdentificationHandler)
		r.Get("/identify/{sessionID}/stream", identHandlers.IdentificationStreamHandler)
		r.Post("/identify/{sessionID}/feedback", identHandlers.UpdateFeedbackHandler)
	}

	fileServer := http.FileServer(http.Dir("./web/static"))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	return r
}
