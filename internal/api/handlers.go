package api

import (
	"html/template"
	"net/http"
	"path/filepath"
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
		Title string
		Message string
	}{
		Title: "VShazam",
		Message: "Hello, VShazam!",
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}