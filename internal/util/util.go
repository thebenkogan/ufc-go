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

func Decode[T any](r *http.Request, v *T) {
	_ = json.NewDecoder(r.Body).Decode(v)
}

func Distinct[T comparable](slice []T) []T {
	keys := make(map[T]struct{})
	list := make([]T, 0)
	for _, entry := range slice {
		if _, ok := keys[entry]; !ok {
			keys[entry] = struct{}{}
			list = append(list, entry)
		}
	}
	return list
}
