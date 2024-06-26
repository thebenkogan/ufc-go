package events

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/thebenkogan/ufc/internal/model"
)

func TestFreshTime(t *testing.T) {
	now := time.Now()

	freshTimeTests := []struct {
		startTime time.Time
		expected  time.Duration
	}{
		{now.Add(2 * beforeFreshTime), beforeFreshTime},
		{now.Add(beforeFreshTime / 2), beforeFreshTime / 2},
		{now.Add(-2 * time.Hour), 0},
	}

	for _, tt := range freshTimeTests {
		t.Run(fmt.Sprintf("event start time: %v, expected duration: %v", tt.startTime.Format(time.RFC1123), tt.expected), func(t *testing.T) {
			event := &model.Event{StartTime: tt.startTime.Format(time.RFC3339), Fights: []model.Fight{{Fighters: []string{"A", "B"}}}}
			got := freshTime(event)
			if math.Abs(got.Seconds()-tt.expected.Seconds()) > 1 {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}

	t.Run("Should return duringFreshTime when event is LIVE", func(t *testing.T) {
		event := &model.Event{StartTime: "LIVE", Fights: []model.Fight{{Fighters: []string{"A", "B"}}}}
		got := freshTime(event)
		if got != duringFreshTime {
			t.Errorf("got %v, want %v", got, duringFreshTime)
		}
	})
}

func TestValidatePicks(t *testing.T) {
	event := &model.Event{StartTime: "LIVE", Fights: []model.Fight{
		{Fighters: []string{"A", "B"}},
		{Fighters: []string{"C", "D"}},
		{Fighters: []string{"E", "F"}},
	}}

	validateTests := []struct {
		picks []string
		valid bool
	}{
		{[]string{}, true},
		{[]string{"A", "C", "E", "D"}, false},
		{[]string{"A", "B", "C"}, false},
		{[]string{"G"}, false},
		{[]string{"A"}, true},
		{[]string{"A", "C", "E"}, true},
	}

	for _, tt := range validateTests {
		t.Run(fmt.Sprintf("picks: %v, valid: %v", tt.picks, tt.valid), func(t *testing.T) {
			err := validatePicks(event, tt.picks)
			if tt.valid && err != nil {
				t.Errorf("expected valid event, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("expected invalid event, got nil error")
			}
		})
	}
}

func TestScorePicks(t *testing.T) {
	event := &model.Event{StartTime: "LIVE", Fights: []model.Fight{
		{Fighters: []string{"A", "B"}, Winner: "A"},
		{Fighters: []string{"C", "D"}, Winner: "D"},
		{Fighters: []string{"E", "F"}, Winner: "E"},
	}}

	scoreTests := []struct {
		picks []string
		score int
	}{
		{[]string{"A", "D", "E"}, 3},
		{[]string{"A", "D", "F"}, 2},
		{[]string{"A"}, 1},
		{[]string{"B", "C", "F"}, 0},
		{[]string{"F"}, 0},
	}

	for _, tt := range scoreTests {
		t.Run(fmt.Sprintf("picks: %v, score: %v", tt.picks, tt.score), func(t *testing.T) {
			got := scorePicks(event, tt.picks)
			if got != tt.score {
				t.Errorf("got %v, want %v", got, tt.score)
			}
		})
	}
}

func TestScrape(t *testing.T) {
	eventScraper := ESPNEventScraper{}
	es, _ := eventScraper.ScrapeSchedule()
	for _, e := range es {
		fmt.Println(e)
	}
}
