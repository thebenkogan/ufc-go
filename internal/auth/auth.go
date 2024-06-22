package auth

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"time"

	"github.com/thebenkogan/ufc/internal/util"
)

type OIDCAuth interface {
	HandleBeginAuth() util.Handler
	HandleAuthCallback() util.Handler
	Middleware(h util.Handler) util.Handler
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

func HandleMe(auth OIDCAuth) util.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) error {
		user := r.Context().Value("user").(User)
		util.Encode(w, http.StatusOK, user)
		return nil
	}
	return auth.Middleware(handler)
}
