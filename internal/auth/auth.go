package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"time"

	"github.com/thebenkogan/ufc/internal/util/api_util"
)

type OIDCAuth interface {
	HandleBeginAuth() api_util.Handler
	HandleAuthCallback() api_util.Handler
	Middleware(h api_util.Handler) api_util.Handler
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

func HandleMe(auth OIDCAuth) api_util.Handler {
	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		user := ctx.Value("user").(User)
		api_util.Encode(w, http.StatusOK, user)
		return nil
	}
	return auth.Middleware(handler)
}
