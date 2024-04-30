package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/thebenkogan/ufc/internal/parser"
)

func addRoutes(
	mux *http.ServeMux,
) {
	mux.Handle("GET /events/{id}", handleGetEvent())
	mux.Handle("/", http.NotFoundHandler())
}

func encode[T any](w http.ResponseWriter, status int, v T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error(err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func handleGetEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		slog.Info(fmt.Sprintf("Getting event, ID: %s", id))
		if id == "latest" {
			id = ""
		}
		event, err := parser.ParseEvent(id)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		encode(w, http.StatusOK, event)
	}
}
