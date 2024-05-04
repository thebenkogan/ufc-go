package server

import (
	"net/http"

	"github.com/thebenkogan/ufc/internal/auth"
	"github.com/thebenkogan/ufc/internal/cache"
)

func addRoutes(
	mux *http.ServeMux,
	auth *auth.Auth,
	eventCache cache.EventCacheRepository,
) {
	mux.Handle("/login", auth.HandleBeginAuth())
	mux.Handle("/auth/google/callback", auth.HandleAuthCallback())

	mux.Handle("GET /events/{id}", auth.Middleware(handleGetEvent(eventCache)))
	mux.Handle("/", http.NotFoundHandler())
}
