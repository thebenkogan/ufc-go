package parser

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

func makeUrl(id string) string {
	if id == "" {
		return "https://www.espn.com/mma/fightcenter"
	}
	return fmt.Sprintf("https://www.espn.com/mma/fightcenter/_/id/%s/league/ufc", id)
}

type Event struct {
	Id        string
	StartTime string
	Fights    []Fight
}

type Fight struct {
	Fighters []string
	Winner   string
}

func ParseEvent(id string) (*Event, error) {
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
				winner = name
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

	fmt.Printf("Event ID: %s\n", event.Id)
	fmt.Printf("Event Date: %s\n", event.StartTime)
	for i, fight := range event.Fights {
		fmt.Printf("Fight %d: %s vs %s\n", i+1, fight.Fighters[0], fight.Fighters[1])
		if fight.Winner != "" {
			fmt.Printf("Winner: %s\n", fight.Winner)
		}
	}

	return &event, nil
}
