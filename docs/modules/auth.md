# auth Module

Authentication and authorization utilities.

## Provides

- `JWTManager` for HS256 token signing and verification
- `Auth(...)` middleware for bearer token auth
- `RequireRole(...)` and `RequirePerm(...)` RBAC guards
- OAuth2/OIDC helper providers:
  - auth code URL generation
  - token exchange
  - userinfo fetch
  - OIDC discovery and ID token claims parsing

## Primary API

- `NewJWTManager(secret string) *JWTManager`
- `func Auth(manager *JWTManager) elgon.Middleware`
- `func RequireRole(role string) elgon.Middleware`
- `func RequirePerm(perm string) elgon.Middleware`
- `func NewOAuth2Provider(cfg OAuth2Config, client HTTPClient) *OAuth2Provider`
- `func DiscoverOIDC(ctx context.Context, issuer string, client HTTPClient) (*OIDCDiscoveryDoc, error)`
