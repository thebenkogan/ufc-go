package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
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
	return func(w http.ResponseWriter, r *http.Request) error {
		panic("not implemented")
	}
}

func (a testOAuth) HandleAuthCallback() util.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		panic("not implemented")
	}
}

func (a testOAuth) Middleware(h util.Handler) util.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		user := auth.User{Id: "user", Email: "user@gmail.com", Name: "user"}
		ctx := context.WithValue(r.Context(), "user", user)
		rWithUser := r.WithContext(ctx)
		return h(w, rWithUser)
	}
}

type testEventScraper struct {
	maker func() *model.Event
}

func (s testEventScraper) ScrapeEvent(id string) (*model.Event, error) {
	return s.maker(), nil
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

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status code 200, got %d", resp.StatusCode)
		}

		var gotEvent *model.Event
		if err := json.NewDecoder(resp.Body).Decode(&gotEvent); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(gotEvent, testEvent) {
			t.Errorf("expected event %+v, got %+v", testEvent, gotEvent)
		}

		if numScrapes != 1 {
			t.Errorf("expected 1 scrape, got %d", numScrapes)
		}

		// second request should hit the cache
		resp2, err := http.Get(ts.URL + "/events/" + testEventId)
		if err != nil {
			t.Fatal(err)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			t.Errorf("expected status code 200, got %d", resp2.StatusCode)
		}

		var gotEvent2 model.Event
		if err := json.NewDecoder(resp2.Body).Decode(&gotEvent2); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(gotEvent2, *testEvent) {
			t.Errorf("expected event %+v, got %+v", testEvent, gotEvent)
		}

		if numScrapes != 1 {
			t.Errorf("expected 1 scrape, got %d", numScrapes)
		}
	})

	testEventId := "test-event-id2"
	t.Run("should persist event picks", func(t *testing.T) {
		testEvent := &model.Event{
			Id:        testEventId,
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

		testPicks := events.PostEventPicksRequest{
			Winners: []string{"A", "D", "F"},
		}
		var buf bytes.Buffer
		_ = json.NewEncoder(&buf).Encode(testPicks)

		resp, err := http.Post(fmt.Sprintf("%s/events/%s/picks", ts.URL, testEventId), "application/json", &buf)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status code 200, got %d", resp.StatusCode)
		}

		resp2, err := http.Get(fmt.Sprintf("%s/events/%s/picks", ts.URL, testEventId))
		if err != nil {
			t.Fatal(err)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			t.Errorf("expected status code 200, got %d", resp2.StatusCode)
		}

		var gotPicks picks.Picks
		if err := json.NewDecoder(resp2.Body).Decode(&gotPicks); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(gotPicks.Winners, testPicks.Winners) {
			t.Errorf("expected picks %+v, got %+v", testPicks.Winners, gotPicks.Winners)
		}
	})

	t.Run("should score picks when event is finished", func(t *testing.T) {
		clearEvents()
		testEvent := &model.Event{
			Id:        testEventId,
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

		resp, err := http.Get(fmt.Sprintf("%s/events/%s/picks", ts.URL, testEventId))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status code 200, got %d", resp.StatusCode)
		}

		var gotPicks picks.Picks
		if err := json.NewDecoder(resp.Body).Decode(&gotPicks); err != nil {
			t.Fatal(err)
		}

		if gotPicks.Score == nil {
			t.Errorf("expected picks to be scored, got nil")
		}

		if gotPicks.Score != nil && *gotPicks.Score != 2 {
			t.Errorf("expected score 2, got %d", *gotPicks.Score)
		}
	})
}
