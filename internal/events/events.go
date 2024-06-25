package events

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/thebenkogan/ufc/internal/auth"
	"github.com/thebenkogan/ufc/internal/cache"
	"github.com/thebenkogan/ufc/internal/model"
	"github.com/thebenkogan/ufc/internal/picks"
)

const eventLatest string = "latest"

func getEventWithCache(ctx context.Context, log *slog.Logger, eventScraper EventScraper, eventCache cache.EventCacheRepository, id string) (*model.Event, error) {
	log.Info(fmt.Sprintf("Getting event, ID: %s", id))

	cached, err := eventCache.GetEvent(ctx, id)
	if err != nil {
		log.Warn("failed to get event from cache", "error", err)
	}

	if cached != nil {
		log.Info("cache hit")
		return cached, nil
	}

	log.Info("cache miss, scraping event...")

	event, err := eventScraper.ScrapeEvent(id)
	if err != nil {
		return nil, err
	}

	log.Info("parsed event, storing to cache")

	ttl := freshTime(event)
	if err := eventCache.SetEvent(ctx, event.Id, event, ttl); err != nil {
		log.Warn("failed to cache event", "error", err)
	}
	if id == eventLatest {
		if event.IsFinished() {
			// don't cache latest key forever when event is over
			ttl = time.Hour
		}
		if err := eventCache.SetEvent(ctx, eventLatest, event, ttl); err != nil {
			log.Warn("failed to cache latest event", "error", err)
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

func checkUpdatePicksScore(ctx context.Context, user auth.User, event *model.Event, userPicks *picks.Picks, eventPicks picks.EventPicksRepository) error {
	if userPicks.Score == nil && event.IsFinished() && len(userPicks.Winners) > 0 {
		score := scorePicks(event, userPicks.Winners)
		userPicks.Score = &score
		if err := eventPicks.ScorePicks(ctx, user, event.Id, score); err != nil {
			return err
		}
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
