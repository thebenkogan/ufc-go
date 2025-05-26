package events

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/thebenkogan/ufc/internal/cache"
	"github.com/thebenkogan/ufc/internal/model"
	"github.com/thebenkogan/ufc/internal/util/logs"
	"golang.org/x/sync/errgroup"
)

const eventLatest string = "latest"

func getEventsWithCache(ctx context.Context, eventScraper EventScraper, eventCache cache.EventCacheRepository, ids []string) (map[string]*model.Event, error) {
	events := make(map[string]*model.Event, len(ids))
	group, gCtx := errgroup.WithContext(ctx)
	group.SetLimit(5)
	var mu sync.Mutex
	for _, id := range ids {
		group.Go(func() error {
			event, err := getEventWithCache(gCtx, eventScraper, eventCache, id)
			if err != nil {
				return err
			}
			mu.Lock()
			events[id] = event
			mu.Unlock()
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("error processing events: %w", err)
	}

	return events, nil

}

func getEventWithCache(ctx context.Context, eventScraper EventScraper, eventCache cache.EventCacheRepository, id string) (*model.Event, error) {
	logs.Logger(ctx).Info(fmt.Sprintf("Getting event, ID: %s", id))

	cached, err := eventCache.GetEvent(ctx, id)
	if err != nil {
		logs.Logger(ctx).Warn("failed to get event from cache", "error", err)
	}

	if cached != nil {
		logs.Logger(ctx).Info("cache hit")
		return cached, nil
	}

	logs.Logger(ctx).Info("cache miss, scraping event...")

	event, err := eventScraper.ScrapeEvent(id)
	if err != nil {
		return nil, err
	}

	logs.Logger(ctx).Info("parsed event, storing to cache")

	ttl := freshTime(event)
	if err := eventCache.SetEvent(ctx, event.Id, event, ttl); err != nil {
		logs.Logger(ctx).Warn("failed to cache event", "error", err)
	}
	if id == eventLatest {
		if event.IsFinished() {
			// don't cache latest key forever when event is over
			ttl = time.Hour
		}
		if err := eventCache.SetEvent(ctx, eventLatest, event, ttl); err != nil {
			logs.Logger(ctx).Warn("failed to cache latest event", "error", err)
		}

	}

	return event, nil
}

const (
	beforeFreshTime = time.Hour
	duringFreshTime = 5 * time.Minute
)

// Returns how long this event should remain in the cache
// before start time, it is fresh for beforeFreshTime or until event start, whichever is sooner
// during the event (LIVE), it is fresh for duringFreshTime
// after the event, it is fresh forever (0)
func freshTime(event *model.Event) time.Duration {
	if event.IsFinished() {
		// event is over, keep forever
		return 0
	}

	if event.StartTime == "LIVE" {
		return duringFreshTime
	}

	startTime, err := time.Parse(time.RFC3339, event.StartTime)
	if err != nil {
		return 0
	}
	now := time.Now()

	// before start time
	if startTime.After(now) {
		if now.Add(beforeFreshTime).After(startTime) {
			return time.Until(startTime)
		} else {
			return beforeFreshTime
		}
	}

	return 0
}

func validatePicks(event *model.Event, picks []string) error {
	if len(picks) > len(event.Fights) {
		return fmt.Errorf("too many picks")
	}

	availableFighters := make(map[string]int)
	for i, fight := range event.Fights {
		for _, fighter := range fight.Fighters {
			availableFighters[fighter] = i
		}
	}
	pickedFights := make(map[int]struct{})

	for _, pick := range picks {
		fightId, ok := availableFighters[pick]
		if !ok {
			return fmt.Errorf("unknown fighter: %s", pick)
		}
		if _, ok := pickedFights[fightId]; ok {
			return fmt.Errorf("cannot pick both fighters in the same fight: %v", event.Fights[fightId].Fighters)
		}
		pickedFights[fightId] = struct{}{}
	}

	return nil
}

func scorePicks(event *model.Event, picks []string) int {
	score := 0
	for _, fight := range event.Fights {
		if slices.Contains(picks, fight.Winner) {
			score++
		}
	}
	return score
}
