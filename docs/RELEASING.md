# Releasing

## v0.1.0 release steps

1. Ensure main is green:

```bash
go test ./...
./scripts/bench_guard.sh
```

2. Update changelog and install docs if needed:

- `CHANGELOG.md`
- `INSTALL.md`

3. Create and push tag:

```bash
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

4. GitHub Actions `release` workflow publishes binaries and release notes.
