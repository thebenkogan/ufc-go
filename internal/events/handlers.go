package events

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/thebenkogan/ufc/internal/cache"
	"github.com/thebenkogan/ufc/internal/util"
)

func HandleGetEvent(eventScraper EventScraper, eventCache cache.EventCacheRepository) util.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		slog.Info(fmt.Sprintf("Getting event, ID: %s", id))

		cached, err := eventCache.GetEvent(r.Context(), id)
		if err != nil {
			slog.Warn("failed to get event from cache", "error", err)
		}

		if cached != nil {
			slog.Info("cache hit")
			util.Encode(w, http.StatusOK, cached)
			return nil
		}

		slog.Info("cache miss, parsing event...")

		event, err := eventScraper.ScrapeEvent(id)
		if err != nil {
			return err
		}

		slog.Info("parsed event, storing to cache")

		if err := eventCache.SetEvent(r.Context(), id, event, freshTime(event)); err != nil {
			slog.Warn("failed to cache event", "error", err)
		}

		util.Encode(w, http.StatusOK, event)
		return nil
	}
}
