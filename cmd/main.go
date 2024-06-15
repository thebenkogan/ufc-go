package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/thebenkogan/ufc/internal/auth"
	"github.com/thebenkogan/ufc/internal/cache"
	"github.com/thebenkogan/ufc/internal/events"
	"github.com/thebenkogan/ufc/internal/picks"
	"github.com/thebenkogan/ufc/internal/server"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("error loading .env file: %w", err)
	}

	auth, err := auth.NewGoogleAuth(ctx, os.Getenv("GOOGLE_OAUTH2_CLIENT_ID"), os.Getenv("GOOGLE_OAUTH2_CLIENT_SECRET"))
	if err != nil {
		return fmt.Errorf("error creating auth: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: net.JoinHostPort(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
	})
	defer rdb.Close()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("error pinging redis cache: %w", err)
	}
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
		return fmt.Errorf("error parsing postgres URL: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, pgCfg)
	if err != nil {
		return fmt.Errorf("error creating postgres pool: %w", err)
	}
	defer pool.Close()

	eventPicks := picks.NewPostgresEventPicks(pool)

	srv := server.NewServer(auth, events.NewESPNEventScraper(), eventCache, eventPicks)
	httpServer := &http.Server{
		Addr:    net.JoinHostPort("localhost", "8080"),
		Handler: srv,
	}
	go func() {
		log.Printf("listening on %s\n", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error(fmt.Sprintf("error listening and serving: %s\n", err))
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		slog.Info("shutting down http server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error(fmt.Sprintf("error shutting down http server: %s\n", err))
		}
	}()
	wg.Wait()
	return nil
}
