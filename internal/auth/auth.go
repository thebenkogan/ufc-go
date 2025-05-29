package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"time"

	"github.com/thebenkogan/ufc/internal/util/api"
)

type OIDCAuth interface {
	HandleBeginAuth() api.Handler
	HandleAuthCallback() api.Handler
	Middleware(h api.Handler) api.Handler
}

type User struct {
	Id    string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	_, _ = io.ReadFull(rand.Reader, b)
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func setCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   int(time.Hour.Seconds()),
		Secure:   r.TLS != nil,
		HttpOnly: true,
		Path:     "/",
	}
	http.SetCookie(w, c)
}

func HandleMe(auth OIDCAuth) api.Handler {
	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		api.Encode(w, http.StatusOK, GetUser(ctx))
		return nil
	}
	return auth.Middleware(handler)
}

type authKey string

const userKey authKey = "user"

func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func GetUser(ctx context.Context) *User {
	if user, ok := ctx.Value(userKey).(*User); ok {
		return user
	}
	return nil
}
