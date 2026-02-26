SHELL := /bin/zsh

.PHONY: help tidy fmt lint test itest bench bench-ci build run dev

help:
	@echo "Available targets:"
	@echo "  make tidy      - go mod tidy"
	@echo "  make fmt       - gofmt all go files"
	@echo "  make lint      - go vet ./..."
	@echo "  make test      - unit tests"
	@echo "  make itest     - integration tests (placeholder)"
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

bench:
	mkdir -p benchmarks/results
	go test ./benchmarks/... -run=^$$ -bench=. -benchmem -count=3 | tee benchmarks/results/bench-full.out

bench-ci:
	mkdir -p benchmarks/results
	go test ./benchmarks/router ./benchmarks/middleware -run=^$$ -bench=. -benchmem -count=1 | tee benchmarks/results/bench-ci.out

build:
	mkdir -p bin
	go build -o bin/elgon-api ./cmd/api
	go build -o bin/elgon ./cmd/elgon

run:
	go run ./cmd/api

dev:
	go run ./cmd/api
