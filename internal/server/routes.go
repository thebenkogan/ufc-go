package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/thebenkogan/ufc/internal/cache"
)

func addRoutes(
	mux *http.ServeMux,
	eventCache cache.EventCacheRepository,
) {
	mux.Handle("GET /events/{id}", handleGetEvent(eventCache))
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
