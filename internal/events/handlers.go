package events

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/thebenkogan/ufc/internal/auth"
	"github.com/thebenkogan/ufc/internal/cache"
	"github.com/thebenkogan/ufc/internal/model"
	"github.com/thebenkogan/ufc/internal/picks"
	"github.com/thebenkogan/ufc/internal/util"
	"golang.org/x/sync/errgroup"
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

func HandleGetPicks(eventScraper EventScraper, eventCache cache.EventCacheRepository, eventPicks picks.EventPicksRepository) util.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		user, ok := r.Context().Value("user").(auth.User)
		if !ok {
			return fmt.Errorf("no user in context")
		}

		eventId := r.PathValue("id")
		event, err := getEventWithCache(r.Context(), eventScraper, eventCache, eventId)
		if err != nil {
			return err
		}

		userPicks, err := eventPicks.GetPicks(r.Context(), user, event.Id)
		if err != nil {
			return err
		}
		if userPicks == nil {
			userPicks = &picks.Picks{EventId: eventId, Winners: []string{}}
		}

		if err := checkUpdatePicksScore(r.Context(), user, event, userPicks, eventPicks); err != nil {
			return err
		}

		util.Encode(w, http.StatusOK, userPicks)
		return nil
	}
}

type GetAllPicksResponse struct {
	*picks.Picks
	Event *model.Event `json:"event"`
}

func HandleGetAllPicks(eventScraper EventScraper, eventCache cache.EventCacheRepository, eventPicks picks.EventPicksRepository) util.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		user, ok := r.Context().Value("user").(auth.User)
		if !ok {
			return fmt.Errorf("no user in context")
		}

		userPicks, err := eventPicks.GetAllPicks(r.Context(), user)
		if err != nil {
			return fmt.Errorf("error getting all picks: %w", err)
		}

		eventIds := make([]string, 0, len(userPicks))
		for _, pick := range userPicks {
			eventIds = append(eventIds, pick.EventId)
		}

		eventMap, err := eventCache.GetEvents(r.Context(), eventIds)
		if err != nil {
			return fmt.Errorf("error getting events from IDs: %w", err)
		}

		group, gCtx := errgroup.WithContext(r.Context())
		group.SetLimit(5)
		var mu sync.Mutex
		for _, up := range userPicks {
			up := up
			group.Go(func() error {
				mu.Lock()
				event, ok := eventMap[up.EventId]
				mu.Unlock()
				if !ok {
					event, err = getEventWithCache(gCtx, eventScraper, eventCache, up.EventId)
					if err != nil {
						return err
					}
					mu.Lock()
					eventMap[event.Id] = event
					mu.Unlock()
				}
				return checkUpdatePicksScore(gCtx, user, event, up, eventPicks)
			})
		}

		if err := group.Wait(); err != nil {
			return fmt.Errorf("error processing events: %w", err)
		}

		res := make([]*GetAllPicksResponse, 0, len(userPicks))
		for _, up := range userPicks {
			res = append(res, &GetAllPicksResponse{Picks: up, Event: eventMap[up.EventId]})
		}

		util.Encode(w, http.StatusOK, res)
		return nil
	}
}

type PostEventPicksRequest struct {
	Winners []string `json:"winners"`
}

func HandlePostPicks(eventScraper EventScraper, eventCache cache.EventCacheRepository, eventPicks picks.EventPicksRepository) util.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var picks PostEventPicksRequest
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

		if err := eventPicks.SavePicks(r.Context(), user, event.Id, pickedFighters); err != nil {
			return fmt.Errorf("error saving picks: %w", err)
		}

		return nil
	}
}
