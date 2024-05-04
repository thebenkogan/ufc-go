package util

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type Handler func(w http.ResponseWriter, r *http.Request) error

func Encode[T any](w http.ResponseWriter, status int, v T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error(err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
