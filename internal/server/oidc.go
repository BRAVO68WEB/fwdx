package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type discoveryDocument struct {
	Issuer                      string `json:"issuer"`
	AuthorizationEndpoint       string `json:"authorization_endpoint"`
	TokenEndpoint               string `json:"token_endpoint"`
	UserinfoEndpoint            string `json:"userinfo_endpoint"`
	JWKSURI                     string `json:"jwks_uri"`
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
}

type oidcManager struct {
	enabled        bool
	issuer         string
	deviceClientID string
	clientSecret   string
	scopes         []string
	provider       *oidc.Provider
	verifier       *oidc.IDTokenVerifier
	oauth2Config   oauth2.Config
	discovery      discoveryDocument
	httpClient     *http.Client
}

type devicePollResult struct {
	Claims   OIDCClaims
	Pending  bool
	Interval int
}

func newOIDCManager(ctx context.Context, cfg Config) (*oidcManager, error) {
	issuer := strings.TrimSpace(cfg.OIDCIssuer)
	if issuer == "" {
		return &oidcManager{enabled: false}, nil
	}
	if strings.TrimSpace(cfg.OIDCClientID) == "" || strings.TrimSpace(cfg.OIDCRedirectURL) == "" {
		return nil, fmt.Errorf("oidc requires issuer, client id, and redirect url")
	}
	m := &oidcManager{
		enabled:        true,
		issuer:         issuer,
		deviceClientID: strings.TrimSpace(cfg.OIDCDeviceClientID),
		clientSecret:   strings.TrimSpace(cfg.OIDCClientSecret),
		scopes:         cfg.OIDCScopes,
		httpClient:     &http.Client{Timeout: 10 * time.Second},
	}
	if m.deviceClientID == "" {
		m.deviceClientID = cfg.OIDCClientID
	}
	if len(m.scopes) == 0 {
		m.scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}
	m.provider = provider
	m.verifier = provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID})
	m.oauth2Config = oauth2.Config{
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.OIDCRedirectURL,
		Scopes:       m.scopes,
	}
	if err := m.loadDiscovery(ctx); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *oidcManager) loadDiscovery(ctx context.Context) error {
	wellKnown := strings.TrimSuffix(m.issuer, "/") + "/.well-known/openid-configuration"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, wellKnown, nil)
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("oidc discovery document: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("oidc discovery document: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&m.discovery); err != nil {
		return fmt.Errorf("decode oidc discovery document: %w", err)
	}
	return nil
}

func (m *oidcManager) Enabled() bool {
	return m != nil && m.enabled
}

func randomString(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generatePKCEVerifier() (verifier, challenge string, err error) {
	verifier, err = randomString(32)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func (m *oidcManager) authCodeURL(state, challenge string) string {
	return m.oauth2Config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func (m *oidcManager) exchangeCode(ctx context.Context, code, verifier string) (OIDCClaims, error) {
	tok, err := m.oauth2Config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		return OIDCClaims{}, fmt.Errorf("oidc code exchange: %w", err)
	}
	return m.claimsFromToken(ctx, tok)
}

func (m *oidcManager) claimsFromToken(ctx context.Context, tok *oauth2.Token) (OIDCClaims, error) {
	claims := OIDCClaims{}
	if raw, _ := tok.Extra("id_token").(string); raw != "" {
		idTok, err := m.verifier.Verify(ctx, raw)
		if err != nil {
			return OIDCClaims{}, fmt.Errorf("verify id token: %w", err)
		}
		var parsed struct {
			Sub    string   `json:"sub"`
			Email  string   `json:"email"`
			Name   string   `json:"name"`
			Groups []string `json:"groups"`
		}
		if err := idTok.Claims(&parsed); err != nil {
			return OIDCClaims{}, fmt.Errorf("parse id token claims: %w", err)
		}
		claims.Subject = parsed.Sub
		claims.Email = parsed.Email
		claims.DisplayName = parsed.Name
		claims.Groups = parsed.Groups
	}
	if claims.Subject != "" && claims.Email != "" && claims.DisplayName != "" {
		return claims, nil
	}
	ui, err := m.provider.UserInfo(ctx, oauth2.StaticTokenSource(tok))
	if err != nil {
		if claims.Subject != "" {
			return claims, nil
		}
		return OIDCClaims{}, fmt.Errorf("userinfo: %w", err)
	}
	var parsed struct {
		Sub    string   `json:"sub"`
		Email  string   `json:"email"`
		Name   string   `json:"name"`
		Groups []string `json:"groups"`
	}
	if err := ui.Claims(&parsed); err != nil {
		return OIDCClaims{}, fmt.Errorf("userinfo claims: %w", err)
	}
	if claims.Subject == "" {
		claims.Subject = parsed.Sub
	}
	if claims.Email == "" {
		claims.Email = parsed.Email
	}
	if claims.DisplayName == "" {
		claims.DisplayName = parsed.Name
	}
	if len(claims.Groups) == 0 {
		claims.Groups = parsed.Groups
	}
	if claims.Subject == "" {
		return OIDCClaims{}, errors.New("oidc subject missing")
	}
	return claims, nil
}

func (m *oidcManager) startDeviceAuthorization(ctx context.Context) (*DeviceAuthorization, error) {
	if strings.TrimSpace(m.discovery.DeviceAuthorizationEndpoint) == "" {
		return nil, errors.New("oidc provider does not expose device authorization endpoint")
	}
	form := url.Values{}
	form.Set("client_id", m.deviceClientID)
	form.Set("scope", strings.Join(m.scopes, " "))
	if m.clientSecret != "" {
		form.Set("client_secret", m.clientSecret)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, m.discovery.DeviceAuthorizationEndpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("device authorization: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var out DeviceAuthorization
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Interval <= 0 {
		out.Interval = 5
	}
	return &out, nil
}

func (m *oidcManager) pollDeviceAuthorization(ctx context.Context, deviceCode string) (*devicePollResult, error) {
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	form.Set("device_code", deviceCode)
	form.Set("client_id", m.deviceClientID)
	if m.clientSecret != "" {
		form.Set("client_secret", m.clientSecret)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, m.discovery.TokenEndpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
			Interval         int    `json:"interval"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&e); err == nil {
			switch e.Error {
			case "authorization_pending", "slow_down":
				interval := e.Interval
				if interval <= 0 {
					interval = 5
				}
				return &devicePollResult{Pending: true, Interval: interval}, nil
			}
			if e.Error != "" {
				return nil, fmt.Errorf("device token exchange: %s: %s", e.Error, e.ErrorDescription)
			}
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("device token exchange: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		IDToken     string `json:"id_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}
	tok := &oauth2.Token{AccessToken: tokenResp.AccessToken, TokenType: tokenResp.TokenType}
	if tokenResp.ExpiresIn > 0 {
		tok.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	claims, err := m.claimsFromToken(ctx, tok.WithExtra(map[string]any{"id_token": tokenResp.IDToken}))
	if err != nil {
		return nil, err
	}
	return &devicePollResult{Claims: claims}, nil
}
