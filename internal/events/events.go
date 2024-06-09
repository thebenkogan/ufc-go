package events

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/thebenkogan/ufc/internal/cache"
	"github.com/thebenkogan/ufc/internal/model"
)

func getEventWithCache(ctx context.Context, eventScraper EventScraper, eventCache cache.EventCacheRepository, id string) (*model.Event, error) {
	slog.Info(fmt.Sprintf("Getting event, ID: %s", id))

	cached, err := eventCache.GetEvent(ctx, id)
	if err != nil {
		slog.Warn("failed to get event from cache", "error", err)
	}

	if cached != nil {
		slog.Info("cache hit")
		return cached, nil
	}

	slog.Info("cache miss, parsing event...")

	event, err := eventScraper.ScrapeEvent(id)
	if err != nil {
		return nil, err
	}

	slog.Info("parsed event, storing to cache")

	if err := eventCache.SetEvent(ctx, id, event, freshTime(event)); err != nil {
		slog.Warn("failed to cache event", "error", err)
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
	if event.StartTime == "LIVE" {
		return duringFreshTime
	}

	startTime, err := time.Parse(time.RFC3339, event.StartTime)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to parse start time: %v", err))
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

	// event is over, keep forever
	return 0
}

func validatePicks(event *model.Event, picks []string) error {
	if len(picks) == 0 {
		return fmt.Errorf("no picks provided")
	}

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
