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
	"github.com/thebenkogan/ufc/internal/util/api_util"
)

type testOAuth struct{}

func (a testOAuth) HandleBeginAuth() api_util.Handler {
	return func(_ context.Context, w http.ResponseWriter, r *http.Request) error {
		panic("not implemented")
	}
}

func (a testOAuth) HandleAuthCallback() api_util.Handler {
	return func(_ context.Context, w http.ResponseWriter, r *http.Request) error {
		panic("not implemented")
	}
}

func (a testOAuth) Middleware(h api_util.Handler) api_util.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		user := auth.User{Id: "user", Email: "user@gmail.com", Name: "user"}
		ctx = auth.WithUser(ctx, &user)
		rWithUser := r.WithContext(ctx)
		return h(ctx, w, rWithUser)
	}
}

type testEventScraper struct {
	maker func(id string) *model.Event
}

func (s testEventScraper) ScrapeEvent(id string) (*model.Event, error) {
	return s.maker(id), nil
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
			maker: func(_ string) *model.Event {
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
	testEvent2 := &model.Event{
		Id:        testEventId2,
		StartTime: time.Now().Add(4 * time.Hour).Format(time.RFC3339),
		Fights: []model.Fight{
			{Fighters: []string{"A", "B"}},
			{Fighters: []string{"C", "D"}},
			{Fighters: []string{"E", "F"}},
		},
	}
	t.Run("should persist event picks and allow updates", func(t *testing.T) {
		testScraper := testEventScraper{
			maker: func(_ string) *model.Event {
				return testEvent2
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

	testEventId3 := "test-event-id3"
	testEvent3 := &model.Event{
		Id:        testEventId3,
		StartTime: time.Now().Add(4 * time.Hour).Format(time.RFC3339),
		Fights: []model.Fight{
			{Fighters: []string{"1", "2"}},
			{Fighters: []string{"3", "4"}},
			{Fighters: []string{"5", "6"}},
		},
	}
	t.Run("should get all picks across multiple events", func(t *testing.T) {
		clearEvents()

		testScraper := testEventScraper{
			maker: func(id string) *model.Event {
				if id == testEventId3 {
					return testEvent3
				} else if id == testEventId2 {
					return testEvent2
				}
				return nil
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

		b, _ := json.Marshal(gotPicks)
		fmt.Println(string(b))

		assert.Len(t, gotPicks, 2)
		assert.Equal(t, testEventId3, gotPicks[0].Picks.EventId)
		assert.Equal(t, testEventId2, gotPicks[1].Picks.EventId)
		assert.Equal(t, testEvent3, gotPicks[0].Event)
		assert.Equal(t, testEvent2, gotPicks[1].Event)

		if len(gotPicks) != 2 {
			t.Errorf("expected 2 picks, got %d", len(gotPicks))
		}
	})
}
