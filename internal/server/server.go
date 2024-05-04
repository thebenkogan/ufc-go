package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/thebenkogan/ufc/internal/auth"
	"github.com/thebenkogan/ufc/internal/cache"
	"github.com/thebenkogan/ufc/internal/events"
)

func NewServer(auth *auth.Auth, eventCache cache.EventCacheRepository) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, auth, eventCache)
	return mux
}

// wrapper for http.HandlerFuncs that return errors
func handler(h func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			slog.Error(err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

func encode[T any](w http.ResponseWriter, status int, v T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error(err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func handleGetEvent(eventCache cache.EventCacheRepository) http.HandlerFunc {
	return handler(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		slog.Info(fmt.Sprintf("Getting event, ID: %s", id))

		cached, err := eventCache.GetEvent(r.Context(), id)
		if err != nil {
			slog.Warn("failed to get event from cache: %s", err)
		}

		if cached != nil {
			slog.Info("cache hit")
			encode(w, http.StatusOK, cached)
			return nil
		}

		slog.Info("cache miss, parsing event...")

		event, err := events.ScrapeEvent(id)
		if err != nil {
			return err
		}

		slog.Info("parsed event, storing to cache")

		if err := eventCache.SetEvent(r.Context(), id, event); err != nil {
			slog.Warn("failed to cache event: %s", err)
		}

		encode(w, http.StatusOK, event)
		return nil
	})
}
