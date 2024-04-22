package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

func main() {
	parseEvent("600041053")
}

func makeUrl(id string) string {
	if id == "" {
		return "https://www.espn.com/mma/fightcenter"
	}
	return fmt.Sprintf("https://www.espn.com/mma/fightcenter/_/id/%s/league/ufc", id)
}

type Event struct {
	id        string
	startTime string
	fights    []Fight
}

type Fight struct {
	fighters []string
	winner   string
}

func parseEvent(id string) {
	event := Event{fights: make([]Fight, 0)}
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
		event.fights = append(event.fights, Fight{fighters: fighters, winner: winner})
	})

	c.OnHTML("div.MMAEventHeader__Event select.dropdown__select", func(e *colly.HTMLElement) {
		selectType := e.ChildText("option[hidden]")
		if selectType != "Events" {
			return
		}
		id := e.ChildAttr("option[selected]", "value")
		event.id = id
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
		log.Fatalf("Failed to visit: %v", err)
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
		log.Fatalf("Failed to parse time: %v", err)
	}
	event.startTime = t.UTC().Format(time.RFC3339)

	fmt.Printf("Event ID: %s\n", event.id)
	fmt.Printf("Event Date: %s\n", event.startTime)
	for i, fight := range event.fights {
		fmt.Printf("Fight %d: %s vs %s\n", i+1, fight.fighters[0], fight.fighters[1])
		if fight.winner != "" {
			fmt.Printf("Winner: %s\n", fight.winner)
		}
	}
}
