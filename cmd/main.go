package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gocolly/colly"
)

func main() {
	parseEvent("")
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
	fights    [][]string
}

func parseEvent(id string) {
	event := Event{fights: make([][]string, 0)}
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
		e.ForEach("span.truncate", func(_ int, el *colly.HTMLElement) {
			fighters = append(fighters, el.Text)
		})
		event.fights = append(event.fights, fighters)
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
	t, err := time.ParseInLocation("January 2, 2006 at 3:04 PM", eventDate+" at "+earliestTime, loc)
	if err != nil {
		log.Fatalf("Failed to parse time: %v", err)
	}
	event.startTime = t.UTC().Format(time.RFC3339)

	fmt.Printf("Event ID: %s\n", event.id)
	fmt.Printf("Event Date: %s\n", event.startTime)
	for i, fight := range event.fights {
		fmt.Printf("Fight %d: %s vs %s\n", i+1, fight[0], fight[1])
	}
}
