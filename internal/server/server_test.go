package server_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/thebenkogan/ufc/internal/cache"
	"github.com/thebenkogan/ufc/internal/model"
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
	return h
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

	rdb := redis.NewClient(&redis.Options{
		Addr: net.JoinHostPort(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
	})
	defer rdb.Close()
	if _, err := rdb.FlushAll(context.Background()).Result(); err != nil {
		t.Fatal(err)
	}
	eventCache := cache.NewRedisEventCache(rdb)

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

	srv := server.NewServer(testOAuth{}, testScraper, eventCache)

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
}
