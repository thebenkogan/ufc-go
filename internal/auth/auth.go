package auth

import (
	"context"
	"log"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

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
