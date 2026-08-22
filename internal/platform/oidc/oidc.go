// Package oidc adapts the externally managed Keycloak OIDC client to local identity keys.
package oidc

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
)

type Config struct {
	IssuerURL        string
	ClientID         string
	ClientSecretFile string
	PublicBaseURL    string
}

type Identity struct {
	Key         access.IdentityKey
	DisplayName string
	Email       string
	Nonce       string
}

type Client struct {
	issuer   string
	oauth2   oauth2.Config
	verifier *coreoidc.IDTokenVerifier
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.IssuerURL == "" || cfg.ClientID == "" || cfg.PublicBaseURL == "" {
		return nil, errors.New("OIDC issuer, client ID, and public base URL are required")
	}
	base, err := url.Parse(cfg.PublicBaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, errors.New("OIDC public base URL must be absolute")
	}
	secret := ""
	if cfg.ClientSecretFile != "" {
		contents, readErr := os.ReadFile(cfg.ClientSecretFile)
		if readErr != nil {
			return nil, fmt.Errorf("read OIDC client secret: %w", readErr)
		}
		secret = strings.TrimSpace(string(contents))
	}
	provider, err := coreoidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}
	base.Path = "/auth/callback"
	base.RawQuery = ""
	base.Fragment = ""
	return &Client{
		issuer: cfg.IssuerURL,
		oauth2: oauth2.Config{
			ClientID: cfg.ClientID, ClientSecret: secret, Endpoint: provider.Endpoint(),
			RedirectURL: base.String(), Scopes: []string{coreoidc.ScopeOpenID, "profile", "email"},
		},
		verifier: provider.Verifier(&coreoidc.Config{ClientID: cfg.ClientID}),
	}, nil
}

func (c *Client) AuthorizationURL(state, nonce, challenge string) string {
	return c.oauth2.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func (c *Client) Exchange(ctx context.Context, code, verifier, expectedNonce string) (Identity, error) {
	token, err := c.oauth2.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return Identity{}, fmt.Errorf("exchange OIDC authorization code: %w", err)
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok || raw == "" {
		return Identity{}, errors.New("OIDC response did not contain an ID token")
	}
	return c.verify(ctx, raw, expectedNonce)
}

func (c *Client) VerifyBearer(ctx context.Context, raw string) (Identity, error) {
	return c.verify(ctx, raw, "")
}

func (c *Client) verify(ctx context.Context, raw, expectedNonce string) (Identity, error) {
	token, err := c.verifier.Verify(ctx, raw)
	if err != nil {
		return Identity{}, fmt.Errorf("verify OIDC token: %w", err)
	}
	var claims struct {
		Subject string `json:"sub"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Nonce   string `json:"nonce"`
	}
	if err := token.Claims(&claims); err != nil {
		return Identity{}, fmt.Errorf("decode OIDC claims: %w", err)
	}
	if expectedNonce != "" && claims.Nonce != expectedNonce {
		return Identity{}, errors.New("OIDC nonce does not match the login flow")
	}
	key := access.IdentityKey{Issuer: c.issuer, Subject: claims.Subject}
	if err := key.Validate(); err != nil {
		return Identity{}, fmt.Errorf("validate OIDC identity: %w", err)
	}
	return Identity{Key: key, DisplayName: claims.Name, Email: claims.Email, Nonce: claims.Nonce}, nil
}

func PKCEChallenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func ValidateReturnTo(value string) (string, error) {
	if value == "" {
		return "/en", nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") {
		return "", errors.New("return_to must be a local absolute path")
	}
	if !strings.HasPrefix(parsed.Path, "/en") && !strings.HasPrefix(parsed.Path, "/pt-br") {
		return "", errors.New("return_to must use a supported localized route")
	}
	return parsed.RequestURI(), nil
}
