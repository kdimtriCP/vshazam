package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

	fileServer := http.FileServer(http.Dir("./web/static"))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	return r
}
