package events

import (
	"fmt"
	"log"
	"log/slog"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

func makeUrl(id string) string {
	if id == "latest" {
		return "https://www.espn.com/mma/fightcenter"
	}
	return fmt.Sprintf("https://www.espn.com/mma/fightcenter/_/id/%s/league/ufc", id)
}

type Event struct {
	Id        string  `json:"id"`
	StartTime string  `json:"start_time"`
	Fights    []Fight `json:"fights"`
}

type Fight struct {
	Fighters []string `json:"fighters"`
	Winner   string   `json:"winner,omitempty"`
}

func ScrapeEvent(id string) (*Event, error) {
	event := Event{Fights: make([]Fight, 0)}
	var eventDate string
	var earliestTime string

	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL)
	})

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Something went wrong:", err)
	})

	c.OnResponse(func(r *colly.Response) {
		log.Println("Visited", r.Request.URL)
	})

	c.OnHTML("div.MMAGamestrip", func(e *colly.HTMLElement) {
		fighters := make([]string, 0)
		var winner string
		e.ForEach("h2.h4", func(_ int, el *colly.HTMLElement) {
			name := el.Text
			fighters = append(fighters, name)
			if strings.Contains(el.Attr("class"), "clr-gray-02") {
				if winner == "" {
					winner = name
				} else {
					// if both are the same color, no winner yet
					winner = ""
				}
			}
		})
		event.Fights = append(event.Fights, Fight{Fighters: fighters, Winner: winner})
	})

	c.OnHTML("div.MMAEventHeader__Event select.dropdown__select", func(e *colly.HTMLElement) {
		selectType := e.ChildText("option[hidden]")
		if selectType != "Events" {
			return
		}
		id := e.ChildAttr("option[selected]", "value")
		event.Id = id
	})

	c.OnHTML("span.MMAHeaderUpsellTunein__Meta", func(e *colly.HTMLElement) {
		earliestTime = e.Text
	})

	c.OnHTML("div.MMAEventHeader__Event div.flex-column", func(e *colly.HTMLElement) {
		e.ForEach("*", func(_ int, el *colly.HTMLElement) {
			if el.Index == 1 {
				eventDate = el.Text
			}
		})
	})

	if err := c.Visit(makeUrl(id)); err != nil {
		return nil, fmt.Errorf("failed to visit URL: %v", err)
	}
	c.Wait()

	loc, _ := time.LoadLocation("Local")
	layout := "January 2, 2006"
	if earliestTime != "" {
		layout += " at 3:04 PM"
		eventDate += " at " + earliestTime
	}
	t, err := time.ParseInLocation(layout, eventDate, loc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse date: %v", err)
	}
	event.StartTime = t.UTC().Format(time.RFC3339)

	return &event, nil
}

const (
	beforeFreshTime = time.Hour
	duringFreshTime = 5 * time.Minute
	eventDuration   = 10 * time.Hour
)

// Returns how long this event should remain in the cache
// before start time, it is fresh for beforeFreshTime or until event start, whichever is sooner
// during the event, it is fresh for duringFreshTime
// after the event, it is fresh forever (0)
// event is considered active for eventDuration after start time
func (e *Event) FreshTime() time.Duration {
	startTime, err := time.Parse(time.RFC3339, e.StartTime)
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

	// during event
	if now.Before(startTime.Add(eventDuration)) {
		return duringFreshTime
	}

	// event is over, keep forever
	return 0
}
