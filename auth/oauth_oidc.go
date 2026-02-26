package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPClient allows custom HTTP client injection.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// OAuth2Config configures an OAuth2/OIDC provider.
type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string

	AuthURL     string
	TokenURL    string
	UserInfoURL string
}

// OAuth2Token contains token endpoint response payload.
type OAuth2Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// OAuth2Provider provides OAuth2 authorization and token exchange helpers.
type OAuth2Provider struct {
	cfg    OAuth2Config
	client HTTPClient
}

func NewOAuth2Provider(cfg OAuth2Config, client HTTPClient) *OAuth2Provider {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &OAuth2Provider{cfg: cfg, client: client}
}

func (p *OAuth2Provider) AuthCodeURL(state string, extra map[string]string) (string, error) {
	if p.cfg.AuthURL == "" {
		return "", errors.New("auth: oauth auth url is required")
	}
	u, err := url.Parse(p.cfg.AuthURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", p.cfg.ClientID)
	q.Set("redirect_uri", p.cfg.RedirectURL)
	if state != "" {
		q.Set("state", state)
	}
	if len(p.cfg.Scopes) > 0 {
		q.Set("scope", strings.Join(p.cfg.Scopes, " "))
	}
	for k, v := range extra {
		if strings.TrimSpace(k) == "" {
			continue
		}
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (p *OAuth2Provider) ExchangeCode(ctx context.Context, code string) (OAuth2Token, error) {
	if p.cfg.TokenURL == "" {
		return OAuth2Token{}, errors.New("auth: oauth token url is required")
	}
	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", code)
	values.Set("redirect_uri", p.cfg.RedirectURL)
	values.Set("client_id", p.cfg.ClientID)
	values.Set("client_secret", p.cfg.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.TokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return OAuth2Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return OAuth2Token{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return OAuth2Token{}, fmt.Errorf("auth: token exchange failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var token OAuth2Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return OAuth2Token{}, err
	}
	if token.AccessToken == "" {
		return OAuth2Token{}, errors.New("auth: missing access_token in token response")
	}
	return token, nil
}

func (p *OAuth2Provider) FetchUserInfo(ctx context.Context, accessToken string) (map[string]any, error) {
	if p.cfg.UserInfoURL == "" {
		return nil, errors.New("auth: userinfo url not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("auth: userinfo request failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	out := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// OIDCDiscoveryDoc is OpenID Connect discovery metadata.
type OIDCDiscoveryDoc struct {
	Issuer        string   `json:"issuer"`
	Authorization string   `json:"authorization_endpoint"`
	Token         string   `json:"token_endpoint"`
	UserInfo      string   `json:"userinfo_endpoint"`
	JWKSURI       string   `json:"jwks_uri"`
	Algs          []string `json:"id_token_signing_alg_values_supported"`
}

// OIDCProvider wraps OAuth2 with OIDC discovery and ID token parsing.
type OIDCProvider struct {
	Issuer string
	oauth  *OAuth2Provider
	jwks   string
}

func DiscoverOIDC(ctx context.Context, issuer string, client HTTPClient) (*OIDCDiscoveryDoc, error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	endpoint := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("auth: oidc discovery failed status=%d", resp.StatusCode)
	}
	var doc OIDCDiscoveryDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}
	if doc.Issuer == "" || doc.Authorization == "" || doc.Token == "" {
		return nil, errors.New("auth: incomplete oidc discovery document")
	}
	return &doc, nil
}

func NewOIDCProvider(doc OIDCDiscoveryDoc, cfg OAuth2Config, client HTTPClient) *OIDCProvider {
	cfg.AuthURL = doc.Authorization
	cfg.TokenURL = doc.Token
	cfg.UserInfoURL = doc.UserInfo
	return &OIDCProvider{
		Issuer: doc.Issuer,
		oauth:  NewOAuth2Provider(cfg, client),
		jwks:   doc.JWKSURI,
	}
}

func (p *OIDCProvider) AuthCodeURL(state string, nonce string) (string, error) {
	extra := map[string]string{}
	if nonce != "" {
		extra["nonce"] = nonce
	}
	return p.oauth.AuthCodeURL(state, extra)
}

func (p *OIDCProvider) ExchangeCode(ctx context.Context, code string) (OAuth2Token, error) {
	return p.oauth.ExchangeCode(ctx, code)
}

func (p *OIDCProvider) FetchUserInfo(ctx context.Context, accessToken string) (map[string]any, error) {
	return p.oauth.FetchUserInfo(ctx, accessToken)
}

// ParseIDTokenClaims decodes OIDC ID token claims without signature verification.
// Use only when token is obtained directly from trusted provider token endpoint over TLS.
func (p *OIDCProvider) ParseIDTokenClaims(idToken, expectedAudience string) (map[string]any, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims := map[string]any{}
	if err := json.NewDecoder(bytes.NewReader(payload)).Decode(&claims); err != nil {
		return nil, ErrInvalidToken
	}
	if iss, _ := claims["iss"].(string); p.Issuer != "" && iss != p.Issuer {
		return nil, errors.New("auth: oidc issuer mismatch")
	}
	if expectedAudience != "" && !audienceContains(claims["aud"], expectedAudience) {
		return nil, errors.New("auth: oidc audience mismatch")
	}
	if exp, ok := toInt64(claims["exp"]); ok && time.Now().Unix() > exp {
		return nil, ErrExpiredToken
	}
	return claims, nil
}

func audienceContains(raw any, audience string) bool {
	s, ok := raw.(string)
	if ok {
		return s == audience
	}
	arr, ok := raw.([]any)
	if !ok {
		return false
	}
	for _, v := range arr {
		if as, ok := v.(string); ok && as == audience {
			return true
		}
	}
	return false
}

func toInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case float64:
		return int64(x), true
	case int64:
		return x, true
	case int:
		return int64(x), true
	default:
		return 0, false
	}
}
