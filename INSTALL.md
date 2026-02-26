# Installation

## Go module

```bash
go get github.com/meshackkazimoto/elgon@v0.1.0
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

## Optional adapters

To include Redis/Kafka concrete adapters:

```bash
go build -tags adapters ./...
```
