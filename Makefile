SHELL := /bin/zsh

.PHONY: help tidy fmt lint test itest adapters-itest bench bench-ci build run dev

help:
	@echo "Available targets:"
	@echo "  make tidy      - go mod tidy"
	@echo "  make fmt       - gofmt all go files"
	@echo "  make lint      - go vet ./..."
	@echo "  make test      - unit tests"
	@echo "  make itest     - integration tests (placeholder)"
	@echo "  make adapters-itest - adapter integration tests (requires services)"
	@echo "  make bench     - full benchmark suite"
	@echo "  make bench-ci  - quick benchmark subset"
	@echo "  make build     - build API and CLI binaries"
	@echo "  make run       - run API server"
	@echo "  make dev       - run API server (dev mode)"

tidy:
	go mod tidy

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.gocache/*' -not -path './.gomodcache/*')

lint:
	go vet ./...

test:
	go test ./...

itest:
	go test ./... -run Integration

adapters-itest:
	go test -tags "adapters integration" ./jobs/redisadapter ./jobs/kafkaadapter -v

bench:
	mkdir -p benchmarks/results
	go test ./benchmarks/... -run=^$$ -bench=. -benchmem -count=3 | tee benchmarks/results/bench-full.out

bench-ci:
	mkdir -p benchmarks/results
	./scripts/bench_guard.sh

build:
	mkdir -p bin
	go build -o bin/elgon-api ./cmd/api
	go build -o bin/elgon ./cmd/elgon

run:
	go run ./cmd/api

dev:
	go run ./cmd/api
