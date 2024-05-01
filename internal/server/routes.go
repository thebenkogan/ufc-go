package server

import (
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
