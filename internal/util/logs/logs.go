package logs

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/google/uuid"
)

type contextKey string

const loggerKey contextKey = "logger"

func WithRequestLogger(r *http.Request) context.Context {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	requestId := uuid.New().String()
	logger = logger.With(slog.String("request_id", requestId))
	logger.Info("request", "method", r.Method, "path", r.URL.Path)
	return context.WithValue(r.Context(), loggerKey, logger)
}

// Logger retrieves the logger from context, returns default logger if not found
func Logger(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}
