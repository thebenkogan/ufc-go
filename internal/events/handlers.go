package events

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/samber/lo"
	"github.com/thebenkogan/ufc/internal/auth"
	"github.com/thebenkogan/ufc/internal/cache"
	"github.com/thebenkogan/ufc/internal/model"
	"github.com/thebenkogan/ufc/internal/picks"
	"github.com/thebenkogan/ufc/internal/util/api"
	"github.com/thebenkogan/ufc/internal/util/conv"
	"github.com/thebenkogan/ufc/internal/util/logs"
)

const scheduleTTL = time.Hour

func HandleGetSchedule(eventScraper EventScraper, eventCache cache.EventCacheRepository) api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		cached, err := eventCache.GetSchedule(ctx)
		if err != nil {
			logs.Logger(ctx).Warn("failed to get schedule from cache", "error", err)
		}

		if cached != nil {
			logs.Logger(ctx).Info("cache hit")
			api.Encode(w, http.StatusOK, cached)
			return nil
		}

		logs.Logger(ctx).Info("cache miss, scraping schedule...")

		schedule, err := eventScraper.ScrapeSchedule()
		if err != nil {
			return err
		}

		logs.Logger(ctx).Info("parsed schedule, storing to cache")

		if err := eventCache.SetSchedule(ctx, schedule, scheduleTTL); err != nil {
			logs.Logger(ctx).Warn("failed to cache schedule", "error", err)
		}

		api.Encode(w, http.StatusOK, schedule)
		return nil
	}
}

func HandleGetEvent(eventScraper EventScraper, eventCache cache.EventCacheRepository) api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		event, err := getEventWithCache(ctx, eventScraper, eventCache, id)
		if err != nil {
			return err
		}
		api.Encode(w, http.StatusOK, event)
		return nil
	}
}

func HandleGetPicks(eventScraper EventScraper, eventCache cache.EventCacheRepository, eventPicks picks.EventPicksRepository) api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		user := auth.GetUser(ctx)
		if user == nil {
			return fmt.Errorf("no user in context")
		}

		eventId := r.PathValue("id")
		event, err := getEventWithCache(ctx, eventScraper, eventCache, eventId)
		if err != nil {
			return err
		}

		userPicks, err := eventPicks.GetUserPicksByEvent(ctx, user, event.Id)
		if err != nil {
			return err
		}
		if userPicks == nil {
			userPicks = &picks.Picks{UserId: user.Id, EventId: eventId, Winners: []string{}}
		}

		api.Encode(w, http.StatusOK, userPicks)
		return nil
	}
}

type GetAllPicksResponse struct {
	*picks.Picks
	Event *model.Event `json:"event"`
}

func HandleGetAllPicks(eventScraper EventScraper, eventCache cache.EventCacheRepository, eventPicks picks.EventPicksRepository) api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		user := auth.GetUser(ctx)
		if user == nil {
			return fmt.Errorf("no user in context")
		}

		userPicks, err := eventPicks.GetAllUserPicks(ctx, user)
		if err != nil {
			return fmt.Errorf("error getting all picks: %w", err)
		}

		if len(userPicks) == 0 {
			api.Encode(w, http.StatusOK, []GetAllPicksResponse{})
			return nil
		}

		eventIds := make([]string, 0, len(userPicks))
		for _, pick := range userPicks {
			eventIds = append(eventIds, pick.EventId)
		}

		fmt.Println("event ids", eventIds)

		eventMap, err := getEventsWithCache(ctx, eventScraper, eventCache, eventIds)
		if err != nil {
			return fmt.Errorf("error getting events from IDs: %w", err)
		}

		res := make([]*GetAllPicksResponse, 0, len(userPicks))
		for _, up := range userPicks {
			res = append(res, &GetAllPicksResponse{Picks: up, Event: eventMap[up.EventId]})
		}

		api.Encode(w, http.StatusOK, res)
		return nil
	}
}

type PostEventPicksRequest struct {
	Winners []string `json:"winners"`
}

func HandlePostPicks(eventScraper EventScraper, eventCache cache.EventCacheRepository, eventPicks picks.EventPicksRepository) api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var picks PostEventPicksRequest
		api.Decode(r, &picks)

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

		user := auth.GetUser(ctx)
		if user == nil {
			return fmt.Errorf("no user in context")
		}

		if err := eventPicks.SavePicks(ctx, user, event.Id, pickedFighters); err != nil {
			return fmt.Errorf("error saving picks: %w", err)
		}

		return nil
	}
}

func HandleScoreJob(eventScraper EventScraper, eventCache cache.EventCacheRepository, eventPicks picks.EventPicksRepository) api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		key := r.Header.Get("api-key")
		cronjobKey := os.Getenv("CRONJOB_API_KEY")
		if key != cronjobKey {
			api.Encode(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
			return nil
		}

		latestEvent, err := getEventWithCache(ctx, eventScraper, eventCache, eventLatest)
		if err != nil {
			return err
		}

		if !latestEvent.IsFinished() {
			logs.Logger(ctx).Info("latest event is not finished, skipping job")
			return nil
		}

		filter := &picks.PicksFilter{
			EventIDs: []string{latestEvent.Id},
			HasScore: conv.Ptr(false),
		}
		allPicks, err := eventPicks.GetPicksByFilter(ctx, filter)
		if err != nil {
			return err
		}

		if len(allPicks) == 0 {
			logs.Logger(ctx).Info("all picks scored, skipping job")
			return nil
		}

		logs.Logger(ctx).Info("scoring picks", "total", len(allPicks), "event ID", latestEvent.Id)

		for _, p := range allPicks {
			p.Score = conv.Ptr(scorePicks(latestEvent, p.Winners))
		}

		errs := eventPicks.BatchScorePicks(ctx, allPicks)
		if len(errs) > 0 {
			logs.Logger(ctx).Warn("failed to save some picks", "errors", errs)
		}

		api.Encode(w, http.StatusOK, fmt.Sprintf("scored %d picks", len(allPicks)))
		return nil
	}
}
