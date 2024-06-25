package events

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/thebenkogan/ufc/internal/model"
)

type EventScraper interface {
	ScrapeEvent(id string) (*model.Event, error)
	ScrapeSchedule() ([]*model.EventInfo, error)
}

type ESPNEventScraper struct{}

func NewESPNEventScraper() *ESPNEventScraper {
	return &ESPNEventScraper{}
}

func (_ ESPNEventScraper) makeUrl(id string) string {
	if id == eventLatest {
		return "https://www.espn.com/mma/fightcenter"
	}
	return fmt.Sprintf("https://www.espn.com/mma/fightcenter/_/id/%s/league/ufc", id)
}

func (e ESPNEventScraper) ScrapeEvent(id string) (*model.Event, error) {
	event := model.Event{Fights: make([]model.Fight, 0)}
	var eventDate string
	var earliestTime string

	c := colly.NewCollector()

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
		event.Fights = append(event.Fights, model.Fight{Fighters: fighters, Winner: winner})
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
			if el.Index == 0 {
				event.Name = el.Text
			}
			if el.Index == 1 {
				eventDate = el.Text
			}
		})
	})

	if err := c.Visit(e.makeUrl(id)); err != nil {
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
		event.StartTime = "LIVE"
	} else {
		event.StartTime = t.UTC().Format(time.RFC3339)
	}

	return &event, nil
}

func (_ ESPNEventScraper) scheduleURL() string {
	return "https://www.espn.com/mma/schedule/_/league/ufc"
}

func (e ESPNEventScraper) ScrapeSchedule() ([]*model.EventInfo, error) {
	events := make([]*model.EventInfo, 0)

	c := colly.NewCollector()

	c.OnHTML("tr.Table__TR", func(e *colly.HTMLElement) {
		date := e.ChildText("span.date__innerCell")
		name := e.ChildText("td.event__col")
		link := e.ChildAttr("td.event__col a", "href")
		if date == "" {
			return
		}
		id := strings.Split(link, "/")[5]
		loc, _ := time.LoadLocation("Local")
		layout := "Jan 2"
		t, err := time.ParseInLocation(layout, date, loc)
		if err != nil {
			return
		}
		events = append(events, &model.EventInfo{Id: id, Name: name, Date: t})
	})

	if err := c.Visit(e.scheduleURL()); err != nil {
		return nil, fmt.Errorf("failed to visit URL: %v", err)
	}
	c.Wait()

	slices.SortFunc(events, func(a, b *model.EventInfo) int {
		return a.Date.Compare(b.Date)
	})

	return events, nil
}
