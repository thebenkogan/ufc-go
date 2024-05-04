package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

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

type Auth struct {
	provider      *oidc.Provider
	config        oauth2.Config
	tokenVerifier *oidc.IDTokenVerifier
}

type User struct {
	Id    string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func NewAuth(ctx context.Context, clientId, clientSecret string) (*Auth, error) {
	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		log.Fatal(err)
	}
	oidcConfig := &oidc.Config{
		ClientID: clientId,
	}
	verifier := provider.Verifier(oidcConfig)
	config := oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	return &Auth{provider, config, verifier}, nil
}

func (a *Auth) HandleBeginAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := randString(16)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		nonce, err := randString(16)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		setCookie(w, r, "state", state)
		setCookie(w, r, "nonce", nonce)
		http.Redirect(w, r, a.config.AuthCodeURL(state, oidc.Nonce(nonce)), http.StatusFound)
	}
}

func (a *Auth) HandleAuthCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := r.Cookie("state")
		if err != nil {
			http.Error(w, "state not found", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("state") != state.Value {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}

		oauth2Token, err := a.config.Exchange(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)
			return
		}
		idToken, err := a.tokenVerifier.Verify(r.Context(), rawIDToken)
		if err != nil {
			http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		nonce, err := r.Cookie("nonce")
		if err != nil {
			http.Error(w, "nonce not found", http.StatusBadRequest)
			return
		}
		if idToken.Nonce != nonce.Value {
			http.Error(w, "nonce did not match", http.StatusBadRequest)
			return
		}

		var user User
		if err := idToken.Claims(&user); err != nil {
			slog.Error("Error when getting claims: " + err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		setCookie(w, r, "id_token", rawIDToken)
		w.Write([]byte("Hello, " + user.Name))
	}
}

func (a *Auth) Middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idTokenCookie, err := r.Cookie("id_token")
		if err == http.ErrNoCookie {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		idToken, err := a.tokenVerifier.Verify(r.Context(), idTokenCookie.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		var user User
		if err := idToken.Claims(&user); err != nil {
			slog.Error("Error when getting claims: " + err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), "user", user)
		rWithUser := r.WithContext(ctx)
		h.ServeHTTP(w, rWithUser)
	})
}
