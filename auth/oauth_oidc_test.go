package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type mockHTTPClient struct {
	handler func(*http.Request) (*http.Response, error)
}

func (m mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.handler(req)
}

func jsonResp(status int, body any) *http.Response {
	b, _ := json.Marshal(body)
	return &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewReader(b)), Header: make(http.Header)}
}

func TestOAuth2ProviderFlow(t *testing.T) {
	client := mockHTTPClient{handler: func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/token":
			return jsonResp(http.StatusOK, OAuth2Token{AccessToken: "at", TokenType: "bearer"}), nil
		case "/userinfo":
			if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				t.Fatalf("missing bearer header")
			}
			return jsonResp(http.StatusOK, map[string]any{"sub": "u1", "email": "u@example.com"}), nil
		default:
			return jsonResp(http.StatusNotFound, map[string]any{"error": "not found"}), nil
		}
	}}

	p := NewOAuth2Provider(OAuth2Config{
		ClientID:     "cid",
		ClientSecret: "sec",
		RedirectURL:  "http://localhost/cb",
		Scopes:       []string{"openid", "profile"},
		AuthURL:      "https://idp.example.com/auth",
		TokenURL:     "https://idp.example.com/token",
		UserInfoURL:  "https://idp.example.com/userinfo",
	}, client)

	authURL, err := p.AuthCodeURL("st", map[string]string{"prompt": "consent"})
	if err != nil || !strings.Contains(authURL, "response_type=code") || !strings.Contains(authURL, "prompt=consent") {
		t.Fatalf("invalid auth url: %s err=%v", authURL, err)
	}

	tok, err := p.ExchangeCode(context.Background(), "abc")
	if err != nil || tok.AccessToken != "at" {
		t.Fatalf("exchange failed tok=%+v err=%v", tok, err)
	}
	user, err := p.FetchUserInfo(context.Background(), tok.AccessToken)
	if err != nil || user["sub"] != "u1" {
		t.Fatalf("userinfo failed user=%v err=%v", user, err)
	}
}

func TestOIDCDiscoveryAndParseIDToken(t *testing.T) {
	now := time.Now().Add(time.Hour).Unix()
	issuerURL := "https://issuer.example.com"
	client := mockHTTPClient{handler: func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			return jsonResp(http.StatusOK, OIDCDiscoveryDoc{
				Issuer:        issuerURL,
				Authorization: issuerURL + "/auth",
				Token:         issuerURL + "/token",
				UserInfo:      issuerURL + "/userinfo",
				JWKSURI:       issuerURL + "/jwks",
			}), nil
		case "/token":
			return jsonResp(http.StatusOK, OAuth2Token{AccessToken: "at", IDToken: makeTestIDToken(issuerURL, "cid", now)}), nil
		default:
			return jsonResp(http.StatusNotFound, map[string]any{"error": "not found"}), nil
		}
	}}

	doc, err := DiscoverOIDC(context.Background(), issuerURL, client)
	if err != nil {
		t.Fatal(err)
	}
	p := NewOIDCProvider(*doc, OAuth2Config{ClientID: "cid", ClientSecret: "sec", RedirectURL: "http://localhost/cb"}, client)
	tok, err := p.ExchangeCode(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := p.ParseIDTokenClaims(tok.IDToken, "cid")
	if err != nil {
		t.Fatal(err)
	}
	if claims["iss"] != issuerURL {
		t.Fatalf("unexpected claims: %v", claims)
	}
}

func makeTestIDToken(issuer, aud string, exp int64) string {
	header, _ := json.Marshal(map[string]any{"alg": "none", "typ": "JWT"})
	payload, _ := json.Marshal(map[string]any{"iss": issuer, "aud": aud, "exp": exp})
	return base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload) + "."
}
