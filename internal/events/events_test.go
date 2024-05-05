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
		{now, duringFreshTime},
		{now.Add(-eventDuration / 3), duringFreshTime},
		{now.Add(2 * beforeFreshTime), beforeFreshTime},
		{now.Add(beforeFreshTime / 2), beforeFreshTime / 2},
		{now.Add(-2 * eventDuration), 0},
	}

	for _, tt := range freshTimeTests {
		t.Run(fmt.Sprintf("event start time: %v, expected duration: %v", tt.startTime.Format(time.RFC1123), tt.expected), func(t *testing.T) {
			event := &model.Event{StartTime: tt.startTime.Format(time.RFC3339)}
			got := freshTime(event)
			if math.Abs(got.Seconds()-tt.expected.Seconds()) > 1 {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}