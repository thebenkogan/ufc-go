package main

import (
	"fmt"
	"log"

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
	id     string
	fights [][]string
}

func parseEvent(id string) {
	event := Event{fights: make([][]string, 0)}

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

	if err := c.Visit(makeUrl(id)); err != nil {
		log.Fatalf("Failed to visit: %v", err)
	}
	c.Wait()

	fmt.Printf("Event ID: %s\n", event.id)
	for i, fight := range event.fights {
		fmt.Printf("Fight %d: %s vs %s\n", i+1, fight[0], fight[1])
	}
}
