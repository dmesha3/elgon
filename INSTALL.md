# Installation

## Go module

```bash
go get github.com/meshackkazimoto/elgon@v0.1.1
```

## Build from source

```bash
git clone https://github.com/meshackkazimoto/elgon.git
cd elgon
go build ./...
```

## CLI (local)

```bash
go build -o ./bin/elgon ./cmd/elgon
./bin/elgon --help
```

## Optional hot reload (developer experience)

Run dev mode with reload:

```bash
make dev HOT_RELOAD=1
```

If `air` is not installed locally, the CLI falls back to `go run github.com/air-verse/air@latest`.

## Optional adapters

To include Redis/Kafka concrete adapters:

```bash
go build -tags adapters ./...
```
