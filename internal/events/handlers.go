package events

import (
	"fmt"
	"net/http"

	"github.com/thebenkogan/ufc/internal/auth"
	"github.com/thebenkogan/ufc/internal/cache"
	"github.com/thebenkogan/ufc/internal/picks"
	"github.com/thebenkogan/ufc/internal/util"
)

func HandleGetEvent(eventScraper EventScraper, eventCache cache.EventCacheRepository) util.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		event, err := getEventWithCache(r.Context(), eventScraper, eventCache, id)
		if err != nil {
			return err
		}
		util.Encode(w, http.StatusOK, event)
		return nil
	}
}

type EventPicks struct {
	Winners []string `json:"winners"`
}

func HandlePostPicks(eventScraper EventScraper, eventCache cache.EventCacheRepository, eventPicks picks.EventPicksRepository) util.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var picks EventPicks
		util.Decode(r, &picks)
		pickedFighters := util.Distinct(picks.Winners)

		id := r.PathValue("id")
		event, err := getEventWithCache(r.Context(), eventScraper, eventCache, id)
		if err != nil {
			return err
		}

		if event.HasStarted() {
			http.Error(w, "picks for this event are closed", http.StatusBadRequest)
			return nil
		}

		if err := validatePicks(event, pickedFighters); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil
		}

		user, ok := r.Context().Value("user").(auth.User)
		if !ok {
			return fmt.Errorf("no user in context")
		}

		if err := eventPicks.SavePicks(r.Context(), user, id, pickedFighters); err != nil {
			return fmt.Errorf("error saving picks: %w", err)
		}

		return nil
	}
}
