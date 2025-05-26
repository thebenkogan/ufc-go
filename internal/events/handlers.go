package events

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/samber/lo"
	"github.com/thebenkogan/ufc/internal/auth"
	"github.com/thebenkogan/ufc/internal/cache"
	"github.com/thebenkogan/ufc/internal/model"
	"github.com/thebenkogan/ufc/internal/picks"
	"github.com/thebenkogan/ufc/internal/util/api_util"
	"github.com/thebenkogan/ufc/internal/util/logs"
	"golang.org/x/sync/errgroup"
)

const SCHEDULE_TTL = time.Hour

func HandleGetSchedule(eventScraper EventScraper, eventCache cache.EventCacheRepository) api_util.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		cached, err := eventCache.GetSchedule(ctx)
		if err != nil {
			logs.Logger(ctx).Warn("failed to get schedule from cache", "error", err)
		}

		if cached != nil {
			logs.Logger(ctx).Info("cache hit")
			api_util.Encode(w, http.StatusOK, cached)
			return nil
		}

		logs.Logger(ctx).Info("cache miss, scraping schedule...")

		schedule, err := eventScraper.ScrapeSchedule()
		if err != nil {
			return err
		}

		logs.Logger(ctx).Info("parsed schedule, storing to cache")

		if err := eventCache.SetSchedule(ctx, schedule, SCHEDULE_TTL); err != nil {
			logs.Logger(ctx).Warn("failed to cache schedule", "error", err)
		}

		api_util.Encode(w, http.StatusOK, schedule)
		return nil
	}
}

func HandleGetEvent(eventScraper EventScraper, eventCache cache.EventCacheRepository) api_util.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		event, err := getEventWithCache(ctx, eventScraper, eventCache, id)
		if err != nil {
			return err
		}
		api_util.Encode(w, http.StatusOK, event)
		return nil
	}
}

func HandleGetPicks(eventScraper EventScraper, eventCache cache.EventCacheRepository, eventPicks picks.EventPicksRepository) api_util.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		user, ok := ctx.Value("user").(auth.User)
		if !ok {
			return fmt.Errorf("no user in context")
		}

		eventId := r.PathValue("id")
		event, err := getEventWithCache(ctx, eventScraper, eventCache, eventId)
		if err != nil {
			return err
		}

		userPicks, err := eventPicks.GetPicks(ctx, user, event.Id)
		if err != nil {
			return err
		}
		if userPicks == nil {
			userPicks = &picks.Picks{UserId: user.Id, EventId: eventId, Winners: []string{}}
		}

		if err := checkUpdatePicksScore(ctx, user, event, userPicks, eventPicks); err != nil {
			return err
		}

		api_util.Encode(w, http.StatusOK, userPicks)
		return nil
	}
}

type GetAllPicksResponse struct {
	*picks.Picks
	Event *model.Event `json:"event"`
}

func HandleGetAllPicks(eventScraper EventScraper, eventCache cache.EventCacheRepository, eventPicks picks.EventPicksRepository) api_util.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		user, ok := ctx.Value("user").(auth.User)
		if !ok {
			return fmt.Errorf("no user in context")
		}

		userPicks, err := eventPicks.GetAllPicks(ctx, user)
		if err != nil {
			return fmt.Errorf("error getting all picks: %w", err)
		}

		if len(userPicks) == 0 {
			api_util.Encode(w, http.StatusOK, []GetAllPicksResponse{})
			return nil
		}

		eventIds := make([]string, 0, len(userPicks))
		for _, pick := range userPicks {
			eventIds = append(eventIds, pick.EventId)
		}

		eventMap, err := eventCache.GetEvents(ctx, eventIds)
		if err != nil {
			return fmt.Errorf("error getting events from IDs: %w", err)
		}

		group, gCtx := errgroup.WithContext(ctx)
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

		api_util.Encode(w, http.StatusOK, res)
		return nil
	}
}

type PostEventPicksRequest struct {
	Winners []string `json:"winners"`
}

func HandlePostPicks(eventScraper EventScraper, eventCache cache.EventCacheRepository, eventPicks picks.EventPicksRepository) api_util.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var picks PostEventPicksRequest
		api_util.Decode(r, &picks)

		pickedFighters := lo.Uniq(picks.Winners)

		id := r.PathValue("id")
		event, err := getEventWithCache(ctx, eventScraper, eventCache, id)
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

		user, ok := ctx.Value("user").(auth.User)
		if !ok {
			return fmt.Errorf("no user in context")
		}

		if err := eventPicks.SavePicks(ctx, user, event.Id, pickedFighters); err != nil {
			return fmt.Errorf("error saving picks: %w", err)
		}

		return nil
	}
}
