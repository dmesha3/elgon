package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/meshackkazimoto/elgon"
)

const userContextKey = "auth.principal"

var (
	ErrInvalidToken = errors.New("auth: invalid token")
	ErrExpiredToken = errors.New("auth: token expired")
)

// Principal is the authenticated user identity.
type Principal struct {
	ID    string   `json:"id"`
	Email string   `json:"email,omitempty"`
	Roles []string `json:"roles,omitempty"`
	Perms []string `json:"perms,omitempty"`
}

// Claims contains JWT payload fields.
type Claims struct {
	Sub   string   `json:"sub"`
	Email string   `json:"email,omitempty"`
	Roles []string `json:"roles,omitempty"`
	Perms []string `json:"perms,omitempty"`
	Exp   int64    `json:"exp,omitempty"`
	Iat   int64    `json:"iat,omitempty"`
}

// JWTManager signs and verifies HS256 JWTs.
type JWTManager struct {
	secret []byte
	now    func() time.Time
}

func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{secret: []byte(secret), now: time.Now}
}

func (m *JWTManager) Sign(claims Claims, ttl time.Duration) (string, error) {
	if len(m.secret) == 0 {
		return "", errors.New("auth: secret is required")
	}
	now := m.now().Unix()
	if claims.Iat == 0 {
		claims.Iat = now
	}
	if ttl > 0 {
		claims.Exp = now + int64(ttl.Seconds())
	}

	headerJSON, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	header := b64Encode(headerJSON)
	payload := b64Encode(claimsJSON)
	signingInput := header + "." + payload
	sig := signHS256(signingInput, m.secret)
	return signingInput + "." + sig, nil
}

func (m *JWTManager) Verify(token string) (Claims, error) {
	if len(m.secret) == 0 {
		return Claims{}, errors.New("auth: secret is required")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}
	signingInput := parts[0] + "." + parts[1]
	expected := signHS256(signingInput, m.secret)
	if !hmac.Equal([]byte(parts[2]), []byte(expected)) {
		return Claims{}, ErrInvalidToken
	}

	headerRaw, err := b64Decode(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var header map[string]any
	if err := json.Unmarshal(headerRaw, &header); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if header["alg"] != "HS256" {
		return Claims{}, fmt.Errorf("auth: unsupported alg: %v", header["alg"])
	}

	payloadRaw, err := b64Decode(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(payloadRaw, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if claims.Exp > 0 && m.now().Unix() > claims.Exp {
		return Claims{}, ErrExpiredToken
	}
	return claims, nil
}

// Auth validates bearer tokens and injects principal into request context.
func Auth(manager *JWTManager) elgon.Middleware {
	return func(next elgon.HandlerFunc) elgon.HandlerFunc {
		return func(c *elgon.Ctx) error {
			if manager == nil {
				return elgon.ErrUnauthorized("auth manager is not configured")
			}
			authHeader := c.Header("Authorization")
			const prefix = "Bearer "
			if !strings.HasPrefix(authHeader, prefix) {
				return elgon.ErrUnauthorized("missing bearer token")
			}
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
			claims, err := manager.Verify(token)
			if err != nil {
				return elgon.ErrUnauthorized("invalid token")
			}
			c.Set(userContextKey, Principal{ID: claims.Sub, Email: claims.Email, Roles: claims.Roles, Perms: claims.Perms})
			return next(c)
		}
	}
}

func FromCtx(c *elgon.Ctx) (Principal, bool) {
	v, ok := c.Get(userContextKey)
	if !ok {
		return Principal{}, false
	}
	p, ok := v.(Principal)
	return p, ok
}

// RequireRole blocks requests if principal does not have a specific role.
func RequireRole(role string) elgon.Middleware {
	return func(next elgon.HandlerFunc) elgon.HandlerFunc {
		return func(c *elgon.Ctx) error {
			p, ok := FromCtx(c)
			if !ok {
				return elgon.ErrUnauthorized("authentication required")
			}
			if !contains(p.Roles, role) {
				return elgon.ErrForbidden("required role missing")
			}
			return next(c)
		}
	}
}

// RequirePerm blocks requests if principal does not have a specific permission.
func RequirePerm(perm string) elgon.Middleware {
	return func(next elgon.HandlerFunc) elgon.HandlerFunc {
		return func(c *elgon.Ctx) error {
			p, ok := FromCtx(c)
			if !ok {
				return elgon.ErrUnauthorized("authentication required")
			}
			if !contains(p.Perms, perm) {
				return elgon.ErrForbidden("required permission missing")
			}
			return next(c)
		}
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func signHS256(input string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	_, _ = h.Write([]byte(input))
	return b64Encode(h.Sum(nil))
}

func b64Encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func b64Decode(data string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(data)
}
