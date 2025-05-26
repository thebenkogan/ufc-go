package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/thebenkogan/ufc/internal/util/api_util"
	"golang.org/x/oauth2"
)

type GoogleAuth struct {
	provider      *oidc.Provider
	config        oauth2.Config
	tokenVerifier *oidc.IDTokenVerifier
}

func NewGoogleAuth(ctx context.Context, clientId, clientSecret string) (*GoogleAuth, error) {
	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, err
	}
	oidcConfig := &oidc.Config{
		ClientID: clientId,
	}
	verifier := provider.Verifier(oidcConfig)
	config := oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  "http://localhost:5173/auth/google/callback",
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	return &GoogleAuth{provider, config, verifier}, nil
}

func (a *GoogleAuth) HandleBeginAuth() api_util.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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

func (a *GoogleAuth) HandleAuthCallback() api_util.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		state, err := r.Cookie("state")
		if err != nil {
			http.Error(w, "state not found", http.StatusBadRequest)
			return nil
		}
		if r.URL.Query().Get("state") != state.Value {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return nil
		}

		oauth2Token, err := a.config.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			return fmt.Errorf("failed to exchange token: %w", err)
		}
		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			return fmt.Errorf("no id_token field in oauth2 token")
		}
		idToken, err := a.tokenVerifier.Verify(ctx, rawIDToken)
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
		_, _ = w.Write([]byte("Hello, " + user.Name))

		return nil
	}
}

func (a *GoogleAuth) Middleware(h api_util.Handler) api_util.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		idTokenCookie, err := r.Cookie("id_token")
		if err == http.ErrNoCookie {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return nil
		}

		idToken, err := a.tokenVerifier.Verify(ctx, idTokenCookie.Value)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return nil
		}

		var user User
		if err := idToken.Claims(&user); err != nil {
			return fmt.Errorf("failed to get claims: %w", err)
		}

		ctx = context.WithValue(ctx, "user", user)
		rWithUser := r.WithContext(ctx)

		return h(ctx, w, rWithUser)
	}
}
