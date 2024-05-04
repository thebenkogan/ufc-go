package server

import (
	"log/slog"
	"net/http"

	"github.com/thebenkogan/ufc/internal/auth"
	"github.com/thebenkogan/ufc/internal/cache"
	"github.com/thebenkogan/ufc/internal/events"
	"github.com/thebenkogan/ufc/internal/util"
)

func NewServer(auth *auth.Auth, eventCache cache.EventCacheRepository) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, auth, eventCache)
	return mux
}

// wrapper for http.HandlerFuncs that return errors
func handler(h util.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			slog.Error(err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

func addRoutes(
	mux *http.ServeMux,
	auth *auth.Auth,
	eventCache cache.EventCacheRepository,
) {
	mux.Handle("/login", handler(auth.HandleBeginAuth()))
	mux.Handle("/auth/google/callback", handler(auth.HandleAuthCallback()))

	mux.Handle("GET /events/{id}", handler(auth.Middleware(events.HandleGetEvent(eventCache))))
	mux.Handle("/", http.NotFoundHandler())
}
