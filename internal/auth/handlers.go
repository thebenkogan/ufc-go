package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/thebenkogan/ufc/internal/util"
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

func (a *Auth) HandleBeginAuth() util.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		state, err := randString(16)
		if err != nil {
			return err
		}
		nonce, err := randString(16)
		if err != nil {
			return err
		}
		setCookie(w, r, "state", state)
		setCookie(w, r, "nonce", nonce)
		http.Redirect(w, r, a.config.AuthCodeURL(state, oidc.Nonce(nonce)), http.StatusFound)
		return nil
	}
}

func (a *Auth) HandleAuthCallback() util.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		state, err := r.Cookie("state")
		if err != nil {
			http.Error(w, "state not found", http.StatusBadRequest)
			return nil
		}
		if r.URL.Query().Get("state") != state.Value {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return nil
		}

		oauth2Token, err := a.config.Exchange(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			return fmt.Errorf("failed to exchange token: %w", err)
		}
		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			return fmt.Errorf("no id_token field in oauth2 token")
		}
		idToken, err := a.tokenVerifier.Verify(r.Context(), rawIDToken)
		if err != nil {
			return fmt.Errorf("failed to verify ID Token: %w", err)
		}

		nonce, err := r.Cookie("nonce")
		if err != nil {
			http.Error(w, "nonce not found", http.StatusBadRequest)
			return nil
		}
		if idToken.Nonce != nonce.Value {
			http.Error(w, "nonce did not match", http.StatusBadRequest)
			return nil
		}

		var user User
		if err := idToken.Claims(&user); err != nil {
			return fmt.Errorf("failed to get claims: %w", err)
		}

		setCookie(w, r, "id_token", rawIDToken)
		w.Write([]byte("Hello, " + user.Name))

		return nil
	}
}

func (a *Auth) Middleware(h util.Handler) util.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		idTokenCookie, err := r.Cookie("id_token")
		if err == http.ErrNoCookie {
			http.Redirect(w, r, "/login", http.StatusFound)
			return nil
		}

		idToken, err := a.tokenVerifier.Verify(r.Context(), idTokenCookie.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return nil
		}

		var user User
		if err := idToken.Claims(&user); err != nil {
			return fmt.Errorf("failed to get claims: %w", err)
		}

		ctx := context.WithValue(r.Context(), "user", user)
		rWithUser := r.WithContext(ctx)

		return h(w, rWithUser)
	}
}
