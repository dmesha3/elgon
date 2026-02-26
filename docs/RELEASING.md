# Releasing

## Release steps

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
git tag -a vX.Y.Z -m "vX.Y.Z"
git push origin vX.Y.Z
```

4. GitHub Actions `release` workflow publishes binaries and release notes.
