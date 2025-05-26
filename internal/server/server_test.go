package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/thebenkogan/ufc/internal/auth"
	"github.com/thebenkogan/ufc/internal/cache"
	"github.com/thebenkogan/ufc/internal/events"
	"github.com/thebenkogan/ufc/internal/model"
	"github.com/thebenkogan/ufc/internal/picks"
	"github.com/thebenkogan/ufc/internal/server"
	"github.com/thebenkogan/ufc/internal/util"
)

type testOAuth struct{}

func (a testOAuth) HandleBeginAuth() util.Handler {
	return func(_ *slog.Logger, w http.ResponseWriter, r *http.Request) error {
		panic("not implemented")
	}
}

func (a testOAuth) HandleAuthCallback() util.Handler {
	return func(_ *slog.Logger, w http.ResponseWriter, r *http.Request) error {
		panic("not implemented")
	}
}

func (a testOAuth) Middleware(h util.Handler) util.Handler {
	return func(log *slog.Logger, w http.ResponseWriter, r *http.Request) error {
		user := auth.User{Id: "user", Email: "user@gmail.com", Name: "user"}
		ctx := context.WithValue(r.Context(), "user", user)
		rWithUser := r.WithContext(ctx)
		return h(log, w, rWithUser)
	}
}

type testEventScraper struct {
	maker func() *model.Event
}

func (s testEventScraper) ScrapeEvent(id string) (*model.Event, error) {
	return s.maker(), nil
}

func (s testEventScraper) ScrapeSchedule() ([]*model.EventInfo, error) {
	panic("unimplemented")
}

func TestServer(t *testing.T) {
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr: net.JoinHostPort(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
	})
	defer rdb.Close()

	clearEvents := func() {
		if _, err := rdb.FlushAll(ctx).Result(); err != nil {
			t.Fatal(err)
		}
	}
	clearEvents()

	eventCache := cache.NewRedisEventCache(rdb)

	pgUrl := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
	)
	pgCfg, err := pgxpool.ParseConfig(pgUrl)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, pgCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, "TRUNCATE TABLE picks"); err != nil {
		t.Fatal(err)
	}

	eventPicks := picks.NewPostgresEventPicks(pool)

	t.Run("should scrape event and use the cache", func(t *testing.T) {
		testEventId := "test-event-id"
		testEvent := &model.Event{
			Id:        testEventId,
			StartTime: time.Now().Format(time.RFC3339),
			Fights:    []model.Fight{},
		}
		numScrapes := 0

		testScraper := testEventScraper{
			maker: func() *model.Event {
				numScrapes += 1
				return testEvent
			},
		}

		srv := server.NewServer(testOAuth{}, testScraper, eventCache, nil)
		ts := httptest.NewServer(srv)
		defer ts.Close()

		// first request should scrape the event
		resp, err := http.Get(ts.URL + "/events/" + testEventId)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var gotEvent *model.Event
		if err := json.NewDecoder(resp.Body).Decode(&gotEvent); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, testEvent, gotEvent)
		assert.Equal(t, 1, numScrapes)

		// second request should hit the cache
		resp2, err := http.Get(ts.URL + "/events/" + testEventId)
		if err != nil {
			t.Fatal(err)
		}
		defer resp2.Body.Close()

		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		var gotEvent2 model.Event
		if err := json.NewDecoder(resp2.Body).Decode(&gotEvent2); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, *testEvent, gotEvent2)
		assert.Equal(t, 1, numScrapes)
	})

	testEventId2 := "test-event-id2"
	t.Run("should persist event picks and allow updates", func(t *testing.T) {
		testEvent := &model.Event{
			Id:        testEventId2,
			StartTime: time.Now().Add(4 * time.Hour).Format(time.RFC3339),
			Fights: []model.Fight{
				{Fighters: []string{"A", "B"}},
				{Fighters: []string{"C", "D"}},
				{Fighters: []string{"E", "F"}},
			},
		}

		testScraper := testEventScraper{
			maker: func() *model.Event {
				return testEvent
			},
		}

		srv := server.NewServer(testOAuth{}, testScraper, eventCache, eventPicks)
		ts := httptest.NewServer(srv)
		defer ts.Close()

		makePicksAndCheck := func(winners []string) {
			t.Helper()
			testPicks := events.PostEventPicksRequest{
				Winners: winners,
			}
			var buf bytes.Buffer
			_ = json.NewEncoder(&buf).Encode(testPicks)

			resp, err := http.Post(fmt.Sprintf("%s/events/%s/picks", ts.URL, testEventId2), "application/json", &buf)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			resp2, err := http.Get(fmt.Sprintf("%s/events/%s/picks", ts.URL, testEventId2))
			if err != nil {
				t.Fatal(err)
			}
			defer resp2.Body.Close()

			assert.Equal(t, http.StatusOK, resp2.StatusCode)

			var gotPicks picks.Picks
			if err := json.NewDecoder(resp2.Body).Decode(&gotPicks); err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, testPicks.Winners, gotPicks.Winners)
		}

		makePicksAndCheck([]string{"A", "D", "E"})
		makePicksAndCheck([]string{"A", "D", "F"})
	})

	t.Run("should score picks when event is finished", func(t *testing.T) {
		clearEvents()
		testEvent := &model.Event{
			Id:        testEventId2,
			StartTime: time.Now().Add(-12 * time.Hour).Format(time.RFC3339),
			Fights: []model.Fight{
				{Fighters: []string{"A", "B"}, Winner: "A"},
				{Fighters: []string{"C", "D"}, Winner: "D"},
				{Fighters: []string{"E", "F"}, Winner: "E"},
			},
		}

		testScraper := testEventScraper{
			maker: func() *model.Event {
				return testEvent
			},
		}

		srv := server.NewServer(testOAuth{}, testScraper, eventCache, eventPicks)
		ts := httptest.NewServer(srv)
		defer ts.Close()

		resp, err := http.Get(fmt.Sprintf("%s/events/%s/picks", ts.URL, testEventId2))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var gotPicks picks.Picks
		if err := json.NewDecoder(resp.Body).Decode(&gotPicks); err != nil {
			t.Fatal(err)
		}

		assert.NotNil(t, gotPicks.Score)
		assert.Equal(t, 2, *gotPicks.Score)
	})

	testEventId3 := "test-event-id3"
	t.Run("should get all picks across multiple events", func(t *testing.T) {
		clearEvents()
		testEvent := &model.Event{
			Id:        testEventId3,
			StartTime: time.Now().Add(4 * time.Hour).Format(time.RFC3339),
			Fights: []model.Fight{
				{Fighters: []string{"1", "2"}},
				{Fighters: []string{"3", "4"}},
				{Fighters: []string{"5", "6"}},
			},
		}

		testScraper := testEventScraper{
			maker: func() *model.Event {
				return testEvent
			},
		}

		srv := server.NewServer(testOAuth{}, testScraper, eventCache, eventPicks)
		ts := httptest.NewServer(srv)
		defer ts.Close()

		testPicks := events.PostEventPicksRequest{
			Winners: []string{"1", "4", "5"},
		}
		var buf bytes.Buffer
		_ = json.NewEncoder(&buf).Encode(testPicks)

		resp, err := http.Post(fmt.Sprintf("%s/events/%s/picks", ts.URL, testEventId3), "application/json", &buf)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		resp2, err := http.Get(fmt.Sprintf("%s/events/picks", ts.URL))
		if err != nil {
			t.Fatal(err)
		}
		defer resp2.Body.Close()

		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		var gotPicks []*events.GetAllPicksResponse
		if err := json.NewDecoder(resp2.Body).Decode(&gotPicks); err != nil {
			t.Fatal(err)
		}

		assert.Len(t, gotPicks, 2)
		assert.Equal(t, testEventId3, gotPicks[0].Picks.EventId)
		assert.Equal(t, testEventId2, gotPicks[1].Picks.EventId)

		if len(gotPicks) != 2 {
			t.Errorf("expected 2 picks, got %d", len(gotPicks))
		}
	})

	t.Run("should score picks when getting all events", func(t *testing.T) {
		clearEvents()
		testEvent := &model.Event{
			Id:        testEventId3,
			StartTime: time.Now().Add(-12 * time.Hour).Format(time.RFC3339),
			Fights: []model.Fight{
				{Fighters: []string{"1", "2"}, Winner: "1"},
				{Fighters: []string{"3", "4"}, Winner: "4"},
				{Fighters: []string{"5", "6"}, Winner: "5"},
			},
		}

		testScraper := testEventScraper{
			maker: func() *model.Event {
				return testEvent
			},
		}

		srv := server.NewServer(testOAuth{}, testScraper, eventCache, eventPicks)
		ts := httptest.NewServer(srv)
		defer ts.Close()

		resp, err := http.Get(fmt.Sprintf("%s/events/picks", ts.URL))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var gotPicks []*events.GetAllPicksResponse
		if err := json.NewDecoder(resp.Body).Decode(&gotPicks); err != nil {
			t.Fatal(err)
		}

		assert.Len(t, gotPicks, 2)
		assert.Equal(t, 3, *gotPicks[0].Picks.Score)
	})
}
