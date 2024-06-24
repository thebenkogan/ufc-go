package server

import (
	"log/slog"
	"net/http"

	"github.com/rs/cors"
	"github.com/thebenkogan/ufc/internal/auth"
	"github.com/thebenkogan/ufc/internal/cache"
	"github.com/thebenkogan/ufc/internal/events"
	"github.com/thebenkogan/ufc/internal/picks"
	"github.com/thebenkogan/ufc/internal/util"
)

func NewServer(oauth auth.OIDCAuth, eventScraper events.EventScraper, eventCache cache.EventCacheRepository, eventPicks picks.EventPicksRepository) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, oauth, eventScraper, eventCache, eventPicks)
	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowCredentials: true,
	}).Handler(mux)
	return handler
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
	oauth auth.OIDCAuth,
	eventScraper events.EventScraper,
	eventCache cache.EventCacheRepository,
	eventPicks picks.EventPicksRepository,
) {
	mux.Handle("/login", handler(oauth.HandleBeginAuth()))
	mux.Handle("/auth/google/callback", handler(oauth.HandleAuthCallback()))
	mux.Handle("/me", handler(auth.HandleMe(oauth)))

	mux.Handle("GET /schedule", handler((events.HandleGetSchedule(eventScraper, eventCache))))

	mux.Handle("GET /events/picks", handler(oauth.Middleware(events.HandleGetAllPicks(eventScraper, eventCache, eventPicks))))
	mux.Handle("GET /events/{id}", handler((events.HandleGetEvent(eventScraper, eventCache))))

	mux.Handle("GET /events/{id}/picks", handler(oauth.Middleware(events.HandleGetPicks(eventScraper, eventCache, eventPicks))))
	mux.Handle("POST /events/{id}/picks", handler(oauth.Middleware(events.HandlePostPicks(eventScraper, eventCache, eventPicks))))

	mux.Handle("/", http.NotFoundHandler())
}
