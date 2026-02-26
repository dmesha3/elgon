# API Stability

`elgon` follows semantic versioning.

## Stable surfaces (frozen as of v0.1.0)

- `elgon` core handler/middleware contracts and routing methods
- `middleware` built-in middleware function signatures
- `config` loader entrypoints
- `auth` JWT auth middleware, RBAC guards, OAuth/OIDC provider entrypoints
- `openapi` generator entrypoints
- `jobs` queue contracts (`Queue`, `Message`, `Handler`)
- `migrate` engine entrypoints (`Up`, `Down`, `Status`, `Load`)

## Compatibility policy

- Patch releases (`v0.1.x`) will not break stable APIs.
- Minor releases (`v0.x+1.0`) may add APIs but avoid breaking stable APIs unless clearly documented.
- Any breaking change will be documented in `CHANGELOG.md` with migration notes.
