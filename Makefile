.PHONY: build lint test test-integration fmt vet tidy all

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS  = -w -s \
	-X github.com/bschimke95/jara/internal/cmd.version=$(VERSION) \
	-X github.com/bschimke95/jara/internal/cmd.commit=$(COMMIT) \
	-X github.com/bschimke95/jara/internal/cmd.date=$(DATE)

all: lint test build

build:
	go build -ldflags "$(LDFLAGS)" -o jara ./cmd/jara

lint:
	golangci-lint run ./...

test:
	go test -race ./...

test-integration:
	go test -race -tags integration ./...

fmt:
	gofumpt -w .

vet:
	go vet ./...

tidy:
	go mod tidy
